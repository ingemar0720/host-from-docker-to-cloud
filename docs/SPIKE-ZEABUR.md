# Phase 0 Spike: Zeabur GitHub Integration (Merged Runbook)

This is the single deployment runbook for this repository.
Current scope is intentionally narrow: deploy this repo to Zeabur via GitHub Integration.

ECR private-image flow is explicitly deferred.

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

- ECR private image path (deferred)
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
make build
./bin/d2z check -workdir examples -f examples/docker-compose.yml
./bin/d2z analyze -workdir examples -f examples/docker-compose.yml
./bin/d2z render -workdir examples -f examples/docker-compose.yml -out /tmp/zeabur.generated.yaml
```

## Validation Matrix (Current Scope)

### A) Public image path

Target: service uses public `image:` and no private registry auth.

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

### C) Private image via ECR path (Deferred)

Not part of this spike. Track as follow-up after GitHub Integration baseline is stable.

## Dependency and Health Behavior

Capture how `depends_on` plus `healthcheck` can be represented in deployment behavior:

- [ ] Dependency-first creation order is preserved where possible
- [ ] Supported health probe types are documented
- [ ] Any gap/workaround is documented

Findings:

- Supported probes:
- Unsupported probes:
- Workaround:

## Evidence Checklist

Attach links, screenshots, or outputs for:

- [ ] Successful push-triggered deploy (public image)
- [ ] Successful push-triggered deploy (local build)
- [ ] Branch and watch-path config screenshot/notes
- [ ] Health/readiness observation
- [ ] Final generated Zeabur YAML sample used for reference

## Decisions for Next Phase

- Deploy integration mechanism: `GitHub Integration (push-to-deploy)`
- Deploy branch:
- Watch paths:
- Health/readiness policy:
- Bitwarden pattern decision (if needed):
- Deferred items:
  - ECR private-image path

## Exit Criteria

Phase 0 is complete when all are true:

- [ ] GitHub App allowlist is configured for this repo
- [ ] Push to deploy branch triggers Zeabur deploy reliably
- [ ] Public image and local build paths are validated
- [ ] Dependency/health behavior notes are captured
- [ ] Follow-up backlog (including ECR) is explicit
