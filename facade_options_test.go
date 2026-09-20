package limoni

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/testkit"
)

// The options are the public way to configure an application, and each one
// only sets a field — which is exactly the kind of wiring that goes wrong
// silently. Reading the config back is the whole test.
func configOf(opts ...AppOption) appConfig {
	var cfg appConfig
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

func TestAppOptionsSetWhatTheyName(t *testing.T) {
	if cfg := configOf(WithFPS(30)); cfg.fps != 30 {
		t.Errorf("WithFPS(30) → fps %d", cfg.fps)
	}
	if cfg := configOf(WithSuspend()); !cfg.suspendOnCtrlZ {
		t.Error("WithSuspend did not enable Ctrl+Z")
	}
	if cfg := configOf(WithInline(7)); cfg.inlineHeight != 7 {
		t.Errorf("WithInline(7) → height %d", cfg.inlineHeight)
	}
	if cfg := configOf(WithTitle("zest")); !cfg.hasTitle || cfg.title != "zest" {
		t.Errorf("WithTitle → %q (set: %v)", cfg.title, cfg.hasTitle)
	}
	if cfg := configOf(WithCatchCtrlC(true)); !cfg.catchCtrlC {
		t.Error("WithCatchCtrlC(true) did not take")
	}
	if cfg := configOf(WithoutDefaultQuitKeys()); !cfg.catchCtrlC {
		t.Error("WithoutDefaultQuitKeys should hand Ctrl+C to the application")
	}

	policy := AutomationPolicy{ExposeScreen: true, AllowInput: true}
	cfg := configOf(WithAutomation("/tmp/x.sock", policy))
	if cfg.automationPath != "/tmp/x.sock" || cfg.automationPolicy != policy {
		t.Errorf("WithAutomation → %q %+v", cfg.automationPath, cfg.automationPolicy)
	}

	// Later options win, and a nil option is ignored rather than panicking.
	if cfg := configOf(WithFPS(10), nil, WithFPS(60)); cfg.fps != 60 {
		t.Errorf("the last option should win: fps %d", cfg.fps)
	}
}

// The state constructors are what an application holds between frames. A
// wrong zero value here shows up as a widget that will not scroll or select.
func TestStateConstructorsStartUsable(t *testing.T) {
	// Selected starts at -1, meaning nothing is selected — a fresh list must
	// not act as though its first row had been chosen.
	if s := NewListState(); s == nil || s.Selected != -1 || s.Offset != 0 {
		t.Errorf("NewListState = %+v", s)
	}
	if s := NewTableState(); s == nil || s.Selected != -1 || s.SortColumn != -1 {
		t.Errorf("NewTableState = %+v", s)
	}
	if s := NewTextInputState(); s == nil || len(s.Text) != 0 || s.Cursor != 0 {
		t.Errorf("NewTextInputState = %+v", s)
	}
	if s := NewTextAreaState(); s == nil {
		t.Error("NewTextAreaState = nil")
	}
	if s := NewSelectState(); s == nil {
		t.Error("NewSelectState = nil")
	}
	if s := NewSliderState(42); s == nil || s.Value != 42 {
		t.Errorf("NewSliderState(42) = %+v", s)
	}
	if s := NewViewportState(); s == nil {
		t.Error("NewViewportState = nil")
	}
}

// Text builders: the pieces a rich line is assembled from.
func TestTextBuildersRenderTheirContent(t *testing.T) {
	line := NewLine(NewSpan("red", Fg(ColorRed)), NewSpan(" plain", NewStyle()))
	text := NewText(line)

	term := testkit.NewTerminal(20, 2)
	term.Render(text, term.Area())
	got := strings.TrimSpace(strings.Split(term.Snapshot(), "\n")[0])
	if got != "red plain" {
		t.Errorf("rendered %q, want %q", got, "red plain")
	}
}

func TestScrollbarMetricsGrowAndClamp(t *testing.T) {
	const track = 10

	// Half the content visible: the thumb takes half the track.
	start, length := ScrollbarMetrics(20, 10, 0, track)
	if length != 5 {
		t.Errorf("thumb length %d for half the content, want 5", length)
	}
	if start != 0 {
		t.Errorf("at the top the thumb starts at 0, got %d", start)
	}

	// At the bottom it sits against the end of the track and no further.
	start, length = ScrollbarMetrics(20, 10, 10, track)
	if start+length != track {
		t.Errorf("thumb %d at %d does not reach the end of a %d-row track", length, start, track)
	}

	// An offset past the end is clamped, not extrapolated.
	if s2, _ := ScrollbarMetrics(20, 10, 999, track); s2 != start {
		t.Errorf("an over-scrolled offset moved the thumb to %d, want %d", s2, start)
	}

	// Content that fits needs no thumb at all.
	if start, length = ScrollbarMetrics(5, 10, 0, track); start != 0 || length != 0 {
		t.Errorf("content that fits gave a thumb of %d at %d", length, start)
	}

	// A thumb never disappears while there is something to scroll.
	if _, length = ScrollbarMetrics(10000, 10, 0, track); length < 1 {
		t.Error("a very long document left no thumb to grab")
	}
}

// The style helpers each produce exactly their own modifier and nothing else,
// which is the one way a facade like this goes wrong.
func TestStyleHelpersEachSetOnlyTheirOwnBit(t *testing.T) {
	cases := []struct {
		name string
		got  Style
		want Modifier
	}{
		{"Bold", Bold(), ModifierBold},
		{"Italic", Italic(), ModifierItalic},
		{"Underline", Underline(), ModifierUnderline},
		{"Dim", Dim(), ModifierDim},
		{"Reverse", Reverse(), ModifierReverse},
	}
	for _, c := range cases {
		if c.got.Modifier != c.want {
			t.Errorf("%s() set %v, want exactly %v", c.name, c.got.Modifier, c.want)
		}
		if c.got.Fg != ColorDefault || c.got.Bg != ColorDefault {
			t.Errorf("%s() also set a colour: %+v", c.name, c.got)
		}
	}

	if got := Fg(ColorRed); got.Fg != ColorRed || got.Bg != ColorDefault {
		t.Errorf("Fg set %+v", got)
	}
	if got := Bg(ColorBlue); got.Bg != ColorBlue || got.Fg != ColorDefault {
		t.Errorf("Bg set %+v", got)
	}
	if got := ANSI(200); got != ANSI(200) || got.ANSI() != 200 {
		t.Errorf("ANSI(200) = %v", got)
	}
	if got := Hyperlink("https://example.com/facade"); got.LinkURL() != "https://example.com/facade" {
		t.Errorf("Hyperlink → %q", got.LinkURL())
	}
	// Hex tolerates the forms people actually write, and refuses the rest.
	for _, bad := range []string{"", "#12", "nope", "#12345"} {
		if got := Hex(bad); got != ColorDefault {
			t.Errorf("Hex(%q) = %v, want the default colour", bad, got)
		}
	}
	if got := Hex("00FF00"); got != RGB(0, 255, 0) {
		t.Errorf("Hex without '#' = %v", got)
	}
}
