package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// agentDef is a coding agent squeeze can race: the command, and how it is
// given the task so that it starts working on it and stays interactive.
type agentDef struct {
	Name string
	Bin  string
	Args func(prompt, dir string) []string
}

// builtins are the agents squeeze knows. Each runs as the user runs it, with
// the user's own login: squeeze needs no API key of its own.
var builtins = []agentDef{
	{Name: "claude", Bin: "claude", Args: func(p, _ string) []string { return []string{p} }},
	// codex [PROMPT] opens the TUI with the prompt as its first message;
	// --cd names the workspace outright.
	// https://learn.chatgpt.com/docs/developer-commands?surface=cli
	{Name: "codex", Bin: "codex", Args: func(p, dir string) []string { return []string{"--cd", dir, p} }},
	// agy 1.4.11 starts with no workspace, its working directory included,
	// unless it is named.
	{Name: "agy", Bin: "agy", Args: func(p, dir string) []string { return []string{"--add-dir", dir, "-i", p} }},
	// -i runs the prompt, then stays interactive; the working directory is
	// the workspace. (A bare positional prompt is interactive in a TTY too.)
	// https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/cli-reference.md
	{Name: "gemini", Bin: "gemini", Args: func(p, _ string) []string { return []string{"-i", p} }},
}

// installed is the built-in agents found on PATH.
func installed() []agentDef {
	var out []agentDef
	for _, a := range builtins {
		if _, err := exec.LookPath(a.Bin); err == nil {
			out = append(out, a)
		}
	}
	return out
}

// parseAgents reads -agents: names of built-in agents, comma separated; a
// name twice races two of that agent.
func parseAgents(list string) ([]agentDef, error) {
	var out []agentDef
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		found := false
		for _, a := range builtins {
			if a.Name == name {
				if _, err := exec.LookPath(a.Bin); err != nil {
					return nil, fmt.Errorf("%s is not installed (no %q on PATH)", name, a.Bin)
				}
				out = append(out, a)
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown agent %q: squeeze knows claude, codex, agy and gemini", name)
		}
	}
	return out, nil
}

// racerNames names each racer after its agent, numbering repeats:
// claude, claude-2.
func racerNames(defs []agentDef) []string {
	seen := map[string]int{}
	names := make([]string, len(defs))
	for i, d := range defs {
		seen[d.Name]++
		names[i] = d.Name
		if n := seen[d.Name]; n > 1 {
			names[i] = fmt.Sprintf("%s-%d", d.Name, n)
		}
	}
	return names
}
