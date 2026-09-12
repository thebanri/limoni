// Benchmark runner for Ratatui.
//
// # What this measures, and why it was rebuilt
//
// The first version of this runner drove `TestBackend` and derived
// `BytesPerFrame` from `cell.symbol().len() + 10` over every cell of the
// buffer — a fabricated constant (21,120 bytes for an empty 80x24 frame),
// not anything Ratatui would emit. The Limoni runner reports the real output
// of `buffer.Diff`, so the two byte columns described different quantities and
// were never comparable.
//
// This version drives `CrosstermBackend` over an in-memory `Sink` behind a
// fixed viewport, so the measured pipeline is: widgets draw into a cell buffer,
// Ratatui diffs against the previous frame, and the backend encodes the result
// as ANSI into the sink. That is the same pipeline the Limoni runner measures —
// `buffer.Diff` also syncs its back buffer, so both sides diff against the
// frame they last emitted. `Viewport::Fixed` keeps `Terminal` from querying a
// terminal size, so no tty is required.
//
// Four further corrections:
//
//   - Fixtures are built once. The old runner cloned a 10,000-element
//     `Vec<Row>` *inside* the timed region and rebuilt `"A".repeat(4800)` every
//     frame, so `table-10000` reported 80,049 allocations per frame and a p50
//     of 2.1 ms that was mostly memcpy.
//   - `table-10000` scrolls. The methodology requires advancing the selection
//     (`(i * 7) % N`); the old runner re-rendered a static window.
//   - Text workloads wrap, matching the Limoni runner's `Wrap: true`.
//   - `full-redraw-120x40` and `hundred-layers` now render what the Limoni
//     runner renders. Both were static here and dynamic there: full-redraw drew
//     one unchanging glyph, so Ratatui's diff emitted 25 bytes a frame against
//     Limoni's 4897, and hundred-layers drew a borderless, titleless
//     `Block::default()` — that is, nothing — at fixed positions. Correcting
//     the latter reversed its result: Ratatui is faster than Limoni there.
//
// # What is still not like-for-like
//
// `table-10000` is a structural comparison, not a like-for-like one, and is
// marked `comparable: false` in the output. Limoni's `Table` holds its rows and
// redraws the visible window; Ratatui's `Table` takes ownership of an iterator
// of `Row`, so an application rebuilds its rows every frame. Constructing rows
// per frame is idiomatic Ratatui, not a harness artifact — but it is a
// different amount of work, and quoting a ratio would hide that.
//
// The workloads with no Ratatui equivalent at all — `empty-frame`,
// `mouse-hit-test`, `async-update-burst`, `native-image-capability` — are
// likewise marked `comparable: false`.

use ratatui::{
    backend::CrosstermBackend,
    layout::{Constraint, Rect},
    style::Color,
    widgets::{Block, Cell, Paragraph, Row, Table, TableState, Wrap},
    Frame, Terminal, TerminalOptions, Viewport,
};
use serde::{Deserialize, Serialize};
use std::alloc::{GlobalAlloc, Layout, System};
use std::cell::RefCell;
use std::io::Write;
use std::rc::Rc;
use std::sync::atomic::{AtomicU64, Ordering};
use std::{env, fs, io, time::Instant};

struct Counter;
static ALLOCATED: AtomicU64 = AtomicU64::new(0);
static ALLOCS: AtomicU64 = AtomicU64::new(0);

unsafe impl GlobalAlloc for Counter {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        let ret = System.alloc(layout);
        if !ret.is_null() {
            // Relaxed is sufficient for a counter nothing synchronises on, and
            // keeps the allocator hook from serialising the measured region.
            ALLOCATED.fetch_add(layout.size() as u64, Ordering::Relaxed);
            ALLOCS.fetch_add(1, Ordering::Relaxed);
        }
        ret
    }
    unsafe fn dealloc(&self, ptr: *mut u8, layout: Layout) {
        System.dealloc(ptr, layout);
    }
}

#[global_allocator]
static A: Counter = Counter;

const RUNNER_VERSION: &str = "v2.0.0";
const WARMUP: usize = 10;

/// Workloads Ratatui has no equivalent for, or that exercise a structurally
/// different amount of work. Reported, never quoted as a ratio.
const NON_COMPARABLE: &[&str] = &[
    "empty-frame",
    "mouse-hit-test",
    "async-update-burst",
    "native-image-capability",
    "table-10000",
];

#[derive(Serialize, Deserialize, Clone)]
struct Spec {
    name: String,
    width: u16,
    height: u16,
    #[serde(default)]
    rows: usize,
    #[serde(default)]
    unicode: bool,
    #[serde(default, rename = "full_draw")]
    full_draw: bool,
    #[serde(default)]
    mouse: bool,
    #[serde(default, rename = "async_burst")]
    async_burst: usize,
    #[serde(default, rename = "output_mode")]
    output_mode: String,
    #[serde(default, rename = "color_mode")]
    color_mode: String,
    iterations: usize,
}

#[derive(Serialize)]
struct Summary {
    #[serde(rename = "Frames")]
    frames: usize,
    #[serde(rename = "P50NS")]
    p50_ns: u128,
    #[serde(rename = "P95NS")]
    p95_ns: u128,
    #[serde(rename = "P99NS")]
    p99_ns: u128,
    #[serde(rename = "MinNS")]
    min_ns: u128,
    #[serde(rename = "MaxNS")]
    max_ns: u128,
    #[serde(rename = "MeanNS")]
    mean_ns: u128,
    #[serde(rename = "StdDevNS")]
    std_dev_ns: u128,
    #[serde(rename = "BytesPerFrame")]
    bytes_per_frame: f64,
    #[serde(rename = "AllocBytes")]
    alloc_bytes: u64,
    #[serde(rename = "Allocs")]
    allocs: u64,
}

#[derive(Serialize)]
struct Workload {
    spec: Spec,
    summary: Summary,
    /// False when a ratio against another implementation would be misleading.
    comparable: bool,
}

#[derive(Serialize)]
struct EnvMetadata {
    os: String,
    arch: String,
    go: Option<String>,
    cpu: Option<String>,
    output: Option<String>,
    #[serde(rename = "manifest_hash")]
    manifest_hash: Option<String>,
    #[serde(rename = "git_commit")]
    git_commit: Option<String>,
    #[serde(rename = "runner_version")]
    runner_version: Option<String>,
    #[serde(rename = "warmup_count")]
    warmup_count: Option<usize>,
    #[serde(rename = "build_mode")]
    build_mode: Option<String>,
    #[serde(rename = "ratatui_version")]
    ratatui_version: Option<String>,
    #[serde(rename = "rustc_version")]
    rustc_version: Option<String>,
}

#[derive(Serialize)]
struct Report {
    implementation: String,
    environment: EnvMetadata,
    valid: bool,
    workloads: Vec<Workload>,
}

fn command_output(program: &str, args: &[&str]) -> Option<String> {
    let output = std::process::Command::new(program)
        .args(args)
        .output()
        .ok()?;
    if !output.status.success() {
        return None;
    }
    let text = String::from_utf8(output.stdout).ok()?;
    Some(text.trim().to_string())
}

fn sha256_file(path: &str) -> String {
    command_output("sha256sum", &[path])
        .and_then(|out| out.split_whitespace().next().map(str::to_string))
        .unwrap_or_else(|| "unknown".to_string())
}

fn cpu_model() -> Option<String> {
    let info = fs::read_to_string("/proc/cpuinfo").ok()?;
    for line in info.lines() {
        if let Some((key, value)) = line.split_once(':') {
            if key.trim() == "model name" {
                return Some(value.trim().to_string());
            }
        }
    }
    None
}

/// Data the scenes render, built once so no fixture construction lands inside
/// a timed region.
struct Fixtures {
    layer_titles: Vec<String>,
    table_rows: Vec<[String; 3]>,
    virtual_rows: Vec<[String; 3]>,
}

impl Fixtures {
    fn new() -> Self {
        Self {
            layer_titles: (0..100).map(|i| format!("Layer {i}")).collect(),
            table_rows: (0..10_000)
                .map(|i| [i.to_string(), "process".into(), "running".into()])
                .collect(),
            virtual_rows: (0..40)
                .map(|i| {
                    [
                        format!("#{i:06}"),
                        format!("örnek kayıt {i}"),
                        "viewport cache".into(),
                    ]
                })
                .collect(),
        }
    }
}

fn rows_from(data: &[[String; 3]]) -> Vec<Row<'_>> {
    data.iter()
        .map(|cells| {
            Row::new(vec![
                Cell::from(cells[0].as_str()),
                Cell::from(cells[1].as_str()),
                Cell::from(cells[2].as_str()),
            ])
        })
        .collect()
}

/// An in-memory sink shared with the caller, because `CrosstermBackend` keeps
/// its writer private. It retains the bytes rather than only counting them, so
/// Ratatui pays the same copy into an output buffer that the Limoni runner does.
#[derive(Clone)]
struct Sink(Rc<RefCell<Vec<u8>>>);

impl Sink {
    fn new() -> Self {
        Self(Rc::new(RefCell::new(Vec::with_capacity(1 << 16))))
    }
    fn len(&self) -> usize {
        self.0.borrow().len()
    }
    /// Clears the bytes while keeping the capacity, so sink growth never lands
    /// in a measured allocation count.
    fn reset(&self) {
        self.0.borrow_mut().clear();
    }
}

impl Write for Sink {
    fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
        self.0.borrow_mut().extend_from_slice(buf);
        Ok(buf.len())
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

type Backend = CrosstermBackend<Sink>;

fn render_scene(
    name: &str,
    frame: &mut Frame,
    step: usize,
    fx: &Fixtures,
    table_state: &mut TableState,
) {
    let area = frame.area();
    match name {
        "empty-frame" | "async-update-burst" => {}
        "full-redraw-120x40" => {
            // Mirrors the Limoni runner cell for cell: the glyph and foreground
            // advance with the step and the background varies by row, so the
            // diff has real work every frame. Rendering static content here
            // instead made Ratatui emit 25 bytes a frame against Limoni's 4897
            // — a difference in the workload, not in the engines.
            let glyph = char::from(b'A' + (step % 26) as u8);
            let fg = Color::Indexed(((step * 3) % 256) as u8);
            let buffer = frame.buffer_mut();
            for y in area.top()..area.bottom() {
                let bg = Color::Indexed((y % 256) as u8);
                for x in area.left()..area.right() {
                    if let Some(cell) = buffer.cell_mut((x, y)) {
                        cell.set_char(glyph).set_fg(fg).set_bg(bg);
                    }
                }
            }
        }
        "single-cell-update" => {
            let symbol = if step.is_multiple_of(2) { "X" } else { "Y" };
            frame.render_widget(Paragraph::new(symbol), Rect::new(0, 0, 1, 1));
        }
        "text-heavy-120x40" => {
            frame.render_widget(
                Paragraph::new("Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis.")
                    .wrap(Wrap { trim: false }),
                area,
            );
        }
        "unicode-emoji" => {
            frame.render_widget(
                Paragraph::new("Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols verification.")
                    .wrap(Wrap { trim: false }),
                area,
            );
        }
        "table-10000" => {
            // Rebuilding rows every frame is how a Ratatui application works:
            // Table takes ownership of the row iterator. See the header note on
            // why this makes the workload structural rather than like-for-like.
            table_state.select(Some((step * 7) % fx.table_rows.len()));
            let table = Table::new(
                rows_from(&fx.table_rows),
                [
                    Constraint::Length(8),
                    Constraint::Percentage(40),
                    Constraint::Min(0),
                ],
            );
            frame.render_stateful_widget(table, area, table_state);
        }
        "virtual-1000000" => {
            // Virtual paging: only the visible window is ever materialised.
            let table = Table::new(
                rows_from(&fx.virtual_rows),
                [
                    Constraint::Length(10),
                    Constraint::Length(30),
                    Constraint::Min(0),
                ],
            );
            frame.render_widget(table, area);
        }
        "mouse-hit-test" | "native-image-capability" => {
            frame.render_widget(Block::default(), area);
        }
        "hundred-layers" => {
            // Bordered and titled, and moving with the step, matching the
            // Limoni runner. Plain `Block::default()` draws nothing at all.
            for (i, title) in fx.layer_titles.iter().enumerate() {
                let offset = i + step;
                let layer = Rect::new((offset % 70) as u16, (offset % 20) as u16, 10, 3);
                frame.render_widget(Block::bordered().title(title.as_str()), layer);
            }
        }
        _ => {}
    }
}

/// Runs one frame and returns the bytes the backend emitted for it.
fn run_frame(
    name: &str,
    terminal: &mut Terminal<Backend>,
    step: usize,
    fx: &Fixtures,
    table_state: &mut TableState,
    spec: &Spec,
) -> io::Result<()> {
    if name == "resize" {
        let size = if step.is_multiple_of(2) {
            Rect::new(0, 0, spec.width + 20, spec.height + 10)
        } else {
            Rect::new(0, 0, spec.width, spec.height)
        };
        terminal.resize(size)?;
    }
    terminal.draw(|frame| render_scene(name, frame, step, fx, table_state))?;
    Ok(())
}

fn percentile(sorted: &[u128], pct: usize) -> u128 {
    if sorted.is_empty() {
        return 0;
    }
    let index = (sorted.len() * pct / 100).min(sorted.len() - 1);
    sorted[index]
}

fn main() -> io::Result<()> {
    let output = env::args().nth(1).unwrap_or_else(|| "ratatui.json".into());

    let manifest_path = [
        "benchmarks/workloads.json",
        "../../workloads.json",
        "../../../workloads.json",
    ]
    .into_iter()
    .find(|path| fs::metadata(path).is_ok())
    .expect("failed to locate workloads.json")
    .to_string();

    let data = fs::read_to_string(&manifest_path).expect("failed to read workloads.json");
    let specs: Vec<Spec> = serde_json::from_str(&data).expect("failed to parse workloads.json");

    let mut workloads = Vec::with_capacity(specs.len());

    for spec in specs {
        let fx = Fixtures::new();
        let mut table_state = TableState::default();

        let sink = Sink::new();
        let mut terminal = Terminal::with_options(
            CrosstermBackend::new(sink.clone()),
            TerminalOptions {
                viewport: Viewport::Fixed(Rect::new(0, 0, spec.width, spec.height)),
            },
        )?;

        for step in 0..WARMUP {
            sink.reset();
            run_frame(
                &spec.name,
                &mut terminal,
                step,
                &fx,
                &mut table_state,
                &spec,
            )?;
        }

        let mut durations = Vec::with_capacity(spec.iterations);
        let mut total_bytes = 0usize;

        ALLOCATED.store(0, Ordering::Relaxed);
        ALLOCS.store(0, Ordering::Relaxed);

        for step in 0..spec.iterations {
            sink.reset();
            let start = Instant::now();
            run_frame(
                &spec.name,
                &mut terminal,
                step,
                &fx,
                &mut table_state,
                &spec,
            )?;
            durations.push(start.elapsed().as_nanos());
            total_bytes += sink.len();
        }

        let alloc_bytes = ALLOCATED.load(Ordering::Relaxed);
        let allocs = ALLOCS.load(Ordering::Relaxed);

        durations.sort_unstable();
        let sum: u128 = durations.iter().sum();
        let mean = sum / durations.len() as u128;
        let variance: f64 = durations
            .iter()
            .map(|&value| {
                let diff = value as f64 - mean as f64;
                diff * diff
            })
            .sum::<f64>()
            / durations.len() as f64;

        let comparable = !NON_COMPARABLE.contains(&spec.name.as_str());
        workloads.push(Workload {
            summary: Summary {
                frames: durations.len(),
                p50_ns: percentile(&durations, 50),
                p95_ns: percentile(&durations, 95),
                p99_ns: percentile(&durations, 99),
                min_ns: durations[0],
                max_ns: durations[durations.len() - 1],
                mean_ns: mean,
                std_dev_ns: variance.sqrt() as u128,
                bytes_per_frame: total_bytes as f64 / durations.len() as f64,
                alloc_bytes,
                allocs,
            },
            spec,
            comparable,
        });
    }

    let report = Report {
        implementation: "ratatui".into(),
        environment: EnvMetadata {
            os: env::consts::OS.into(),
            arch: env::consts::ARCH.into(),
            go: None,
            cpu: cpu_model(),
            output: Some("memory".into()),
            manifest_hash: Some(sha256_file(&manifest_path)),
            git_commit: Some(
                command_output("git", &["rev-parse", "HEAD"]).unwrap_or_else(|| "unknown".into()),
            ),
            runner_version: Some(RUNNER_VERSION.into()),
            warmup_count: Some(WARMUP),
            build_mode: Some("release".into()),
            ratatui_version: Some(env!("RATATUI_VERSION").into()),
            rustc_version: command_output("rustc", &["--version"]),
        },
        valid: true,
        workloads,
    };

    fs::write(output, serde_json::to_vec_pretty(&report).unwrap())?;
    Ok(())
}
