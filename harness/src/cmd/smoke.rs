use crate::util::{
    CommandError, OutputBundle, command_with_override, render_command_steps, require_success,
    run_exec, run_shell,
};
use serde_json::json;
use std::path::Path;

pub fn run(repo_root: &Path) -> Result<OutputBundle, CommandError> {
    let tmp_binary = std::env::temp_dir().join("impactable-harness-smoke-bin");
    let tmp_binary_str = tmp_binary.to_string_lossy().to_string();
    let resolved = command_with_override(
        "HARNESS_SMOKE_CMD",
        "default",
        "go",
        &["build", "-o", &tmp_binary_str, "./cmd/ralph-loop"],
    );
    let result = if resolved[0].starts_with("override:") {
        require_success(
            "smoke",
            run_shell(repo_root, &resolved[1]).map_err(shell_error("smoke"))?,
        )?
    } else {
        let args: Vec<&str> = resolved[2..].iter().map(String::as_str).collect();
        require_success(
            "smoke",
            run_exec(repo_root, &resolved[1], &args).map_err(shell_error("smoke"))?,
        )?
    };

    let steps = vec![json!({
        "label": "smoke",
        "status": "ok",
        "command": result.command,
    })];
    Ok(OutputBundle {
        text: render_command_steps("smoke", &steps),
        json: json!({
            "command": "smoke",
            "status": "ok",
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "smoke",
            "status": "ok",
            "step": "smoke",
            "executed": result.command,
        })],
    })
}

fn shell_error(command: &'static str) -> impl Fn(anyhow::Error) -> CommandError {
    move |err| CommandError::new(command, "command_start_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn supports_override_resolution() {
        unsafe {
            std::env::set_var("HARNESS_SMOKE_CMD", "echo smoke");
        }
        let resolved = command_with_override("HARNESS_SMOKE_CMD", "default", "go", &["build"]);
        assert_eq!(resolved[0], "override:HARNESS_SMOKE_CMD");
        unsafe {
            std::env::remove_var("HARNESS_SMOKE_CMD");
        }
    }
}
