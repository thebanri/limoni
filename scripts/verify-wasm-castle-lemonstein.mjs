// Smoke-tests Castle Lemonstein's WebAssembly build by running it, as
// verify-wasm.mjs does for the playground.
//
// It boots the module under Node with a stub xterm.js bridge and a stub Web
// Audio context, types a name, starts a game, and quits with Esc. Each step
// asserts on what the engine did: the name entry (Node has no localStorage,
// so nothing is remembered), the title and the HUD reached the screen in
// truecolor, the
// synthesised clips were handed to Web Audio and one was played, and Esc
// ended the program. Esc is the one that had a bug behind it — the browser
// never delivered a lone ESC — so the exit is checked, not assumed.
//
//   node scripts/verify-wasm-castle-lemonstein.mjs "$(go env GOROOT)" path/to/castle-lemonstein.wasm

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

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
let exited = false;
const onExit = () => {
  exited = true;
  // wasm_exec.js leaves the timers of sleeping goroutines set, and they
  // throw into the dead instance; the page clears them the same way.
  for (const timer of go._scheduledTimeouts.values()) clearTimeout(timer);
};
go.run(instance).then(onExit, onExit);

await sleep(600);
globalThis.__limoni_resize?.(100, 30);
await sleep(600);
const asked = captured;

globalThis.__limoni_input?.("Tester\r"); // the name, then on to the title
await sleep(800);
const title = captured.slice(asked.length);

globalThis.__limoni_input?.("\r"); // ENTER CASTLE
await sleep(800);
const game = captured.slice(asked.length + title.length);

globalThis.__limoni_input?.("\x1b"); // Esc, alone, as xterm.js sends it
await sleep(800);

const checks = [
  ["alternate screen (?1049h)", asked.includes("\x1b[?1049h")],
  ["truecolor SGR (38;2 / 48;2)", /\x1b\[[0-9;]*?[34]8;2;/.test(asked)],
  ["half blocks (▀)", asked.includes("▀")],
  ["name entry first (NAME YOUR KNIGHT)", asked.includes("NAME YOUR KNIGHT")],
  ["title menu with the name (ENTER CASTLE, Tester)", title.includes("ENTER CASTLE") && title.includes("Tester")],
  ["HUD after Enter (LEMONS)", game.includes("LEMONS")],
  ["clips loaded into Web Audio", audio.buffers > 0],
  ["a sound played", audio.started > 0],
  ["Esc ended the program", exited],
];

console.log(`captured ${captured.length} bytes from the engine\n`);
let missing = 0;
for (const [label, ok] of checks) {
  if (!ok) missing++;
  console.log(`  ${ok ? "OK     " : "MISSING"}  ${label}`);
}
console.log(`\n  Web Audio: ${audio.buffers} clips, ${audio.started} played`);
process.exit(missing === 0 ? 0 : 1);
