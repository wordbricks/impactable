use crate::util::{CommandError, OutputBundle, repo_root, run_exec};
use serde::Serialize;
use serde_json::json;
use std::path::{Path, PathBuf};

#[derive(Debug, Serialize)]
struct AuditCheck {
    label: String,
    path: String,
    kind: String,
    passed: bool,
}

pub fn run(path: PathBuf) -> Result<OutputBundle, CommandError> {
    let root = repo_root(&path)
        .or_else(|_| path.canonicalize().map_err(anyhow::Error::from))
        .map_err(audit_error)?;

    let mut checks = required_file_checks(&root);
    checks.extend(required_dir_checks(&root));
    checks.push(build_check(&root));

    let failed = checks.iter().filter(|check| !check.passed).count();
    let passed = failed == 0;
    let summary = json!({
        "passed": passed,
        "failed": failed,
        "total": checks.len(),
    });

    let text = render_text(&checks, passed);
    let json_body = json!({
        "command": "audit",
        "status": if passed { "ok" } else { "failed" },
        "passed": passed,
        "summary": summary,
        "checks": checks,
    });

    if !passed {
        return Err(
            CommandError::new("audit", "audit_failed", "Harness audit failed").with_details(
                json!({
                    "summary": summary,
                    "checks": checks,
                }),
            ),
        );
    }

    let mut ndjson = checks
        .iter()
        .map(|check| {
            json!({
                "command": "audit",
                "kind": check.kind,
                "path": check.path,
                "label": check.label,
                "passed": check.passed,
            })
        })
        .collect::<Vec<_>>();
    ndjson.push(json!({
        "command": "audit",
        "summary": summary,
        "passed": true,
    }));

    Ok(OutputBundle {
        text,
        json: json_body,
        ndjson,
    })
}

fn required_file_checks(root: &Path) -> Vec<AuditCheck> {
    let files = vec![
        "AGENTS.md",
        "ARCHITECTURE.md",
        "NON_NEGOTIABLE_RULES.md",
        "docs/PLANS.md",
        "docs/design-docs/index.md",
        "docs/design-docs/local-operations.md",
        "docs/design-docs/worktree-isolation.md",
        "docs/design-docs/observability-shim.md",
        "docs/exec-plans/tech-debt-tracker.md",
        "docs/product-specs/index.md",
        "docs/product-specs/harness-demo-app.md",
        "Makefile.harness",
        "harness/Cargo.toml",
        ".github/workflows/harness.yml",
    ];
    files
        .into_iter()
        .map(|path| AuditCheck {
            label: format!("{path} exists"),
            path: path.to_string(),
            kind: "file".to_string(),
            passed: root.join(path).exists(),
        })
        .collect()
}

fn required_dir_checks(root: &Path) -> Vec<AuditCheck> {
    let dirs = vec![
        "docs/design-docs",
        "docs/exec-plans/active",
        "docs/exec-plans/completed",
        "docs/product-specs",
        "docs/references",
        "docs/generated",
    ];
    dirs.into_iter()
        .map(|path| AuditCheck {
            label: format!("{path} exists"),
            path: path.to_string(),
            kind: "directory".to_string(),
            passed: root.join(path).is_dir(),
        })
        .collect()
}

fn build_check(root: &Path) -> AuditCheck {
    let passed = run_exec(
        root,
        "cargo",
        &[
            "build",
            "--release",
            "--manifest-path",
            "harness/Cargo.toml",
        ],
    )
    .map(|result| result.status == 0)
    .unwrap_or(false);
    AuditCheck {
        label: "harnesscli builds successfully".to_string(),
        path: "harness/Cargo.toml".to_string(),
        kind: "build".to_string(),
        passed,
    }
}

fn render_text(checks: &[AuditCheck], passed: bool) -> String {
    let mut lines = Vec::new();
    for check in checks {
        let status = if check.passed { "[ok]" } else { "[missing]" };
        lines.push(format!("{status} {}", check.label));
    }
    if passed {
        lines.push("Harness audit passed.".to_string());
    } else {
        lines.push("Harness audit failed.".to_string());
    }
    lines.join("\n")
}

fn audit_error(err: anyhow::Error) -> CommandError {
    CommandError::new("audit", "audit_failed", err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn required_files_include_agents() {
        let checks = required_file_checks(Path::new("/tmp"));
        assert!(checks.iter().any(|check| check.path == "AGENTS.md"));
    }
}
