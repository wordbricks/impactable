use crate::util::{
    CommandError, OutputBundle, ensure_runtime_dirs, render_command_steps, runtime_manifest_path,
    worktree_context, write_json_file,
};
use serde_json::json;
use std::path::Path;

pub fn run(current_dir: &Path) -> Result<OutputBundle, CommandError> {
    let ctx = worktree_context(current_dir).map_err(init_error)?;
    ensure_runtime_dirs(&ctx).map_err(init_error)?;

    write_json_file(
        &runtime_manifest_path(&ctx),
        &json!({
            "repo_root": ctx.repo_root,
            "worktree_id": ctx.worktree_id,
            "runtime_root": ctx.runtime_root,
            "selected_port": ctx.selected_port,
        }),
    )
    .map_err(init_error)?;

    let steps = vec![
        json!({"label": "resolve-worktree", "status": "ok"}),
        json!({"label": "create-runtime-root", "status": "ok"}),
        json!({"label": "write-runtime-manifest", "status": "ok"}),
    ];
    Ok(OutputBundle {
        text: render_command_steps("init", &steps),
        json: json!({
            "command": "init",
            "status": "ok",
            "worktree_id": ctx.worktree_id,
            "repo_root": ctx.repo_root,
            "runtime_root": ctx.runtime_root,
            "selected_port": ctx.selected_port,
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "init",
            "status": "ok",
            "worktree_id": ctx.worktree_id,
            "runtime_root": ctx.runtime_root,
        })],
    })
}

fn init_error(err: anyhow::Error) -> CommandError {
    CommandError::new("init", "init_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    use crate::util::derive_worktree_id;
    use std::path::Path;

    #[test]
    fn worktree_id_uses_path() {
        let id = derive_worktree_id(Path::new("/tmp/impactable"));
        assert!(id.starts_with("impactable-"));
    }
}
