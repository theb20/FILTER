# Contrat, génération et build. Le code serveur et le client du frontend sont GÉNÉRÉS depuis le contrat,
# jamais écrits à la main en parallèle, jamais commités (pkg/gen, services/gateway/internal/api).
SHELL := /bin/bash
PY    ?= python3
BASE  ?= origin/main
OGEN  ?= ogen
MODULES := pkg services/identity services/gateway services/catalog services/content

.PHONY: help deps contract-check states-check openapi-check proto-lint proto-breaking generated-go \
        generate generate-proto generate-gateway generate-client fmt-check build test test-integration

help:  ## liste des cibles
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*##/\t/'

deps:  ## dépendances (Python du contrat, plugins Go pinés)
	$(PY) -m pip install -r tools/requirements.txt
	@echo "buf : brew install buf"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
	go install github.com/ogen-go/ogen/cmd/ogen@v1.25.0

# ── Contrat ─────────────────────────────────────────────────────────────────
contract-check: states-check openapi-check proto-lint generated-go  ## tous les contrôles du contrat

states-check:  ## machine à états commande × paiement (cohérence exhaustive)
	$(PY) tools/check_order_states.py

openapi-check: ## OpenAPI valide + règles du projet + correspondance route ↔ RPC (périmètre validé)
	$(PY) tools/check_contract.py

proto-lint:    ## buf lint (STANDARD, sans exception)
	cd proto && buf lint

proto-breaking: ## buf breaking contre BASE (contrats VALIDÉS uniquement ; brouillons exclus)
	cd proto && if git cat-file -e $(BASE):backend/proto/buf.yaml 2>/dev/null; then \
	  buf breaking --against "../../.git#ref=$(BASE),subdir=backend/proto"; \
	else echo "Aucune baseline sur $(BASE) : contrôle ignoré (premier contrat)."; fi

generated-go:  ## les .proto génèrent-ils du Go qui compile ? (n'écrit rien dans le dépôt)
	tools/check_generated_go.sh

# ── Génération (jamais écrite à la main, jamais commitée) ───────────────────
generate: generate-proto generate-gateway  ## tout le code dérivé du contrat

generate-proto:   ## .proto -> pkg/gen (messages + gRPC)
	cd proto && buf generate

generate-gateway: ## openapi.yaml -> services/gateway/internal/api (serveur ogen, routes VALIDÉES : api/ogen.yml)
	$(OGEN) --config api/ogen.yml --target services/gateway/internal/api --package api --clean api/openapi.yaml 2>&1 | { grep -v -E "^(INFO|WARN)" || true; }
	$(PY) tools/gen_permissions.py

generate-client: ## client TypeScript du frontend depuis openapi.yaml — à brancher en phase 7
	@echo "Phase 7 : génération du client typé du frontend depuis api/openapi.yaml (jamais écrit à la main)."

# ── Qualité du code ─────────────────────────────────────────────────────────
fmt-check:     ## gofmt : aucun fichier mal formaté (code généré exclu)
	@bad="$$(gofmt -l $$(find $(MODULES) -name '*.go' -not -path '*/gen/*' -not -path '*/internal/api/*'))"; \
	if [ -n "$$bad" ]; then echo "À formater :"; echo "$$bad"; exit 1; fi

build: generate ## compile tous les modules
	@for m in $(MODULES); do (cd $$m && go build ./...) || exit 1; done

test: generate  ## vet + tests de tous les modules (les tests d'intégration sont ignorés sans Docker ni TEST_*_URL)
	@for m in $(MODULES); do (cd $$m && go vet ./... && go test ./...) || exit 1; done

test-integration: generate  ## tests avec PostgreSQL + NATS réels (testcontainers : Docker requis) ; AUCUN test ignoré
	@for m in $(MODULES); do (cd $$m && REQUIRE_INTEGRATION=1 go test -race -count=1 ./...) || exit 1; done
