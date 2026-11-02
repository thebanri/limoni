package main

import (
	"testing"
	"time"
)

// fake starts a racer whose agent runs script (sh) and then waits for a line.
func fake(t *testing.T, script string) *racer {
	t.Helper()
	agent := agentDef{Name: "fake", Bin: "sh", Args: func(string, string) []string {
		return []string{"-c", script + "; read line", "sh"}
	}}
	r, err := startRacer(agent, "fake", "", t.TempDir(), "b", 40, 6, func() {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.stop)
	return r
}

func TestWaitingSignals(t *testing.T) {
	for _, c := range []struct {
		name, script string
		want         state
		text         string
	}{
		{"bare bell", `printf 'hi\007'`, waiting, ""},
		{"bell inside OSC", `printf '\033]0;title\007\033]2;x\033\\'`, working, ""},
		{"OSC 9", `printf '\033]9;needs you\007'`, waiting, "needs you"},
		{"OSC 9 with ST", `printf '\033]9;st\033\\'`, waiting, "st"},
		{"OSC 9;4 progress", `printf '\033]9;4;1;50\007'`, working, ""},
		{"OSC 777", `printf '\033]777;notify;Codex;done\033\\'`, waiting, "Codex: done"},
		{"OSC 99", `printf '\033]99;i=1:d=0;Claude is done\033\\'`, waiting, "Claude is done"},
		{"gemini title", `printf '\033]0;✋ Action Required\007'`, waiting, ""},
		{"bell in DCS", `printf '\033Pq\007\033\\'`, working, ""},
		{"split OSC", `printf '\033]'; sleep .1; printf '9;sp'; sleep .1; printf 'lit\033'; sleep .1; printf '\\'`, waiting, "split"},
		{"split inside OSC then BEL", `printf '\033]0;a'; sleep .1; printf 'b\007'`, working, ""},
		{"silent thinking", `printf 'thinking'; sleep 3`, working, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := fake(t, c.script)
			time.Sleep(700 * time.Millisecond)
			if c.name == "silent thinking" {
				time.Sleep(2500 * time.Millisecond) // past the old two seconds
			}
			if got := r.state(time.Now()); got != c.want {
				t.Errorf("state = %v, want %v", got, c.want)
			}
			if c.want == waiting && c.text != "" && r.ask() != c.text {
				t.Errorf("ask = %q, want %q", r.ask(), c.text)
			}
		})
	}
}

func TestUserInputClearsWaiting(t *testing.T) {
	r := fake(t, `printf 'x\007'`)
	time.Sleep(500 * time.Millisecond)
	if r.state(time.Now()) != waiting {
		t.Fatal("bell did not make it wait")
	}
	r.userInput()
	if r.state(time.Now()) != working || r.ask() != "" {
		t.Error("input did not clear waiting")
	}
}
