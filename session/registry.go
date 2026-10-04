package session

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/thebanri/limoni/core/engine"
)

// A message is an interface value, so writing one to a file loses its type.
// The registry maps a stable name back to a concrete type, which is what lets
// a replay hand Update the same Go type the original run did.
//
// Registration is also the privacy boundary for application messages: a type
// that is not registered is recorded by name only, never by content. An
// application's own messages can carry anything — a login form's password, an
// API response — and the recorder has no way to know, so it records nothing it
// was not explicitly given permission to.

var (
	registryMu sync.RWMutex
	registry   = map[string]registeredType{}
)

type registeredType struct {
	typ    reflect.Type
	redact func(engine.Msg) engine.Msg
}

// Register allows messages of type T to be recorded in full and replayed.
//
// T must round-trip through encoding/json: exported fields only. A field JSON
// cannot carry comes back as its zero value, and the replay reports the
// resulting divergence rather than hiding it.
func Register[T any]() {
	registerType(reflect.TypeFor[T](), nil)
}

// RegisterRedacted is Register with a function applied before the message is
// written, for a type that is needed for replay but carries something that
// must not reach disk.
func RegisterRedacted[T any](redact func(T) T) {
	registerType(reflect.TypeFor[T](), func(msg engine.Msg) engine.Msg {
		return redact(msg.(T))
	})
}

func registerType(t reflect.Type, redact func(engine.Msg) engine.Msg) {
	name := typeName(t)
	if name == "" {
		panic(fmt.Sprintf("session: cannot register unnamed type %v; declare a named type for the message", t))
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = registeredType{typ: t, redact: redact}
}

// typeName is the package path and name of a type, with a leading star per
// level of pointer. Unnamed composite types have no stable name and return "".
func typeName(t reflect.Type) string {
	prefix := ""
	for t.Kind() == reflect.Pointer {
		prefix += "*"
		t = t.Elem()
	}
	if t.Name() == "" {
		return ""
	}
	if t.PkgPath() == "" {
		return prefix + t.Name()
	}
	return prefix + t.PkgPath() + "." + t.Name()
}

func lookup(name string) (registeredType, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	rt, ok := registry[name]
	return rt, ok
}

// decode rebuilds a message from its recorded name and JSON body.
func decode(name string, data json.RawMessage) (engine.Msg, error) {
	if name == nilMessage {
		return nil, nil
	}
	rt, ok := lookup(name)
	if !ok {
		return nil, fmt.Errorf("session: message type %s is not registered; call session.Register for it before replaying", name)
	}
	target := rt.typ
	depth := 0
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
		depth++
	}
	value := reflect.New(target)
	if len(data) > 0 {
		if err := json.Unmarshal(data, value.Interface()); err != nil {
			return nil, fmt.Errorf("session: decode %s: %w", name, err)
		}
	}
	// value is *target. Peel or add pointer levels to match the recorded type.
	result := value.Elem()
	for i := 0; i < depth; i++ {
		ptr := reflect.New(result.Type())
		ptr.Elem().Set(result)
		result = ptr
	}
	return result.Interface(), nil
}

const nilMessage = "<nil>"

func init() {
	for _, t := range []reflect.Type{
		reflect.TypeFor[engine.KeyPressMsg](),
		reflect.TypeFor[engine.KeyReleaseMsg](),
		reflect.TypeFor[engine.MousePressMsg](),
		reflect.TypeFor[engine.MouseReleaseMsg](),
		reflect.TypeFor[engine.MouseMotionMsg](),
		reflect.TypeFor[engine.MouseWheelMsg](),
		reflect.TypeFor[engine.PasteMsg](),
		reflect.TypeFor[engine.ResizeMsg](),
		reflect.TypeFor[engine.FocusMsg](),
		reflect.TypeFor[engine.BlurMsg](),
		reflect.TypeFor[engine.TimeMsg](),
	} {
		registerType(t, nil)
	}
}
