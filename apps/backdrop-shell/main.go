// backdrop-shell runs your shell in front of an animated scene, in any
// terminal with 256 colours or more.
//
//	backdrop-shell                        # $SHELL over the aurora
//	backdrop-shell -scene city -opacity 0.5
//	backdrop-shell -scene synthwave -still  # a picture, not an animation
//	backdrop-shell -image ~/Pictures/wall.jpg
//	backdrop-shell -art my-art.txt        # your own ASCII art, still or animated
//	backdrop-shell -scene-cmd ./my-scene.py # a scene your own program draws
//	backdrop-shell -- htop                # any command instead of the shell
//
// A terminal has no layers: one grid of cells, and whatever writes last is
// what shows. So the scene cannot run beside the shell; it has to know where
// the shell's text is. backdrop-shell runs the shell on a pseudo-terminal,
// keeps an emulated screen of what it writes, and draws that over the scene:
// every cell the shell leaves without a background colour shows the scene
// behind it, faded into the terminal's own background by -opacity.
//
// What reaches the terminal is Limoni's cell diff, so the scene costs only
// the cells it changes. It stops while the window is out of focus, and while
// a full-screen program covers it all; a still scene costs nothing at all.
//
// The wheel, Shift+PageUp and Shift+PageDown scroll back through the
// history, since the terminal's own scrollback is set aside while the
// wrapper runs. Dragging selects and copies only what the shell wrote —
// the terminal's own selection would copy the background's characters too
// (see select.go).
//
// To have it in every new terminal, "backdrop-shell enable" adds a few marked
// lines to the start-up files of fish, bash and zsh; "backdrop-shell disable"
// takes exactly those lines out again, and "backdrop-shell uninstall" removes
// the settings and the binary too. install.sh builds it and enables it.
//
// It starts the plain shell instead when it is already running inside
// itself, when it is not on a terminal at all, or on a 16-colour one.
package main

import (
	"flag"
	"fmt"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/thebanri/limoni/backdrop"
	"github.com/thebanri/limoni/core/terminal"
)

// envNested names the terminal of a running backdrop-shell, for the shell
// inside it: a backdrop-shell started on that same terminal runs the plain
// shell instead of a scene inside a scene. On any other terminal — a window
// opened from inside, which inherits the variable — it is ignored.
const envNested = "LIMONI_BACKDROP_SHELL"

// nested reports whether this process runs on the terminal of a
// backdrop-shell already.
func nested() bool {
	v := os.Getenv(envNested)
	return v != "" && v == ttyName(os.Stdin)
}

func main() {
	if handled, err := command(os.Args[1:]); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, "backdrop-shell:", err)
			os.Exit(1)
		}
		return
	}

	// The settings file gives the defaults; flags override them.
	s, err := loadSettings(settingsPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "backdrop-shell:", err)
	}
	opts := options{}
	flag.StringVar(&opts.scene, "scene", s.Scene, "scene: "+strings.Join(sceneNames(), ", "))
	flag.Float64Var(&opts.opacity, "opacity", s.Opacity, "how strongly the scene shows, from 0 to 1")
	flag.Float64Var(&opts.fps, "fps", s.FPS, "cap the scene's frame rate (0: the scene's own)")
	flag.BoolVar(&opts.still, "still", s.Still, "a still picture of the scene instead of an animation")
	flag.StringVar(&opts.image, "image", s.Image, "a picture (PNG, JPEG, GIF) as the background instead of a scene")
	flag.StringVar(&opts.art, "art", s.Art, "a text file of ASCII art as the background; see docs/backdrop-art.md")
	flag.StringVar(&opts.sceneCmd, "scene-cmd", s.SceneCmd, "a program that draws the background, in any language; see docs/backdrop-art.md")
	flag.BoolVar(&opts.selection, "select", s.Select, "select and copy with the mouse without the background (false: the terminal's own selection)")
	flag.Usage = usage
	flag.Parse()
	// A background named on the command line is the one wanted, over one
	// the settings name: -scene city shows the city even with art saved.
	flag.Visit(func(f *flag.Flag) { chooseOnly(f.Name, &opts.image, &opts.art, &opts.sceneCmd) })
	// Inside backdrop-shell already, flags alone are a wish to change the
	// background here, which a nested plain shell would silently ignore:
	// "backdrop-shell -opacity 0.3" opened a shell and changed nothing.
	if nested() && flag.NArg() == 0 && flag.NFlag() > 0 {
		var set []*flag.Flag
		flag.Visit(func(f *flag.Flag) { set = append(set, f) })
		fmt.Fprint(os.Stderr, nestedHint(set))
		os.Exit(2)
	}
	opts.argv = flag.Args()
	if len(opts.argv) == 0 {
		opts.argv = defaultShell()
	}

	// The plain shell wherever a scene cannot or should not be drawn: inside
	// backdrop-shell already, turned off, not on a terminal, or on one with
	// 16 colours, where a gradient is blocks of eight.
	caps := terminal.DetectCapabilities()
	if nested() || os.Getenv("LIMONI_BACKDROP") == "off" ||
		!isTerminal(os.Stdin) || !isTerminal(os.Stdout) ||
		(!caps.TrueColor && !caps.Colors256) {
		execPlain(opts.argv)
	}
	if opts.backdrop, err = chooseScene(opts.scene, opts.image, opts.art, opts.sceneCmd, opts.still); err != nil {
		// A background that cannot be drawn is no reason to lose the shell.
		fmt.Fprintln(os.Stderr, "backdrop-shell:", err)
		opts.backdrop, _ = chooseScene("aurora", "", "", "", opts.still)
	}
	code, err := run(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backdrop-shell:", err)
		execPlain(opts.argv)
	}
	os.Exit(code)
}

// chooseScene loads the background the settings name: ASCII art first,
// then a picture, then a program's scene, then a built-in one. -still
// freezes whichever it is, but a program's: that one moves as it wants.
func chooseScene(scene, image, art, sceneCmd string, still bool) (terminal.Backdrop, error) {
	var bd terminal.Backdrop
	switch {
	case art != "":
		a, err := backdrop.LoadArt(art)
		if err != nil {
			return nil, err
		}
		bd = a
	case image != "":
		img, err := backdrop.LoadImage(image)
		if err != nil {
			return nil, err
		}
		bd = img
	case sceneCmd != "":
		c, err := newCmdScene(sceneCmd)
		if err != nil {
			return nil, err
		}
		bd = c
	default:
		bd = backdrop.New(scene)
		if bd != nil {
			break
		}
		path, ok := localScenes()[scene]
		if !ok {
			return nil, fmt.Errorf("no scene %q; there are %s", scene, strings.Join(sceneNames(), ", "))
		}
		c, err := newCmdScene(path)
		if err != nil {
			return nil, err
		}
		bd = c
	}
	if still && bd.Interval() > 0 {
		// A moment well into it, when everything is on stage.
		bd = backdrop.Still(bd, 20*time.Second)
	}
	return bd, nil
}

// sceneNames lists the built-in scenes, then the user's own.
func sceneNames() []string {
	return append(backdrop.Names(), slices.Sorted(maps.Keys(localScenes()))...)
}

// localScenes finds the user's scene programs in ~/.config/limoni/scenes,
// which -scene names like the built-in ones: an executable file there is
// named without its extension (fireflies.py is "fireflies"), and a folder
// holding an executable of its own name (cosmos/cosmos) by the folder's.
// What counts as executable is the platform's: a permission bit on Unix, a
// PATHEXT extension on Windows (platform_unix.go, platform_windows.go).
func localScenes() map[string]string {
	dir := filepath.Join(configDir(), "limoni", "scenes")
	entries, _ := os.ReadDir(dir)
	found := map[string]string{}
	for _, e := range entries {
		name, path := e.Name(), filepath.Join(dir, e.Name())
		if e.IsDir() {
			if path = sceneProgramIn(path, name); path == "" {
				continue // a folder with no program of its name is not a scene
			}
			found[name] = path
			continue
		}
		name = strings.TrimSuffix(name, filepath.Ext(name))
		if isRunnable(path) {
			found[name] = path
		}
	}
	return found
}

// chooseOnly clears the other backgrounds when the flag named chose one, as
// the order of precedence in chooseScene would otherwise overrule it. It
// reports whether the flag chose a background.
func chooseOnly(name string, image, art, sceneCmd *string) bool {
	switch name {
	case "scene":
		*image, *art, *sceneCmd = "", "", ""
	case "image":
		*art, *sceneCmd = "", ""
	case "art":
		*image, *sceneCmd = "", ""
	case "scene-cmd":
		*image, *art = "", ""
	default:
		return false
	}
	return true
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  backdrop-shell [flags] [-- command args]   run a shell (or command) in front of a scene
  backdrop-shell enable [flags]              start it in every new terminal
  backdrop-shell disable                     stop starting it; stays installed
  backdrop-shell status                      what is on, and the settings
  backdrop-shell reset                       the default settings again: the aurora, moving
  backdrop-shell opacity [0.3 | +0.1 | -0.1] show or change how strongly the background shows
  backdrop-shell uninstall                   disable, and remove the settings and this binary

Flags (enable saves them as the defaults):
`)
	flag.PrintDefaults()
}

// execPlain replaces the process with the command, as if the wrapper had
// never been there.
// nestedHint says how to change the background of the terminal the flags
// were typed in: the opacity command, else enable with the same flags.
func nestedHint(set []*flag.Flag) string {
	var flags []string
	opacity := ""
	for _, f := range set {
		if f.Name == "opacity" {
			opacity = f.Value.String()
		}
		flags = append(flags, "-"+f.Name+" "+f.Value.String())
	}
	var b strings.Builder
	b.WriteString("backdrop-shell: this terminal already shows a background, so the flags would start\n")
	b.WriteString("a plain shell inside it and change nothing. To change it here and in every open terminal:\n")
	if opacity != "" && len(set) == 1 {
		b.WriteString("    backdrop-shell opacity " + opacity + "\n")
	} else {
		b.WriteString("    backdrop-shell enable " + strings.Join(flags, " ") + "\n")
	}
	return b.String()
}

func execPlain(argv []string) {
	code, err := execProcess(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backdrop-shell:", err)
		os.Exit(127)
	}
	os.Exit(code)
}

func rgba(r, g, b uint8) color.Color { return color.RGBA{r, g, b, 0xff} }
