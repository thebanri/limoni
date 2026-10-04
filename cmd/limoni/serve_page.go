package main

// servePage is the page limoni serve hands the browser: xterm.js, fitted to
// the window, talking to /ws. Output arrives as binary messages; keys go back
// as binary, resizes as {"type":"resize","cols":…,"rows":…}.
const servePage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{TITLE}} · limoni serve</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.css">
<style>
  html, body { margin: 0; height: 100%; overflow: hidden; background: #010409; color: #c9d1d9;
    font: 14px ui-sans-serif, system-ui, sans-serif; }
  body { display: flex; flex-direction: column; }
  #wrap { flex: 1; min-height: 0; padding: 6px; }
  #terminal { width: 100%; height: 100%; overflow: hidden; }
  /* A full-screen program has no scrollback to show; the wheel still
     scrolls a shell's. */
  #terminal .xterm-viewport { overflow: hidden !important; scrollbar-width: none; }
  #terminal .xterm-viewport::-webkit-scrollbar { display: none; }
  #status { display: none; padding: 8px 12px; background: #161b22; border-top: 1px solid #30363d; }
  #status button { margin-left: 8px; font: inherit; color: inherit; background: #21262d;
    border: 1px solid #30363d; border-radius: 6px; padding: 2px 10px; cursor: pointer; }
</style>
</head>
<body>
<div id="wrap"><div id="terminal"></div></div>
<div id="status"><span id="message"></span><button id="restart" type="button">Start again</button></div>
<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-unicode11@0.8.0/lib/addon-unicode11.js"></script>
<script>
(() => {
  const term = new Terminal({
    cursorBlink: true,
    scrollback: 1000,
    fontFamily: 'ui-monospace, Menlo, Monaco, "Courier New", monospace',
    fontSize: 14,
    lineHeight: 1.0,
    theme: { background: '#010409', foreground: '#c9d1d9' },
    allowProposedApi: true // the unicode API below
  });
  // xterm.js measures with Unicode 6 unless told otherwise, where 🍋 is one
  // column; the program lays it out in two, and the rest of the row shifts.
  term.loadAddon(new Unicode11Addon.Unicode11Addon());
  term.unicode.activeVersion = '11';
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(document.getElementById('terminal'));
  fit.fit();
  term.focus();

  const token = new URLSearchParams(location.search).get('token') || '';
  const scheme = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = scheme + '//' + location.host + '/ws?token=' + encodeURIComponent(token) +
    '&cols=' + term.cols + '&rows=' + term.rows;
  const ws = new WebSocket(url);
  ws.binaryType = 'arraybuffer';
  const encoder = new TextEncoder();

  ws.onmessage = (ev) => {
    term.write(typeof ev.data === 'string' ? ev.data : new Uint8Array(ev.data));
  };
  ws.onclose = () => {
    document.getElementById('message').textContent = 'The program has ended.';
    document.getElementById('status').style.display = 'block';
    fit.fit();
  };
  term.onData((data) => {
    if (ws.readyState === WebSocket.OPEN) ws.send(encoder.encode(data));
  });
  term.onBinary((data) => {
    if (ws.readyState !== WebSocket.OPEN) return;
    const bytes = new Uint8Array(data.length);
    for (let i = 0; i < data.length; i++) bytes[i] = data.charCodeAt(i) & 0xff;
    ws.send(bytes);
  });
  term.onResize(({ cols, rows }) => {
    if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'resize', cols, rows }));
  });
  new ResizeObserver(() => fit.fit()).observe(document.getElementById('terminal'));
  document.getElementById('restart').onclick = () => location.reload();
})();
</script>
</body>
</html>
`
