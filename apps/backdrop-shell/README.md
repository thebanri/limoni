# backdrop-shell

<p align="center"><img src="../../assets/backdrop-shell.gif" alt="backdrop-shell: a fish prompt over an animated aurora; ls lists files in front of it, backdrop-shell opacity 0.9 brightens it at once, and backdrop-shell enable -scene synthwave switches the running terminal to a striped sunset over a neon grid" width="100%" /></p>

Your shell, in front of an animated scene, a picture, your own ASCII art, or
a scene your own program draws — in any terminal with 256 colours or more.

```bash
backdrop-shell                          # $SHELL over the aurora
backdrop-shell -scene city              # aurora, city, starfield, synthwave
backdrop-shell -opacity 0.3             # quieter (0 to 1, default 0.45)
backdrop-shell -fps 10                  # cap the scene's frame rate
backdrop-shell -still                   # a still picture instead of an animation
backdrop-shell -image ~/Pictures/wall.jpg   # a picture as wallpaper
backdrop-shell -art art/cat.txt         # your own ASCII art, still or animated
backdrop-shell -scene-cmd scenes/fireflies.py  # a scene your own program draws
backdrop-shell -- btop                  # any command instead of the shell
```

The files in [`art/`](art) are examples: a lemon, a cat that blinks, a bird
that flies, clouds and rain. **[docs/backdrop-art.md](../../docs/backdrop-art.md)**
(Türkçe: [docs/tr/backdrop-art.md](../../docs/tr/backdrop-art.md)) shows how
to draw your own, colour it, and animate it frame by frame or across the
screen — and how to turn the output of chafa, jp2a or lolcat into one.

For a scene that fills the screen at any size and moves like the built-in
ones, write a program in any language that draws on a terminal, and name it
with `-scene-cmd`. It is run on a terminal the size of the screen, started
again when the window changes size, stopped while the background cannot be
seen, and ended with the terminal. [`scenes/fireflies.py`](scenes/fireflies.py)
is an example to copy; section 7 of the guide has the rules. Kept in
`~/.config/limoni/scenes`, it is a scene by name: `-scene fireflies`.

## Install

It needs Go 1.25 or newer.

```bash
curl -fsSL https://raw.githubusercontent.com/thebanri/limoni/main/apps/backdrop-shell/install.sh | sh
# with settings:
curl -fsSL https://raw.githubusercontent.com/thebanri/limoni/main/apps/backdrop-shell/install.sh | sh -s -- -scene city -opacity 0.3
# from a checkout of this repository:
sh apps/backdrop-shell/install.sh
```

The installer builds the binary into `$(go env GOPATH)/bin` and runs
`backdrop-shell enable`, which makes every new terminal start with the scene —
Alacritty, kitty, Konsole, an editor's terminal, all of them — by adding a few
marked lines to each installed shell's start-up file:

| Shell | Where |
| :--- | :--- |
| fish | `~/.config/fish/conf.d/limoni-backdrop.fish`, a file of its own |
| bash | a block at the top of `~/.bashrc` |
| zsh  | a block at the top of `~/.zshrc` (above a prompt theme's instant prompt) |

The lines hand an interactive shell over to backdrop-shell, which starts the
same shell again inside it. Scripts, SSH sessions and shells already inside
are left alone, and if the binary is gone the shell starts as usual.

## Windows

The native build uses ConPTY and needs Windows 10 1809 or newer. From a
checkout, install it with:

```powershell
powershell -ExecutionPolicy Bypass -File .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1 -Scene city -Opacity 0.3
```

That builds the binary into Go's bin directory and runs `backdrop-shell enable`.
You can also build it yourself with `go build -o backdrop-shell.exe .` and run
`backdrop-shell.exe`. With no command named it starts `pwsh.exe` or
`powershell.exe`, falling back to `cmd.exe`.

The settings commands (`opacity`, `enable` flags, `reset`, `status`) work, and
changes reach open terminals through the same socket as on Unix (AF_UNIX,
available since Windows 10 17063). Copying a selection uses `clip`.

A few things are different, by necessity:

- `enable` and `disable` write fish, bash and zsh start-up files as on Unix,
  plus PowerShell's `Profile.ps1` for `pwsh` (7) and `powershell` (5.1).
  cmd.exe has no start-up hook; start `backdrop-shell.exe` from there yourself.
- A scene (`-scene-cmd`) and a live region keep drawing while the window is out
  of focus: Windows has no SIGSTOP to pause a program.
- Ending a scene stops the program itself. Ending everything it started would
  need a job object, which this build does not use yet.
- The terminal tab is not renamed after the foreground program.
- `clip` reads its input in the console's code page, so non-ASCII text copied
  from a selection may arrive on the clipboard garbled.

## Turn it off, turn it on, remove it

```bash
backdrop-shell disable      # new terminals open without it; stays installed
backdrop-shell enable       # back on (flags change the settings: -scene city ...)
backdrop-shell status       # what is on, and the settings
backdrop-shell opacity 0.3  # how strongly it shows; +0.1 / -0.1 step it
backdrop-shell reset        # the default settings again: the aurora, moving
backdrop-shell uninstall    # remove the lines, the settings and the binary
```

**The background stopped moving?** `still = true` is set — `status` says so.
`backdrop-shell enable -still=false` makes it move again, and choosing a
background (`enable -scene …`, `-art …`, `-image …`, `-scene-cmd …`) without `-still` does
too. `backdrop-shell reset` starts over from the defaults.

Changes apply at once to every open terminal, not only to new ones: each
running wrapper listens on a socket of its own in `$XDG_RUNTIME_DIR` (a
directory only you can enter), and `opacity`, `enable` and `reset` ask them
all to read the settings again. A terminal window opened from inside one
gets a background of its own: the marker the wrapper leaves for its shell
names its own terminal, so a new window, which inherits it, is not mistaken
for the inside of the old one.

`uninstall.sh` next to `install.sh` does the last one, also over `curl | sh`.
`disable` takes out exactly the lines `enable` put in; the rest of each file is
left byte for byte as it was. For one terminal only, `LIMONI_BACKDROP=off`
starts the plain shell.

Settings live in `~/.config/limoni/backdrop.conf`:

```ini
scene = aurora     # aurora, city, starfield, synthwave
opacity = 0.45     # 0 to 1
fps = 0            # cap the frame rate; 0 is the scene's own
still = false      # a still picture instead of an animation
image =            # a picture file instead of a scene
art =              # an ASCII art file instead of a scene; beats image
scene-cmd =        # a program that draws the scene; image and art beat it
select = true      # select with the mouse, leaving the background out
```

`backdrop-shell enable -art ~/my-art.txt` (or `-image`, `-scene-cmd`, or
`-scene`) switches between them. On the command line too, the background
named there is the one shown, whatever the settings name.

## Why a wrapper

A terminal has no layers: it is one grid of cells, and whatever writes last is
what shows. A scene drawn by another program beside the shell would paint over
your text, and your text would wipe it. So backdrop-shell runs the shell on a
pseudo-terminal, keeps an emulated screen of what it writes, and draws that
over the scene. Every cell the shell leaves without a background colour shows
the scene, faded into the terminal's own background (asked for with OSC 11);
a program that paints its own background — btop, a vim colour scheme — covers
it.

Owning the screen also lets a program keep what it printed moving after it
exits, scrolling with the output: a system fetch's 3D logo can go on spinning
while you type below it. The program marks its rows
with `OSC 7799 ; live ; id ; width ; height` and streams frames for them over
the wrapper's socket (`$LIMONI_BACKDROP_SOCKET`, readable by its user alone),
as a `-scene-cmd` program draws; `live.go` has the details.

## What it costs

What reaches the terminal is Limoni's cell diff, so a background costs only
the cells it changes. Measured behind fish in kitty at 120×40, twenty seconds
each, CPU time of all threads, as a share of one core:

| Background | backdrop-shell | kitty |
| :--- | ---: | ---: |
| a picture (`-image`) | 0.00% | 0.01% |
| aurora, `-still` | 0.01% | 1.82% |
| aurora | 0.54% | 1.26% |
| synthwave | 0.77% | 1.45% |
| `art/rain.txt` | 0.22% | 2.30% |

kitty costs about the same with the aurora still and moving: most of its
share is its own. The scene stops, and the wrapper uses nothing, while the
window is out of focus, and while a full-screen program covers the whole
scene. A still background is drawn once and costs nothing while the shell is
idle. The full table is in the guide.

## Mouse and keys

- **Drag** selects, and letting go copies — only what the shell wrote, never
  the stars or the art behind it. **Double-click** takes a word (a whole path
  or address), **triple-click** a line. Typing drops the highlight.
- **Middle-click** pastes what was copied last.
- **Ctrl+click** opens a link (OSC 8) a program printed — a fetch's distro
  name, say — in the browser, as the terminal would if it had the mouse.
  Only web addresses: anything printed, a file being cat, can carry a link.
- **The wheel**, and **Shift+PageUp / Shift+PageDown**, scroll back through
  history. The terminal's own scrollback is set aside while the wrapper runs
  (it draws on the alternate screen); typing returns to the live screen. In
  a full-screen program without mouse support (less, man) the wheel sends
  arrow keys, as terminals do.

Why the wrapper selects instead of the terminal: a terminal copies whatever
is in the cells, and the background's characters are in the cells like any
text. The copy goes through the system's clipboard tool — `wl-copy` on
Wayland, `xclip` or `xsel` on X11, `pbcopy` on macOS — onto the clipboard
(Ctrl+V) and the primary selection (middle-click in other programs), so it
works in any terminal and for any length. Over SSH, or with none of those
installed, it goes to the terminal as OSC 52, which Alacritty, kitty,
WezTerm, foot and Ghostty accept. A program that wants the mouse itself —
vim with `mouse=a`, btop — gets it while it runs. **Shift+drag** still makes
the terminal's own selection, background included; `-select=false` (or
`select = false` in the settings) leaves the mouse to the terminal
altogether.

Everything else goes to the shell as the terminal sent it.

## Resizing

Nothing the shell wrote is lost when the window changes size. A narrower
window wraps long lines onto the next row instead of cutting them, and a
wider one joins the rows it split — fastfetch's output survives a trip down
to 30 columns and back as it was. A shorter window moves the top rows into
the history instead of dropping the bottom ones, where the prompt is. A
line a program itself let wrap stays split when the window grows again;
kitty and Alacritty would join that too.

The history is kept by the wrapper rather than the emulator, so reflowing
10,000 lines of it takes about 2 ms and a drag stays smooth. `clear` erases
it, as it does in the terminal itself.

## kitty

- kitty tells programs a new window size only 0.1 s after a drag pauses
  (`resize_debounce_time`), for every program, not only this one; until
  then the part of the window that grew stays blank. With
  `resize_debounce_time 0 0` in `~/.config/kitty/kitty.conf` it follows the
  drag.
- `hide_window_decorations yes` there removes the title bar and borders
  (`titlebar-only` keeps the borders for resizing). A kitty window opened
  from inside a backdrop-shell — `kitty &` — gets a background of its own.

## Off switches

`LIMONI_BACKDROP=off` starts the plain shell. So does running backdrop-shell
inside itself, or anywhere that is not a terminal (scripts, `ssh host cmd`).
