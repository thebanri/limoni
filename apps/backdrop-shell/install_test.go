package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeHome points HOME and XDG_CONFIG_HOME at a temporary directory.
func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("ZDOTDIR", "")
	return home
}

func TestSettingsRoundTrip(t *testing.T) {
	fakeHome(t)
	want := settings{Scene: "city", Opacity: 0.3, FPS: 12, Still: true, Select: true}
	if err := want.save(settingsPath()); err != nil {
		t.Fatal(err)
	}
	got, err := loadSettings(settingsPath())
	if err != nil || got != want {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
	// Written by hand, as the README shows it.
	hand := "# mine\nscene = city     # a comment\nopacity=0.3\nfps = 12\nstill = true\n"
	if err := os.WriteFile(settingsPath(), []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := loadSettings(settingsPath()); got != want {
		t.Fatalf("hand-written file gave %+v", got)
	}
	os.Remove(settingsPath())
	if got, _ := loadSettings(settingsPath()); got != defaultSettings() {
		t.Fatalf("missing file gave %+v", got)
	}
}

// Enabling twice leaves one block, at the top, and disabling gives back the
// file byte for byte — through a symlink, as dotfile managers keep them.
func TestHookBlocksAreAddedOnceAndRemovedCleanly(t *testing.T) {
	home := fakeHome(t)
	dotfiles := filepath.Join(home, "dotfiles")
	os.Mkdir(dotfiles, 0o755)
	original := "# my bashrc\nalias ll='ls -l'\n"
	real := filepath.Join(dotfiles, "bashrc")
	if err := os.WriteFile(real, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".bashrc")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	h := hook{shell: "bash", file: link}
	for i := 0; i < 2; i++ {
		if err := h.install("/opt/bin/backdrop-shell"); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(real)
	text := string(b)
	if strings.Count(text, blockStart) != 1 || !strings.HasPrefix(text, blockStart) || !strings.HasSuffix(text, original) {
		t.Fatalf("after enabling twice:\n%s", text)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the symlink was replaced by a file")
	}
	// Windows has no mode bits: a file's access is its ACL's, and Go reports
	// 0666 for every writable file there.
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(real); fi.Mode().Perm() != 0o600 {
			t.Fatalf("permissions changed to %v", fi.Mode().Perm())
		}
	}
	if removed, err := h.remove(); !removed || err != nil {
		t.Fatalf("remove = %v, %v", removed, err)
	}
	if b, _ := os.ReadFile(real); string(b) != original {
		t.Fatalf("after disabling:\n%q\nwant\n%q", b, original)
	}
	if removed, _ := h.remove(); removed {
		t.Fatal("removed a block that was not there")
	}
}

// The hooks are run by real shells: an interactive one hands itself over to
// the binary with the right arguments; one already inside does not.
func TestHooksHandOverInRealShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("these hooks hand a POSIX shell over to the wrapper; Windows gets a profile hook of its own")
	}
	dir := t.TempDir()
	markerFile := filepath.Join(dir, "ran")
	bin := filepath.Join(dir, "fake-backdrop-shell")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + markerFile + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		shell string
		args  []string
		want  string // the shell's name in what the binary was given
	}{
		{"bash", []string{"--norc", "-i", "-c", "true"}, "bash"},
		{"zsh", []string{"-f", "-i", "-c", "true"}, "zsh"},
		{"fish", []string{"--no-config", "-i", "-c", "true"}, "fish"},
	}
	for _, tc := range cases {
		path, err := exec.LookPath(tc.shell)
		if err != nil {
			t.Logf("%s not installed; skipped", tc.shell)
			continue
		}
		hookFile := filepath.Join(dir, tc.shell+"-hook")
		if err := os.WriteFile(hookFile, []byte(hookText(tc.shell, bin)), 0o644); err != nil {
			t.Fatal(err)
		}
		// Syntax first: a hook that does not parse would break the shell.
		check := map[string][]string{"bash": {"-n", hookFile}, "zsh": {"-n", hookFile}, "fish": {"--no-execute", hookFile}}
		if out, err := exec.Command(path, check[tc.shell]...).CombinedOutput(); err != nil {
			t.Fatalf("%s rejects the hook: %v\n%s", tc.shell, err, out)
		}
		// The terminal these shells run on, as tty names it: the marker a
		// backdrop-shell sets for the shell inside it names its terminal.
		here, _ := exec.Command("tty").Output()
		run := func(marker string, interactive bool) (string, bool) {
			os.Remove(markerFile)
			source := map[string]string{"bash": ". ", "zsh": ". ", "fish": "source "}[tc.shell]
			var args []string
			for _, a := range tc.args[:len(tc.args)-1] {
				if a != "-i" || interactive {
					args = append(args, a)
				}
			}
			args = append(args, source+hookFile+"; true")
			cmd := exec.Command(path, args...)
			cmd.Env = withoutEnv(withoutEnv(os.Environ(), "SSH_TTY"), envNested)
			if marker != "" {
				cmd.Env = append(cmd.Env, envNested+"="+marker)
			}
			_ = cmd.Run()
			b, err := os.ReadFile(markerFile)
			return string(b), err == nil
		}
		got, ran := run("", true)
		if !ran || !strings.HasPrefix(got, "--\n") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: interactive shell did not hand over (ran=%v, args %q)", tc.shell, ran, got)
		}
		if _, ran := run(strings.TrimSpace(string(here)), true); ran {
			t.Errorf("%s: a shell already inside backdrop-shell handed over again", tc.shell)
		}
		// A terminal window opened from inside one inherits the marker, but
		// it names another terminal: that window gets a background too.
		if _, ran := run("/dev/pts/9999", true); !ran {
			t.Errorf("%s: a new terminal opened from inside backdrop-shell was left without one", tc.shell)
		}
		if _, ran := run("", false); ran {
			t.Errorf("%s: a script (a shell that is not interactive) handed over", tc.shell)
		}
	}
}

// A file that held nothing but the block was created by enable, and disable
// takes it away again rather than leaving an empty rc file behind.
func TestDisableRemovesAFileEnableCreated(t *testing.T) {
	home := fakeHome(t)
	h := hook{shell: "zsh", file: filepath.Join(home, ".zshrc")}
	if err := h.install("/opt/bin/backdrop-shell"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(h.file); !os.IsNotExist(err) {
		t.Fatalf("%s is still there (%v)", h.file, err)
	}
}
