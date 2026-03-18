use anyhow::{Context, Result, anyhow};
use clap::ValueEnum;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::fs::{self, File};
use std::io::{self, IsTerminal, Read, Write};
use std::net::TcpStream;
use std::path::{Path, PathBuf};
use std::process::{Command, Output, Stdio};
use std::thread;
use std::time::{Duration, Instant};

const DEFAULT_APP_PORT_BASE: u16 = 4100;
const PORT_RANGE: u16 = 20;
const FALLBACK_PORT_RANGE: u16 = 2000;

#[derive(Clone, Copy, Debug, Eq, PartialEq, ValueEnum)]
pub enum OutputFormat {
    Text,
    Json,
    Ndjson,
}

#[derive(Debug)]
pub struct OutputBundle {
    pub text: String,
    pub json: Value,
    pub ndjson: Vec<Value>,
}

#[derive(Debug, Serialize, Clone)]
pub struct CommandError {
    pub code: String,
    pub message: String,
    pub command: String,
    #[serde(skip_serializing_if = "Value::is_null")]
    pub details: Value,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WorktreeContext {
    pub repo_root: PathBuf,
    pub worktree_id: String,
    pub runtime_root: PathBuf,
    pub selected_port: u16,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct BootMetadata {
    pub worktree_id: String,
    pub runtime_root: PathBuf,
    pub pid: u32,
    pub app_url: String,
    pub healthcheck_url: String,
    pub selected_port: u16,
    pub stdout_log: PathBuf,
    pub stderr_log: PathBuf,
}

#[derive(Debug)]
pub struct CmdResult {
    pub command: String,
    pub status: i32,
    pub stdout: String,
    pub stderr: String,
}

impl CommandError {
    pub fn new(
        command: impl Into<String>,
        code: impl Into<String>,
        message: impl Into<String>,
    ) -> Self {
        Self {
            code: code.into(),
            message: message.into(),
            command: command.into(),
            details: Value::Null,
        }
    }

    pub fn with_details(mut self, details: Value) -> Self {
        self.details = details;
        self
    }
}

pub fn resolve_output(explicit: Option<OutputFormat>) -> OutputFormat {
    explicit.unwrap_or_else(|| {
        if io::stdout().is_terminal() {
            OutputFormat::Text
        } else {
            OutputFormat::Json
        }
    })
}

pub fn emit(bundle: OutputBundle, format: OutputFormat) -> Result<()> {
    match format {
        OutputFormat::Text => {
            println!("{}", bundle.text);
        }
        OutputFormat::Json => {
            println!("{}", serde_json::to_string_pretty(&bundle.json)?);
        }
        OutputFormat::Ndjson => {
            for item in bundle.ndjson {
                println!("{}", serde_json::to_string(&item)?);
            }
        }
    }
    Ok(())
}

pub fn emit_error(err: &CommandError, format: OutputFormat) -> Result<()> {
    match format {
        OutputFormat::Text => {
            if let Some(checks) = err.details.get("checks").and_then(Value::as_array) {
                for check in checks {
                    let passed = check
                        .get("passed")
                        .and_then(Value::as_bool)
                        .unwrap_or(false);
                    let label = check
                        .get("label")
                        .and_then(Value::as_str)
                        .unwrap_or("check");
                    let status = if passed { "[ok]" } else { "[missing]" };
                    eprintln!("{status} {label}");
                }
            }
            eprintln!("{}: {}", err.command, err.message);
        }
        OutputFormat::Json | OutputFormat::Ndjson => {
            let payload = json!({ "error": err });
            println!("{}", serde_json::to_string_pretty(&payload)?);
        }
    }
    Ok(())
}

pub fn repo_root(path: &Path) -> Result<PathBuf> {
    let output = Command::new("git")
        .arg("rev-parse")
        .arg("--show-toplevel")
        .current_dir(path)
        .output()
        .context("failed to run git rev-parse")?;
    if !output.status.success() {
        return Err(anyhow!(
            "git rev-parse failed: {}",
            String::from_utf8_lossy(&output.stderr)
        ));
    }
    canonicalize_utf8(Path::new(String::from_utf8_lossy(&output.stdout).trim()))
}

pub fn worktree_context(path: &Path) -> Result<WorktreeContext> {
    let repo_root = repo_root(path)?;
    let worktree_id = std::env::var("DISCODE_WORKTREE_ID")
        .ok()
        .filter(|value| !value.trim().is_empty())
        .unwrap_or_else(|| derive_worktree_id(&repo_root));
    let runtime_root = repo_root.join(".worktree").join(&worktree_id);
    let selected_port = resolve_port(&worktree_id)?;
    Ok(WorktreeContext {
        repo_root,
        worktree_id,
        runtime_root,
        selected_port,
    })
}

pub fn derive_worktree_id(path: &Path) -> String {
    let canonical = canonicalize_utf8(path).unwrap_or_else(|_| path.to_path_buf());
    let base = canonical
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("worktree");
    let hash = fnv1a64(canonical.to_string_lossy().as_bytes());
    format!("{base}-{hash:08x}")
}

pub fn ensure_runtime_dirs(ctx: &WorktreeContext) -> Result<()> {
    for dir in [
        ctx.runtime_root.join("run"),
        ctx.runtime_root.join("logs"),
        ctx.runtime_root.join("tmp"),
        ctx.runtime_root.join("demo-app"),
        ctx.runtime_root.join("observability"),
    ] {
        fs::create_dir_all(dir)?;
    }
    Ok(())
}

pub fn runtime_manifest_path(ctx: &WorktreeContext) -> PathBuf {
    ctx.runtime_root.join("run").join("runtime.json")
}

pub fn boot_metadata_path(ctx: &WorktreeContext) -> PathBuf {
    ctx.runtime_root.join("run").join("boot.json")
}

pub fn observability_metadata_path(ctx: &WorktreeContext) -> PathBuf {
    ctx.runtime_root.join("run").join("observability.json")
}

pub fn write_json_file<T: Serialize>(path: &Path, value: &T) -> Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }
    let body = serde_json::to_vec_pretty(value)?;
    fs::write(path, body)?;
    Ok(())
}

pub fn read_json_file<T: for<'de> Deserialize<'de>>(path: &Path) -> Result<T> {
    let body = fs::read(path)?;
    Ok(serde_json::from_slice(&body)?)
}

pub fn run_exec(current_dir: &Path, program: &str, args: &[&str]) -> Result<CmdResult> {
    let output = Command::new(program)
        .args(args)
        .current_dir(current_dir)
        .output()
        .with_context(|| format!("failed to run {program}"))?;
    cmd_result(program, args, output)
}

pub fn run_shell(current_dir: &Path, command: &str) -> Result<CmdResult> {
    let output = Command::new("sh")
        .arg("-lc")
        .arg(command)
        .current_dir(current_dir)
        .output()
        .with_context(|| format!("failed to run shell command: {command}"))?;
    cmd_result("sh", &["-lc", command], output)
}

pub fn require_success(command: &str, result: CmdResult) -> Result<CmdResult, CommandError> {
    if result.status == 0 {
        Ok(result)
    } else {
        Err(CommandError::new(
            command,
            "command_failed",
            format!("{} failed with exit code {}", result.command, result.status),
        )
        .with_details(json!({
            "stdout": result.stdout,
            "stderr": result.stderr,
            "executed": result.command,
        })))
    }
}

pub fn spawn_background_http_server(
    serve_dir: &Path,
    port: u16,
    stdout_log: &Path,
    stderr_log: &Path,
) -> Result<u32> {
    let stdout = File::create(stdout_log)?;
    let stderr = File::create(stderr_log)?;
    let child = Command::new("python3")
        .arg("-m")
        .arg("http.server")
        .arg(port.to_string())
        .arg("--bind")
        .arg("127.0.0.1")
        .current_dir(serve_dir)
        .stdout(Stdio::from(stdout))
        .stderr(Stdio::from(stderr))
        .spawn()
        .context("failed to start demo app server")?;
    Ok(child.id())
}

pub fn is_pid_alive(pid: u32) -> bool {
    Command::new("kill")
        .arg("-0")
        .arg(pid.to_string())
        .status()
        .map(|status| status.success())
        .unwrap_or(false)
}

pub fn stop_pid(pid: u32) -> Result<()> {
    let status = Command::new("kill")
        .arg("-TERM")
        .arg(pid.to_string())
        .status()
        .context("failed to stop process")?;
    if status.success() {
        Ok(())
    } else {
        Err(anyhow!("kill -TERM {} failed", pid))
    }
}

pub fn wait_for_http_ok(url: &str, timeout: Duration) -> bool {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if http_ok(url) {
            return true;
        }
        thread::sleep(Duration::from_millis(200));
    }
    false
}

pub fn http_ok(url: &str) -> bool {
    let stripped = url.strip_prefix("http://").unwrap_or(url);
    let mut parts = stripped.splitn(2, '/');
    let host_port = parts.next().unwrap_or_default();
    let path = format!("/{}", parts.next().unwrap_or_default());
    let mut stream = match TcpStream::connect(host_port) {
        Ok(stream) => stream,
        Err(_) => return false,
    };
    let request = format!("GET {path} HTTP/1.1\r\nHost: {host_port}\r\nConnection: close\r\n\r\n");
    if stream.write_all(request.as_bytes()).is_err() {
        return false;
    }
    let mut response = String::new();
    if stream.read_to_string(&mut response).is_err() {
        return false;
    }
    response.starts_with("HTTP/1.0 200") || response.starts_with("HTTP/1.1 200")
}

pub fn resolve_port(worktree_id: &str) -> Result<u16> {
    for key in ["DISCODE_APP_PORT", "APP_PORT", "PORT"] {
        if let Ok(value) = std::env::var(key) {
            if !value.trim().is_empty() {
                return value
                    .parse::<u16>()
                    .with_context(|| format!("invalid port in {key}"));
            }
        }
    }

    let base = std::env::var("APP_PORT_BASE")
        .ok()
        .and_then(|raw| raw.parse::<u16>().ok())
        .unwrap_or(DEFAULT_APP_PORT_BASE);
    let hash_offset = (fnv1a64(worktree_id.as_bytes()) % 1000) as u16;
    let start = base.saturating_add(hash_offset);
    for offset in 0..PORT_RANGE {
        let candidate = start.saturating_add(offset);
        if can_bind_port(candidate) {
            return Ok(candidate);
        }
    }
    for offset in 0..FALLBACK_PORT_RANGE {
        let candidate = base.saturating_add(offset);
        if can_bind_port(candidate) {
            return Ok(candidate);
        }
    }
    Ok(base)
}

pub fn render_command_steps(command: &str, steps: &[Value]) -> String {
    let mut lines = vec![format!("{command}: ok")];
    for step in steps {
        let label = step.get("label").and_then(Value::as_str).unwrap_or("step");
        let status = step.get("status").and_then(Value::as_str).unwrap_or("ok");
        lines.push(format!("- {label}: {status}"));
    }
    lines.join("\n")
}

pub fn command_with_override(
    env_key: &str,
    default_label: &str,
    default_program: &str,
    default_args: &[&str],
) -> Vec<String> {
    if let Ok(value) = std::env::var(env_key) {
        if !value.trim().is_empty() {
            return vec![format!("override:{env_key}"), value];
        }
    }
    let mut result = vec![default_label.to_string(), default_program.to_string()];
    result.extend(default_args.iter().map(|value| value.to_string()));
    result
}

fn cmd_result(program: &str, args: &[&str], output: Output) -> Result<CmdResult> {
    Ok(CmdResult {
        command: std::iter::once(program.to_string())
            .chain(args.iter().map(|value| value.to_string()))
            .collect::<Vec<_>>()
            .join(" "),
        status: output.status.code().unwrap_or(-1),
        stdout: String::from_utf8(output.stdout)?,
        stderr: String::from_utf8(output.stderr)?,
    })
}

fn canonicalize_utf8(path: &Path) -> Result<PathBuf> {
    Ok(path.canonicalize()?)
}

fn fnv1a64(input: &[u8]) -> u64 {
    let mut hash = 0xcbf29ce484222325u64;
    for byte in input {
        hash ^= u64::from(*byte);
        hash = hash.wrapping_mul(0x100000001b3);
    }
    hash
}

fn can_bind_port(port: u16) -> bool {
    std::net::TcpListener::bind(("127.0.0.1", port)).is_ok()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn derive_worktree_id_is_stable() {
        let path = Path::new("/tmp/example-repo");
        assert_eq!(derive_worktree_id(path), derive_worktree_id(path));
    }

    #[test]
    fn command_override_prefers_env() {
        unsafe {
            std::env::set_var("HARNESS_TEST_KEY", "echo custom");
        }
        let resolved = command_with_override("HARNESS_TEST_KEY", "default", "go", &["test"]);
        assert_eq!(resolved[0], "override:HARNESS_TEST_KEY");
        unsafe {
            std::env::remove_var("HARNESS_TEST_KEY");
        }
    }

    #[test]
    fn resolve_port_returns_a_value() {
        let port = resolve_port("impactable-test").unwrap();
        assert!(port >= DEFAULT_APP_PORT_BASE);
    }
}
