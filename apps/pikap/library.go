package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// A track is one side's song. What the file's name and folders say is
// known at once; tags and length come from ffprobe in the background and
// arrive as tagResults, which only the interface goroutine applies.
type track struct {
	path   string
	title  string
	artist string
	album  string
	num    int
	dur    float64 // seconds; 0 until known
	artKey string  // tracks with the same key share a cover
	hasPic bool    // the file carries its own picture
	demo   bool
}

var audioExt = map[string]bool{
	".mp3": true, ".flac": true, ".ogg": true, ".oga": true, ".opus": true,
	".m4a": true, ".aac": true, ".wav": true, ".aif": true, ".aiff": true,
	".wma": true, ".mka": true, ".webm": true, ".alac": true, ".ape": true,
	".wv": true, ".mpc": true,
}

const maxTracks = 5000

// musicDir is where to look when no files are given: XDG's music directory
// if it is set, ~/Music otherwise.
func musicDir() string {
	if d := os.Getenv("XDG_MUSIC_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if b, err := os.ReadFile(filepath.Join(home, ".config", "user-dirs.dirs")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "XDG_MUSIC_DIR="); ok {
				v = strings.Trim(v, `"`)
				v = strings.Replace(v, "$HOME", home, 1)
				if v != home {
					return v
				}
			}
		}
	}
	return filepath.Join(home, "Music")
}

// scan collects the tracks under the given files and directories, in
// order: a directory's files sorted by name, so an album plays as numbered.
func scan(args []string) []*track {
	var out []*track
	for _, a := range args {
		info, err := os.Stat(a)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if audioExt[strings.ToLower(filepath.Ext(a))] {
				out = append(out, fromName(a, filepath.Dir(filepath.Dir(a))))
			}
			continue
		}
		root := filepath.Clean(a)
		var paths []string
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if !d.IsDir() && audioExt[strings.ToLower(filepath.Ext(p))] && len(paths) < maxTracks {
				paths = append(paths, p)
			}
			return nil
		})
		slices.SortFunc(paths, naturalCompare)
		for _, p := range paths {
			out = append(out, fromName(p, root))
		}
	}
	if len(out) > maxTracks {
		out = out[:maxTracks]
	}
	return out
}

// fromName reads what it can from the path: "07 - Title.opus" in
// "Artist/Album/", with root the directory the search started from, which
// is not an artist.
func fromName(path, root string) *track {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	t := &track{path: path, artKey: filepath.Dir(path)}
	num, rest := leadingNumber(base)
	t.num = num
	t.title = strings.TrimSpace(rest)
	if t.title == "" {
		t.title = base
	}
	dir := filepath.Dir(path)
	if filepath.Clean(dir) != filepath.Clean(root) {
		t.album = filepath.Base(dir)
		if parent := filepath.Dir(dir); filepath.Clean(parent) != filepath.Clean(root) && strings.HasPrefix(parent, filepath.Clean(root)) {
			t.artist = filepath.Base(parent)
		}
	}
	// "Artist - Title" in a folder that did not name the artist. A numbered
	// file is an album's track, where a dash is part of the title:
	// "09 - Rondo a Capriccio - Rage over a Lost Penny".
	if t.artist == "" && num == 0 {
		if a, b, ok := strings.Cut(t.title, " - "); ok && a != "" && b != "" {
			t.artist, t.title = strings.TrimSpace(a), strings.TrimSpace(b)
		}
	}
	return t
}

// leadingNumber splits "07 - Title", "07. Title" and "07 Title".
func leadingNumber(s string) (int, string) {
	i := 0
	for i < len(s) && i < 4 && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i == len(s) {
		return 0, s
	}
	n, _ := strconv.Atoi(s[:i])
	rest := s[i:]
	trimmed := strings.TrimLeftFunc(rest, func(r rune) bool {
		return r == ' ' || r == '-' || r == '.' || r == '_' || r == ')'
	})
	if trimmed == rest {
		return 0, s // "1984" is a title, not track 1984
	}
	return n, trimmed
}

// naturalCompare orders "2 x" before "10 x".
func naturalCompare(a, b string) int {
	for a != "" && b != "" {
		da, db := unicode.IsDigit(rune(a[0])), unicode.IsDigit(rune(b[0]))
		if da && db {
			i, j := 0, 0
			for i < len(a) && unicode.IsDigit(rune(a[i])) {
				i++
			}
			for j < len(b) && unicode.IsDigit(rune(b[j])) {
				j++
			}
			na, _ := strconv.Atoi(a[:i])
			nb, _ := strconv.Atoi(b[:j])
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			a, b = a[i:], b[j:]
			continue
		}
		ca, cb := unicode.ToLower(rune(a[0])), unicode.ToLower(rune(b[0]))
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
		a, b = a[1:], b[1:]
	}
	return len(a) - len(b)
}

type tagResult struct {
	t      *track
	title  string
	artist string
	album  string
	num    int
	dur    float64
	hasPic bool
}

func (r tagResult) apply() {
	t := r.t
	if r.title != "" {
		t.title = r.title
	}
	if r.artist != "" {
		t.artist = r.artist
	}
	if r.album != "" {
		t.album = r.album
	}
	if r.num > 0 {
		t.num = r.num
	}
	if r.dur > 0 {
		t.dur = r.dur
	}
	if r.hasPic {
		t.hasPic = true
		t.artKey = t.path
	}
}

// probeAll asks ffprobe about every track, a few at a time, and sends what
// it learns to out. The current track goes first.
func probeAll(ctx context.Context, tracks []*track, first int, out chan<- tagResult) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return
	}
	order := make([]*track, 0, len(tracks))
	for i := range tracks {
		order = append(order, tracks[(first+i)%len(tracks)])
	}
	work := make(chan *track)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range work {
				if r, ok := probe(ctx, t); ok {
					select {
					case out <- r:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	for _, t := range order {
		if t.demo {
			continue
		}
		select {
		case work <- t:
		case <-ctx.Done():
		}
	}
	close(work)
	wg.Wait()
}

func probe(ctx context.Context, t *track) (tagResult, bool) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration:format_tags:stream=codec_type:stream_disposition=attached_pic:stream_tags",
		"-of", "json", t.path).Output()
	if err != nil {
		return tagResult{}, false
	}
	var v struct {
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			CodecType   string            `json:"codec_type"`
			Disposition map[string]int    `json:"disposition"`
			Tags        map[string]string `json:"tags"`
		} `json:"streams"`
	}
	if json.Unmarshal(b, &v) != nil {
		return tagResult{}, false
	}
	r := tagResult{t: t}
	r.dur, _ = strconv.ParseFloat(v.Format.Duration, 64)
	// Tag names differ in case between formats; Vorbis comments often sit
	// on the stream rather than the container.
	tags := map[string]string{}
	for _, s := range v.Streams {
		if s.CodecType == "video" && s.Disposition["attached_pic"] == 1 {
			r.hasPic = true
		}
		for k, val := range s.Tags {
			tags[strings.ToLower(k)] = val
		}
	}
	for k, val := range v.Format.Tags {
		tags[strings.ToLower(k)] = val
	}
	r.title = strings.TrimSpace(tags["title"])
	r.artist = strings.TrimSpace(tags["artist"])
	if r.artist == "" {
		r.artist = strings.TrimSpace(tags["album_artist"])
	}
	r.album = strings.TrimSpace(tags["album"])
	if n, _ := leadingNumber(tags["track"] + " "); n > 0 {
		r.num = n
	} else if n, err := strconv.Atoi(strings.SplitN(tags["track"], "/", 2)[0]); err == nil {
		r.num = n
	}
	return r, true
}
