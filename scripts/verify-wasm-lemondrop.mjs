// Smoke-tests Lemon Drop's WebAssembly build by running it, as
// verify-wasm-lemonhunt.mjs does for the other game.
//
// It boots the module under Node with a stub xterm.js bridge, starts a run,
// moves and drops pieces, pauses, and quits with Esc. Each step asserts on
// what reached the screen: the title, the panel of a run, the board in
// truecolor half blocks, the pause, and that Esc ended the program. The
// releases the page sends (CSI 1;1:3B for ↓ let go) go through as well, so
// a parser that read one as a second press would show here as a crash or a
// run that never ends.
//
//   node scripts/verify-wasm-lemondrop.mjs "$(go env GOROOT)" path/to/lemondrop.wasm

import { readFileSync } from "node:fs";
import path from "node:path";

const goroot = process.argv[2];
const wasmPath = process.argv[3];

globalThis.crypto ??= (await import("node:crypto")).webcrypto;
await import(path.join(goroot, "lib/wasm/wasm_exec.js"));

let captured = "";
globalThis.__limoni_output = (s) => {
  captured += s;
};

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
let exited = false;
const onExit = () => {
  exited = true;
  for (const timer of go._scheduledTimeouts.values()) clearTimeout(timer);
};
go.run(instance).then(onExit, onExit);

const since = (mark) => captured.slice(mark);
const send = (s) => globalThis.__limoni_input?.(s);

await sleep(600);
globalThis.__limoni_resize?.(100, 40);
await sleep(600);
const title = captured;

let mark = captured.length;
send("\r"); // ENTER plays
await sleep(700);
const run = since(mark);

mark = captured.length;
for (const key of ["\x1b[D", "\x1b[D", "\x1b[1;1:3D", "\x1b[A", " ", "\x1b[C", "\x1b[1;1:3C", "x", "z", " "]) {
  send(key);
  await sleep(60);
}
send("\x1b[B"); // ↓ held…
await sleep(300);
send("\x1b[1;1:3B"); // …and let go, as the page spells it
await sleep(300);
const played = since(mark);

mark = captured.length;
send("p");
await sleep(300);
const paused = since(mark);
send("p");
await sleep(200);

send("\x1b"); // Esc, alone, as xterm.js sends it
await sleep(800);

const checks = [
  ["alternate screen (?1049h)", title.includes("\x1b[?1049h")],
  ["truecolor SGR (38;2 / 48;2)", /\x1b\[[0-9;]*?[34]8;2;/.test(title)],
  ["half blocks (▀)", title.includes("▀")],
  ["title and panel (ENTER  play, SCORE, NEXT)", ["ENTER  play", "SCORE", "NEXT"].every((s) => title.includes(s))],
  // The panel is already on the title, so the diff does not send it again;
  // the pause below is only reachable in a run.
  ["a run after Enter redrew the board", run.length > 1000],
  ["the board moved under the keys", played.length > 500],
  ["the pause (PAUSED)", paused.includes("PAUSED")],
  ["Esc ended the program", exited],
];

console.log(`captured ${captured.length} bytes from the engine\n`);
let missing = 0;
for (const [label, ok] of checks) {
  if (!ok) missing++;
  console.log(`  ${ok ? "OK     " : "MISSING"}  ${label}`);
}
process.exit(missing === 0 ? 0 : 1);
