use crate::util::{
    CommandError, OutputBundle, render_command_steps, require_success, run_exec, run_shell,
};
use serde_json::json;
use std::path::Path;

pub fn run(repo_root: &Path) -> Result<OutputBundle, CommandError> {
    if let Ok(value) = std::env::var("HARNESS_TEST_CMD") {
        if !value.trim().is_empty() {
            let result = require_success(
                "test",
                run_shell(repo_root, &value).map_err(shell_error("test"))?,
            )?;
            return Ok(bundle(
                result.command,
                vec![json!({"label": "override-test", "status": "ok"})],
            ));
        }
    }

    let go = require_success(
        "test",
        run_exec(repo_root, "go", &["test", "./..."]).map_err(shell_error("test"))?,
    )?;
    let cargo = require_success(
        "test",
        run_exec(
            repo_root,
            "cargo",
            &["test", "--manifest-path", "harness/Cargo.toml"],
        )
        .map_err(shell_error("test"))?,
    )?;
    Ok(bundle(
        "go test ./... && cargo test --manifest-path harness/Cargo.toml".to_string(),
        vec![
            json!({"label": "go-tests", "status": "ok", "command": go.command}),
            json!({"label": "cargo-tests", "status": "ok", "command": cargo.command}),
        ],
    ))
}

fn bundle(executed: String, steps: Vec<serde_json::Value>) -> OutputBundle {
    OutputBundle {
        text: render_command_steps("test", &steps),
        json: json!({
            "command": "test",
            "status": "ok",
            "executed": executed,
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "test",
            "status": "ok",
            "executed": executed,
        })],
    }
}

fn shell_error(command: &'static str) -> impl Fn(anyhow::Error) -> CommandError {
    move |err| CommandError::new(command, "command_start_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    use crate::util::command_with_override;

    #[test]
    fn default_command_prefers_go_and_cargo() {
        let resolved =
            command_with_override("HARNESS_TEST_CMD", "default", "go", &["test", "./..."]);
        assert_eq!(resolved[1], "go");
    }
}
