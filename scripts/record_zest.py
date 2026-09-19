#!/usr/bin/env python3
"""Record the zest demo as an asciicast, for assets/zest.gif.

    go build -o /tmp/zest ./cmd/zest
    ZEST=/tmp/zest python3 scripts/record_zest.py /tmp/zest.cast
    agg --font-family "DejaVu Sans Mono,Noto Sans Mono CJK JP,Noto Color Emoji" \
        --font-size 14 --theme monokai --fps-cap 15 --idle-time-limit 2 \
        --last-frame-duration 2 /tmp/zest.cast assets/zest.gif

It runs zest in a pseudo-terminal and types at a human pace. REP and the
capability probe are off because the recording has no terminal to answer and
agg's emulator does not implement REP.
"""
import pty,os,time,struct,fcntl,termios,json,threading,codecs,sys
ZEST=os.environ.get('ZEST','zest')
W,H=110,26
out=open(sys.argv[1],'w')
out.write(json.dumps({"version":2,"width":W,"height":H,"env":{"TERM":"xterm-256color"}})+"\n")
pid,fd=pty.fork()
if pid==0:
    os.environ.update(TERM='xterm-256color',COLORTERM='truecolor',LIMONI_REP='0',LIMONI_PROBE='0')
    os.execvp(ZEST,[ZEST,'-demo','1000000'])
fcntl.ioctl(fd,termios.TIOCSWINSZ,struct.pack('HHHH',H,W,0,0))
t0=time.time(); dec=codecs.getincrementaldecoder('utf-8')('replace'); lock=threading.Lock()
def pump():
    while True:
        try: b=os.read(fd,65536)
        except OSError: return
        if not b: return
        s=dec.decode(b)
        if s:
            with lock: out.write(json.dumps([round(time.time()-t0,3),"o",s])+"\n")
threading.Thread(target=pump,daemon=True).start()
def key(s,wait=0.0):
    os.write(fd,s.encode()); time.sleep(wait)
def typeslow(s,gap=0.12):
    for ch in s: key(ch,gap)
time.sleep(3.0)                 # a million lines load; watch it follow
key('5',1.8)                    # errors and worse
key('/',0.4); typeslow('billing'); time.sleep(1.0); key('\r',0.8)
for _ in range(3): key('\x1b[A',0.35)
key('\r',2.6)                   # details of the chosen line
key('\x1b',3.0)                 # clear the filter: the line stays, now in context
key('\x1b',0.8)                 # close details
key('G',2.2)                    # back to following
key('q',0.8)
try: os.waitpid(pid,0)
except Exception: pass
time.sleep(0.3); out.close()
