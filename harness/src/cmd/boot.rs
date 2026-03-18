use crate::util::{
    BootMetadata, CommandError, OutputBundle, boot_metadata_path, ensure_runtime_dirs, http_ok,
    is_pid_alive, read_json_file, render_command_steps, spawn_background_http_server, stop_pid,
    wait_for_http_ok, worktree_context, write_json_file,
};
use serde_json::json;
use std::fs;
use std::path::Path;
use std::time::Duration;

pub fn start(current_dir: &Path) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(boot_error)?;
    ensure_runtime_dirs(&ctx).map_err(boot_error)?;

    if let Ok(existing) = read_json_file::<BootMetadata>(&boot_metadata_path(&ctx)) {
        if is_pid_alive(existing.pid) && http_ok(&existing.healthcheck_url) {
            return Ok(start_bundle("reused", &existing));
        }
    }

    render_demo_app(
        &ctx.runtime_root.join("demo-app"),
        &ctx.worktree_id,
        &ctx.runtime_root,
        ctx.selected_port,
    )
    .map_err(boot_error)?;
    let stdout_log = ctx.runtime_root.join("logs").join("demo-app.stdout.log");
    let stderr_log = ctx.runtime_root.join("logs").join("demo-app.stderr.log");
    let pid = spawn_background_http_server(
        &ctx.runtime_root.join("demo-app"),
        ctx.selected_port,
        &stdout_log,
        &stderr_log,
    )
    .map_err(boot_error)?;

    let healthcheck_url = format!("http://127.0.0.1:{}/healthz", ctx.selected_port);
    if !wait_for_http_ok(&healthcheck_url, Duration::from_secs(15)) {
        let stderr_excerpt = fs::read_to_string(&stderr_log)
            .ok()
            .and_then(|body| body.lines().last().map(str::to_string));
        return Err(CommandError::new(
            "boot start",
            "boot_timeout",
            format!("demo app failed readiness probe at {healthcheck_url}"),
        )
        .with_details(json!({
            "healthcheck_url": healthcheck_url,
            "stderr_log": stderr_log,
            "stderr_excerpt": stderr_excerpt,
        })));
    }

    let metadata = BootMetadata {
        worktree_id: ctx.worktree_id.clone(),
        runtime_root: ctx.runtime_root.clone(),
        pid,
        app_url: format!("http://127.0.0.1:{}/", ctx.selected_port),
        healthcheck_url,
        selected_port: ctx.selected_port,
        stdout_log,
        stderr_log,
    };
    write_json_file(&boot_metadata_path(&ctx), &metadata).map_err(boot_error)?;
    Ok(start_bundle("started", &metadata))
}

pub fn status(current_dir: &Path) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(boot_error)?;
    let metadata: BootMetadata = read_json_file(&boot_metadata_path(&ctx)).map_err(|err| {
        CommandError::new("boot status", "missing_boot_metadata", err.to_string())
    })?;
    let healthy = is_pid_alive(metadata.pid) && http_ok(&metadata.healthcheck_url);

    let steps = vec![json!({
        "label": "healthcheck",
        "status": if healthy { "ok" } else { "failed" },
    })];
    Ok(OutputBundle {
        text: render_command_steps("boot status", &steps),
        json: json!({
            "command": "boot status",
            "status": if healthy { "ok" } else { "failed" },
            "app_url": metadata.app_url,
            "healthcheck_url": metadata.healthcheck_url,
            "healthcheck_status": if healthy { "ok" } else { "failed" },
            "selected_port": metadata.selected_port,
            "worktree_id": metadata.worktree_id,
            "runtime_root": metadata.runtime_root,
            "pid": metadata.pid,
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "boot status",
            "status": if healthy { "ok" } else { "failed" },
            "pid": metadata.pid,
        })],
    })
}

pub fn stop(current_dir: &Path) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(boot_error)?;
    let metadata: BootMetadata = read_json_file(&boot_metadata_path(&ctx))
        .map_err(|err| CommandError::new("boot stop", "missing_boot_metadata", err.to_string()))?;
    if is_pid_alive(metadata.pid) {
        stop_pid(metadata.pid).map_err(boot_error)?;
    }
    let _ = fs::remove_file(boot_metadata_path(&ctx));
    let steps = vec![json!({"label": "stop-demo-app", "status": "ok"})];
    Ok(OutputBundle {
        text: render_command_steps("boot stop", &steps),
        json: json!({
            "command": "boot stop",
            "status": "ok",
            "stopped_pid": metadata.pid,
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "boot stop",
            "status": "ok",
            "stopped_pid": metadata.pid,
        })],
    })
}

fn start_bundle(state: &str, metadata: &BootMetadata) -> OutputBundle {
    let steps = vec![json!({"label": "demo-app", "status": state})];
    OutputBundle {
        text: render_command_steps("boot start", &steps),
        json: json!({
            "command": "boot start",
            "status": "ok",
            "result": state,
            "app_url": metadata.app_url,
            "healthcheck_url": metadata.healthcheck_url,
            "healthcheck_status": "ok",
            "selected_port": metadata.selected_port,
            "worktree_id": metadata.worktree_id,
            "runtime_root": metadata.runtime_root,
            "pid": metadata.pid,
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "boot start",
            "status": "ok",
            "result": state,
            "pid": metadata.pid,
        })],
    }
}

fn render_demo_app(
    dir: &Path,
    worktree_id: &str,
    runtime_root: &Path,
    selected_port: u16,
) -> Result<(), anyhow::Error> {
    fs::create_dir_all(dir)?;
    fs::write(
        dir.join("index.html"),
        format!(
            "<!doctype html><html><head><meta charset=\"utf-8\"><title>Impactable Harness Demo</title></head><body><h1>Impactable Harness Demo</h1><ul><li>worktree_id: {worktree_id}</li><li>runtime_root: {}</li><li>selected_port: {selected_port}</li></ul></body></html>",
            runtime_root.display()
        ),
    )?;
    fs::write(dir.join("healthz"), "ok\n")?;
    Ok(())
}

fn boot_error(err: anyhow::Error) -> CommandError {
    CommandError::new("boot", "boot_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn demo_app_contains_expected_title() {
        let base =
            std::env::temp_dir().join(format!("impactable-boot-test-{}", std::process::id()));
        let _ = fs::remove_dir_all(&base);
        fs::create_dir_all(&base).unwrap();
        render_demo_app(&base, "worktree-1", Path::new("/tmp/runtime"), 4100).unwrap();
        let body = fs::read_to_string(base.join("index.html")).unwrap();
        assert!(body.contains("Impactable Harness Demo"));
        let _ = fs::remove_dir_all(&base);
    }
}
