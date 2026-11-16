# Image d'un service Go (gateway, identity…). Contexte de build : le dossier backend/.
#
#   make generate                       # le code dérivé du contrat doit exister (il n'est pas commité)
#   docker build -f deploy/Dockerfile --build-arg SERVICE=identity -t filterfinder/identity .
#
# NON TESTÉ : Docker n'est pas installé sur la machine de développement (la compilation, elle, l'est).
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.work ./
COPY pkg/ pkg/
COPY services/ services/
ARG SERVICE
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./services/${SERVICE}/cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
