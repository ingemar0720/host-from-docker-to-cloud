# host-from-docker-to-cloud

CLI and Makefile helpers to validate **Docker Compose** projects, classify services (public image vs local build vs private/ECR), analyze **`depends_on`** / **`healthcheck`**, and emit **Zeabur-oriented** YAML. See **PLAN.md** for full design, secrets (including Bitwarden), and phases.

## Name: `d2z`

The command-line tool is **`d2z`**: short for **Docker (Compose) → Zeabur**. The binary is built as `./bin/d2z` from `cmd/d2z`. The Makefile exposes the same tool via the `D2Z` / `D2Z_FLAGS` variables.

## Requirements

- Go **1.25+** (to build)
- **Docker** + **docker compose** for `make check` and local stack targets
- Compose files should define a top-level **`name:`** (or set `COMPOSE_PROJECT_NAME` in the environment) so compose-go can load the project

## Validation

**Automated (no Docker needed for unit tests):**

```bash
make validate   # go vet + go test -race ./... + build ./cmd/d2z
# or
go vet ./...
go test -race -count=1 ./...
```

Tests cover Compose **load + `depends_on` / `service_healthy`**, **dependency cycle detection**, **Zeabur render** (YAML shape, health + ordering), **strategy classification**, and **env mapping**. CI runs the same checks and additionally runs `./bin/d2z check` on `examples/` (needs Docker on the runner).

**Integration / manual:**

```bash
make build
./bin/d2z check -workdir examples -f examples/docker-compose.yml
./bin/d2z analyze -workdir examples -f examples/docker-compose.yml
```

`d2z check` verifies `git`, **Docker**, **`docker compose`**, loads your compose file(s), and fails on **`depends_on` cycles**.

## Build

```bash
make build    # produces ./bin/d2z
make test     # go test ./... (no rebuild)
```

## CLI

```bash
./bin/d2z check   -workdir /path/to/repo [-f compose.yml] [-optional aws,zeabur,helm,bw]
./bin/d2z analyze -workdir /path/to/repo [-f compose.yml] [-strategy zeabur.strategy.yaml]
./bin/d2z render  -workdir /path/to/repo [-f compose.yml] -out zeabur.generated.yaml
./bin/d2z clone   -repo "$REPO_URL" -dir "$WORK_DIR"
```

Compose `-f` paths may be relative to the current working directory or to `-workdir`.

## Makefile (after `make clone` or copying a project into `work/`)

```bash
export WORK_DIR=$(pwd)/work
export D2Z_FLAGS='-f docker-compose.yml'   # optional if auto-detect finds the file
make check analyze render
make compose-up    # docker compose up in WORK_DIR
```

Set `OPTIONAL_TOOLS=aws,zeabur` on `make check` when you want those binaries verified.

## Examples

```bash
make build
./bin/d2z analyze -workdir examples -f examples/docker-compose.yml
./bin/d2z render  -workdir examples -f examples/docker-compose.yml -out /tmp/out.yaml
```

## Generated Zeabur YAML

Output uses `apiVersion: zeabur.com/v1` / `kind: Project` as a **starting point**. Confirm fields against the current [Zeabur documentation](https://zeabur.com/docs) before applying in production.
