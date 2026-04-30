# See PLAN.md. Variables: WORK_DIR, D2Z_FLAGS (e.g. -f docker-compose.yml -strategy zeabur.strategy.yaml), REPO_URL, ECR_REGISTRY, AWS_REGION.

D2Z       ?= $(CURDIR)/bin/d2z
WORK_DIR  ?= $(CURDIR)/work
# Pass compose/strategy flags to d2z, e.g. D2Z_FLAGS=-f compose.yaml
D2Z_FLAGS ?=
GO        ?= go
RENDER_OUT ?= $(WORK_DIR)/zeabur.generated.yaml
ZEABUR_DEPLOY_BRANCH ?= main

.PHONY: all build check analyze render test validate compose-up compose-down clone build-private push-ecr deploy-ready deploy

all: build

# Static + unit tests (no Docker required). CI runs the same checks.
validate:
	$(GO) vet ./...
	$(GO) test -race -count=1 ./...
	$(GO) build -o $(D2Z) ./cmd/d2z

build:
	mkdir -p bin
	$(GO) build -o $(D2Z) ./cmd/d2z

check: build
	$(D2Z) check -workdir "$(WORK_DIR)" $(D2Z_FLAGS) $(if $(OPTIONAL_TOOLS),-optional "$(OPTIONAL_TOOLS)",)

analyze: build
	$(D2Z) analyze -workdir "$(WORK_DIR)" $(D2Z_FLAGS)

render: build
	$(D2Z) render -workdir "$(WORK_DIR)" $(D2Z_FLAGS) -out "$(RENDER_OUT)"

test:
	$(GO) test ./...

COMPOSE_FILE ?= docker-compose.yml

compose-up: build
	docker compose -f "$(WORK_DIR)/$(COMPOSE_FILE)" --project-directory "$(WORK_DIR)" up -d

compose-down: build
	docker compose -f "$(WORK_DIR)/$(COMPOSE_FILE)" --project-directory "$(WORK_DIR)" down

clone:
	@test -n "$(REPO_URL)" || (echo "Set REPO_URL"; exit 1)
	@test -n "$(WORK_DIR)" || (echo "Set WORK_DIR"; exit 1)
	$(D2Z) clone -repo "$(REPO_URL)" -dir "$(WORK_DIR)"

build-private:
	@test -n "$(ECR_REGISTRY)" || (echo "Set ECR_REGISTRY"; exit 1)
	@test -n "$(PRIVATE_SERVICES)" || (echo "Set PRIVATE_SERVICES (space-separated service names)"; exit 1)
	docker compose -f "$(WORK_DIR)/$(COMPOSE_FILE)" --project-directory "$(WORK_DIR)" build $(PRIVATE_SERVICES)
	@echo "Tag images for $(ECR_REGISTRY) and docker push (see push-ecr)."

push-ecr:
	@test -n "$(AWS_REGION)" || (echo "Set AWS_REGION"; exit 1)
	@test -n "$(ECR_REGISTRY)" || (echo "Set ECR_REGISTRY"; exit 1)
	aws ecr get-login-password --region "$(AWS_REGION)" | docker login --username AWS --password-stdin "$(ECR_REGISTRY)"

deploy-ready: validate check analyze render
	@echo "Local pre-deploy checks passed."
	@echo "Push to branch '$(ZEABUR_DEPLOY_BRANCH)' to trigger Zeabur deploy."

deploy:
	@echo "Zeabur deploy mode: GitHub Integration (push-to-deploy)."
	@echo "Ensure the service is linked in Zeabur: Add Service -> GitHub."
	@echo "Push to branch '$(ZEABUR_DEPLOY_BRANCH)' to trigger deployment."
	@echo "Generated template available for reference: $(RENDER_OUT)"
