package accessibility

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

func TestRedact(t *testing.T) {
	secret := Node{ID: "pw", Role: RoleInput, Label: "Password", Value: "hunter2", Description: "hint", State: StateSensitive}
	field := Node{ID: "name", Role: RoleInput, Label: "Name", Value: "Ada", Description: "Full name"}
	slider := Node{ID: "vol", Role: RoleSlider, Label: "Volume", Value: "40"}

	tests := []struct {
		name         string
		node         Node
		exposeInputs bool
		wantValue    string
		wantDesc     string
	}{
		{name: "sensitive field", node: secret, wantValue: "", wantDesc: ""},
		{name: "sensitive field with inputs exposed", node: secret, exposeInputs: true, wantValue: "", wantDesc: ""},
		{name: "sensitive field outside the input role", node: Node{Role: RoleSlider, Value: "1", State: StateSensitive}, wantValue: ""},
		{name: "input", node: field, wantValue: "", wantDesc: "Full name"},
		{name: "input with inputs exposed", node: field, exposeInputs: true, wantValue: "Ada", wantDesc: "Full name"},
		{name: "input with no value", node: Node{Role: RoleInput, Label: "Name"}, exposeInputs: true, wantValue: ""},
		{name: "not an input", node: slider, wantValue: "40"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Redact([]Node{tt.node}, tt.exposeInputs)[0]
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
			if got.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDesc)
			}
			if got.Label != tt.node.Label {
				t.Errorf("Label = %q, want %q", got.Label, tt.node.Label)
			}
		})
	}
}

func TestRedactNilTree(t *testing.T) {
	if got := Redact(nil, false); got != nil {
		t.Errorf("Redact(nil) = %#v, want nil", got)
	}
}

func TestRedactReachesChildren(t *testing.T) {
	tree := []Node{{Role: RoleDialog, Children: []Node{
		{Role: RoleGeneric, Children: []Node{{Role: RoleInput, Value: "deep", State: StateSensitive}}},
	}}}

	got := Redact(tree, true)

	if value := got[0].Children[0].Children[0].Value; value != "" {
		t.Errorf("nested sensitive value = %q, want it removed", value)
	}
}

func TestRedactLeavesInputTreeAlone(t *testing.T) {
	build := func() []Node {
		return []Node{{Role: RoleDialog, Children: []Node{
			{Role: RoleInput, Value: "secret", Description: "hint", State: StateSensitive},
		}}}
	}
	tree := build()

	Redact(tree, false)

	if !reflect.DeepEqual(tree, build()) {
		t.Errorf("Redact changed its input: %#v", tree)
	}
}

func TestModeSwitches(t *testing.T) {
	tests := []struct {
		name        string
		mode        Mode
		wantAnimate bool
		wantMouse   bool
	}{
		{name: "defaults", mode: Mode{}, wantAnimate: true, wantMouse: true},
		{name: "reduced motion", mode: Mode{ReducedMotion: true}, wantAnimate: false, wantMouse: true},
		{name: "no mouse", mode: Mode{NoMouse: true}, wantAnimate: true, wantMouse: false},
		{name: "screen reader turns the mouse off", mode: Mode{ScreenReader: true}, wantAnimate: true, wantMouse: false},
		{name: "visual settings change neither", mode: Mode{HighContrast: true, NoColor: true, ASCIIOnly: true}, wantAnimate: true, wantMouse: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.ShouldAnimate(); got != tt.wantAnimate {
				t.Errorf("ShouldAnimate() = %v, want %v", got, tt.wantAnimate)
			}
			if got := tt.mode.AllowsMouse(); got != tt.wantMouse {
				t.Errorf("AllowsMouse() = %v, want %v", got, tt.wantMouse)
			}
		})
	}
}

// With no PATH the speech tools cannot start, so this reports the failure
// without the test machine reading anything aloud.
func TestPlatformAnnounceReportsMissingTool(t *testing.T) {
	t.Setenv("PATH", "")

	adapters := []struct {
		name    string
		adapter ScreenReaderAdapter
	}{
		{name: "linux", adapter: LinuxScreenReaderAdapter{}},
		{name: "darwin", adapter: MacOSScreenReaderAdapter{}},
		{name: "windows", adapter: WindowsScreenReaderAdapter{}},
	}
	for _, tt := range adapters {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.adapter.Announce("it's saved"); !errors.Is(err, exec.ErrNotFound) {
				t.Errorf("Announce() error = %v, want %v", err, exec.ErrNotFound)
			}
		})
	}
}

func TestRoleStringNamesEveryRole(t *testing.T) {
	want := map[Role]string{
		RoleGeneric:     "generic",
		RoleButton:      "button",
		RoleCheckbox:    "checkbox",
		RoleInput:       "input",
		RoleList:        "list",
		RoleListItem:    "list-item",
		RoleTable:       "table",
		RoleDialog:      "dialog",
		RoleProgress:    "progress",
		RoleImage:       "image",
		RoleRadioButton: "radio-button",
		RoleSlider:      "slider",
		RoleTree:        "tree",
		RoleTreeItem:    "tree-item",
		RoleRow:         "row",
		RoleCell:        "cell",
		RoleTabList:     "tab-list",
		RoleTab:         "tab",
	}
	for role, name := range want {
		if got := role.String(); got != name {
			t.Errorf("Role(%d).String() = %q, want %q", role, got, name)
		}
	}
}
