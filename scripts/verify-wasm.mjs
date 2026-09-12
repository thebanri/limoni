// Smoke-tests the WebAssembly build by running it, not by building it.
//
// Two bugs have shipped in this demo because someone checked that it compiled
// and stopped there: it called Program.Run instead of RunTerminal and rendered
// nothing, and capability detection fell back to 16 colours because it reads
// environment variables that do not exist under js/wasm. A third followed the
// same pattern — the WASM backend registered its JS callbacks but never emitted
// the terminal setup sequence, so the browser had a blinking cursor over the
// render, no mouse reporting, and auto-wrap corrupting full-width frames.
//
// None of those are visible to the compiler. This boots the real module under
// Node with a stub xterm.js bridge, drives a resize and a keypress, and asserts
// on the bytes the engine actually emitted.
//
//   node scripts/verify-wasm.mjs "$(go env GOROOT)" path/to/limoni.wasm

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

const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);

// go.run resolves only when the Go program exits; the demo runs forever, so
// let it start, drive a couple of frames, then inspect what came out.
go.run(instance).catch(() => {});

await new Promise((r) => setTimeout(r, 400));
if (typeof globalThis.__limoni_resize === "function") {
  globalThis.__limoni_resize(100, 30);
}
await new Promise((r) => setTimeout(r, 400));
if (typeof globalThis.__limoni_input === "function") {
  globalThis.__limoni_input("2"); // switch scene
}
await new Promise((r) => setTimeout(r, 600));

const checks = [
  ["alternate screen  (?1049h)", "\x1b[?1049h"],
  ["HIDE CURSOR       (?25l)", "\x1b[?25l"],
  ["mouse any-event   (?1003h)", "\x1b[?1003h"],
  ["mouse SGR         (?1006h)", "\x1b[?1006h"],
  ["focus reporting   (?1004h)", "\x1b[?1004h"],
  ["bracketed paste   (?2004h)", "\x1b[?2004h"],
  ["auto-wrap off     (?7l)", "\x1b[?7l"],
];

console.log(`captured ${captured.length} bytes from the engine\n`);
let missing = 0;
for (const [label, seq] of checks) {
  const ok = captured.includes(seq);
  if (!ok) missing++;
  console.log(`  ${ok ? "OK     " : "MISSING"}  ${label}`);
}

const inputRegistered = typeof globalThis.__limoni_input === "function";
const resizeRegistered = typeof globalThis.__limoni_resize === "function";
console.log(`\n  ${inputRegistered ? "OK     " : "MISSING"}  __limoni_input bridge`);
console.log(`  ${resizeRegistered ? "OK     " : "MISSING"}  __limoni_resize bridge`);

// Truecolor SGR proves capability detection did not fall back to 16 colours,
// the other bug this demo shipped with.
const truecolor = /\x1b\[[0-9;]*?[34]8;2;/.test(captured);
console.log(`  ${truecolor ? "OK     " : "MISSING"}  truecolor SGR (38;2 / 48;2)`);

console.log(`\nfirst 120 bytes emitted: ${JSON.stringify(captured.slice(0, 120))}`);
process.exit(missing === 0 && inputRegistered && resizeRegistered ? 0 : 1);
