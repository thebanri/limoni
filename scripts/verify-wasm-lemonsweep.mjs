// Smoke-tests Lemon Sweep's WebAssembly build by running it, as
// verify-wasm-lemonsnake.mjs does for Lemon Snake.
//
// It boots the module under Node with a stub xterm.js bridge and a stub Web
// Audio context, starts a run with Enter, then plays it the way the page's
// pointer lock does: the lock taken, movement, a click that fires, the lock
// let go (which pauses), and QUIT from the pause. Each step asserts on what
// reached the screen or the speakers.
//
//   node scripts/verify-wasm-lemonsweep.mjs "$(go env GOROOT)" path/to/lemonsweep.wasm

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

// Just enough of an AudioContext to count what the game does with it.
const audio = { buffers: 0, started: 0 };
const node = () => ({ connect: () => {}, gain: {}, pan: {} });
globalThis.__limoni_audio = {
  destination: {},
  createBuffer: () => {
    audio.buffers++;
    return { copyToChannel: () => {} };
  },
  createBufferSource: () => ({ ...node(), start: () => audio.started++ }),
  createGain: node,
  createStereoPanner: node,
};

// The page's side of the pointer lock: the game asks for it through
// __lemonsweep_requestLock and lets it go through document.exitPointerLock.
let lockRequests = 0;
let lockExits = 0;
globalThis.__lemonsweep_requestLock = () => lockRequests++;
globalThis.document = { exitPointerLock: () => lockExits++ };

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
const pointer = (...args) => globalThis.__lemonsweep_pointer?.(...args);

await sleep(600);
globalThis.__limoni_resize?.(120, 36);
await sleep(900);
const title = captured;

let mark = captured.length;
send("\r"); // Enter starts a run
await sleep(700);
const run = since(mark);
const wanted = globalThis.__lemonsweep_wantLock === true;
const asked = lockRequests > 0;

pointer("lock", true); // the page took the lock
pointer("move", 240, 0); // a turn
await sleep(300);
const soundsBefore = audio.started;
mark = captured.length;
pointer("button", 0, true); // fire
await sleep(500);
pointer("button", 0, false);
await sleep(200);
const fired = audio.started > soundsBefore;
const firing = since(mark);

mark = captured.length;
pointer("lock", false); // Esc let the pointer go: the run pauses
await sleep(500);
const paused = since(mark);
const stillRunning = !exited;

send("s"); // down to RECIPES
await sleep(100);
send("s"); // down to SETTINGS
await sleep(100);
send("s"); // down to QUIT
await sleep(100);
send("\r");
await sleep(800);

const checks = [
  ["alternate screen (?1049h)", title.includes("\x1b[?1049h")],
  ["truecolor SGR (38;2 / 48;2)", /\x1b\[[0-9;]*?[34]8;2;/.test(title)],
  ["the title (the garbage collector, SWEEP, the frameworks)", ["garbage collector", "SWEEP", "Bubble Tea", "Ratatui"].every((s) => title.includes(s))],
  ["a run after Enter: the HUD (THE HEAP, SWEPT, WARDEN)", ["THE HEAP", "SWEPT", "WARDEN"].every((s) => run.includes(s))],
  ["the run wants the pointer locked, and asked for it", wanted && asked],
  ["a click on the locked pointer fired (a sound, the frame redrawn)", fired && firing.length > 500],
  ["letting the pointer go pauses the run (the pause card: stopped, RESUME)", paused.includes("stopped") && paused.includes("RESUME") && stillRunning],
  ["clips loaded into Web Audio", audio.buffers > 0],
  ["QUIT from the pause ended the program", exited],
];

console.log(`captured ${captured.length} bytes from the engine\n`);
let missing = 0;
for (const [label, ok] of checks) {
  if (!ok) missing++;
  console.log(`  ${ok ? "OK     " : "MISSING"}  ${label}`);
}
console.log(`\n  Web Audio: ${audio.buffers} clips, ${audio.started} played; lock asked ${lockRequests}, let go ${lockExits}`);
process.exit(missing === 0 ? 0 : 1);
