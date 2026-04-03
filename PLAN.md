# Host from Docker Compose to Zeabur — Project Plan

This document is the authoritative plan for tooling that clones a GitHub repository, reads its Docker Compose file, validates and tests locally, and deploys workloads to **Zeabur** (k3s-backed). It supports **public / open-image** flows and **private** flows via **AWS ECR**.

---

## 1. Goals

| Goal | Description |
|------|-------------|
| **Ingest** | Clone or update a Git repo; discover `docker-compose.yml` (and optional overrides). |
| **Validate** | Run prerequisite checks and Compose sanity after clone. |
| **Local parity** | Run tests and `docker compose` via **Makefile** targets. |
| **Classify** | Per service: public image / build from Dockerfile / Helm chart (optional) / private → ECR. |
| **Deploy** | Produce Zeabur-compatible definitions and deploy (CLI, API, or template workflow — finalized in Phase 0). |
| **Dependencies** | Use Compose **`depends_on`** and **`healthcheck`** for deployment ordering and readiness (see §6). |

Non-goals for v1 (unless reprioritized): full Swarm `deploy:` semantics, every Compose extension field, automatic license/OSI “open source” detection (use heuristics + overrides; see §5).

---

## 2. Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Makefile / CLI  │────▶│ Compose parser   │────▶│ Strategy router │
│ (entrypoints)   │     │ + classifier     │     │ per service      │
└────────┬────────┘     └────────┬─────────┘     └────────┬────────┘
         │                       │                          │
         │                       ▼                          ▼
         │               ┌───────────────┐          ┌──────────────┐
         │               │ Dependency    │          │ Artifacts:   │
         │               │ planner       │          │ Zeabur YAML, │
         │               │ (depends_on + │          │ image refs,  │
         │               │  healthcheck) │          │ Helm values  │
         │               └───────────────┘          └──────┬───────┘
         │                                                 │
         ▼                                                 ▼
   docker compose / tests                           ECR (private) │
                                                                   ▼
                                                           Zeabur (k3s)
```

| Component | Responsibility |
|-----------|----------------|
| **Makefile** | `check`, `test`, `compose-*`, `analyze`, `render`, `build-private`, `push-ecr`, `deploy`. |
| **Compose parser** | Parse v2/v3 Compose; normalize services, networks, volumes, `build`, `image`, env, secrets. |
| **Classifier** | Decide public-image vs build vs private/ECR (rules + override file). |
| **Dependency planner** | DAG from `depends_on`; readiness from `healthcheck` and `condition: service_healthy`. |
| **Renderer** | Emit Zeabur template/service definitions; optional Helm values for chart-based paths. |
| **Registry** | ECR create (optional), build, tag, push; credentials for Zeabur image pull. |

---

## 3. Makefile contract (initial)

Variables (examples): `REPO_URL`, `WORK_DIR`, `AWS_REGION`, `ECR_REGISTRY`, `ZEABUR_PROJECT`, `COMPOSE_FILE`.

| Target | Purpose |
|--------|---------|
| `make clone` | Clone or pull `REPO_URL` into `WORK_DIR`. |
| `make check` | Prerequisites + Compose validation + report. |
| `make test` | Delegate to repo tests (`go test`, `npm test`, etc.) or no-op with clear message. |
| `make compose-up` / `compose-down` | Local `docker compose` for parity. |
| `make analyze` | Print per-service classification and dependency graph. |
| `make render` | Generate Zeabur-oriented YAML (no deploy). |
| `make build-private` | Build images for private services; tag for ECR. |
| `make push-ecr` | Push to ECR. |
| `make deploy` | Deploy to Zeabur (mechanism from Phase 0 spike). |

Implementation order: `check` → `analyze` → `render` → ECR path → `deploy`.

---

## 4. Prerequisites (`make check`)

After clone, before heavy work:

- **Binaries:** `git`, `docker`, `docker compose`, `make`; optional `helm`, `aws`, Zeabur CLI.
- **Auth:** Git (SSH/HTTPS), AWS credential for ECR, Zeabur token/API key if applicable.
- **Files:** `docker-compose.yml` (or `COMPOSE_FILE`) readable; optional `.env.example` if required by policy.
- **Compose:** Valid YAML; detect unsupported features (documented matrix); **detect cycles in `depends_on`** and fail fast.

Exit non-zero on failure; print a single actionable summary.

---

## 5. Public vs private classification

Compose does not encode “open source.” Use **rules + overrides**.

| Signal | Typical classification |
|--------|-------------------------|
| `image:` from public registry (Docker Hub, GHCR public, etc.), no `build:` | Use prebuilt image (pin tag or digest when possible). |
| `build:` present; repo/policy treats stack as rebuildable | Build from Dockerfile (or suggest known Helm chart if in catalog). |
| `image:` points at private registry / ECR, or repo private, or override `PRIVATE=1` / strategy file | **Private path:** build → tag → **ECR** → Zeabur pulls with registry credentials. |

**Override file (recommended):** e.g. `zeabur.strategy.yaml` or Compose labels such as `zeabur.sourcing: private|public|auto` so edge cases are explicit.

---

## 6. Dependency management during deployment

**Policy:** Use **`depends_on`** for ordering and **`healthcheck`** for readiness. together they define how deployment steps are sequenced and when a dependent service may be considered ready.

### 6.1 `depends_on`

- Parse all `depends_on` entries per service; build a **directed acyclic graph** (DAG). If cycles exist, **error** with the cycle path.
- **Deployment / apply order:** Apply or create services in an order where **dependencies come before dependents** (topological sort). Document the chosen sort tie-breaking (e.g. lexical service name).

### 6.2 `healthcheck`

- For each service with a Compose `healthcheck`, map fields to Zeabur-supported health probes (HTTP/TCP/cmd — **confirm** Zeabur capabilities in Phase 0).
- Treat passing health as **service ready** for gating downstream steps.

### 6.3 `depends_on` conditions

- **`condition: service_healthy`:** Do not proceed with dependent service deployment (or mark dependent as blocked) until dependency healthcheck has succeeded per platform semantics.
- **`condition: service_started`:** Order only; **no** readiness guarantee—document that apps should retry connections unless a healthcheck is also defined.
- **`condition: service_completed_successfully`:** For one-shot jobs; align with Zeabur job/run-once patterns if available, otherwise document manual/jenkins-style workaround.

### 6.4 Fallbacks

- Service with `depends_on` but **no** `healthcheck` on dependency: honor **startup order** only; recommend adding `healthcheck` or `service_healthy` for production stacks.
- Zeabur cannot express a probe: preserve order; emit warning; rely on app retries and document gap.

### 6.5 Kubernetes / k3s reality

Zeabur runs on k3s; pods do not implement Compose `depends_on` natively. The tool’s job is to **translate** intent into:

1. Correct **creation order** (and retries between steps if using a script/CLI), and  
2. **Health-based readiness** where Zeabur allows, mirroring `service_healthy` + `healthcheck`.

---

## 7. Conversion strategies (per service)

1. **Public image:** Map `image`, `ports`, `environment`, volumes (note platform volume semantics).
2. **Build from Dockerfile:** `docker build` with build args/context from Compose; tag for Zeabur or ECR.
3. **Helm (optional):** Small catalog mapping common images/services to charts + values; only if Zeabur or ops process accepts it.
4. **Private:** Build → push **ECR** → reference image in Zeabur; configure **private registry** credentials on Zeabur for pull (**verify ECR pull auth** in Phase 0).

---

## 8. Security and secrets

### 8.1 Required secrets inventory

Use this as the checklist for operators and CI. **Scope** = when the secret is required; **Store in** = where it must live (never commit values to Git).

| Secret | Purpose | Required when | Typical store |
|--------|---------|---------------|----------------|
| **Git: HTTPS username + password or PAT** | Clone/pull **private** GitHub (or other Git HTTPS) repos | Private `REPO_URL` over HTTPS | CI secret vault; local env or credential helper |
| **Git: SSH private key (+ passphrase if used)** | Clone/pull private repos over SSH | Private `REPO_URL` over SSH | SSH agent; CI `SSH_PRIVATE_KEY`; `~/.ssh` locally |
| **GitHub fine-grained or classic PAT** | Submodules, LFS, API, or HTTPS clone where PAT is mandated | Repo uses submodules/LFS or HTTPS needs token | `GITHUB_TOKEN` / PAT in CI; gh auth locally |
| **AWS access key ID + secret access key** (or **session token** for STS) | `aws ecr get-login-password`, `ecr:BatchCheckLayerAvailability`, `PutImage`, `BatchGetImage`, push/pull automation | `make push-ecr`, optional ECR repo creation | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`; or OIDC role in CI |
| **AWS IAM role ARN / OIDC** (alternative to long-lived keys) | Same as above, in CI | GitHub Actions / GitLab CI with OIDC to AWS | Federated role; no static keys in repo |
| **ECR registry identity** | Not a secret by itself; **URL** is `*.dkr.ecr.<region>.amazonaws.com/<repo>` | Private image path | Config vars (non-secret) |
| **Zeabur API token / account token** | Zeabur CLI, REST API, or deploy hooks | `make deploy` via API/CLI | `ZEABUR_TOKEN` or vendor-documented name; CI vault |
| **Zeabur project / service identifiers** | Target deploy scope | Any automated deploy | Env vars (often non-secret IDs; treat per org policy) |
| **Zeabur private registry credentials** | Pull images from **ECR** (or other private registry) inside Zeabur | Private images hosted on ECR after push | Zeabur dashboard “container registry” / image pull secret; robot account **username + password** or **token** per Zeabur docs |
| **Docker Hub username + password or access token** | Pull/push if Compose uses private Docker Hub images or avoids anonymous rate limits | Private `image:` on `docker.io`; high pull volume | `DOCKER_AUTH_CONFIG` or login locally; CI secrets |
| **GHCR / GCR / other registry token** | Pull **private** base or runtime images during `docker build` or compose | `build:` or `image:` references private registry | `GITHUB_TOKEN`, JSON key, or registry-specific env |
| **`.env` / `env_file` values referenced by Compose** | Runtime DB passwords, API keys, signing keys for **local** `docker compose` | Local parity and tests that need real credentials | Local `.env` (gitignored); CI secrets injected before `compose up` |
| **Compose `secrets:` file contents** | Files mounted into containers per Compose spec | Stack uses `secrets:` | Same as above—never commit; map to Zeabur secret store at deploy |
| **Application runtime secrets** (from Compose `environment` that reference `${VAR}`) | Production DB URLs, JWT secrets, third-party API keys | Every deploy where the app needs them | **Primary:** Bitwarden Secrets Manager projects (see §8.3). **Alternative:** Zeabur env / platform secret manager |
| **Bitwarden Secrets Manager: machine account credential** | Non-interactive auth so `bw`/API can read only allowed projects | Runtime or init fetches secrets from Bitwarden Cloud | Zeabur env (bootstrap only): token / client id+secret per [Bitwarden Secrets Manager](https://bitwarden.com/help/secrets-manager/) docs—**never** commit |
| **Bitwarden: project / org identifiers** | Select vault scope in CLI/API | Using Secrets Manager | Config (often non-secret); confirm in Bitwarden UI |
| **Helm repository credentials** (optional) | `helm dependency build` / private chart repo | Helm path enabled + private chart | `HELM_REGISTRY_*` or repo `username`/`password` in CI |

Secrets **not** required for a minimal **public-only** path (public repo, public images only): typically **Git** auth optional, **no** ECR push keys, **no** Zeabur registry pull secret (unless Zeabur still needs a token for deploy). Still usually need **Zeabur deploy token** if deploy is automated. **Bitwarden** bootstrap is only needed if the stack uses §8.3 (fetch-at-runtime or CI-based SM access).

### 8.2 Principles

- Do not bake secrets into images; runtime values come from **Bitwarden Secrets Manager** (preferred) or Zeabur env as a fallback.
- Map Compose `environment` / `env_file` / `secrets` to **named secrets in Bitwarden projects** where possible; `analyze` / `render` should list required keys so operators create matching SM entries.
- **Zeabur surface:** minimize per-project duplication—ideally **one bootstrap credential** (machine account) on Zeabur per environment, not every DB password and API key (see §8.3).
- **ECR:** separate IAM: **push** principal (operator/CI) vs **pull** principal (Zeabur registry integration); least privilege on both.
- **Rotation:** document owners for Git credentials, AWS keys, Zeabur deploy token, Bitwarden machine accounts, and registry robot accounts; prefer OIDC + short-lived AWS for CI over static keys.

### 8.3 Bitwarden Cloud as source of truth (Secrets Manager)

**Goal:** Keep **application and integration secrets** in **Bitwarden Cloud** so operators do not configure **every** key again in **Zeabur** for each project—while still accepting a **small bootstrap** on Zeabur.

**Why Secrets Manager, not personal “folders”:** On a **free personal** Password Manager vault, **folders** are UI organization only—they are **not** an access-control boundary for automation. A personal **API key** / automation identity can typically see **everything** that identity can access, which is poor for a Zeabur-wide integration. **Bitwarden Secrets Manager** uses **projects** and **machine accounts** with **scoped** access—appropriate for “Zeabur may only read these secrets.”

**Free tier (verify current limits on official pricing):** Bitwarden offers a **Secrets Manager free plan** with capped **projects**, **machine accounts**, and users—sufficient for modest Zeabur footprints if split sensibly (e.g. one project per environment or per app). Re-check [Secrets Manager plans](https://bitwarden.com/help/secrets-manager-plans) before locking counts.

| Concept | Use in this project |
|---------|---------------------|
| **Project** | Slice of secrets (e.g. `zeabur-prod-myapp`, `zeabur-staging-myapp`). Analog to a “folder” for operations, but **ACL-backed**. |
| **Machine account** | Credential used on Zeabur (or in CI). Grant **read-only** access to **only** the projects this workload needs. **One account per environment** (or per service) to limit blast radius. |
| **Secrets / keys** | Map Compose `${VAR}` names to SM secret names so render/deploy docs stay predictable. |

**Bootstrap on Zeabur (unavoidable):** Zeabur must receive **something** to authenticate to Bitwarden (machine account token or equivalent per Bitwarden docs). That is **one integration secret per env** (or per service), **not** a full duplicate of every app secret in the Zeabur UI.

**Runtime patterns (choose one per stack):**

1. **Fetch at container start:** Entrypoint or init runs Bitwarden CLI or HTTP API → loads secrets → exports env or writes files → starts the app. **Tradeoff:** Startup depends on Bitwarden API availability; implement retries and logging.
2. **CI / `make deploy` only:** Pipeline reads Secrets Manager and **pushes** values into Zeabur **once per deploy**. **Tradeoff:** Copies exist on Zeabur, but operators still **edit once** in Bitwarden; less runtime dependency on Bitwarden.

**Local dev:** Developers use `bw login` / personal access or a **non-production** machine account; `.env` stays gitignored. `make check` can verify `bw` session or document bypass for offline work.

**Phase 0:** Confirm exact env var names for machine accounts, API rate limits, and whether Zeabur’s process model favors init fetch vs CI sync for target services.

---

## 9. Phase roadmap

| Phase | Outcome |
|-------|---------|
| **0 — Spike** | Manual deploy on Zeabur: public image, local build, ECR image; document exact steps, health probes, private registry; validate Bitwarden Secrets Manager machine account + project scoping and bootstrap env on Zeabur. |
| **1 — Skeleton** | Repo layout, Makefile scaffold, `check`, parser + `analyze` (DAG + health summary). |
| **2 — Local** | `compose-up`/`down`, `test` delegation. |
| **3 — Render** | Generate Zeabur YAML/template from normalized model + dependency order. |
| **4 — Private path** | ECR tag/push; Zeabur pull credentials. |
| **5 — Deploy** | `make deploy` wired to chosen Zeabur integration. |
| **6 — Hardening** | Override file, integration tests, Compose compatibility matrix in `docs/COMPOSE-MATRIX.md` (optional). |

---

## 10. Risks and compatibility

| Risk | Mitigation |
|------|------------|
| Compose features unsupported on Zeabur | Maintained compatibility matrix; fail or warn per field. |
| `depends_on` without health | Document ordering-only semantics; suggest healthchecks. |
| Helm vs Zeabur native | Defer Helm until spike confirms fit. |
| ECR + Zeabur auth | Spike in Phase 0; document required Zeabur settings. |
| Bitwarden API unavailable at container start | Retries/backoff; healthchecks; optional CI-sync pattern to Zeabur env as fallback. |
| Secrets Manager free-tier caps (projects / machine accounts) | Model env boundaries; upgrade or split workloads if limits block. |

---

## 11. Deliverables checklist

- [x] `PLAN.md` (this document) committed in repo root.
- [ ] Phase 0 spike notes (add `docs/SPIKE-ZEABUR.md` after spike).
- [x] Makefile + `cmd/d2z` implementing `check`, `analyze`, `render`, `clone` (ECR/deploy stubs in Makefile).
- [x] Optional: `zeabur.strategy.yaml` example under `examples/`.
- [x] Bitwarden pointer in `docs/BITWARDEN.md`.

---

## 12. Document history

| Date | Change |
|------|--------|
| 2026-04-02 | Initial plan; dependency policy uses `depends_on` + `healthcheck`. |
| 2026-04-02 | §8.1: full required-secrets inventory (Git, AWS ECR, Zeabur, registries, Compose/runtime). |
| 2026-04-02 | §8.3: Bitwarden Secrets Manager as primary secret store; free-tier notes; bootstrap + runtime/CI patterns; §10 risks. |
