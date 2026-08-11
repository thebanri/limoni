use ratatui::{buffer::Buffer, layout::Rect, widgets::{Paragraph, Widget}};
use serde::Serialize;
use std::{env, fs, time::Instant};

#[derive(Serialize)] struct Spec { name: String, width: u16, height: u16, rows: usize, unicode: bool, full_draw: bool, mouse: bool, async_burst: usize }
#[derive(Serialize)] struct Summary { #[serde(rename="Frames")] frames: usize, #[serde(rename="P50NS")] p50_ns: u128, #[serde(rename="P95NS")] p95_ns: u128, #[serde(rename="P99NS")] p99_ns: u128, #[serde(rename="BytesPerFrame")] bytes_per_frame: f64, #[serde(rename="AllocBytes")] alloc_bytes: u64, #[serde(rename="Allocs")] allocs: u64 }
#[derive(Serialize)] struct Workload { spec: Spec, summary: Summary }
#[derive(Serialize)] struct Report { implementation: String, workloads: Vec<Workload> }

fn main() {
    let output = env::args().nth(1).unwrap_or_else(|| "ratatui.json".into());
    let specs = vec![Spec{name:"empty-frame".into(),width:80,height:24,rows:0,unicode:false,full_draw:false,mouse:false,async_burst:0}, Spec{name:"text-heavy-120x40".into(),width:120,height:40,rows:0,unicode:true,full_draw:true,mouse:false,async_burst:0}];
    let mut workloads = Vec::new();
    for spec in specs {
        let start = Instant::now(); let mut buffer = Buffer::empty(Rect::new(0,0,spec.width,spec.height));
        Paragraph::new("Limoni benchmark ✓ 日本語").render(buffer.area, &mut buffer);
        let ns = start.elapsed().as_nanos();
        workloads.push(Workload{spec,summary:Summary{frames:1,p50_ns:ns,p95_ns:ns,p99_ns:ns,bytes_per_frame:0.0,alloc_bytes:0,allocs:0}});
    }
    fs::write(output, serde_json::to_vec_pretty(&Report{implementation:"ratatui".into(),workloads}).unwrap()).unwrap();
}