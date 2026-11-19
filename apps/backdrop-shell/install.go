package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// The permanent setup: a settings file, and a few lines in each shell's
// start-up file that hand an interactive shell over to backdrop-shell. The
// start-up file rather than the terminal's configuration, so the scene is
// there in every terminal the user opens — Alacritty, kitty, Konsole, an
// editor's built-in terminal — without configuring each.
//
// The lines are marked, and disable removes exactly them. A fish hook is a
// file of its own in conf.d; bash and zsh get a block at the top of their rc
// file, above anything slow such as a prompt theme, since everything before
// the hand-over runs twice.

const (
	blockStart = "# >>> limoni backdrop >>>"
	blockEnd   = "# <<< limoni backdrop <<<"
)

// settings are the defaults backdrop-shell starts with; flags override them.
type settings struct {
	Scene   string
	Opacity float64
	FPS     float64
	Still   bool
	Image   string // a picture file; beats Scene
	Art     string // an ASCII art file; beats Image and Scene
	// SceneCmd is a program that draws the scene (scenecmd.go); it beats
	// Scene, and Art and Image beat it.
	SceneCmd string
	Select   bool // the wrapper's own mouse selection, which leaves the background out
}

func defaultSettings() settings { return settings{Scene: "aurora", Opacity: 0.45, Select: true} }

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func settingsPath() string { return filepath.Join(configDir(), "limoni", "backdrop.conf") }

// loadSettings reads the settings file; a missing one gives the defaults.
func loadSettings(path string) (settings, error) {
	s := defaultSettings()
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#") // comments, whole-line or after a value
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "scene":
			s.Scene = value
		case "opacity":
			s.Opacity, _ = strconv.ParseFloat(value, 64)
		case "fps":
			s.FPS, _ = strconv.ParseFloat(value, 64)
		case "still":
			s.Still = value == "true"
		case "image":
			s.Image = value
		case "art":
			s.Art = value
		case "scene-cmd":
			s.SceneCmd = value
		case "select":
			s.Select = value != "false"
		}
	}
	return s, sc.Err()
}

func (s settings) save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	text := fmt.Sprintf(`# backdrop-shell settings. Edit freely; command-line flags override them.
# scene: %s
scene = %s
# opacity: 0 (hidden) to 1 (full strength)
opacity = %g
# fps: cap the scene's frame rate; 0 is the scene's own
fps = %g
# still: a still picture instead of an animation
still = %t
# image: a picture file (PNG, JPEG, GIF) instead of a scene
image = %s
# art: a text file of ASCII art instead of a scene; it beats image
art = %s
# scene-cmd: a program that draws the scene, in any language; image and art beat it
scene-cmd = %s
# select: select and copy with the mouse, leaving the background out;
# false gives the mouse back to the terminal, whose selection copies it too
select = %t
`, strings.Join(sceneNames(), ", "), s.Scene, s.Opacity, s.FPS, s.Still, s.Image, s.Art, s.SceneCmd, s.Select)
	return os.WriteFile(path, []byte(text), 0o644)
}

// hook is where one shell's hand-over lives.
type hook struct {
	shell string
	file  string
	own   bool // the whole file is the hook (fish conf.d), not a block in it
}

func hooks() []hook {
	home, _ := os.UserHomeDir()
	zdot := os.Getenv("ZDOTDIR")
	if zdot == "" {
		zdot = home
	}
	return append([]hook{
		{shell: "fish", file: filepath.Join(configDir(), "fish", "conf.d", "limoni-backdrop.fish"), own: true},
		{shell: "bash", file: filepath.Join(home, ".bashrc")},
		{shell: "zsh", file: filepath.Join(zdot, ".zshrc")},
	}, platformHooks()...)
}

// hookText is the hand-over for a shell: an interactive shell that is not
// already inside backdrop-shell, and not reached over SSH, replaces itself
// with backdrop-shell running the same shell. If the binary is gone, the
// check fails and the shell starts as usual.
func hookText(shell, bin string) string {
	q := "'" + bin + "'"
	switch shell {
	case "fish":
		return `# Added by "backdrop-shell enable"; "backdrop-shell disable" removes this file.
if status is-interactive; and test "$LIMONI_BACKDROP_SHELL" != (tty 2>/dev/null); and not set -q SSH_TTY; and test -x ` + q + `
    if status is-login
        exec ` + q + ` -- (status fish-path) -l
    else
        exec ` + q + ` -- (status fish-path)
    end
end
`
	case "bash":
		return blockStart + ` added by "backdrop-shell enable"; "backdrop-shell disable" removes it
if [[ $- == *i* && ${LIMONI_BACKDROP_SHELL-} != "$(tty 2>/dev/null)" && -z ${SSH_TTY-} && -x ` + q + ` ]]; then
    if shopt -q login_shell; then exec ` + q + ` -- "$BASH" -l; else exec ` + q + ` -- "$BASH"; fi
fi
` + blockEnd + "\n"
	case "zsh":
		return blockStart + ` added by "backdrop-shell enable"; "backdrop-shell disable" removes it
if [[ -o interactive && ${LIMONI_BACKDROP_SHELL-} != "$(tty 2>/dev/null)" && -z ${SSH_TTY-} && -x ` + q + ` ]]; then
    if [[ -o login ]]; then exec ` + q + ` -- zsh -l; else exec ` + q + ` -- zsh; fi
fi
` + blockEnd + "\n"
	case "pwsh", "powershell":
		// A profile also runs for scripts, so hand over only in an
		// interactive session with a real console, and only from the profile
		// itself: $PSCommandPath names the profile while it is running.
		// The marker makes the shell the wrapper starts skip its own
		// profile's hook; otherwise each hand-over would start another one.
		return blockStart + ` added by "backdrop-shell enable"; "backdrop-shell disable" removes it
if (-not $Env:LIMONI_BACKDROP_HOOK -and -not $Env:SSH_TTY -and -not $Env:SSH_CONNECTION -and
    -not [Console]::IsInputRedirected -and -not [Console]::IsOutputRedirected -and
    ([string]::IsNullOrEmpty($PSCommandPath) -or $PSCommandPath -eq $PROFILE.CurrentUserAllHosts) -and
    (Test-Path -LiteralPath ` + q + `)) {
    $Env:LIMONI_BACKDROP_HOOK = '1'
    & ` + q + ` -- (Get-Command ` + shell + `).Source
    [Environment]::Exit($LASTEXITCODE)
}
` + blockEnd + "\n"
	}
	return ""
}

// removeBlock returns text without the marked block, and whether it had one.
func removeBlock(text string) (string, bool) {
	start := strings.Index(text, blockStart)
	if start < 0 {
		return text, false
	}
	end := strings.Index(text[start:], blockEnd)
	if end < 0 {
		return text, false // a start with no end: leave the file alone
	}
	end += start + len(blockEnd)
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return text[:start] + text[end:], true
}

// writeInPlace replaces the contents of path, following a symlink to the
// real file (dotfile managers link rc files) and keeping its permissions.
func writeInPlace(path, text string) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	mode := fs.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	// A profile's directory may never have been created: PowerShell keeps
	// its profiles in Documents, which a new account need not have used.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".limoni-tmp"
	if err := os.WriteFile(tmp, []byte(text), mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (h hook) installed() bool {
	b, err := os.ReadFile(h.file)
	if err != nil {
		return false
	}
	if h.own {
		return true
	}
	_, found := removeBlock(string(b))
	return found
}

func (h hook) install(bin string) error {
	text := hookText(h.shell, bin)
	if h.own {
		if err := os.MkdirAll(filepath.Dir(h.file), 0o755); err != nil {
			return err
		}
		return os.WriteFile(h.file, []byte(text), 0o644)
	}
	old, err := os.ReadFile(h.file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	rest, _ := removeBlock(string(old))
	return writeInPlace(h.file, text+rest)
}

func (h hook) remove() (bool, error) {
	if h.own {
		err := os.Remove(h.file)
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return err == nil, err
	}
	old, err := os.ReadFile(h.file)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	rest, found := removeBlock(string(old))
	if !found {
		return false, nil
	}
	if rest == "" {
		// Nothing but the block: enable created the file, so it goes.
		return true, os.Remove(h.file)
	}
	return true, writeInPlace(h.file, rest)
}

// tilde writes a path under the home directory with ~, as people type it.
func tilde(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || home == "/" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}

// expandHome turns a leading ~ into the home directory, for a path given in
// a way the shell did not expand (-image=~/x.jpg).
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

// selfPath is the absolute path of this binary, which the hooks run.
func selfPath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		p = real
	}
	if strings.ContainsAny(p, "'\n") {
		return "", fmt.Errorf("cannot hook a binary whose path contains a quote: %s", p)
	}
	return p, nil
}

// command runs a subcommand, if args name one, and reports whether it did.
func command(args []string) (handled bool, err error) {
	if len(args) == 0 {
		return false, nil
	}
	switch args[0] {
	case "enable":
		return true, cmdEnable(args[1:])
	case "disable":
		return true, cmdDisable(true)
	case "status":
		return true, cmdStatus()
	case "reset":
		return true, cmdReset()
	case "opacity":
		return true, cmdOpacity(args[1:])
	case "uninstall":
		return true, cmdUninstall()
	}
	return false, nil
}

func cmdEnable(args []string) error {
	s, err := loadSettings(settingsPath())
	if err != nil {
		return err
	}
	fl := flag.NewFlagSet("enable", flag.ContinueOnError)
	fl.StringVar(&s.Scene, "scene", s.Scene, "scene: "+strings.Join(sceneNames(), ", "))
	fl.Float64Var(&s.Opacity, "opacity", s.Opacity, "how strongly the scene shows, from 0 to 1")
	fl.Float64Var(&s.FPS, "fps", s.FPS, "cap the scene's frame rate (0: the scene's own)")
	fl.BoolVar(&s.Still, "still", s.Still, "a still picture instead of an animation")
	fl.StringVar(&s.Image, "image", s.Image, "a picture (PNG, JPEG, GIF) instead of a scene")
	fl.StringVar(&s.Art, "art", s.Art, "a text file of ASCII art instead of a scene")
	fl.StringVar(&s.SceneCmd, "scene-cmd", s.SceneCmd, "a program that draws the scene, in any language")
	fl.BoolVar(&s.Select, "select", s.Select, "select and copy with the mouse without the background")
	shells := fl.String("shells", "", "comma-separated shells to hook (default: fish, bash, zsh, pwsh, powershell, those installed)")
	if err := fl.Parse(args); err != nil {
		return err
	}
	// The background named on the command line is the one wanted: a new
	// -scene drops an earlier picture or art, and so on. A new background
	// also starts moving again unless -still comes with it; an old -still
	// freezing a newly chosen animation would look like a bug.
	stillGiven, chosen := false, false
	fl.Visit(func(f *flag.Flag) {
		if chooseOnly(f.Name, &s.Image, &s.Art, &s.SceneCmd) {
			chosen = true
		}
		if f.Name == "still" {
			stillGiven = true
		}
	})
	if chosen && !stillGiven {
		s.Still = false
	}
	// The hook runs from anywhere, so a file is kept by its full path.
	for _, p := range []*string{&s.Image, &s.Art, &s.SceneCmd} {
		if *p != "" {
			if abs, err := filepath.Abs(expandHome(*p)); err == nil {
				*p = abs
			}
		}
	}
	if _, err := chooseScene(s.Scene, s.Image, s.Art, s.SceneCmd, s.Still); err != nil {
		return err
	}
	bin, err := selfPath()
	if err != nil {
		return err
	}
	if err := s.save(settingsPath()); err != nil {
		return err
	}
	want := map[string]bool{}
	for _, sh := range strings.Split(*shells, ",") {
		if sh = strings.TrimSpace(sh); sh != "" {
			want[sh] = true
		}
	}
	done := 0
	for _, h := range hooks() {
		if len(want) > 0 && !want[h.shell] {
			continue
		}
		if len(want) == 0 {
			if _, err := exec.LookPath(h.shell); err != nil {
				continue
			}
		}
		if err := h.install(bin); err != nil {
			return fmt.Errorf("%s: %w", h.file, err)
		}
		fmt.Printf("  on   %-4s  %s\n", h.shell, tilde(h.file))
		done++
	}
	if done == 0 {
		return errors.New("no shell to hook: none of fish, bash, zsh, pwsh or powershell found (or named with -shells)")
	}
	fmt.Printf("Settings: %s\n", tilde(settingsPath()))
	reportReload()
	fmt.Println("\"backdrop-shell disable\" turns it off.")
	return nil
}

func cmdDisable(hint bool) error {
	removedAny := false
	for _, h := range hooks() {
		removed, err := h.remove()
		if err != nil {
			return fmt.Errorf("%s: %w", h.file, err)
		}
		if removed {
			fmt.Printf("  off  %-4s  %s\n", h.shell, tilde(h.file))
			removedAny = true
		}
	}
	if !removedAny && hint {
		fmt.Println("Nothing to turn off: no shell starts backdrop-shell.")
	} else if hint {
		fmt.Println("New terminals open without the scene. \"backdrop-shell enable\" turns it back on.")
	}
	return nil
}

func cmdStatus() error {
	bin, _ := selfPath()
	s, err := loadSettings(settingsPath())
	if err != nil {
		return err
	}
	fmt.Printf("binary    %s\nsettings  %s\n", tilde(bin), tilde(settingsPath()))
	fmt.Printf("          scene=%s opacity=%g fps=%g still=%t\n", s.Scene, s.Opacity, s.FPS, s.Still)
	if s.Art != "" {
		fmt.Printf("          art=%s\n", s.Art)
	} else if s.Image != "" {
		fmt.Printf("          image=%s\n", s.Image)
	} else if s.SceneCmd != "" {
		fmt.Printf("          scene-cmd=%s\n", s.SceneCmd)
	}
	if s.Still {
		fmt.Println("          still: the background is frozen. \"backdrop-shell enable -still=false\" makes it move.")
	}
	if !s.Select {
		fmt.Println("          select=false: the terminal's own selection copies the background too.")
	}
	for _, h := range hooks() {
		state := "off"
		if h.installed() {
			state = "on"
		}
		fmt.Printf("%-9s %-3s  %s\n", h.shell, state, tilde(h.file))
	}
	if nested() {
		fmt.Println("This shell is running inside backdrop-shell.")
	}
	return nil
}

// cmdReset puts the settings back to the defaults — the aurora, moving —
// and leaves the shells as they are: on stays on, off stays off.
func cmdReset() error {
	if err := defaultSettings().save(settingsPath()); err != nil {
		return err
	}
	fmt.Printf("Settings are back to the defaults (%s).\n", tilde(settingsPath()))
	reportReload()
	return nil
}

func cmdUninstall() error {
	if err := cmdDisable(false); err != nil {
		return err
	}
	path := settingsPath()
	if err := os.Remove(path); err == nil {
		fmt.Printf("  removed %s\n", tilde(path))
		_ = os.Remove(filepath.Dir(path)) // only if empty
	}
	bin, err := selfPath()
	if err != nil {
		return err
	}
	if err := os.Remove(bin); err != nil {
		return fmt.Errorf("removing %s: %w", bin, err)
	}
	fmt.Printf("  removed %s\nbackdrop-shell is uninstalled. Terminals already open keep it until closed.\n", tilde(bin))
	return nil
}
