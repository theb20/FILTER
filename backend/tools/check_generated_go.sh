#!/usr/bin/env bash
# Vérifie que les .proto génèrent du Go qui COMPILE (go build + go vet), sans écrire dans le dépôt.
# Prérequis : buf, go, protoc-gen-go, protoc-gen-go-grpc dans le PATH (voir proto/buf.gen.yaml).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/gen"
sed "s#\.\./pkg/gen#$TMP/gen#" "$ROOT/proto/buf.gen.yaml" > "$TMP/buf.gen.yaml"
(cd "$ROOT/proto" && buf generate --template "$TMP/buf.gen.yaml")
printf 'module github.com/theb20/FILTER/backend/pkg\n\ngo 1.26\n' > "$TMP/go.mod"
(cd "$TMP" && GOFLAGS=-mod=mod go mod tidy >/dev/null 2>&1 && go build ./... && go vet ./...)
echo "Go généré : $(find "$TMP/gen" -name '*.go' | wc -l | tr -d ' ') fichiers, compile et passe go vet."
