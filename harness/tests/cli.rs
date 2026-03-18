use serde_json::Value;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::{SystemTime, UNIX_EPOCH};

fn binary() -> &'static str {
    env!("CARGO_BIN_EXE_harnesscli")
}

fn repo_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .to_path_buf()
}

#[test]
fn smoke_defaults_to_json_when_stdout_is_captured() {
    let output = Command::new(binary())
        .arg("smoke")
        .current_dir(repo_root())
        .output()
        .expect("run smoke");
    assert!(output.status.success());
    let body: Value = serde_json::from_slice(&output.stdout).expect("json output");
    assert_eq!(body["command"], "smoke");
    assert_eq!(body["status"], "ok");
}

#[test]
fn audit_reports_success_in_json_mode() {
    let output = Command::new(binary())
        .args(["--output", "json", "audit", "."])
        .current_dir(repo_root())
        .output()
        .expect("run audit");
    assert!(output.status.success());
    let body: Value = serde_json::from_slice(&output.stdout).expect("json output");
    assert_eq!(body["command"], "audit");
    assert_eq!(body["passed"], true);
}

#[test]
fn audit_reports_structured_json_errors() {
    let temp_dir = std::env::temp_dir().join(format!(
        "impactable-harness-missing-{}",
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir_all(&temp_dir).unwrap();

    let output = Command::new(binary())
        .args([
            "--output",
            "json",
            "audit",
            temp_dir.to_str().expect("temp dir utf-8"),
        ])
        .current_dir(repo_root())
        .output()
        .expect("run audit");

    assert!(!output.status.success());
    let body: Value = serde_json::from_slice(&output.stdout).expect("error json output");
    assert_eq!(body["error"]["code"], "audit_failed");
    assert!(body["error"]["details"]["checks"].is_array());

    let _ = std::fs::remove_dir_all(temp_dir);
}
