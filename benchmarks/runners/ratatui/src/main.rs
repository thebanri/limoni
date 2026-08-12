use ratatui::{
    backend::TestBackend,
    buffer::Buffer,
    layout::{Constraint, Rect},
    widgets::{Block, Paragraph, Table, Row, Cell, Widget},
    Terminal,
};
use serde::{Serialize, Deserialize};
use std::{env, fs, time::Instant};
use std::alloc::{GlobalAlloc, Layout, System};
use std::sync::atomic::{AtomicU64, Ordering};

struct Counter;
static ALLOCATED: AtomicU64 = AtomicU64::new(0);
static ALLOCS: AtomicU64 = AtomicU64::new(0);

unsafe impl GlobalAlloc for Counter {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        let ret = System.alloc(layout);
        if !ret.is_null() {
            ALLOCATED.fetch_add(layout.size() as u64, Ordering::SeqCst);
            ALLOCS.fetch_add(1, Ordering::SeqCst);
        }
        ret
    }
    unsafe fn dealloc(&self, ptr: *mut u8, layout: Layout) {
        System.dealloc(ptr, layout);
    }
}

#[global_allocator]
static A: Counter = Counter;

#[derive(Serialize, Deserialize, Clone)]
struct Spec {
    name: String,
    width: u16,
    height: u16,
    #[serde(default)]
    rows: usize,
    #[serde(default)]
    unicode: bool,
    #[serde(default, rename="full_draw")]
    full_draw: bool,
    #[serde(default)]
    mouse: bool,
    #[serde(default, rename="async_burst")]
    async_burst: usize,
    #[serde(default, rename="output_mode")]
    output_mode: String,
    #[serde(default, rename="color_mode")]
    color_mode: String,
    iterations: usize,
}

#[derive(Serialize)]
struct Summary {
    #[serde(rename="Frames")]
    frames: usize,
    #[serde(rename="P50NS")]
    p50_ns: u128,
    #[serde(rename="P95NS")]
    p95_ns: u128,
    #[serde(rename="P99NS")]
    p99_ns: u128,
    #[serde(rename="MinNS")]
    min_ns: u128,
    #[serde(rename="MaxNS")]
    max_ns: u128,
    #[serde(rename="MeanNS")]
    mean_ns: u128,
    #[serde(rename="StdDevNS")]
    std_dev_ns: u128,
    #[serde(rename="BytesPerFrame")]
    bytes_per_frame: f64,
    #[serde(rename="AllocBytes")]
    alloc_bytes: u64,
    #[serde(rename="Allocs")]
    allocs: u64,
}

#[derive(Serialize)]
struct Workload {
    spec: Spec,
    summary: Summary,
}

#[derive(Serialize)]
struct Report {
    implementation: String,
    environment: EnvMetadata,
    valid: bool,
    workloads: Vec<Workload>,
}

#[derive(Serialize)]
struct EnvMetadata {
    os: String,
    arch: String,
    go: Option<String>,
    cpu: Option<String>,
    output: Option<String>,
}

fn buffer_bytes(buf: &Buffer) -> usize {
    let mut bytes = 0;

    for y in 0..buf.area.height {
        for x in 0..buf.area.width {
            if let Some(cell) = buf.cell((x, y)) {
                bytes += cell.symbol().len();
                bytes += 10; // Estimate style overhead
            }
        }
    }

    bytes
}

fn main() {
    let output = env::args().nth(1).unwrap_or_else(|| "ratatui.json".into());

    // Load specs from workload manifest
    let mut manifest_path = "benchmarks/workloads.json".to_string();
    if !fs::metadata(&manifest_path).is_ok() {
        manifest_path = "../../workloads.json".to_string();
    }
    if !fs::metadata(&manifest_path).is_ok() {
        manifest_path = "../../../workloads.json".to_string();
    }
    let data = fs::read_to_string(&manifest_path).expect("failed to read workloads.json");
    let mut specs: Vec<Spec> = serde_json::from_str(&data).expect("failed to parse workloads.json");

    let mut workloads = Vec::new();

    // Table rows pre-building
    let table_rows: Vec<Row> = (0..10000).map(|i| {
        Row::new(vec![
            Cell::from(format!("{}", i)),
            Cell::from("process"),
            Cell::from("running"),
        ])
    }).collect();

    // Virtual table rows pre-building
    let virtual_rows: Vec<Row> = (0..40).map(|i| {
        Row::new(vec![
            Cell::from(format!("#{:06}", i)),
            Cell::from(format!("örnek kayıt {}", i)),
            Cell::from("viewport cache"),
        ])
    }).collect();

    for spec in specs.drain(..) {
        let mut terminal = Terminal::new(TestBackend::new(spec.width, spec.height)).unwrap();
        let mut toggle = false;

        // Warmup (10 runs)
        for _ in 0..10 {
            match spec.name.as_str() {
                "empty-frame" => {
                    terminal.draw(|_f| {}).unwrap();
                }
                "full-redraw-120x40" => {
                    terminal.draw(|f| {
                        let area = f.area();
                        let text = "A".repeat((area.width * area.height) as usize);
                        Paragraph::new(text).render(area, f.buffer_mut());
                    }).unwrap();
                }
                "single-cell-update" => {
                    terminal.draw(|f| {
                        Paragraph::new("X").render(Rect::new(0, 0, 1, 1), f.buffer_mut());
                    }).unwrap();
                }
                "text-heavy-120x40" => {
                    terminal.draw(|f| {
                        let text = "Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis.";
                        Paragraph::new(text).render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "unicode-emoji" => {
                    terminal.draw(|f| {
                        let text = "Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols verification.";
                        Paragraph::new(text).render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "table-10000" => {
                    let rows_clone = table_rows.clone();
                    terminal.draw(|f| {
                        let table = Table::new(rows_clone, [Constraint::Length(8), Constraint::Percentage(40), Constraint::Min(0)]);
                        table.render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "virtual-1000000" => {
                    let rows_clone = virtual_rows.clone();
                    terminal.draw(|f| {
                        let table = Table::new(rows_clone, [Constraint::Length(10), Constraint::Length(30), Constraint::Min(0)]);
                        table.render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "mouse-hit-test" => {
                    terminal.draw(|f| {
                        Block::default().render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "hundred-layers" => {
                    terminal.draw(|f| {
                        for i in 0..100 {
                            let area = Rect::new((i % 70) as u16, (i % 20) as u16, 10, 3);
                            Block::default().render(area, f.buffer_mut());
                        }
                    }).unwrap();
                }
                "resize" => {
                    terminal.resize(Rect::new(0, 0, spec.width, spec.height)).unwrap();
                    terminal.draw(|_f| {}).unwrap();
                }
                "async-update-burst" => {
                    terminal.draw(|_f| {}).unwrap();
                }
                "native-image-capability" => {
                    terminal.draw(|f| {
                        Block::default().render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                _ => {}
            }
        }

        // Timing & allocation measurements
        let mut durations = Vec::with_capacity(spec.iterations);
        let mut total_bytes = 0;

        // Reset memory counters before measurement
        ALLOCATED.store(0, Ordering::SeqCst);
        ALLOCS.store(0, Ordering::SeqCst);

        for _ in 0..spec.iterations {
            let start = Instant::now();
            match spec.name.as_str() {
                "empty-frame" => {
                    terminal.draw(|_f| {}).unwrap();
                }
                "full-redraw-120x40" => {
                    terminal.draw(|f| {
                        let area = f.area();
                        let text = "A".repeat((area.width * area.height) as usize);
                        Paragraph::new(text).render(area, f.buffer_mut());
                    }).unwrap();
                }
                "single-cell-update" => {
                    terminal.draw(|f| {
                        let sym = if toggle { "X" } else { "Y" };
                        Paragraph::new(sym).render(Rect::new(0, 0, 1, 1), f.buffer_mut());
                    }).unwrap();
                    toggle = !toggle;
                }
                "text-heavy-120x40" => {
                    terminal.draw(|f| {
                        let text = "Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis.";
                        Paragraph::new(text).render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "unicode-emoji" => {
                    terminal.draw(|f| {
                        let text = "Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols verification.";
                        Paragraph::new(text).render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "table-10000" => {
                    let rows_clone = table_rows.clone();
                    terminal.draw(|f| {
                        let table = Table::new(rows_clone, [Constraint::Length(8), Constraint::Percentage(40), Constraint::Min(0)]);
                        table.render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "virtual-1000000" => {
                    let rows_clone = virtual_rows.clone();
                    terminal.draw(|f| {
                        let table = Table::new(rows_clone, [Constraint::Length(10), Constraint::Length(30), Constraint::Min(0)]);
                        table.render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "mouse-hit-test" => {
                    terminal.draw(|f| {
                        Block::default().render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                "hundred-layers" => {
                    terminal.draw(|f| {
                        for i in 0..100 {
                            let area = Rect::new((i % 70) as u16, (i % 20) as u16, 10, 3);
                            Block::default().render(area, f.buffer_mut());
                        }
                    }).unwrap();
                }
                "resize" => {
                    let size = if toggle { Rect::new(0, 0, spec.width + 20, spec.height + 10) } else { Rect::new(0, 0, spec.width, spec.height) };
                    terminal.resize(size).unwrap();
                    terminal.draw(|_f| {}).unwrap();
                    toggle = !toggle;
                }
                "async-update-burst" => {
                    terminal.draw(|_f| {}).unwrap();
                }
                "native-image-capability" => {
                    terminal.draw(|f| {
                        Block::default().render(f.area(), f.buffer_mut());
                    }).unwrap();
                }
                _ => {}
            }
            durations.push(start.elapsed().as_nanos());
            total_bytes += buffer_bytes(terminal.backend().buffer());
        }

        let alloc_bytes_total = ALLOCATED.load(Ordering::SeqCst);
        let allocs_total = ALLOCS.load(Ordering::SeqCst);

        durations.sort_unstable();
        let p50 = durations[durations.len() * 50 / 100];
        let p95 = durations[durations.len() * 95 / 100];
        let p99 = durations[durations.len() * 99 / 100];

        let sum: u128 = durations.iter().sum();
        let mean = sum / durations.len() as u128;
        let variance_sum: f64 = durations.iter().map(|&val| {
            let diff = val as f64 - mean as f64;
            diff * diff
        }).sum();
        let std_dev = variance_sum / durations.len() as f64;
        let std_dev_ns = std_dev.sqrt() as u128;

        workloads.push(Workload {
            spec,
            summary: Summary {
                frames: durations.len(),
                p50_ns: p50,
                p95_ns: p95,
                p99_ns: p99,
                min_ns: durations[0],
                max_ns: durations[durations.len() - 1],
                mean_ns: mean,
                std_dev_ns,
                bytes_per_frame: total_bytes as f64 / durations.len() as f64,
                alloc_bytes: alloc_bytes_total,
                allocs: allocs_total,
            },
        });
    }

    let report = Report {
        implementation: "ratatui".into(),
        environment: EnvMetadata {
            os: env::consts::OS.into(),
            arch: env::consts::ARCH.into(),
            go: None,
            cpu: None,
            output: Some("memory".into()),
        },
        valid: true,
        workloads,
    };

    fs::write(output, serde_json::to_vec_pretty(&report).unwrap()).unwrap();
}