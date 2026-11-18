//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func powershellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// TestPowerShellHookText checks that the profile hook has the guards it needs
// and parses as PowerShell. A hook that does not parse would break the shell.
func TestPowerShellHookText(t *testing.T) {
	bin := `C:\tools\backdrop-shell.exe`
	for _, shell := range []string{"pwsh", "powershell"} {
		text := hookText(shell, bin)
		for _, want := range []string{
			blockStart, blockEnd, bin,
			"Test-Path -LiteralPath '" + bin + "'",
			"& '" + bin + "' -- (Get-Command " + shell + ").Source",
			"$Env:LIMONI_BACKDROP_HOOK", "[Console]::IsInputRedirected", "$PROFILE.CurrentUserAllHosts",
			"[Environment]::Exit($LASTEXITCODE)",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s hook lacks %q", shell, want)
			}
		}
		if strings.Contains(text, "''"+bin) || strings.Contains(text, bin+"''") {
			t.Errorf("%s hook quotes the binary path twice: %s", shell, text)
		}
		if strings.Contains(text, "-not $PSCommandPath") {
			t.Errorf("%s hook refuses to run from its own profile: %s", shell, text)
		}
		if strings.Contains(text, "\n    exit $LASTEXITCODE") {
			t.Errorf("%s hook only exits the profile script, not the shell: %s", shell, text)
		}

		ps, err := exec.LookPath("powershell.exe")
		if err != nil {
			t.Skip("no powershell.exe to check syntax with")
		}
		script := filepath.Join(t.TempDir(), "profile.ps1")
		if err := os.WriteFile(script, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		check := "$errs = $null; [void][System.Management.Automation.Language.Parser]::ParseFile(" +
			powershellQuote(script) + ", [ref]$null, [ref]$errs); if ($errs.Count) { $errs | ForEach-Object { $_.Message }; exit 1 }"
		if out, err := exec.Command(ps, "-NoProfile", "-Command", check).CombinedOutput(); err != nil {
			t.Errorf("%s hook does not parse: %v\n%s", shell, err, out)
		}
	}
}

func testPowerShellHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	return home
}

func findHook(t *testing.T, shell string) hook {
	t.Helper()
	for _, h := range hooks() {
		if h.shell == shell {
			return h
		}
	}
	t.Fatalf("no %s hook on Windows", shell)
	return hook{}
}

// TestPowerShellHookInstall checks that enable writes the profile hook and
// disable removes exactly that block, using a fake home.
func TestPowerShellHookInstall(t *testing.T) {
	home := testPowerShellHome(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	h := findHook(t, "powershell")
	want := filepath.Join(home, "Documents", "WindowsPowerShell", "Profile.ps1")
	if h.file != want {
		t.Fatalf("profile %s, want %s", h.file, want)
	}
	bin := filepath.Join(home, "backdrop-shell.exe")
	if err := h.install(bin); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(h.file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), bin) || !strings.Contains(string(got), blockStart) {
		t.Fatalf("profile does not hold the hook: %s", got)
	}

	removed, err := h.remove()
	if err != nil || !removed {
		t.Fatalf("remove: removed=%v err=%v", removed, err)
	}
	got, err = os.ReadFile(h.file)
	if errors.Is(err, os.ErrNotExist) {
		return // a file holding only the hook is removed whole
	}
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), blockStart) || strings.Contains(string(got), bin) {
		t.Fatalf("remove left the hook behind: %s", got)
	}
}

// TestEnableHooksPowerShell drives the real command so that the shell list,
// the settings file and the hand-over all agree with each other.
func TestEnableHooksPowerShell(t *testing.T) {
	home := testPowerShellHome(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	if handled, err := command([]string{"enable", "-shells", "powershell", "-scene", "city"}); !handled || err != nil {
		t.Fatalf("enable: handled=%v err=%v", handled, err)
	}
	profile := findHook(t, "powershell").file
	got, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("no profile after enable: %v", err)
	}
	if !strings.Contains(string(got), blockStart) || !strings.Contains(string(got), "(Get-Command powershell).Source") {
		t.Fatalf("profile does not hand the shell over: %s", got)
	}
	set, err := loadSettings(settingsPath())
	if err != nil || set.Scene != "city" {
		t.Fatalf("settings scene=%q err=%v", set.Scene, err)
	}

	if handled, err := command([]string{"disable"}); !handled || err != nil {
		t.Fatalf("disable: handled=%v err=%v", handled, err)
	}
	got, err = os.ReadFile(profile)
	if errors.Is(err, os.ErrNotExist) {
		return // a file holding only the hook is removed whole
	}
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), blockStart) {
		t.Fatalf("disable left the hook behind: %s", got)
	}
}
