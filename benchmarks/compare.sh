#!/usr/bin/env bash
#
# Runs every cross-framework runner three times and reports the median p50 per
# workload with its run-to-run spread.
#
# The runners take no -count flag: each executes the workload manifest once. A
# single run is not a result, so this invokes each three times, sequentially —
# running them concurrently would have them compete for the same cores and
# report contention as engine cost.
#
#   ./benchmarks/compare.sh [output-dir]
#
# Requires Go and, for the Ratatui runner, a Rust toolchain.

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

out=${1:-benchmark-results/compare}
rounds=3

mkdir -p "$out/bin" "$out/runs"
# Absolute, because `go -C` resolves -o against the directory it changed into.
out=$(cd "$out" && pwd)

echo "building runners"
go build -o "$out/bin/limoni" ./benchmarks/runners/limoni
go -C benchmarks/runners/ultraviolet build -o "$out/bin/ultraviolet" .
cargo build --release --manifest-path benchmarks/runners/ratatui/Cargo.toml

ratatui_bin=benchmarks/runners/ratatui/target/release/limoni-ratatui-runner

for round in $(seq 1 "$rounds"); do
  echo "round $round/$rounds"
  "$out/bin/limoni" -output "$out/runs/limoni-$round.json" >/dev/null
  "$ratatui_bin" "$out/runs/ratatui-$round.json"
  "$out/bin/ultraviolet" -output "$out/runs/ultraviolet-$round.json" >/dev/null
done

echo
python3 - "$out/runs" "$rounds" <<'PY'
import json, pathlib, statistics, sys

runs = pathlib.Path(sys.argv[1])
rounds = int(sys.argv[2])
impls = ["limoni", "ratatui", "ultraviolet"]


def load(name):
    per_workload = {}
    for round_ in range(1, rounds + 1):
        report = json.loads((runs / f"{name}-{round_}.json").read_text())
        for workload in report["workloads"]:
            entry = per_workload.setdefault(
                workload["spec"]["name"],
                {"p50": [], "bytes": [], "comparable": workload.get("comparable", True)},
            )
            entry["p50"].append(workload["summary"]["P50NS"] / 1000.0)
            entry["bytes"].append(workload["summary"]["BytesPerFrame"])
    return per_workload


data = {name: load(name) for name in impls}
names = [w["spec"]["name"] for w in json.loads((runs / "limoni-1.json").read_text())["workloads"]]


def spread(values):
    median = statistics.median(values)
    return (max(values) - min(values)) / median * 100 if median else 0.0


header = ["workload", "limoni", "ratatui", "ultraviolet", "R/L", "UV/L"]
rows = []
for name in names:
    row, medians = [name], {}
    for impl in impls:
        values = data[impl][name]["p50"]
        medians[impl] = statistics.median(values)
        row.append(f"{medians[impl]:.1f}us+-{spread(values):.0f}%")
    # A runner marks a workload non-comparable when a ratio against it would
    # mislead. That verdict is per-runner: Ratatui withholding `resize` says
    # nothing about whether Limoni and Ultraviolet can be compared on it.
    for impl in ("ratatui", "ultraviolet"):
        usable = data[impl][name]["comparable"] and medians["limoni"]
        row.append(f"{medians[impl] / medians['limoni']:.1f}x" if usable else "-")
    rows.append(row)

widths = [max(len(str(r[i])) for r in [header] + rows) for i in range(len(header))]
for r in [header] + rows:
    print("  ".join(str(c).ljust(widths[i]) for i, c in enumerate(r)))

varying = [
    f"{impl}/{name}: {data[impl][name]['bytes']}"
    for name in names
    for impl in impls
    if len(set(data[impl][name]["bytes"])) != 1
]
print()
print("bytes/frame varied across runs:", "; ".join(varying) if varying else "nowhere")
PY
