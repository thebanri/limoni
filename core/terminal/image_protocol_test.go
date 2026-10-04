package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
)

// The environment cannot see through SSH or tmux; what the terminal says
// about itself can. Only a half-block guess is upgraded, and LIMONI_GRAPHICS
// is never overruled.
func TestWithReportChoosesTheImageProtocol(t *testing.T) {
	half := CapabilityProfile{GraphicsProto: graphics.ProtocolHalfBlock}
	cases := []struct {
		name   string
		from   CapabilityProfile
		report driver.TerminalReport
		env    string
		want   graphics.Protocol
	}{
		{"DA1 lists sixel", half, driver.TerminalReport{Answered: true, Sixel: true}, "", graphics.ProtocolSixel},
		{"no sixel in DA1", half, driver.TerminalReport{Answered: true}, "", graphics.ProtocolHalfBlock},
		{"kitty by name over ssh", half, driver.TerminalReport{Answered: true, Name: "kitty"}, "", graphics.ProtocolKitty},
		{"ghostty by name", half, driver.TerminalReport{Answered: true, Name: "ghostty"}, "", graphics.ProtocolKitty},
		{"iterm2 by name", half, driver.TerminalReport{Answered: true, Name: "iTerm2"}, "", graphics.ProtocolIterm2},
		{"a name beats DA1", half, driver.TerminalReport{Answered: true, Name: "wezterm", Sixel: true}, "", graphics.ProtocolKitty},
		{"an environment guess stays", CapabilityProfile{GraphicsProto: graphics.ProtocolKitty}, driver.TerminalReport{Answered: true, Sixel: true}, "", graphics.ProtocolKitty},
		{"LIMONI_GRAPHICS wins", half, driver.TerminalReport{Answered: true, Sixel: true, Name: "kitty"}, "halfblock", graphics.ProtocolHalfBlock},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("LIMONI_GRAPHICS", c.env)
			if got := c.from.WithReport(c.report).GraphicsProto; got != c.want {
				t.Errorf("GraphicsProto = %d, want %d", got, c.want)
			}
		})
	}
}
