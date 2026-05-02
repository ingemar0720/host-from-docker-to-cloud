# Phase 0 Spike: Zeabur GitHub Integration (Merged Runbook)

This is the single deployment runbook for this repository.
Current scope is intentionally narrow: deploy this repo to Zeabur via GitHub Integration.

References:

- https://zeabur.com/docs/en-US/deploy/methods/github-integration
- https://zeabur.com/docs/en-US/deploy

## Goal

Validate and document the exact push-to-deploy workflow for this repository:

1. Link repo via Zeabur GitHub integration.
2. Push commits to deploy branch.
3. Confirm service boots and health behavior is acceptable.

## Scope

In scope now:

- GitHub Integration setup (including allowlist and watch paths)
- Public image and local build deployment behavior
- Dependency/health behavior notes for this repo

Out of scope now:

- Full production hardening and full compatibility matrix
- Full Bitwarden runtime integration design

## One-Time Setup

### 1) Link GitHub account to Zeabur

- Zeabur Dashboard -> Account Settings -> Integrations -> link GitHub account.

### 2) Install Zeabur GitHub App and set repository allowlist

- GitHub -> Settings -> Integrations -> Applications -> Installed GitHub Apps -> Zeabur -> Configure.
- Set repository access to **Only select repositories**.
- Add this repository to the allowlist.

### 3) Create service from GitHub

- Zeabur Project -> Add Service -> GitHub.
- Select this repository.
- Set deploy branch (default: `main`).

### 4) Configure runtime environment

- Zeabur Service -> Configuration -> Environment Variables.
- Add required runtime variables/secrets for the app.
- See `docs/BITWARDEN.md` and `PLAN.md` section 8.3 for secret strategy.

### 5) Configure watch paths (recommended)

Use watch paths to avoid unnecessary redeploys.

Suggested baseline for this repo:

- `cmd/**`
- `internal/**`
- `go.mod`
- `go.sum`
- `Dockerfile*`
- `docker-compose*.yml`

## Baseline Local Commands

Run locally before pushing:

```bash
make deploy-ready WORK_DIR=examples D2Z_FLAGS='-f examples/docker-compose.yml' RENDER_OUT=/tmp/zeabur.generated.yaml
```

### Latest Local Verification (2026-04-30)

- [x] `make deploy-ready` passes with Docker daemon running.
- [x] `d2z check` passes (tools + compose load + no dependency cycles).
- [x] `d2z analyze` confirms:
  - Deployment order is deterministic with lexical tie-break (`db` before `api` in current example).
  - `api` classified as `build-from-source`, depends on `db` with `condition=service_healthy`.
  - `db` classified as `image-dockerhub-public` (`postgres:16-alpine`) with healthcheck.
- [x] `d2z render` writes output to `/tmp/zeabur.generated.yaml`.

## Validation Matrix (Current Scope)

### A) Public image path

Target: service uses public `image:`.

- [ ] Deploy succeeds from push
- [ ] Service starts successfully
- [ ] Health/readiness checks behave as expected
- [ ] Route/port is reachable

Notes:

- Deploy branch:
- Zeabur service name:
- Observed behavior:

### B) Local build path

Target: service uses `build:` in Compose.

- [ ] Zeabur build succeeds from push
- [ ] Runtime behavior matches local expectation
- [ ] Health/readiness checks behave as expected

Notes:

- Build context and args:
- Divergence from local compose behavior:

## Dependency and Health Behavior

Capture how `depends_on` plus `healthcheck` can be represented in deployment behavior:

- [ ] Dependency-first creation order is preserved where possible
- [ ] Supported health probe types are documented
- [ ] Any gap/workaround is documented

Findings:

- Supported probes (tooling-level): command-style healthcheck (`CMD-SHELL`) and `service_healthy` dependency intent are parsed and rendered.
- Unsupported probes: pending Zeabur runtime confirmation during push-triggered deploy.
- Workaround: if a probe mapping gap is found in Zeabur, keep dependency order and require app-level retries.

## Evidence Checklist

Attach links, screenshots, or outputs for:

- [ ] Successful push-triggered deploy (public image)
- [ ] Successful push-triggered deploy (local build)
- [ ] Branch and watch-path config screenshot/notes
- [ ] Health/readiness observation
- [ ] Final generated Zeabur YAML sample used for reference

## First Push Test (Execute Now)

Use this sequence for the first real Zeabur validation:

1. Confirm Zeabur GitHub service is linked to this repository and branch `main`.
2. Confirm repository allowlist includes this repository.
3. Confirm watch paths are configured (or disabled for broad trigger).
4. Run local gate:

```bash
make deploy-ready WORK_DIR=examples D2Z_FLAGS='-f examples/docker-compose.yml' RENDER_OUT=/tmp/zeabur.generated.yaml
```

If any service is classified `image-dockerhub-private`, configure Docker Hub credentials in Zeabur before pushing.

5. Push a commit to `main` and wait for Zeabur deploy to start.
6. Capture logs and fill the fields below.

Suggested commit message for deploy trigger:

`chore: trigger first zeabur github-integration deploy test`

Record this after the push:

- Commit SHA:
- Push time (local timezone):
- Zeabur deploy ID / URL:
- Build status (pass/fail):
- Runtime status (healthy/unhealthy):
- Service URL:
- Notes:

## Decisions for Next Phase

- Deploy integration mechanism: `GitHub Integration (push-to-deploy)`
- Deploy branch: `main` (current default)
- Watch paths (baseline to configure): `cmd/**`, `internal/**`, `go.mod`, `go.sum`, `Dockerfile*`, `docker-compose*.yml`
- Health/readiness policy:
- Bitwarden pattern decision (if needed):

## Exit Criteria

Phase 0 is complete when all are true:

- [ ] GitHub App allowlist is configured for this repo
- [ ] Push to deploy branch triggers Zeabur deploy reliably
- [ ] Public image and local build paths are validated
- [ ] Dependency/health behavior notes are captured
