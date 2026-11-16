# Backend FilterFinder

Microservices Go + contrat d'API. **Phase 2 en cours** : contrat écrit ; `pkg/`, gateway et un premier lot d'identity (inscription, sessions) implémentés en mémoire (voir `docs/IDENTITY-GATEWAY-MAPPING.md` §7).

| Dossier | Contenu |
|---|---|
| `api/openapi.yaml` | Contrat HTTP du gateway (OpenAPI 3.1). Chaque tag porte `x-status: draft \| validated`. |
| `api/route-rpc-map.yaml` | Correspondance exacte route ↔ RPC du périmètre validé. |
| `proto/` | Contrats gRPC internes et événements (`buf lint` / `buf breaking`). En-tête `STATUS: draft \| validated`. |
| `pkg/`, `services/gateway`, `services/identity` | Code Go (workspace `go.work`). Le code dérivé du contrat (`pkg/gen`, `services/gateway/internal/api`) est GÉNÉRÉ par `make generate` et jamais commité. |
| `deploy/` | Docker Compose (PostgreSQL, Redis, NATS, identity, gateway) et Dockerfile — non testés (Docker absent de la machine de développement). |
| `tools/` | Contrôles de cohérence (OpenAPI, machine à états des commandes, génération Go). |
| `docs/` | Propositions et décisions (voir `DECISIONS-PHASE2.md`, `PROPOSITION-STATUTS.md`, `IDENTITY-GATEWAY-MAPPING.md`). |

## Contrôles

```bash
make deps              # une fois
make contract-check    # états + OpenAPI + buf lint + génération Go qui compile
make proto-breaking    # compatibilité (contrats validés), contre origin/main
```

## Règle de génération

Le code serveur (Go) et le client du frontend sont **générés** depuis `api/openapi.yaml` et `proto/` : jamais écrits à la main en parallèle.
Le code Go des `.proto` se génère avec `make generate` (plugins locaux, voir `proto/buf.gen.yaml`).

## Statut de validation

Validé : conventions communes, `proto/common`, `proto/events`, gateway, service `identity`.
Brouillon : tout le reste, à relire route par route avant chaque phase.

## Développement

```bash
make deps && make generate   # outils + code dérivé du contrat
make test                    # vet + tests de pkg, identity, gateway
```

## Persistance et tests d'intégration

- **identity** : `STORE=postgres` (défaut hors `ENV=dev`) applique les migrations goose au démarrage (`pkg/outbox` puis identity, tables de version séparées), écrit les événements dans l'**outbox** dans la même transaction que les données, et les relaie vers **NATS JetStream** (déduplication par identifiant d'événement ; consommateurs idempotents via `outbox.Process`). `STORE=memory` n'existe que pour `ENV=dev` et les tests unitaires.
- **`pkg/`** réutilisable tel quel par les autres services : `pgxtx` (unité de travail), `outbox` (Publisher, Relay, Process, migrations), `natsbus`, `testsupport`.
- **Tests** : `make test-integration` (ou la CI) lance les tests avec PostgreSQL 17 et NATS réels via testcontainers-go (Docker requis) et `REQUIRE_INTEGRATION=1` : un test d'intégration ignoré fait échouer. En local sans Docker : `TEST_DATABASE_URL=postgres://…/postgres TEST_NATS_URL=nats://…` pointe vers des serveurs déjà lancés. Sans rien, seuls les cas de l'adaptateur mémoire tournent.
