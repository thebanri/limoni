package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// agentDef is a coding agent squeeze can race: the command, and how it is
// given the task so that it starts working on it and stays interactive.
type agentDef struct {
	Name  string
	Bin   string
	Args  func(prompt, dir string) []string
	Color string // its maker's colour, for its card and frame
	Desc  string // a line about it on the setup screen
	// Blurb is the same, in two lines short enough for its card there.
	Blurb [2]string
	// Install installs it for this user, without root: npm's global prefix
	// is often /usr, so packages go to ~/.local, whose bin is on PATH.
	// npm 11 skips install scripts unless allowed, and opencode's picks its
	// binary: without it, opencode only prints that its postinstall did not run.
	Install string
	// Hooked agents report themselves through squeeze's hooks: finished,
	// needs you, working again. For them only those decide whether they
	// work; Enter does not (it may submit nothing, or open a menu).
	Hooked bool
	// Env, if set, is added to the agent's environment; dir is its worktree.
	Env func(dir string) []string
}

// builtins are the agents squeeze knows. Each runs as the user runs it, with
// the user's own login: squeeze needs no API key of its own.
var builtins = []agentDef{
	{Name: "claude", Bin: "claude", Blurb: [2]string{"says when done", "shows subagents"}, Desc: "Claude Code · says when it is done, shows its subagents", Install: `curl -fsSL https://claude.ai/install.sh | bash`, Color: "#D97757", Hooked: true, Args: func(p, _ string) []string { return []string{"--settings", claudeHooks, p} }},
	// codex [PROMPT] opens the TUI with the prompt as its first message;
	// --cd names the workspace outright.
	// https://learn.chatgpt.com/docs/developer-commands?surface=cli
	{Name: "codex", Bin: "codex", Blurb: [2]string{"ChatGPT login", "inside it, once"}, Desc: "Codex CLI · sign in with ChatGPT inside it, once", Install: `npm install -g --prefix "$HOME/.local" @openai/codex`, Color: "#10A37F", Args: func(p, dir string) []string { return []string{"--cd", dir, p} }},
	// agy 1.4.11 starts with no workspace, its working directory included,
	// unless it is named. It has no auto mode that runs commands unasked
	// (--mode auto only accepts edits), so it would stop at every one;
	// in its own worktree it may run them.
	// Its hooks come from a workspace's .agents/hooks.json, not from
	// flags, so the Stop hook lives in a second workspace beside the
	// worktrees, out of the agent's diff (see writeAgyHooks).
	{Name: "agy", Bin: "agy", Blurb: [2]string{"runs commands", "unasked"}, Desc: "Antigravity · install it from antigravity.google", Color: "#4285F4", Hooked: true, Args: func(p, dir string) []string {
		return []string{"--add-dir", dir, "--add-dir", agyHooksDir(filepath.Dir(filepath.Dir(dir))),
			"--dangerously-skip-permissions", "-i", p}
	}},
	// -i runs the prompt, then stays interactive; the working directory is
	// the workspace. (A bare positional prompt is interactive in a TTY too.)
	// https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/cli-reference.md
	// opencode [project] opens the TUI there; --prompt sends the task and
	// --auto lets it run commands unasked. It works with no account, on
	// opencode's free models. It reports itself through a plugin that
	// OPENCODE_CONFIG adds to the user's own config (see writeOpencodePlugin).
	{Name: "opencode", Bin: "opencode", Blurb: [2]string{"free models", "no account"}, Desc: "free models, no account · shows its subagents", Install: `npm install -g --prefix "$HOME/.local" --allow-scripts=opencode-ai opencode-ai`, Color: "#EEEEEE", Hooked: true,
		Args: func(p, dir string) []string { return []string{dir, "--auto", "--prompt", p} },
		Env: func(dir string) []string {
			return []string{"OPENCODE_CONFIG=" + filepath.Join(opencodeDir(filepath.Dir(filepath.Dir(dir))), "config.json")}
		}},
	{Name: "gemini", Bin: "gemini", Blurb: [2]string{"Google login", "inside it, once"}, Desc: "Gemini CLI · sign in with Google inside it, once", Install: `npm install -g --prefix "$HOME/.local" @google/gemini-cli`, Color: "#A87FFB", Args: func(p, _ string) []string { return []string{"-i", p} }},
}

// signalCmd is a hook command that writes an OSC 9 carrying text to the
// terminal squeeze gave the agent: hooks run without one of their own.
func signalCmd(text string) string {
	return `[ -n "$SQUEEZE_TTY" ] && printf '\033]9;` + text + `\007' > "$SQUEEZE_TTY"`
}

// claudeHooks makes Claude Code say, in every terminal, when it has finished
// (Stop), needs the user (Notification), and works again (a prompt sent, a
// tool run, as after a choice made with one key): by itself it notifies only
// in Ghostty, kitty and iTerm2, and says nothing when it starts working,
// while its screen redraws when idle too, so neither silence nor output
// tells. --settings adds to the user's settings for this session only.
var claudeHooks = func() string {
	command := func(cmd string) map[string]any {
		return map[string]any{"hooks": []map[string]string{{"type": "command", "command": cmd}}}
	}
	signal := func(text string) map[string]any { return command(signalCmd(text) + "; true") }
	// Its subagents, in the Agent tool (Task before 2.1), are reported by
	// subCmd: what each is for, its id, and when it ends.
	sub := command(subCmd)
	subTool := map[string]any{"matcher": "Agent|Task", "hooks": sub["hooks"]}
	b, _ := json.Marshal(map[string]any{"hooks": map[string]any{
		"Stop":             []any{signal("Claude has finished")},
		"Notification":     []any{signal("Claude needs you")},
		"UserPromptSubmit": []any{signal(promptMark)},
		"PostToolUse":      []any{signal(workingMark), subTool},
		"PreToolUse":       []any{subTool},
		"SubagentStart":    []any{sub},
		"SubagentStop":     []any{sub},
	}})
	return string(b)
}()

// agyHooksDir is the workspace holding agy's hooks, in squeeze's directory
// (.squeeze, excluded from git).
func agyHooksDir(squeezeDir string) string { return filepath.Join(squeezeDir, "agy-hooks") }

// writeAgyHooks writes agy's hooks, which signal squeeze as Claude Code's
// do: Stop when a turn ends, and working before each model call and after
// each tool. Without them agy says nothing, and its status line redraws
// while it waits. Hooks must print JSON.
func writeAgyHooks(squeezeDir string) error {
	handler := func(text string) map[string]string {
		return map[string]string{"type": "command", "command": signalCmd(text) + "; echo '{}'"}
	}
	b, err := json.Marshal(map[string]any{"squeeze": map[string]any{
		"Stop":          []map[string]string{handler("agy has finished")},
		"PreInvocation": []map[string]string{handler(workingMark)},
		"PostToolUse":   []map[string]any{{"matcher": "*", "hooks": []map[string]string{handler(workingMark)}}},
	}})
	if err != nil {
		return err
	}
	dir := filepath.Join(agyHooksDir(squeezeDir), ".agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "hooks.json"), b, 0o644)
}

// opencodeDir holds opencode's plugin, in squeeze's directory.
func opencodeDir(squeezeDir string) string { return filepath.Join(squeezeDir, "opencode") }

// opencodePlugin signals squeeze from opencode's events: busy is working,
// idle is finished. A subagent's session (one with a parent) going idle is
// not the agent finishing.
const opencodePlugin = `import fs from "fs"
const say = (text) => {
  const tty = process.env.SQUEEZE_TTY
  if (tty) try { fs.appendFileSync(tty, "\x1b]9;" + text + "\x07") } catch {}
}
// report tells squeeze about a subagent as Claude Code's hooks do: the
// hook input in a file of SQUEEZE_SUBS, its name on the terminal.
const report = (input) => {
  const dir = process.env.SQUEEZE_SUBS
  if (!dir) return
  const name = "oc-" + Date.now() + "-" + Math.random().toString(36).slice(2)
  try {
    fs.writeFileSync(dir + "/" + name, JSON.stringify(input))
    say("` + subMark + `" + name)
  } catch {}
}
const children = new Set()
export const Squeeze = async ({ client } = {}) => ({
  event: async ({ event }) => {
    const p = event.properties ?? {}
    if (event.type === "session.created" && p.info?.parentID) {
      children.add(p.sessionID)
      report({ hook_event_name: "SubagentStart", agent_id: p.sessionID, description: p.info.title ?? "" })
    }
    if (children.has(p.sessionID)) {
      if (event.type === "session.idle") {
        let last = ""
        try { // its last words, if this opencode's client can give them
          const res = await client.session.messages({ path: { id: p.sessionID } })
          for (const m of res.data ?? []) {
            const text = (m.parts ?? []).filter((x) => x.type === "text").map((x) => x.text).join("\n")
            if (m.info?.role === "assistant" && text) last = text
          }
        } catch {}
        report({ hook_event_name: "SubagentStop", agent_id: p.sessionID, last_assistant_message: last })
      }
      return
    }
    if (event.type === "session.status" && p.status?.type === "busy") say("` + workingMark + `")
    else if (event.type === "session.idle") say("opencode has finished")
    else if (event.type === "permission.asked" || event.type === "question.asked") say("opencode needs you")
  },
})
`

// writeOpencodePlugin writes the plugin and the config that loads it.
func writeOpencodePlugin(squeezeDir string) error {
	dir := opencodeDir(squeezeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	plugin := filepath.Join(dir, "squeeze.js")
	if err := os.WriteFile(plugin, []byte(opencodePlugin), 0o644); err != nil {
		return err
	}
	b, err := json.Marshal(map[string]any{"plugin": []string{"file://" + plugin}})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), b, 0o644)
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
			return nil, fmt.Errorf("unknown agent %q: squeeze knows claude, codex, agy, opencode and gemini", name)
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
