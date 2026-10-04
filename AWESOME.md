# 🌟 Awesome Limoni

Applications, tools, widgets and writing built with or for [**Limoni**](https://github.com/thebanri/limoni) 🍋.

The list is short on purpose: it names things that exist and says what they
are. Examples that live in this repository are listed as examples, not as
third-party projects. Yours belongs here — see [Submitting](#-submitting-your-project).

---

## 📑 Contents

- [🚀 Applications](#-applications)
- [🧪 Examples in this repository](#-examples-in-this-repository)
- [🛠️ Developer tools](#️-developer-tools)
- [🧩 Third-party widgets and addons](#-third-party-widgets-and-addons)
- [🎓 Writing](#-writing)
- [🤝 Submitting your project](#-submitting-your-project)

---

## 🚀 Applications

- **[zest](https://github.com/thebanri/limoni/tree/main/cmd/zest)** — Log viewer that follows files and pipes, colours by level, and filters a million lines without stalling.
- **[globe](https://github.com/thebanri/limoni/tree/main/apps/globe)** — A searchable, zoomable ASCII globe. `go install github.com/thebanri/limoni/apps/globe@latest`
- **[backdrop-shell](https://github.com/thebanri/limoni/tree/main/apps/backdrop-shell)** — Your shell, in any terminal, in front of an animated backdrop scene.
- **[Castle Lemonstein](https://github.com/thebanri/limoni/tree/main/apps/castle-lemonstein)** — A short raycaster game in half blocks, with a shared leaderboard.
- **[Lemon Drop](https://github.com/thebanri/limoni/tree/main/apps/lemondrop)** — Falling blocks that turn to sand.
- **[Limoni Voice](https://github.com/thebanri/limoni-voice)** — A voice-driven terminal assistant built on Limoni.

---

## 🧪 Examples in this repository

- **[3d_viewer](https://github.com/thebanri/limoni/tree/main/examples/3d_viewer)** — OBJ/STL/PLY viewer with Lambert and Gouraud shading and orbit controls.
- **[dashboard](https://github.com/thebanri/limoni/tree/main/examples/dashboard)** — Live CPU and memory sparklines, a process table and a streaming log.
- **[table_virtual](https://github.com/thebanri/limoni/tree/main/examples/table_virtual)** — A one-million-row table that only touches the visible rows.
- **[todo](https://github.com/thebanri/limoni/tree/main/examples/todo)** — A declarative (Elm architecture) todo app with tags, filters and fuzzy search.
- **[ssh_server](https://github.com/thebanri/limoni/tree/main/examples/ssh_server)** — One Limoni app per SSH session, in one process.
- **[custom_widget](https://github.com/thebanri/limoni/tree/main/examples/custom_widget)** — An analog gauge drawn with Braille arcs, dragged with the mouse: how to write your own widget.
- **[wasm](https://github.com/thebanri/limoni/tree/main/examples/wasm)** — The engine compiled to WebAssembly and drawn by xterm.js; it is the [browser playground](https://thebanri.github.io/limoni/).

---

## 🛠️ Developer tools

- **[limoni](https://github.com/thebanri/limoni/tree/main/cmd/limoni)** — `limoni new` generates a project with a test already written; `limoni doctor` shows what your terminal answered and what Limoni will use.
- **[limoni-mcp](https://github.com/thebanri/limoni/tree/main/cmd/limoni-mcp)** — Lets an MCP client (Claude Code, Cursor, …) drive a running Limoni app by widget role and label.

---

## 🧩 Third-party widgets and addons

None yet. A widget in its own module that implements `widgets.Widget` is
exactly what this section is for.

---

## 🎓 Writing

- [Every number we almost published](https://github.com/thebanri/limoni/blob/main/docs/blog/every-number-we-almost-published.md) — the benchmark mistakes caught before they shipped.
- [Ask the terminal, then measure it anyway](https://github.com/thebanri/limoni/blob/main/docs/blog/measure-the-terminal.md) — how Limoni decides what a terminal can do.
- [An AI agent drove our log viewer](https://github.com/thebanri/limoni/blob/main/docs/blog/an-agent-drove-our-log-viewer.md) — a headless Claude Code agent operating zest over MCP, and where it got stuck.
- [Architecture](https://github.com/thebanri/limoni/blob/main/docs/architecture.md) — the flat cell grid, the diff, and why the draw path does not allocate.

---

## 🤝 Submitting your project

Built something with Limoni? Add it:

1. Fork the [Limoni repository](https://github.com/thebanri/limoni).
2. Add one line to the right section of `AWESOME.md`, in alphabetical order:
   ```markdown
   - **[Project Name](https://github.com/your-name/your-project)** — What it does, in one sentence.
   ```
3. Open a pull request titled `awesome: add <project-name>`.

A screenshot or GIF in your README helps people decide to try it.
