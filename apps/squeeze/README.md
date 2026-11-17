# squeeze

Give one coding task to several agents at once — Claude Code, Codex,
Antigravity (`agy`), opencode, Gemini CLI — and keep the best answer.
opencode needs no account: it has free models of its own.

`agy` is started with `--dangerously-skip-permissions` and opencode with
`--auto`: agy has no mode that runs commands without asking, and in its own
worktree that is what a race needs. The others run with the permissions you gave them.

Each agent works in its own git worktree, on its own branch, in its own
terminal: squeeze runs the agents you already have, as you run them, with
your own logins. No API key, no account.

```
squeeze "add a --json flag to the export command"
squeeze -agents claude,claude,agy "fix the flaky test in store_test.go"
squeeze -check "go test ./..." "make the export streaming"
squeeze -check "npm test" "…"  # whatever tells your project's good code from bad
squeeze                       # write the task and pick the agents on screen
                              # (a click on an agent's card adds one, − takes one away)
squeeze clean                 # remove the worktrees and branches races left
```

- **Race:** every agent's screen, live, and how long each has worked (the
  clock stops while it waits for you). A dot says whether it is working, or
  has gone quiet and waits for you — you get a desktop notification when one
  does. `Enter` or a click on its screen types into the agent (answer its
  questions, approve a command); `F12`, `Ctrl+]` or a click on the list gives
  the keys back. An agent that asked for the mouse gets it — Claude Code
  scrolls its own history with the wheel; for one that did not, the wheel
  scrolls back through what it printed. Dragging over an agent that did not take
  the mouse (agy) selects its text, and letting go copies it (OSC 52); in
  one that did, Shift+drag is the terminal's own selection. A click on a web
  address opens it in your browser (Ctrl+click in an agent that took the
  mouse), wrapped over rows or not. Shift+Enter is a new line in the prompt;
  Ctrl and Shift with arrows, Delete, Home and the F keys reach the agent as
  xterm sends them.
- **Jobs:** `+ New job` (or `n`, or the `+ New job` card under the
  others) opens the setup screen beside the cards, on the task shown. Keep the task
  and the agents you pick join its race; write another and it is a new
  job, raced beside the others in the background. Cards name their job
  (`#2 …`) once there is more than one, and the header shows the task of the
  one selected.
- **Quitting:** `q` (or `Quit` at the top) asks in a box over the dimmed
  screen; yes stops the agents, shows each one stopping, and the box shrinks
  shut. Merging and installing ask the same way.
- **Agents not installed** are on the setup screen too, with an
  `install` button: it shows the command and asks first, then installs for
  you alone, under `~/.local` (npm for codex, opencode and gemini, Claude's
  own installer for claude; agy comes from antigravity.google). codex and
  gemini ask you to sign in, inside their screen, the first time.
- **Subagents:** when Claude Code (its Agent tool) or opencode (a child
  session) starts a subagent, it is listed under the agent's card, ● at work
  or ✓ done, with how long it ran. A click on it shows what it was asked,
  what it does — the tools it runs, live, from its transcript — and its
  answer. A subagent sent to the background keeps its agent working after
  the agent's own turn ends.
- **Check (`-check`):** a shell command run in an agent's worktree each
  time it stops working (and on `t`); ✓ in the sidebar if it exits 0, ✗ if
  not. It is not run again on a worktree it has already checked unchanged.
- **Shared notes (`-notes`):** off by default, since agents that read each
  other drift towards one answer. With it, each agent is told that others
  work on the same task and to write what it learns — tried, worked, dead
  ends — to `.squeeze-notes/<its name>.md` in its worktree. Every two seconds
  squeeze copies that file into the `.squeeze-notes/` of the other agents on
  the same job, so each reads the rest. Each writes only its own file: a
  copy it edits is overwritten. The directory is excluded from git, so notes
  are never part of a diff or a merge.
- **Compare (`c`):** what each agent changed — files, lines added and
  removed, and the diff of each file, highlighted; with `-check`, whether it
  passes, and `o` shows the check's output.
- **Merge (`m`):** the agent you pick has its changes squash-merged into
  your working tree as staged changes. Nothing is committed for you. An agent
  that fails the check asks once more.

Worktrees live in `.squeeze/worktrees/<agent>` inside the repository, the
same path in every race, so an agent that remembers trust per folder asks
once. A new race commits what an earlier one left uncommitted there to that
race's branch (`squeeze/<race>/<agent>`) before it reuses the path; `.squeeze/` is listed in
`.git/info/exclude`, not in your `.gitignore`. They start from `HEAD`:
uncommitted changes are not in them.

## Install

```
go install github.com/thebanri/limoni/apps/squeeze@latest
```

Linux and macOS (it needs a pseudo-terminal per agent).

## Known limits

- One race per repository at a time: a second squeeze there would take
  over the worktrees the first one's agents are working in, so it refuses.

- Agents that ask whether to trust a new folder ask the first time a
  worktree path is used; answer them from the race screen.
- Git-ignored `node_modules`, `.venv`, `venv` and `vendor` at the repository
  root are symlinked into each worktree, so agents can build and test without
  reinstalling. They are shared with the main tree and between racers: an
  install or upgrade in one changes them for all. Nested ones (monorepo
  packages) and build output (`target`, `dist`) are not linked.
- "Waiting" is what the agent says: a bell, an OSC 9, 777 or 99
  notification, or Gemini's "Action Required" title, until you next type to
  it. Claude Code is started with two hooks of its own (`--settings`, for
  that session only, beside yours): Stop and Notification signal squeeze the
  moment it finishes or needs you, in any terminal. `agy` gets a Stop hook
  too, from `.squeeze/agy-hooks/.agents/hooks.json`, a second workspace it
  is given beside its worktree, and opencode a plugin,
  `.squeeze/opencode/squeeze.js`, added to your config by `OPENCODE_CONFIG`
  (busy is working, idle is finished; a subagent's idle is not). All three also report working again through
  their hooks, and only the hooks decide for them: typing, clicking into
  the screen or Enter on an empty prompt is not the agent working. For the
  others, Enter is; Codex
  notifies (`tui.notification_method`, auto: OSC 9 or bell) only while it
  thinks its terminal is unfocused. An
  agent that never signals counts as waiting after 30 seconds of silence,
  which a long silent think can trigger.
- The check runs in the worktree as it is when the agent stops, on your
  machine, with your environment: the same command in every worktree at
  once, so checks that bind a fixed port or share a database collide.
  Answering a question the agent asked starts it working again, and the
  next stop checks again.
