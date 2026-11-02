# squeeze

Give one coding task to several agents at once — Claude Code, Codex,
Antigravity (`agy`), Gemini CLI — and keep the best answer.

Each agent works in its own git worktree, on its own branch, in its own
terminal: squeeze runs the agents you already have, as you run them, with
your own logins. No API key, no account.

```
squeeze "add a --json flag to the export command"
squeeze -agents claude,claude,agy "fix the flaky test in store_test.go"
squeeze                       # write the task and pick the agents on screen
squeeze clean                 # remove the worktrees and branches races left
```

- **Race:** every agent's screen, live. A dot says whether it is working, or
  has gone quiet and waits for you — you get a desktop notification when one
  does. `Enter` types into the agent (answer its questions, approve a
  command); `Ctrl+]` gives the keys back.
- **Compare (`c`):** what each agent changed — files, lines added and
  removed, and the diff of each file, highlighted.
- **Merge (`m`):** the agent you pick has its changes squash-merged into
  your working tree as staged changes. Nothing is committed for you.

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

- Agents that ask whether to trust a new folder ask the first time a
  worktree path is used; answer them from the race screen.
- Git-ignored `node_modules`, `.venv`, `venv` and `vendor` at the repository
  root are symlinked into each worktree, so agents can build and test without
  reinstalling. They are shared with the main tree and between racers: an
  install or upgrade in one changes them for all. Nested ones (monorepo
  packages) and build output (`target`, `dist`) are not linked.
- "Waiting" is what the agent says: a bell, an OSC 9, 777 or 99
  notification, or Gemini's "Action Required" title, until you next type to
  it. Claude Code sends one by default only in Ghostty, kitty and iTerm2
  (`preferredNotifChannel` in its settings changes that); Codex
  notifies (`tui.notification_method`, auto: OSC 9 or bell) only while it
  thinks its terminal is unfocused; for `agy` nothing is documented. An
  agent that never signals counts as waiting after 30 seconds of silence,
  which a long silent think can trigger.
