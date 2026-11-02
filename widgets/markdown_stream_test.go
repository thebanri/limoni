package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// renderRows draws src at width as tall as Rows says, and returns the rows.
func renderRows(src string, width uint16) []string {
	md := NewMarkdown(src)
	h := md.Rows(width, false)
	if h == 0 {
		return nil
	}
	area := cell.NewRect(0, 0, width, uint16(h))
	buf := buffer.NewBuffer(area)
	md.Draw(cell.NewContext(area, cell.Style{}), buf)
	rows := strings.Split(buf.Snapshot(), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return rows
}

func trimBlank(rows []string) []string {
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

// longCode is more than codeBatch lines, with blank ones and a tab among them.
const longCode = "a := 1\n\n\tb := 2\nc := 3\nd := 4\n\ne := 5\nf := 6\ng := 7\nh := 8\ni := 9\nj := 10\n"

// codeSpans lists every code line of src with its highlight spans.
func codeSpans(src string) []string {
	md := NewMarkdown(src)
	md.parse(cell.Style{}, false)
	var out []string
	for _, l := range md.cachedLines {
		if l.code != nil {
			out = append(out, fmt.Sprint(l.code.text, l.code.spans))
		}
	}
	return out
}

// A reply streamed a byte at a time, its settled blocks drawn one by one as
// a chat would commit them, must end up as the rows the whole reply draws.
func TestSettledMarkdownDrawsAsTheWhole(t *testing.T) {
	replies := []string{
		"Hello **there**.\n\nA second paragraph that is long enough to wrap at this width.\n",
		"# Plan\n\n1. Read the code\n2. Change it\n\n- [x] done\n- [ ] not yet\n\nEnd.",
		"Here:\n\n```go\nfunc main() {\n\n\tprintln(\"hi\")\n}\n```\n\nAnd after.\n",
		"```\nunclosed\n\nstill code",
		"| a | b |\n|---|---|\n| 1 | 2 |\n\n> quoted\n> more\n\n---\n\ntail",
		"### Head\ntext right under\n\n\n\nafter blank lines\n",
		"Code:\n\n```go title\n" + longCode + "```\n\nDone.\n",
		"```go\n" + longCode + "\n/* a block comment\n" + longCode + " still inside\n*/\n" + longCode + "x := `raw\n" + longCode + "`\n" + longCode + "\n",
		"~~~py\n" + longCode + "~~~\nafter",
		"````md\n" + longCode + "```\n" + longCode + "````\ntail",
		"```\n" + longCode + "still open",
		"- item\n\n    ```sh\n" + longCode + "    ```\n",
	}
	for _, reply := range replies {
		want := trimBlank(renderRows(reply, 24))
		wantSpans := codeSpans(reply)
		var gotSpans []string

		var got []string
		live := ""
		for i := range len(reply) {
			live += reply[i : i+1]
			var done string
			done, live = SettledMarkdown(live)
			if done != "" {
				got = append(got, renderRows(done, 24)...)
				gotSpans = append(gotSpans, codeSpans(done)...)
			}
		}
		got = trimBlank(append(got, renderRows(live, 24)...))
		gotSpans = append(gotSpans, codeSpans(live)...)
		if strings.Join(gotSpans, "\n") != strings.Join(wantSpans, "\n") {
			t.Errorf("reply %q: highlighting differs\nstreamed: %q\nwhole:    %q", reply, gotSpans, wantSpans)
		}

		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("reply %q\nstreamed:\n%s\nwhole:\n%s", reply, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

func TestSettledMarkdownKeepsOpenBlocks(t *testing.T) {
	for _, c := range []struct{ in, settled string }{
		{"no newline yet", ""},
		{"a line\n", ""},
		{"para\n\nnext", "para\n\n"},
		{"```go\ncode\n\nmore", ""},
		{"```go\n1\n2\n3\n4\n5\n6\n7\n8\n9\n", "```go\n1\n2\n3\n4\n5\n6\n7\n8\n```"},
		{"```go\n1\n2\n3\n4\n5\n6\n7\n/* c\n9\n", ""},
		{"```go\ncode\n```\nafter", "```go\ncode\n```"},
		{"~~~\n```\n~~~\n", ""},
		{"~~~\n```\n~~~\n\n", "~~~\n```\n~~~\n\n"},
		{"\n\n", ""},
	} {
		s, r := SettledMarkdown(c.in)
		whole := strings.TrimLeft(c.in, "\n")
		// A split fence adds a closing fence to settled and a reopening line to rest.
		joined := s + r
		if strings.HasSuffix(s, "```") || strings.HasSuffix(s, "~~~") {
			joined = s + "\n" + r // the newline after a closing fence
		}
		if strings.HasSuffix(s, "\n```") && strings.HasPrefix(r, "```go\n") {
			joined = strings.TrimSuffix(s, "```") + strings.TrimPrefix(r, "```go\n")
		}
		if s != c.settled || joined != whole {
			t.Errorf("SettledMarkdown(%q) = %q, %q; want settled %q", c.in, s, r, c.settled)
		}
	}
}
