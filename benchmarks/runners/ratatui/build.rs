// Stamps the resolved Ratatui version into the binary so the emitted report
// carries an explicit version label rather than one typed by hand. The
// methodology treats an unlabelled comparison as not a claim at all.

use std::{env, fs, path::Path};

fn main() {
    println!("cargo:rerun-if-changed=Cargo.lock");
    let manifest = env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR");
    let lock = Path::new(&manifest).join("Cargo.lock");
    let version = fs::read_to_string(&lock)
        .ok()
        .and_then(|text| ratatui_version(&text))
        .unwrap_or_else(|| "unknown".to_string());
    println!("cargo:rustc-env=RATATUI_VERSION={version}");
}

/// Returns the `version` of the `[[package]]` block named `ratatui`.
fn ratatui_version(lock: &str) -> Option<String> {
    let mut in_ratatui = false;
    for line in lock.lines() {
        let line = line.trim();
        if line == "[[package]]" {
            in_ratatui = false;
        } else if line == r#"name = "ratatui""# {
            in_ratatui = true;
        } else if in_ratatui {
            if let Some(rest) = line.strip_prefix("version = ") {
                return Some(rest.trim_matches('"').to_string());
            }
        }
    }
    None
}
