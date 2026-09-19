package zestapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// detailCache pretty-prints the selected line once, not on every frame.
type detailCache struct {
	line int
	text string
	ok   bool
}

func (c *detailCache) get(line int, raw string) string {
	if c.ok && c.line == line {
		return c.text
	}
	c.line, c.text, c.ok = line, describeLine(raw), true
	return c.text
}

// describeLine lays a JSON log line out one field per line, the common ones
// first; anything else is shown as it is.
func describeLine(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "{") {
		return raw
	}
	var fields map[string]any
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&fields); err != nil {
		return raw
	}
	first := []string{"time", "ts", "timestamp", "level", "lvl", "severity", "msg", "message", "error", "err"}
	rank := map[string]int{}
	for i, k := range first {
		rank[k] = i + 1
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ri, rj := rank[keys[i]], rank[keys[j]]
		switch {
		case ri != 0 && rj != 0:
			return ri < rj
		case ri != 0 || rj != 0:
			return ri != 0
		}
		return keys[i] < keys[j]
	})
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\n", k, formatValue(fields[k]))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatValue(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case nil:
		return "null"
	case map[string]any, []any:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("  ", "  ")
		_ = enc.Encode(v)
		return strings.TrimRight(buf.String(), "\n")
	}
	return fmt.Sprint(v)
}
