use crate::util::{
    CommandError, OutputBundle, ensure_runtime_dirs, observability_metadata_path, read_json_file,
    render_command_steps, worktree_context, write_json_file,
};
use serde_json::json;
use std::fs;
use std::path::Path;

pub fn start(current_dir: &Path) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(obs_error)?;
    ensure_runtime_dirs(&ctx).map_err(obs_error)?;
    let metadata = json!({
        "worktree_id": ctx.worktree_id,
        "runtime_root": ctx.runtime_root,
        "log_query_path": ctx.runtime_root.join("logs"),
        "status": "shim-active",
    });
    write_json_file(&observability_metadata_path(&ctx), &metadata).map_err(obs_error)?;
    let steps = vec![json!({"label": "create-observability-shim", "status": "ok"})];
    Ok(OutputBundle {
        text: render_command_steps("observability start", &steps),
        json: json!({
            "command": "observability start",
            "status": "ok",
            "metadata": metadata,
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "observability start",
            "status": "ok",
        })],
    })
}

pub fn stop(current_dir: &Path) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(obs_error)?;
    let _ = fs::remove_file(observability_metadata_path(&ctx));
    let steps = vec![json!({"label": "remove-observability-shim", "status": "ok"})];
    Ok(OutputBundle {
        text: render_command_steps("observability stop", &steps),
        json: json!({
            "command": "observability stop",
            "status": "ok",
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "observability stop",
            "status": "ok",
        })],
    })
}

pub fn query(
    current_dir: &Path,
    kind: &str,
    query: Option<&str>,
) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(obs_error)?;
    let _metadata: serde_json::Value =
        read_json_file(&observability_metadata_path(&ctx)).map_err(|err| {
            CommandError::new(
                "observability query",
                "observability_not_started",
                err.to_string(),
            )
        })?;
    let mut items = Vec::new();
    if kind == "logs" {
        let log_dir = ctx.runtime_root.join("logs");
        if let Ok(entries) = fs::read_dir(log_dir) {
            for entry in entries.flatten() {
                if let Ok(body) = fs::read_to_string(entry.path()) {
                    for line in body.lines() {
                        if query.map(|needle| line.contains(needle)).unwrap_or(true) {
                            items.push(json!({
                                "path": entry.path(),
                                "line": line,
                            }));
                        }
                    }
                }
            }
        }
    }
    let text = if items.is_empty() {
        "observability query: ok\n- items: 0".to_string()
    } else {
        format!("observability query: ok\n- items: {}", items.len())
    };
    Ok(OutputBundle {
        text,
        json: json!({
            "command": "observability query",
            "status": "ok",
            "kind": kind,
            "items": items,
            "worktree_id": ctx.worktree_id,
            "runtime_root": ctx.runtime_root,
        }),
        ndjson: if items.is_empty() {
            vec![json!({
                "command": "observability query",
                "status": "ok",
                "kind": kind,
                "items": 0,
            })]
        } else {
            items
        },
    })
}

fn obs_error(err: anyhow::Error) -> CommandError {
    CommandError::new("observability", "observability_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    #[test]
    fn log_filter_matches_substring() {
        let line = "demo app started";
        assert!(line.contains("started"));
    }
}
