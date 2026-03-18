# Harness Demo App

## Purpose

The repository does not yet have a user-facing application, so the harness boots a deterministic demo app to validate worktree isolation, port derivation, health checks, and browser automation.

## Required Surface

- Root page: `/`
  - Title contains `Impactable Harness Demo`
  - Body shows:
    - repository name
    - worktree ID
    - runtime root
    - selected port
- Health endpoint: `/healthz`
  - Returns HTTP 200 with body `ok`

## Runtime Contract

- The app is served from `.worktree/<worktree_id>/demo-app/`.
- `harnesscli boot start` creates or refreshes the demo app assets before launch.
- The boot command blocks until `/healthz` returns success.
- `harnesscli boot status` returns the app URL and healthcheck URL in structured form.
