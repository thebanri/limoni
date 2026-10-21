// pikap: a record player in the terminal, drawn in half blocks — two
// pixels to a cell — or, with -ascii, in ASCII.
//
//	pikap                 # follow what is playing: Spotify, a browser, mpv…
//	pikap ~/Music         # play files itself, with the needle in the groove
//	pikap song.flac a/    # files and folders, in that order
//	pikap -demo           # the built-in record, needing nothing
//	pikap -sixel          # the record as a picture (Sixel or kitty's protocol)
//
// With no files it follows another player over MPRIS, the desktop's media
// interface: the record turns while it plays and coasts to a stop when it
// pauses, the arm travels across the record as the song goes on, and
// dragging the record or the arm moves the song back and forth.
//
// Given files, it plays them itself through ffmpeg, and then the record is
// a record: pausing slows the sound down with the platter, and dragging it
// scratches, backwards too.
//
// ("Pikap" is Turkish for a record player.)
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

func main() {
	fps := flag.Int("fps", 30, "frames per second")
	demo := flag.Bool("demo", false, "play the built-in demo record")
	mute := flag.Bool("mute", false, "play files without sound (the record still turns)")
	ascii := flag.Bool("ascii", false, "draw in ASCII characters rather than half blocks")
	vinyl := flag.Bool("vinyl", false, "a black record with the cover on its label")
	blocks := flag.Bool("blocks", false, "stay with half blocks even where the terminal shows pictures")
	sixel := flag.Bool("sixel", false, "draw the record as a picture where the terminal shows one (Sixel or kitty), half blocks elsewhere")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: pikap [flags] [file or folder…]\n\n")
		fmt.Fprintf(os.Stderr, "With no files, pikap follows whatever is playing (MPRIS).\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	src, err := openSource(flag.Args(), *demo, *mute)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pikap:", err)
		os.Exit(1)
	}
	u := newUI(src)
	if *ascii {
		u.glyph = asciiGlyphs
	}
	if *vinyl {
		u.disc = blackVinyl
	}
	// Where the terminal shows pictures, the record is one unless a way of
	// drawing was asked for. The terminal may say so only after the first
	// frames, so this is settled in the loop. -sixel does not force Sixel:
	// a terminal without it (kitty) printed the picture as text.
	// LIMONI_GRAPHICS=sixel does, for one that shows Sixel without saying so.
	autoPix := !*ascii && !*blocks && !*sixel
	if *sixel {
		u.glyph = sixelPixels
	}

	term, err := limoni.New()
	if err != nil {
		src.close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app := limoni.NewApp(term, limoni.WithFPS(*fps), limoni.WithTitle("pikap"))
	err = app.Run(context.Background(), func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev != nil {
			switch ev.Type {
			case limoni.EventKey:
				u.key(ev.Key)
			case limoni.EventMouse:
				u.mouse(ev.Mouse)
			}
		}
		u.tick(time.Now())
		proto := term.Capabilities().GraphicsProto
		u.pixels = proto == graphics.ProtocolSixel || proto == graphics.ProtocolKitty
		if autoPix && u.pixels {
			u.glyph, autoPix = sixelPixels, false
		}
		if cw, ch, err := term.Driver().CellPixelSize(); err == nil {
			u.cellW, u.cellH = int(cw), int(ch)
		}
		u.render(f.Buffer)
		if u.pixImg != nil {
			// In kitty, under the cells with a background of their own (the
			// help card's): at -1 the card's text was over the picture, its
			// background under it.
			u.pic.img, u.pic.z = u.pixImg, math.MinInt32/2-1
			f.RenderWidget(&u.pic, cell.Rect{X: uint16(u.dv.x0), Y: uint16(u.dv.y0), Width: uint16(u.dv.w), Height: uint16(u.dv.h)})
		}
		return !u.quit
	})
	term.Close()
	src.close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// openSource decides what the record plays: files when there are any, the
// demo when asked, and otherwise whatever other player is playing.
func openSource(args []string, demo, mute bool) (source, error) {
	var tracks []*track
	if len(args) > 0 {
		tracks = scan(args)
		if len(tracks) == 0 {
			return nil, fmt.Errorf("no audio files in %v", args)
		}
	} else if !demo {
		m, err := newMPRIS()
		if err == nil {
			return m, nil
		}
		// No session bus: play the music folder instead, or the demo.
		fmt.Fprintln(os.Stderr, "pikap: cannot follow other players:", err)
		tracks = scan([]string{musicDir()})
	}
	if len(tracks) == 0 {
		tracks = []*track{demoTrack()}
	}
	var out *output
	note := ""
	if mute {
		note = "sound off (-mute)"
	} else if out = openOutput(); out == nil {
		note = "no sound: install pw-play, pacat, aplay or sox"
	}
	return newLocal(tracks, out, note), nil
}

// picture places the deck's pixels the way widgets place theirs, through
// the draw context, which takes the z-index as given. It works the same
// against the Limoni pikap pins and against one where the frame's image
// list is private. The ui keeps one, so a frame allocates nothing.
type picture struct {
	img image.Image
	z   int
}

func (p *picture) Draw(ctx cell.Context, _ *buffer.Buffer) {
	if ctx.RegisterImage != nil && p.img != nil {
		ctx.RegisterImage(ctx.Area, p.img, p.z, false)
	}
}

func (*picture) SizeHint(cell.Rect) (uint16, uint16) { return 0, 0 }
