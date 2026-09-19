package zestapp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"
)

// generateDemo fills s with n plausible log lines — structured JSON from a
// few services, logfmt, plain text and the odd stack trace — then keeps
// adding a few dozen a second, so following and filtering can be tried
// without a real log.
func generateDemo(ctx context.Context, s *store, n int, notify func()) error {
	r := rand.New(rand.NewPCG(1, 2))
	t := time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)
	var buf []byte
	emit := func() {
		t = t.Add(time.Duration(r.IntN(40)) * time.Millisecond)
		buf = demoLine(buf[:0], r, t)
		s.add(buf)
		if buf[0] == 'p' { // a panic: follow it with a trace
			for _, frame := range []string{"\tgoroutine 1 [running]:", "\tmain.(*worker).process(0xc000123450)", "\t\t/app/worker.go:88 +0x1a4", "\tmain.main()"} {
				s.add([]byte(frame))
			}
		}
	}
	for i := 0; i < n; i++ {
		emit()
		if i%100_000 == 0 {
			notify()
		}
		if i%1_000_000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
	}
	notify()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			for k := r.IntN(3); k >= 0; k-- {
				emit()
			}
			notify()
		}
	}
}

var (
	demoServices = []string{"api", "auth", "billing", "worker", "search"}
	demoPaths    = []string{"/api/v1/items", "/api/v1/cart", "/login", "/api/v1/search", "/healthz", "/api/v1/orders/42"}
	demoUsers    = []string{"ada", "linus", "grace", "ken", "田中", "Émilie", "sören"}
)

func demoLine(b []byte, r *rand.Rand, t time.Time) []byte {
	ts := t.Format("2006-01-02T15:04:05.000Z")
	svc := demoServices[r.IntN(len(demoServices))]
	switch p := r.IntN(1000); {
	case p < 2:
		return append(b, "panic: runtime error: index out of range [5] with length 5"...)
	case p < 30:
		return fmt.Appendf(b, `{"time":"%s","level":"error","service":"%s","msg":"upstream request failed","error":"dial tcp 10.0.3.%d:5432: connect: connection refused","attempt":%d}`, ts, svc, r.IntN(250), 1+r.IntN(5))
	case p < 110:
		return fmt.Appendf(b, `{"time":"%s","level":"warn","service":"%s","msg":"slow query","duration_ms":%d,"table":"orders"}`, ts, svc, 500+r.IntN(4000))
	case p < 180:
		return fmt.Appendf(b, "time=%s level=debug service=%s msg=\"cache miss\" key=user:%d", ts, svc, r.IntN(100000))
	case p < 220:
		return fmt.Appendf(b, "%s INFO  [%s] user %s signed in from 192.168.%d.%d", ts[:19], svc, demoUsers[r.IntN(len(demoUsers))], r.IntN(255), r.IntN(255))
	default:
		status := 200
		if r.IntN(20) == 0 {
			status = 404
		}
		b = fmt.Appendf(b, `{"time":"%s","level":"info","service":"%s","msg":"request served","method":"GET","path":"%s","status":%d,"duration_ms":%d,"request_id":"`, ts, svc, demoPaths[r.IntN(len(demoPaths))], status, r.IntN(120))
		b = strconv.AppendUint(b, r.Uint64(), 36)
		return append(b, `"}`...)
	}
}
