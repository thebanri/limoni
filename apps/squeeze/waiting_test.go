package main

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
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
	r.userInput(true)
	if r.state(time.Now()) != working || r.ask() != "" {
		t.Error("input did not clear waiting")
	}
}

// The clock counts working time: it stops when the agent waits, from the
// last thing it printed, and goes on when it works again.
func TestTheClockStopsWhileWaiting(t *testing.T) {
	r := fake(t, `printf 'x\007'`)
	u := newUI(t.TempDir(), "", "task", nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	u.racers = []*racer{r}
	time.Sleep(300 * time.Millisecond)
	u.stopped(time.Now())
	first := r.elapsed(time.Now())
	if first > 200*time.Millisecond {
		t.Errorf("worked %v, want about the time to print one character", first)
	}
	time.Sleep(300 * time.Millisecond)
	if got := r.elapsed(time.Now()); got != first {
		t.Errorf("the clock went on while waiting: %v, then %v", first, got)
	}
	r.userInput(true)
	u.stopped(time.Now())
	time.Sleep(200 * time.Millisecond)
	if got := r.elapsed(time.Now()); got < first+150*time.Millisecond {
		t.Errorf("the clock did not go on after input: %v", got)
	}
}

// Typing to an agent that waits is not the agent working, though the
// terminal echoes every key; Enter sets it working.
func TestTypingIsNotWork(t *testing.T) {
	r := fake(t, `printf 'x\007'`)
	time.Sleep(1200 * time.Millisecond) // past the second in which a printed screen is the ask's own
	for _, ch := range "hello" {
		k, _ := toEmulator(driver.KeyForRune(ch))
		r.emu.SendKey(k)
		r.userInput(false)
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(400 * time.Millisecond)
	if got := r.state(time.Now()); got != waiting {
		t.Errorf("after typing: %v, want waiting", got)
	}
	k, _ := toEmulator(driver.KeyEvent{Type: driver.KeyEnter})
	r.emu.SendKey(k)
	r.userInput(true)
	if got := r.state(time.Now()); got != working {
		t.Errorf("after Enter: %v, want working", got)
	}
}

// Output does not answer an agent that asked for the user: agents redraw
// while they wait. Its own word that it works again (squeeze's hooks) does.
func TestOnlyTheAgentsWordEndsAnAsk(t *testing.T) {
	r := fake(t, `printf 'x\007'; sleep .5; printf 'a suggestion'; sleep .5; printf '\033]9;squeeze:working\007'`)
	time.Sleep(800 * time.Millisecond)
	if got := r.state(time.Now()); got != waiting {
		t.Errorf("after a redraw: %v, want waiting", got)
	}
	time.Sleep(600 * time.Millisecond)
	if got := r.state(time.Now()); got != working {
		t.Errorf("after the working mark: %v, want working", got)
	}
	if r.ask() != "" {
		t.Errorf("ask = %q after the working mark", r.ask())
	}
}

// A hooked agent that waits stays waiting through typing, Enter and clicks
// (a click into its screen to start typing used to set Claude Code
// working); its hook's word sets it working.
func TestAHookedAgentWorksOnlyOnItsWord(t *testing.T) {
	r := fake(t, `printf 'x\007'; read a; sleep .5; printf '\033]9;squeeze:working\007'`)
	r.hooked = true
	time.Sleep(300 * time.Millisecond)
	for _, submit := range []bool{false, true} {
		r.userInput(submit)
		if got := r.state(time.Now()); got != waiting {
			t.Errorf("after userInput(%v): %v, want waiting", submit, got)
		}
	}
	k, _ := toEmulator(driver.KeyEvent{Type: driver.KeyEnter})
	r.emu.SendKey(k)
	time.Sleep(900 * time.Millisecond)
	if got := r.state(time.Now()); got != working {
		t.Errorf("after the working mark: %v, want working", got)
	}
}
