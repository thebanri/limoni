//go:build unix

package main

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

// A sixel picture goes to a kitty terminal as a placement where the sixel
// would be, is cut as text covers its last row, and goes when written over;
// the program's own kitty commands pass through untouched.
func TestKittyOut(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	emu := vt.NewEmulator(80, 24)
	go io.Copy(io.Discard, emu)
	s := &session{out: w, emu: emu, cols: 80, rows: 24, cellW: 10, cellH: 20, pixels: true,
		kout: &kittyOut{placed: map[*picture]*kittyPlaced{}}}
	s.pics = pictures{emu: emu, byID: map[int]*picture{}}
	out := func(in string) string {
		t.Helper()
		if err := s.output([]byte(in)); err != nil {
			t.Fatal(err)
		}
		r.SetReadDeadline(time.Now().Add(time.Second))
		b := make([]byte, 1<<16)
		n, _ := r.Read(b)
		return string(b[:n])
	}

	got := out("\x1b[3;5H\x1bP0;1;0q\"1;1;20;45#1;2;100;0;0#1!20~-!20~-!20~-!20~-!20~-!20~-!20~-!20~\x1b\\")
	id := "i=2113929217" // kittyOutBase+1
	for _, want := range []string{"\x1b_Ga=t,f=32,o=z,q=2," + id + ",s=20,v=48,m=0;",
		"\x1b[3;5H\x1b_Ga=p,p=1,q=2,C=1," + id + ",c=2,r=3,h=48\x1b\\\x1b[5;5H"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sixel sent as\n%q\nwant it to hold %q", got, want)
		}
	}
	if strings.Contains(got, "\x1bP") || strings.ContainsRune(got, '▀') {
		t.Fatal("the sixel itself, or half blocks, went to the terminal too")
	}
	if got := out("\r\n"); strings.Contains(got, "\x1b_G") {
		t.Fatalf("an unchanged picture was sent again: %q", got)
	}
	if got := out("\x1b[5;1H$ ls -la"); !strings.Contains(got, "\x1b[3;5H\x1b_Ga=p,p=1,q=2,C=1,"+id+",c=2,r=2,h=32\x1b\\") {
		t.Fatalf("a prompt over its last row did not cut it: %q", got)
	}
	if got := out("\x1b[3;5Hx"); !strings.Contains(got, "\x1b_Ga=d,d=I,q=2,"+id+"\x1b\\") {
		t.Fatalf("written over in its first row, it stayed: %q", got)
	}
	own := "\x1b_Ga=T,f=100,i=5;AAAA\x1b\\"
	if got := out(own); got != own {
		t.Fatalf("the program's kitty command came out as %q", got)
	}
}

func TestKittyAnswer(t *testing.T) {
	if ok, rest := kittyAnswer([]byte("a\x1b_Gi=31;OK\x1b\\b")); !ok || string(rest) != "ab" {
		t.Fatalf("OK: %v %q", ok, rest)
	}
	if ok, rest := kittyAnswer([]byte("a\x1b_Gi=31;EINVAL:bad\x1b\\")); ok || string(rest) != "a" {
		t.Fatalf("an error: %v %q", ok, rest)
	}
	if ok, rest := kittyAnswer([]byte("x")); ok || string(rest) != "x" {
		t.Fatalf("no answer: %v %q", ok, rest)
	}
}
