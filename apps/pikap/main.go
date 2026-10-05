// pikap: a record player in the terminal, drawn in half blocks — two
// pixels to a cell — or, with -ascii, in ASCII.
//
//	pikap                 # follow what is playing: Spotify, a browser, mpv…
//	pikap ~/Music         # play files itself, with the needle in the groove
//	pikap song.flac a/    # files and folders, in that order
//	pikap -demo           # the built-in record, needing nothing
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
	"os"
	"time"

	"github.com/thebanri/limoni"
)

func main() {
	fps := flag.Int("fps", 30, "frames per second")
	demo := flag.Bool("demo", false, "play the built-in demo record")
	mute := flag.Bool("mute", false, "play files without sound (the record still turns)")
	ascii := flag.Bool("ascii", false, "draw in ASCII characters rather than half blocks")
	vinyl := flag.Bool("vinyl", false, "a black record with the cover on its label")
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
		u.render(f.Buffer)
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
