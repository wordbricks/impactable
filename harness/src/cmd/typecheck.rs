use crate::util::{
    CommandError, OutputBundle, render_command_steps, require_success, run_exec, run_shell,
};
use serde_json::json;
use std::path::Path;

pub fn run(repo_root: &Path) -> Result<OutputBundle, CommandError> {
    if let Ok(value) = std::env::var("HARNESS_TYPECHECK_CMD") {
        if !value.trim().is_empty() {
            let result = require_success(
                "typecheck",
                run_shell(repo_root, &value).map_err(shell_error("typecheck"))?,
            )?;
            return Ok(bundle(vec![
                json!({"label": "override-typecheck", "status": "ok", "command": result.command}),
            ]));
        }
    }

    let go = require_success(
        "typecheck",
        run_exec(repo_root, "go", &["build", "./..."]).map_err(shell_error("typecheck"))?,
    )?;
    let cargo = require_success(
        "typecheck",
        run_exec(
            repo_root,
            "cargo",
            &["check", "--manifest-path", "harness/Cargo.toml"],
        )
        .map_err(shell_error("typecheck"))?,
    )?;

    Ok(bundle(vec![
        json!({"label": "go-build", "status": "ok", "command": go.command}),
        json!({"label": "cargo-check", "status": "ok", "command": cargo.command}),
    ]))
}

fn bundle(steps: Vec<serde_json::Value>) -> OutputBundle {
    OutputBundle {
        text: render_command_steps("typecheck", &steps),
        json: json!({
            "command": "typecheck",
            "status": "ok",
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "typecheck",
            "status": "ok",
        })],
    }
}

fn shell_error(command: &'static str) -> impl Fn(anyhow::Error) -> CommandError {
    move |err| CommandError::new(command, "command_start_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bundle_contains_command_name() {
        let steps = vec![json!({"label": "go-build", "status": "ok"})];
        let bundle = bundle(steps);
        assert_eq!(bundle.json["command"], "typecheck");
    }
}
