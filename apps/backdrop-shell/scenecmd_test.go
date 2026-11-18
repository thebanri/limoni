//go:build unix

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// sceneProgram writes a shell script that draws a scene.
func sceneProgram(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "scene.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// waitFrame renders the scene until row y starts with want.
func waitFrame(t *testing.T, c *cmdScene, b *buffer.Buffer, y int, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	got := ""
	for time.Now().Before(deadline) {
		c.Render(b, 0)
		if got = rowText(b, y); strings.HasPrefix(got, want) {
			return
		}
		select {
		case <-c.Changed():
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatalf("row %d is %q, want it to start with %q", y, got, want)
}

// The program is told the screen's size, and started again with the new one
// when the window changes; what it writes is laid out as on a terminal,
// newlines returning to the first column.
func TestSceneProgramFollowsTheScreenSize(t *testing.T) {
	c, err := newCmdScene(sceneProgram(t, `printf '%sx%s\n%s\n' "$LIMONI_COLS" "$LIMONI_ROWS" "$(stty size)"; sleep 30`))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	b := newBuf(20, 3)
	waitFrame(t, c, b, 0, "20x3")
	waitFrame(t, c, b, 1, "3 20") // the pseudo-terminal's own size
	b = newBuf(30, 4)
	waitFrame(t, c, b, 0, "30x4")
}

// A frame is shown whole. This program clears the screen before each of its
// frames, one every 10 ms or so, while the scene is drawn; had a frame been shown where the clear begins the next one rather
// than before it, the scene would flash empty.
func TestSceneProgramFramesAreShownWhole(t *testing.T) {
	c, err := newCmdScene(sceneProgram(t, `i=0
while [ $i -lt 150 ]; do printf '\033[2J\033[HAB%03d' $i; i=$((i+1)); sleep 0.01; done
sleep 30`))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	b := newBuf(20, 2)
	seen := map[string]bool{}
	deadline := time.Now().Add(10 * time.Second)
	for !seen["AB149"] { // the last one, which no frame follows
		if time.Now().After(deadline) {
			t.Fatalf("the last frame never showed; saw %d", len(seen))
		}
		c.Render(b, 0)
		got := rowText(b, 0)
		if len(seen) > 0 && !strings.HasPrefix(got, "AB") {
			t.Fatalf("after %d frames, one showed %q", len(seen), got)
		}
		if got != "" {
			seen[got] = true
		}
		time.Sleep(2 * time.Millisecond)
	}
	// Frames show while the program draws, not only once it stops.
	if len(seen) < 20 {
		t.Fatalf("only %d of 150 frames showed", len(seen))
	}
}

// Print ends every line with a newline, the last too; on a screen the size
// of the program's, that scrolled the first row away.
func TestSceneProgramMayEndWithANewline(t *testing.T) {
	c, err := newCmdScene(sceneProgram(t, `printf 'top\nmid\nend\n'; sleep 30`))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	b := newBuf(10, 3)
	waitFrame(t, c, b, 2, "end")
	if got := rowText(b, 0); got != "top" {
		t.Fatalf("first row %q", got)
	}
}

// Colours become RGB so -opacity fades them; a cell drawn with no
// background keeps none, so the terminal's shows.
func TestSceneProgramColours(t *testing.T) {
	c, err := newCmdScene(sceneProgram(t, `printf '\033[38;5;196;48;2;1;2;3mX\033[0mY'; sleep 30`))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	b := newBuf(4, 1)
	waitFrame(t, c, b, 0, "XY")
	x, y := b.Content[0], b.Content[1]
	if x.Style.Fg != cell.NewColorRGB(255, 0, 0) || x.Style.Bg != cell.NewColorRGB(1, 2, 3) {
		t.Errorf("X is %+v", x.Style)
	}
	if y.Style.Bg.Type() != cell.ColorDefault {
		t.Errorf("Y has a background: %+v", y.Style)
	}
}

// Closing the scene ends the program and what it started; pausing stops it
// and going on lets it run again.
func TestSceneProgramIsStoppedAndKilled(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	c, err := newCmdScene(sceneProgram(t, `sleep 30 & echo $! > `+pidFile+`; echo up; wait`))
	if err != nil {
		t.Fatal(err)
	}
	b := newBuf(10, 2)
	waitFrame(t, c, b, 0, "up")
	raw, _ := os.ReadFile(pidFile)
	child, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if child == 0 {
		t.Fatal("no pid")
	}
	c.Pause(true)
	waitState(t, child, "T")
	c.Pause(false)
	waitState(t, child, "S")
	c.Close()
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(child, 0) == nil && !zombie(child) {
		if time.Now().After(deadline) {
			t.Fatal("the program's child outlived the scene")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func procState(pid int) string {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return ""
	}
	s := string(b)
	if i := strings.LastIndexByte(s, ')'); i >= 0 && i+2 < len(s) {
		return s[i+2 : i+3]
	}
	return ""
}

func zombie(pid int) bool { return procState(pid) == "Z" }

func waitState(t *testing.T, pid int, want string) {
	t.Helper()
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("no /proc")
	}
	deadline := time.Now().Add(3 * time.Second)
	for procState(pid) != want {
		if time.Now().After(deadline) {
			t.Fatalf("process state %q, want %q", procState(pid), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSceneProgramMustBeExecutable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "scene.py")
	os.WriteFile(p, []byte("print(1)\n"), 0o644)
	if _, err := chooseScene("aurora", "", "", p, false); err == nil || !strings.Contains(err.Error(), "chmod +x") {
		t.Fatalf("a file that cannot run was accepted: %v", err)
	}
	if _, err := chooseScene("aurora", "", "", p+".missing", false); err == nil {
		t.Fatal("a missing program was accepted")
	}
}

func TestFrameStartsCutAcrossReads(t *testing.T) {
	for _, in := range []string{"ab\x1b", "ab\x1b[", "ab\x1b[1", "ab\x1b[1;", "ab\x1b[1;1", "ab\x1b[2"} {
		rest, carry := splitCarry([]byte(in))
		if string(rest) != "ab" || string(carry) != in[2:] {
			t.Errorf("%q split as %q + %q", in, rest, carry)
		}
	}
	if rest, carry := splitCarry([]byte("ab\x1b[31m")); string(rest) != "ab\x1b[31m" || carry != nil {
		t.Errorf("a colour was held back: %q + %q", rest, carry)
	}
	if at, n := nextFrameStart([]byte("x\x1b[31my\x1b[1;1Hz")); at != 7 || n != 6 {
		t.Errorf("found at %d, %d long", at, n)
	}
}

// Enabling a program's scene drops an earlier picture or art, and keeps
// the program by its full path, for the hook that runs from anywhere.
func TestEnableKeepsTheSceneProgram(t *testing.T) {
	fakeHome(t)
	prog := sceneProgram(t, "true\n")
	s := defaultSettings()
	s.Art = "/somewhere/art.txt"
	if err := s.save(settingsPath()); err != nil {
		t.Fatal(err)
	}
	// No shell to hook, so it fails — after saving the settings.
	_ = cmdEnable([]string{"-scene-cmd", prog, "-shells", "nosuchshell"})
	got, _ := loadSettings(settingsPath())
	if got.SceneCmd != prog || got.Art != "" {
		t.Fatalf("settings %+v", got)
	}
}

// The example scenes that ship next to the program, and that the guide
// points to, run and draw.
func TestExampleScenesDraw(t *testing.T) {
	files, _ := filepath.Glob("scenes/*")
	if len(files) == 0 {
		t.Fatal("no example scenes")
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".py") {
			if _, err := exec.LookPath("python3"); err != nil {
				t.Logf("%s: no python3; skipped", f)
				continue
			}
		}
		c, err := newCmdScene(f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		b := newBuf(80, 24)
		deadline := time.Now().Add(5 * time.Second)
		for drawn := false; !drawn; {
			if time.Now().After(deadline) {
				t.Errorf("%s drew nothing", f)
				break
			}
			select {
			case <-c.Changed():
			case <-time.After(100 * time.Millisecond):
			}
			c.Render(b, 0)
			for y := 0; y < 24 && !drawn; y++ {
				drawn = rowText(b, y) != ""
			}
		}
		c.Close()
	}
}
