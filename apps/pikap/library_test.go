package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFromName(t *testing.T) {
	root := "/music"
	for _, c := range []struct {
		path                 string
		num                  int
		title, artist, album string
	}{
		{"/music/Beethoven/Masterpieces/02 - Für Elise.opus", 2, "Für Elise", "Beethoven", "Masterpieces"},
		{"/music/Album/07. Song.flac", 7, "Song", "", "Album"},
		{"/music/Album/1984.mp3", 0, "1984", "", "Album"},
		{"/music/Feldup - Waters.mp3", 0, "Waters", "Feldup", ""},
		{"/music/A/B/10 Title - With Dash.ogg", 10, "Title - With Dash", "A", "B"},
		{"/music/B/09 - Rondo a Capriccio - Rage.opus", 9, "Rondo a Capriccio - Rage", "", "B"},
	} {
		tr := fromName(c.path, root)
		if tr.num != c.num || tr.title != c.title || tr.artist != c.artist || tr.album != c.album {
			t.Errorf("%s: got %d %q by %q on %q, want %d %q by %q on %q",
				c.path, tr.num, tr.title, tr.artist, tr.album, c.num, c.title, c.artist, c.album)
		}
	}
}

func TestNaturalOrder(t *testing.T) {
	got := []string{"10 b", "2 a", "1 c", "b", "A"}
	slices.SortFunc(got, naturalCompare)
	want := []string{"1 c", "2 a", "10 b", "A", "b"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestScanFindsAudioInOrder(t *testing.T) {
	dir := t.TempDir()
	album := filepath.Join(dir, "Artist", "Album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"10 - Ten.opus", "2 - Two.mp3", "cover.jpg", "notes.txt", "1 - One.FLAC"} {
		if err := os.WriteFile(filepath.Join(album, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, ".hidden", "x.mp3"), nil, 0o644)
	tr := scan([]string{dir})
	var titles []string
	for _, x := range tr {
		titles = append(titles, x.title)
	}
	if !slices.Equal(titles, []string{"One", "Two", "Ten"}) {
		t.Fatalf("got %q", titles)
	}
	if tr[0].artist != "Artist" || tr[0].album != "Album" {
		t.Fatalf("artist %q album %q", tr[0].artist, tr[0].album)
	}
}

func TestFolderCoverIsFound(t *testing.T) {
	dir := t.TempDir()
	red := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range 16 {
		red.Set(i%4, i/4, color.RGBA{220, 20, 20, 255})
	}
	f, err := os.Create(filepath.Join(dir, "Folder.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, red); err != nil {
		t.Fatal(err)
	}
	f.Close()
	_ = os.WriteFile(filepath.Join(dir, "zzz.jpg"), []byte("not a picture"), 0o644)
	img := folderPicture(dir)
	if img == nil {
		t.Fatal("Folder.png was not found or not read")
	}
	a := fromImage(img)
	if a.made || a.tint.r <= a.tint.g {
		t.Fatalf("a red cover gave tint %+v (made=%v)", a.tint, a.made)
	}
}

func TestGeneratedArtIsStableAndVaried(t *testing.T) {
	a, b, c := generatedArt("x"), generatedArt("x"), generatedArt("y")
	if a.px[1234] != b.px[1234] {
		t.Fatal("the same name painted two different covers")
	}
	same := 0
	for i := range a.px {
		if a.px[i] == c.px[i] {
			same++
		}
	}
	if same > len(a.px)/10 {
		t.Fatal("two names painted the same cover")
	}
	// Not a flat colour: some light and some dark.
	lo, hi := 1.0, 0.0
	for _, p := range a.px {
		lo, hi = min(lo, p.lum()), max(hi, p.lum())
	}
	if hi-lo < 0.3 {
		t.Fatalf("generated cover is flat: luminance %.2f…%.2f", lo, hi)
	}
}
