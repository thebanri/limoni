package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thebanri/limoni/automation"
	"github.com/thebanri/limoni/core/accessibility"
)

// tool is one MCP tool: its advertised shape and the function behind it.
type tool struct {
	name        string
	title       string
	description string
	// properties and required make up the JSON Schema of the arguments.
	properties map[string]any
	required   []string
	readOnly   bool
	run        func(ctx context.Context, args json.RawMessage) (string, error)
}

func (t tool) descriptor() map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           t.properties,
		"additionalProperties": false,
	}
	if len(t.required) > 0 {
		schema["required"] = t.required
	}
	if t.properties == nil {
		schema["properties"] = map[string]any{}
	}
	return map[string]any{
		"name":        t.name,
		"title":       t.title,
		"description": t.description,
		"inputSchema": schema,
		// Hints for the client's confirmation UI. Input tools are marked
		// destructive because what a click does is up to the application: it
		// may delete a file as easily as open a menu.
		"annotations": map[string]any{
			"title":           t.title,
			"readOnlyHint":    t.readOnly,
			"destructiveHint": !t.readOnly,
			"idempotentHint":  t.readOnly,
			"openWorldHint":   false,
		},
	}
}

const serverInstructions = `These tools drive a running terminal application built with Limoni through its semantic tree — the same tree a screen reader uses. Address widgets by what they are, never by screen position.

Start with tree to see what is on screen. Each line is: role#id "label" value=... position=n/m state=... bounds=x,y WxH, children indented.

Select a widget with any combination of id, role, label, label_contains and value; every field given must match. A selector that matches several widgets is refused for click — add a field, or nth to pick one. click, press_key and type_text return the tree after the application has redrawn, so there is usually no need to call tree again. Use wait_for rather than retrying when something appears asynchronously.

Keyboard input goes to the focused widget; check which one has state=focused before typing, and use press_key tab or click to move focus.

The application decides what leaves the process. Secret fields never show a value; other input values, the screen text and input itself may be disabled by its policy — status reports which, and a refused call says why.`

var roleNames = []string{
	"button", "checkbox", "input", "list", "list-item", "table", "dialog",
	"progress", "image", "radio-button", "slider", "tree", "tree-item",
	"row", "cell", "tab-list", "tab", "generic",
}

// selectorProperties is the schema shared by every tool that takes a selector.
// The fields sit at the top level of the arguments rather than in a nested
// object, which is how models most reliably fill them in.
func selectorProperties(extra map[string]any) map[string]any {
	props := map[string]any{
		"id":             map[string]any{"type": "string", "description": "The widget's ID, exactly as shown after # in the tree."},
		"role":           map[string]any{"type": "string", "enum": roleNames, "description": "The semantic role."},
		"label":          map[string]any{"type": "string", "description": "The label, matched exactly."},
		"label_contains": map[string]any{"type": "string", "description": "A substring of the label, for labels that carry live data."},
		"value":          map[string]any{"type": "string", "description": "The current value, matched exactly — for example the selected item of a list."},
		"nth":            map[string]any{"type": "integer", "description": "Which match to use when several qualify, 0-based; negative counts from the end."},
	}
	for k, v := range extra {
		props[k] = v
	}
	return props
}

type selectorArgs struct {
	ID            string `json:"id"`
	Role          string `json:"role"`
	Label         string `json:"label"`
	LabelContains string `json:"label_contains"`
	Value         string `json:"value"`
	Nth           *int   `json:"nth"`
}

func (s selectorArgs) selector() automation.Selector {
	return automation.Selector{ID: s.ID, Role: s.Role, Label: s.Label, LabelContains: s.LabelContains, Value: s.Value, Nth: s.Nth}
}

// decodeArgs rejects unknown fields. A misspelled selector field would
// otherwise vanish silently and widen the selector to something else.
func decodeArgs(raw json.RawMessage, into any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("invalid arguments: %v", err)
	}
	return nil
}

func newTools(a *app) []tool {
	return []tool{
		{
			name:        "status",
			title:       "Application status",
			description: "Reports whether the application is reachable, its screen size, the focused widget, and what its automation policy allows: input, screen text, and input values.",
			readOnly:    true,
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				if err := decodeArgs(raw, &struct{}{}); err != nil {
					return "", err
				}
				hello, err := a.do(ctx, automation.Request{Op: automation.OpHello}, true)
				if err != nil {
					return "", err
				}
				focused, err := a.do(ctx, automation.Request{Op: automation.OpFocused}, true)
				if err != nil {
					return "", err
				}
				focus := focused.Focused
				if focus == "" {
					focus = "(nothing)"
				}
				return fmt.Sprintf("connected to %s\nprotocol version %d\nscreen %dx%d\nfocused %s\ninput allowed: %t\nscreen text exposed: %t\ninput values exposed: %t",
					a.socket, hello.Version, hello.Width, hello.Height, focus,
					hello.AllowInput, hello.ExposeScreen, hello.ExposeInputValues), nil
			},
		},
		{
			name:        "tree",
			title:       "Semantic tree",
			description: "Returns the semantic tree of the current frame: every widget's role, ID, label, value, state and bounds. Start here.",
			properties: map[string]any{
				"format": map[string]any{"type": "string", "enum": []string{"outline", "json"}, "description": "outline (default) is one widget per line; json is structured."},
			},
			readOnly: true,
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				var args struct {
					Format string `json:"format"`
				}
				if err := decodeArgs(raw, &args); err != nil {
					return "", err
				}
				nodes, err := a.tree(ctx)
				if err != nil {
					return "", err
				}
				switch args.Format {
				case "", "outline":
					return outline(nodes), nil
				case "json":
					return treeJSON(nodes)
				default:
					return "", fmt.Errorf("unknown format %q: use outline or json", args.Format)
				}
			},
		},
		{
			name:        "find",
			title:       "Find widgets",
			description: "Returns every widget matching a selector, with its subtree. An empty result is not an error.",
			properties:  selectorProperties(nil),
			readOnly:    true,
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				var args selectorArgs
				if err := decodeArgs(raw, &args); err != nil {
					return "", err
				}
				resp, err := a.do(ctx, automation.Request{Op: automation.OpFind, Selector: args.selector()}, true)
				if err != nil {
					return "", err
				}
				if len(resp.Nodes) == 0 {
					return fmt.Sprintf("nothing matches %s", args.selector()), nil
				}
				return fmt.Sprintf("%d match(es) for %s:\n%s", len(resp.Nodes), args.selector(), outline(resp.Nodes)), nil
			},
		},
		{
			name:        "screen",
			title:       "Screen text",
			description: "Returns the rendered character grid. Prefer tree; use this only for what the tree cannot express, such as chart contents or layout. Disabled unless the application's policy exposes the screen.",
			readOnly:    true,
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				if err := decodeArgs(raw, &struct{}{}); err != nil {
					return "", err
				}
				resp, err := a.do(ctx, automation.Request{Op: automation.OpSnapshot}, true)
				return resp.Snapshot, err
			},
		},
		{
			name:        "click",
			title:       "Click a widget",
			description: "Clicks the centre of the one widget the selector matches, then returns the tree after the application redraws. Refused if the selector matches more than one widget. " +
				"Clicking toggles a checkbox; to make a step safe to repeat, pass ensure: checked, unchecked or selected, and the widget is clicked only if it is not already in that state.",
			properties: selectorProperties(map[string]any{
				"ensure": map[string]any{"type": "string", "enum": []string{"checked", "unchecked", "selected"},
					"description": "Click only if the widget is not already in this state, then confirm it reached it. Makes the call idempotent."},
			}),
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				var args struct {
					selectorArgs
					Ensure string `json:"ensure"`
				}
				if err := decodeArgs(raw, &args); err != nil {
					return "", err
				}
				sel := args.selector()
				if sel.IsEmpty() {
					return "", fmt.Errorf("click needs a selector: give at least one of id, role, label, label_contains, value")
				}
				var inState func(accessibility.NodeState) bool
				switch args.Ensure {
				case "":
				case "checked":
					inState = func(s accessibility.NodeState) bool { return s&accessibility.StateChecked != 0 }
				case "unchecked":
					inState = func(s accessibility.NodeState) bool { return s&accessibility.StateChecked == 0 }
				case "selected":
					inState = func(s accessibility.NodeState) bool { return s&accessibility.StateSelected != 0 }
				default:
					return "", fmt.Errorf("ensure must be checked, unchecked or selected, not %q", args.Ensure)
				}
				if inState != nil {
					found, err := a.do(ctx, automation.Request{Op: automation.OpFind, Selector: sel}, true)
					if err != nil {
						return "", err
					}
					if len(found.Nodes) == 1 && inState(found.Nodes[0].State) {
						tree, err := a.tree(ctx)
						if err != nil {
							return "", err
						}
						return fmt.Sprintf("%s is already %s; nothing was clicked.\n\n%s", describe(found.Nodes[0]), args.Ensure, outline(tree)), nil
					}
				}
				resp, nodes, changed, err := a.act(ctx, automation.Request{Op: automation.OpClick, Selector: sel})
				target := sel.String()
				if len(resp.Nodes) == 1 {
					target = describe(resp.Nodes[0])
				}
				what := "clicked " + target
				if inState != nil && err == nil {
					if after, ferr := a.do(ctx, automation.Request{Op: automation.OpFind, Selector: sel}, true); ferr == nil && len(after.Nodes) == 1 && !inState(after.Nodes[0].State) {
						what += fmt.Sprintf(", but it is still not %s — the click may have gone to something else, or the widget does not change state on click", args.Ensure)
					}
				}
				return a.afterAction(ctx, what, nodes, changed, err)
			},
		},
		{
			name:  "press_key",
			title: "Press a key",
			description: "Sends one key press to the focused widget, then returns the tree after the application redraws. " +
				"key is a single character or one of: enter, tab, esc, space, backspace, delete, insert, up, down, left, right, home, end, pgup, pgdn, f1–f12.",
			properties: map[string]any{
				"key":   map[string]any{"type": "string", "description": "A single character, or a key name such as enter, tab, esc, up, down."},
				"ctrl":  map[string]any{"type": "boolean"},
				"alt":   map[string]any{"type": "boolean"},
				"shift": map[string]any{"type": "boolean"},
			},
			required: []string{"key"},
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				var args struct {
					Key   string `json:"key"`
					Ctrl  bool   `json:"ctrl"`
					Alt   bool   `json:"alt"`
					Shift bool   `json:"shift"`
				}
				if err := decodeArgs(raw, &args); err != nil {
					return "", err
				}
				_, nodes, changed, err := a.act(ctx, automation.Request{Op: automation.OpKey, Key: args.Key, Ctrl: args.Ctrl, Alt: args.Alt, Shift: args.Shift})
				name := args.Key
				for _, mod := range []struct {
					on   bool
					name string
				}{{args.Shift, "shift"}, {args.Alt, "alt"}, {args.Ctrl, "ctrl"}} {
					if mod.on {
						name = mod.name + "+" + name
					}
				}
				return a.afterAction(ctx, "pressed "+name, nodes, changed, err)
			},
		},
		{
			name:        "type_text",
			title:       "Type text",
			description: "Types text into the focused widget as one key press per character, then returns the tree after the application redraws. Does not press enter.",
			properties: map[string]any{
				"text": map[string]any{"type": "string", "description": "The text to type."},
			},
			required: []string{"text"},
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				var args struct {
					Text string `json:"text"`
				}
				if err := decodeArgs(raw, &args); err != nil {
					return "", err
				}
				_, nodes, changed, err := a.act(ctx, automation.Request{Op: automation.OpText, Text: args.Text})
				return a.afterAction(ctx, fmt.Sprintf("typed %d character(s)", len([]rune(args.Text))), nodes, changed, err)
			},
		},
		{
			name:        "wait_for",
			title:       "Wait for a widget",
			description: "Waits until a widget matching the selector is on screen, and returns the matches. Use this instead of polling tree when something appears asynchronously.",
			properties: selectorProperties(map[string]any{
				"timeout_ms": map[string]any{"type": "integer", "minimum": 1, "maximum": 60000, "description": "How long to wait, in milliseconds. Defaults to 5000."},
			}),
			readOnly: true,
			run: func(ctx context.Context, raw json.RawMessage) (string, error) {
				var args struct {
					selectorArgs
					TimeoutMS int `json:"timeout_ms"`
				}
				if err := decodeArgs(raw, &args); err != nil {
					return "", err
				}
				sel := args.selector()
				if sel.IsEmpty() {
					return "", fmt.Errorf("wait_for needs a selector: give at least one of id, role, label, label_contains, value")
				}
				timeout := 5 * time.Second
				if args.TimeoutMS > 0 {
					timeout = time.Duration(min(args.TimeoutMS, 60000)) * time.Millisecond
				}
				nodes, err := a.waitFor(ctx, sel, timeout)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%d match(es) for %s:\n%s", len(nodes), sel, outline(nodes)), nil
			},
		},
	}
}

func (a *app) afterAction(ctx context.Context, what string, nodes []accessibility.AccessibilityNode, changed bool, err error) (string, error) {
	if errors.Is(err, errAppGone) {
		return what + "; the application then closed its automation socket — it has most likely exited.", nil
	}
	if err != nil {
		return "", err
	}
	note := "the application redrew"
	if !changed {
		note = fmt.Sprintf("the tree did not change within %s — the input may have had no visible effect, or went to a different widget than intended", a.settle)
		// The likeliest reasons for typed text not showing up are that the
		// field is secret or that the policy withholds input values. An agent
		// that does not know either will type the same text again.
		if focusedSensitive(nodes) {
			note += ". The focused field is secret, so what is typed into it is never shown"
		} else if hello, err := a.do(ctx, automation.Request{Op: automation.OpHello}, true); err == nil && !hello.ExposeInputValues {
			note += ". Note that the application's policy hides input values (ExposeInputValues is off), so text typed into an input never appears in the tree"
		}
	}
	return fmt.Sprintf("%s; %s.\n\n%s", what, note, outline(nodes)), nil
}

func focusedSensitive(nodes []accessibility.AccessibilityNode) bool {
	for _, n := range nodes {
		if n.State&accessibility.StateFocused != 0 && n.State&accessibility.StateSensitive != 0 {
			return true
		}
		if focusedSensitive(n.Children) {
			return true
		}
	}
	return false
}

func outline(nodes []accessibility.AccessibilityNode) string {
	if len(nodes) == 0 {
		return "(the application has not rendered any semantic nodes)"
	}
	return accessibility.Mode{}.LineMode(nodes)
}

func describe(node accessibility.AccessibilityNode) string {
	var b strings.Builder
	b.WriteString(node.Role.String())
	if node.ID != "" {
		b.WriteString("#" + node.ID)
	}
	if node.Label != "" {
		fmt.Fprintf(&b, " %q", node.Label)
	}
	return b.String()
}

// jsonNode is the structured form of a tree node. The in-memory node carries
// the role as a number and the state as a bit set, which are meaningless to a
// reader outside the process.
type jsonNode struct {
	ID          string     `json:"id,omitempty"`
	Role        string     `json:"role"`
	Label       string     `json:"label,omitempty"`
	Value       string     `json:"value,omitempty"`
	Description string     `json:"description,omitempty"`
	States      []string   `json:"states,omitempty"`
	Position    int        `json:"position,omitempty"`
	SetSize     int        `json:"set_size,omitempty"`
	Bounds      jsonRect   `json:"bounds"`
	Children    []jsonNode `json:"children,omitempty"`
}

type jsonRect struct {
	X      uint16 `json:"x"`
	Y      uint16 `json:"y"`
	Width  uint16 `json:"width"`
	Height uint16 `json:"height"`
}

func toJSONNodes(nodes []accessibility.AccessibilityNode) []jsonNode {
	out := make([]jsonNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, jsonNode{
			ID: n.ID, Role: n.Role.String(), Label: n.Label, Value: n.Value, Description: n.Description,
			States: n.State.StateNames(), Position: n.Position, SetSize: n.SetSize,
			Bounds:   jsonRect{n.Bounds.X, n.Bounds.Y, n.Bounds.Width, n.Bounds.Height},
			Children: toJSONNodes(n.Children),
		})
	}
	return out
}

func treeJSON(nodes []accessibility.AccessibilityNode) (string, error) {
	data, err := json.MarshalIndent(toJSONNodes(nodes), "", "  ")
	return string(data), err
}
