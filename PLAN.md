# Host from Docker Compose to Zeabur — Project Plan

This document is the authoritative plan for tooling that clones a GitHub repository, reads its Docker Compose file, validates and tests locally, and deploys workloads to **Zeabur** (k3s-backed) via **GitHub Integration (push-to-deploy)**.

AWS/ECR is out of scope for this project.

---

## 1. Goals

| Goal | Description |
|------|-------------|
| **Ingest** | Clone or update a Git repo; discover `docker-compose.yml` (and optional overrides). |
| **Validate** | Run prerequisite checks and Compose sanity after clone. |
| **Local parity** | Run tests and `docker compose` via Makefile targets. |
| **Classify** | Per service: build from Dockerfile, Docker Hub public image, or Docker Hub private image. |
| **Deploy** | Prepare Zeabur-compatible output and deploy via GitHub Integration push-to-deploy. |
| **Dependencies** | Use Compose `depends_on` and `healthcheck` for deployment ordering/readiness guidance. |

Non-goals for v1 (unless reprioritized): full Swarm `deploy:` semantics, every Compose extension field, and non-Docker-Hub private-registry pipelines.

---

## 2. Architecture

```text
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Makefile / CLI  │────▶│ Compose parser   │────▶│ Strategy router │
│ (entrypoints)   │     │ + classifier     │     │ per service     │
└────────┬────────┘     └────────┬─────────┘     └────────┬────────┘
         │                       │                          │
         │                       ▼                          ▼
         │               ┌───────────────┐          ┌──────────────┐
         │               │ Dependency    │          │ Artifacts:   │
         │               │ planner       │          │ Zeabur YAML  │
         │               │ (depends_on + │          │ + analysis   │
         │               │  healthcheck) │          └──────┬───────┘
         │               └───────────────┘                 │
         ▼                                                 ▼
   docker compose / tests                           GitHub push triggers
                                                    Zeabur deployment
```

| Component | Responsibility |
|-----------|----------------|
| **Makefile** | `check`, `test`, `compose-*`, `analyze`, `render`, `deploy-ready`, `deploy`. |
| **Compose parser** | Parse v2/v3 Compose; normalize services, networks, volumes, `build`, `image`, env, secrets. |
| **Classifier** | Decide build-from-source vs image pull and whether image pull requires Docker Hub credentials. |
| **Dependency planner** | DAG from `depends_on`; readiness intent from `healthcheck` and `condition: service_healthy`. |
| **Renderer** | Emit Zeabur-oriented YAML template for review/reference. |
| **GitHub Integration** | Zeabur GitHub App allowlist, branch binding, watch paths, auto-deploy on push. |

---

## 3. Makefile contract

Variables (examples): `REPO_URL`, `WORK_DIR`, `ZEABUR_DEPLOY_BRANCH`, `COMPOSE_FILE`, `D2Z_FLAGS`, `RENDER_OUT`.

| Target | Purpose |
|--------|---------|
| `make clone` | Clone or pull `REPO_URL` into `WORK_DIR`. |
| `make check` | Prerequisites + Compose validation + report. |
| `make test` / `make validate` | Unit checks (`go test`, `go vet`) and build validation. |
| `make compose-up` / `compose-down` | Local `docker compose` for parity. |
| `make analyze` | Print per-service classification + dependency order + health summary. |
| `make render` | Generate Zeabur-oriented YAML (no deploy). |
| `make deploy-ready` | Local pre-push gate (`validate` + `check` + `analyze` + `render`). |
| `make deploy` | Print/confirm push-to-deploy behavior for Zeabur GitHub Integration. |

Implementation order: `check` -> `analyze` -> `render` -> `deploy-ready` -> `deploy`.

---

## 4. Prerequisites (`make check`)

After clone, before heavy work:

- **Binaries:** `git`, `docker`, `docker compose`, `make`; optional `helm`, Zeabur CLI, `bw`.
- **Access:** GitHub repo access (public or private via GitHub App allowlist), Zeabur project access, and Docker Hub credentials if any `image:` is private.
- **Files:** `docker-compose.yml` (or `COMPOSE_FILE`) readable; optional `.env.example` if required by policy.
- **Compose:** Valid YAML; detect unsupported fields and `depends_on` cycles (fail fast on cycle).

Exit non-zero on failure with a single actionable summary.

---

## 5. Service classification model

Compose does not directly encode deployment strategy. Use rules + optional overrides.

Use two axes:

1. **Repo access**
   - `repo-public`: Zeabur can read source without private-repo grant.
   - `repo-private`: Zeabur needs GitHub App allowlist for this repo.
2. **Service source**
   - `build-from-source`: service has `build:`.
   - `image-dockerhub-public`: service has `image:` from Docker Hub public.
   - `image-dockerhub-private`: service has `image:` from Docker Hub private (requires registry credentials on Zeabur).

| Signal | Typical classification |
|--------|-------------------------|
| `build:` present | `build-from-source` |
| `image:` and no `build:` + public Docker Hub access | `image-dockerhub-public` |
| `image:` and no `build:` + private Docker Hub repo | `image-dockerhub-private` |

Enforcement: `make check` / `d2z check` fails when a service defines both `build` and `image`, or neither.

Override file (recommended): `zeabur.strategy.yaml` or labels (for example `zeabur.sourcing: build|image-public|image-private|auto`) so edge cases are explicit.

---

## 6. Dependency management during deployment

Policy: use `depends_on` for order and `healthcheck` for readiness intent.

### 6.1 `depends_on`

- Parse all `depends_on` entries and build a DAG.
- Detect cycles and fail with cycle details.
- Deployment order uses topological sort with lexical tie-breaks for deterministic output.

### 6.2 `healthcheck`

- Parse Compose `healthcheck` fields and surface them in analysis/render output.
- Treat passing health as readiness intent for downstream services.

### 6.3 `depends_on` conditions

- `condition: service_healthy`: readiness-gated dependency.
- `condition: service_started`: order only.
- `condition: service_completed_successfully`: one-shot job dependency intent.

### 6.4 Fallbacks

- Dependency without healthcheck: order only; recommend app-level retries.
- If Zeabur cannot map a probe directly, keep order and document workaround.

### 6.5 Kubernetes / k3s reality

Zeabur runs on k3s; Compose `depends_on` is not a native Kubernetes primitive. The tool translates intent into deterministic order + probe metadata so operators can verify behavior in Zeabur.

---

## 7. Conversion strategies (per service)

1. **Prebuilt image:** map `image`, `ports`, `environment`, and relevant metadata.
2. **Build from Dockerfile:** capture `build` context/dockerfile/build args.
3. **Helm (optional):** only if accepted by actual Zeabur operating workflow.

---

## 8. Security and secrets

### 8.1 Required secrets inventory (current scope)

| Secret | Purpose | Required when | Typical store |
|--------|---------|---------------|----------------|
| GitHub App repo allowlist | Grant Zeabur access to private source repository | Repo is private | GitHub App configuration |
| Zeabur token (optional) | CLI/API automation (if used) | Non-GitHub deploy automation | CI vault / local env |
| Docker Hub username/token | Pull private Docker Hub images referenced by `image:` | Service uses private Docker Hub image | Zeabur container registry credentials / secret vault |
| Runtime app secrets (`${VAR}`) | App credentials and runtime keys | Service runtime | Bitwarden Secrets Manager (preferred) or Zeabur env |
| Compose `secrets:` content | File-based runtime secrets | Stack uses Compose secrets | Secret manager / gitignored local files |
| Bitwarden machine account credential | Scoped secret fetch from Bitwarden Cloud | If Bitwarden integration is enabled | Zeabur env / CI secret vault |

Notes:

- GitHub App repository allowlist is required configuration.
- Do not commit plaintext secret values.

### 8.2 Principles

- Never bake secrets into images.
- Prefer Bitwarden Secrets Manager as source of truth; Zeabur env as fallback.
- Keep Zeabur-side bootstrap minimal (one integration credential per environment where possible).
- Document ownership and rotation for each credential.

### 8.3 Bitwarden Cloud as source of truth (optional in v1)

If enabled:

- Use Bitwarden Secrets Manager projects and machine accounts with scoped access.
- Keep only bootstrap credential in Zeabur env.
- Choose either runtime fetch or CI sync based on reliability/operational tradeoff.

---

## 9. Phase roadmap

| Phase | Outcome |
|-------|---------|
| **0 — Spike** | Manual deploy via GitHub Integration: configure allowlist/branch/watch paths, validate push-triggered deploy, and record health/dependency observations. |
| **1 — Skeleton** | Makefile scaffold + parser + `check` + `analyze`. |
| **2 — Local** | `compose-up`/`down`, test delegation, local parity workflow. |
| **3 — Render** | Generate Zeabur YAML/template from normalized model + dependency order. |
| **4 — Deploy flow hardening** | `deploy-ready` workflow, spike evidence checklist, watch-path tuning. |
| **5 — Hardening** | Integration tests and Compose compatibility matrix (`docs/COMPOSE-MATRIX.md`, optional). |

---

## 10. Risks and compatibility

| Risk | Mitigation |
|------|------------|
| Compose fields unsupported on Zeabur | Maintain compatibility matrix; fail or warn by field. |
| `depends_on` without health | Document order-only semantics; recommend retries/healthchecks. |
| Watch paths too narrow or too broad | Start with baseline patterns and validate using push tests. |
| Bitwarden/API dependency issues | Add retries/backoff or choose CI-sync fallback pattern. |

---

## 11. Deliverables checklist

- [ ] `PLAN.md` (this document) committed in repo root.
- [ ] Phase 0 spike notes updated in `docs/SPIKE-ZEABUR.md`.
- [ ] Makefile + CLI commands implementing section 3.
- [ ] Optional: `zeabur.strategy.yaml` schema/examples.
- [ ] Optional: Bitwarden layout details in `docs/BITWARDEN.md`.

---

## 12. Document history

| Date | Change |
|------|--------|
| 2026-04-02 | Initial plan with Compose dependency/health policy. |
| 2026-04-30 | Shifted to GitHub Integration as active deploy path. |
| 2026-05-02 | Removed AWS/ECR scope; plan now targets direct GitHub-to-Zeabur deployment only. |
