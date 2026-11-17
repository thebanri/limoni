package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/cell"
)

// subagent is one an agent started: Claude Code's Agent tool, an opencode
// child session. The agent's hooks report it (see subCmd).
type subagent struct {
	tool       string // Claude's tool_use_id, which ties PreToolUse to PostToolUse
	id         string // its own id: SubagentStart's and SubagentStop's agent_id
	kind       string // general-purpose, Explore, …
	desc       string
	prompt     string
	result     string
	transcript string // its JSONL, read to show what it is doing
	async      bool   // it runs in the background, past the main agent's turn
	start, end time.Time
}

func (s *subagent) running() bool { return s.end.IsZero() }

// subMark starts the OSC 9 that names a hook's input file to squeeze, and
// promptMark the one Claude Code's UserPromptSubmit sends.
const (
	subMark    = "squeeze:sub:"
	promptMark = "squeeze:prompt"
)

// subCmd is a hook command that keeps the hook's input, a JSON object, in
// the racer's SQUEEZE_SUBS directory and names the file on squeeze's
// terminal: the input is too long for an OSC.
const subCmd = `[ -n "$SQUEEZE_SUBS" ] && f=$(mktemp "$SQUEEZE_SUBS/h.XXXXXX") && cat > "$f" && printf '\033]9;` + subMark + `%s\007' "${f##*/}" > "$SQUEEZE_TTY"; true`

// hookInput is what Claude Code gives its hooks, and what the opencode
// plugin writes in its image.
type hookInput struct {
	Event      string `json:"hook_event_name"`
	ToolUseID  string `json:"tool_use_id"`
	Transcript string `json:"transcript_path"`
	AgentID    string `json:"agent_id"`
	AgentType  string `json:"agent_type"`
	Last       string `json:"last_assistant_message"`
	Desc       string `json:"description"` // opencode's session title
	Input      struct {
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
		Type        string `json:"subagent_type"`
	} `json:"tool_input"`
	Response json.RawMessage `json:"tool_response"`
}

// subEvent takes in the hook input saved as name in the racer's subs
// directory. Claude Code reports a subagent as PreToolUse (what it is for),
// SubagentStart (its id), PostToolUse (done, or launched to the background)
// and SubagentStop (done, with its last message).
func (r *racer) subEvent(name string) {
	if r.subsDir == "" || name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return
	}
	path := filepath.Join(r.subsDir, name)
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	var in hookInput
	if err != nil || json.Unmarshal(data, &in) != nil {
		return
	}
	now := time.Now()
	r.subMu.Lock()
	defer r.subMu.Unlock()
	find := func(match func(*subagent) bool) *subagent {
		for i := len(r.subs) - 1; i >= 0; i-- {
			if match(r.subs[i]) {
				return r.subs[i]
			}
		}
		return nil
	}
	switch in.Event {
	case "PreToolUse":
		r.subs = append(r.subs, &subagent{tool: in.ToolUseID, kind: in.Input.Type, desc: in.Input.Description,
			prompt: in.Input.Prompt, start: now})
	case "SubagentStart":
		// Its id, unless PostToolUse gave it first (a background subagent).
		s := find(func(s *subagent) bool { return s.id == in.AgentID })
		if s == nil {
			s = find(func(s *subagent) bool {
				return s.id == "" && s.running() && strings.EqualFold(s.kind, in.AgentType)
			})
		}
		if s == nil {
			if in.AgentType == "" && in.Desc == "" {
				return // Claude Code's own helpers start and stop untyped, outside any Agent call
			}
			s = &subagent{kind: in.AgentType, desc: in.Desc, start: now}
			r.subs = append(r.subs, s)
		}
		s.id = in.AgentID
		if in.Transcript != "" && in.AgentID != "" {
			s.transcript = strings.TrimSuffix(in.Transcript, ".jsonl") + "/subagents/agent-" + in.AgentID + ".jsonl"
		}
	case "PostToolUse":
		var resp struct {
			AgentID string          `json:"agentId"`
			Async   bool            `json:"isAsync"`
			Content json.RawMessage `json:"content"`
		}
		_ = json.Unmarshal(in.Response, &resp)
		s := find(func(s *subagent) bool { return s.tool == in.ToolUseID })
		if s == nil {
			return
		}
		if s.id == "" {
			s.id = resp.AgentID
		}
		s.async = resp.Async
		if !resp.Async && s.running() {
			s.end = now
		}
		if !resp.Async && s.result == "" {
			s.result = contentText(resp.Content)
		}
	case "SubagentStop":
		s := find(func(s *subagent) bool { return s.id == in.AgentID })
		if s == nil {
			return // one of Claude Code's own helpers, or one never seen starting
		}
		if s.running() {
			s.end = now
		}
		if in.Last != "" {
			s.result = in.Last
		}
	}
}

// promptSent ends the subagents of the last turn that ran in it: a new
// prompt means the turn is over, even one interrupted before their hooks ran.
func (r *racer) promptSent() {
	r.subMu.Lock()
	defer r.subMu.Unlock()
	for _, s := range r.subs {
		if s.running() && !s.async {
			s.end = time.Now()
		}
	}
}

// background reports whether a subagent the agent sent to the background is
// still at work, which keeps the racer working after its own turn ended.
func (r *racer) background() bool {
	r.subMu.Lock()
	defer r.subMu.Unlock()
	for _, s := range r.subs {
		if s.running() && s.async {
			return true
		}
	}
	return false
}

// subagents is a copy of the racer's subagents, safe to read while the read
// loop adds to them.
func (r *racer) subagents() []subagent {
	r.subMu.Lock()
	defer r.subMu.Unlock()
	out := make([]subagent, len(r.subs))
	for i, s := range r.subs {
		out[i] = *s
	}
	return out
}

// contentText is the text of a tool result: a string, or text blocks.
func contentText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct{ Type, Text string }
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// row is a line of text to draw, in a style.
type row struct {
	text  string
	style limoni.Style
}

// transcriptTail is how much of a transcript's end is read for its activity.
const transcriptTail = 256 << 10

// activity is what a subagent has done, from the end of its transcript:
// what it said, the tools it ran, and the first line of what they gave back;
// and the report it handed back, which Claude Code 2.1 subagents give in a
// SubagentHandback call rather than as their last words.
func activity(path string) (rows []row, answer string) {
	if !strings.HasSuffix(path, ".jsonl") {
		return nil, ""
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, ""
	}
	defer f.Close()
	var off int64
	if fi, err := f.Stat(); err == nil && fi.Size() > transcriptTail {
		off = fi.Size() - transcriptTail
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil, ""
	}
	data, _ := io.ReadAll(f)
	if off > 0 { // the first line is cut
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		var e struct {
			Type    string
			Message struct{ Content json.RawMessage }
		}
		if json.Unmarshal(line, &e) != nil {
			continue
		}
		var blocks []struct {
			Type    string
			Text    string
			Name    string
			Input   map[string]any
			Content json.RawMessage
		}
		if json.Unmarshal(e.Message.Content, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			switch {
			case e.Type == "assistant" && b.Type == "text":
				rows = append(rows, row{text: b.Text})
			case e.Type == "assistant" && b.Type == "tool_use":
				rows = append(rows, row{text: "→ " + b.Name + toolSummary(b.Input), style: cyan})
				if msg, ok := b.Input["message"].(string); ok && b.Name == "SubagentHandback" {
					answer = msg
				}
			case e.Type == "user" && b.Type == "tool_result":
				out, _, _ := strings.Cut(strings.TrimSpace(contentText(b.Content)), "\n")
				rows = append(rows, row{text: "  " + out, style: dim})
			}
		}
	}
	return rows, answer
}

// toolSummary is the part of a tool call's input that says what it does.
func toolSummary(in map[string]any) string {
	for _, k := range []string{"description", "command", "file_path", "pattern", "path", "url", "query", "prompt"} {
		if s, ok := in[k].(string); ok && s != "" {
			s, _, _ = strings.Cut(s, "\n")
			return ": " + s
		}
	}
	return ""
}

// wrap breaks text into lines no wider than w columns, between words where
// it can.
func wrap(text string, w int) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\t", "    "), "\n") {
		for {
			if cell.StringWidth(line) <= w || w < 1 {
				out = append(out, line)
				break
			}
			cut, width := 0, 0
			for i, r := range line {
				rw := cell.StringWidth(string(r))
				if width+rw > w {
					cut = i
					break
				}
				width += rw
			}
			if cut == 0 { // one character wider than the line
				_, cut = utf8.DecodeRuneInString(line)
			}
			if sp := strings.LastIndexByte(line[:cut], ' '); sp > 0 { // between words, if it can
				out = append(out, line[:sp])
				line = line[sp+1:]
				continue
			}
			out = append(out, line[:cut])
			line = line[cut:]
		}
	}
	return out
}
