use crate::util::{
    CommandError, OutputBundle, render_command_steps, require_success, run_exec, run_shell,
};
use serde_json::json;
use std::path::Path;

pub fn run(repo_root: &Path) -> Result<OutputBundle, CommandError> {
    if let Ok(value) = std::env::var("HARNESS_LINT_CMD") {
        if !value.trim().is_empty() {
            let result = require_success(
                "lint",
                run_shell(repo_root, &value).map_err(shell_error("lint"))?,
            )?;
            return Ok(bundle(vec![
                json!({"label": "override-lint", "status": "ok", "command": result.command}),
            ]));
        }
    }

    let gofmt = run_exec(repo_root, "gofmt", &["-l", "."]).map_err(shell_error("lint"))?;
    if !gofmt.stdout.trim().is_empty() {
        return Err(CommandError::new(
            "lint",
            "formatting_required",
            "gofmt reported files that need formatting",
        )
        .with_details(json!({ "files": gofmt.stdout.lines().collect::<Vec<_>>() })));
    }

    let go_vet = require_success(
        "lint",
        run_exec(repo_root, "go", &["vet", "./..."]).map_err(shell_error("lint"))?,
    )?;
    let cargo_fmt = require_success(
        "lint",
        run_exec(
            repo_root,
            "cargo",
            &[
                "fmt",
                "--manifest-path",
                "harness/Cargo.toml",
                "--all",
                "--",
                "--check",
            ],
        )
        .map_err(shell_error("lint"))?,
    )?;

    Ok(bundle(vec![
        json!({"label": "gofmt", "status": "ok", "command": "gofmt -l ."}),
        json!({"label": "go-vet", "status": "ok", "command": go_vet.command}),
        json!({"label": "cargo-fmt", "status": "ok", "command": cargo_fmt.command}),
    ]))
}

fn bundle(steps: Vec<serde_json::Value>) -> OutputBundle {
    OutputBundle {
        text: render_command_steps("lint", &steps),
        json: json!({
            "command": "lint",
            "status": "ok",
            "steps": steps,
        }),
        ndjson: vec![json!({
            "command": "lint",
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
    fn lint_bundle_reports_command_name() {
        let bundle = bundle(vec![json!({"label": "cargo-fmt", "status": "ok"})]);
        assert_eq!(bundle.json["command"], "lint");
    }
}
