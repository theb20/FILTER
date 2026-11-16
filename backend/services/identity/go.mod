module github.com/theb20/FILTER/backend/services/identity

go 1.26.0

require (
	github.com/google/uuid v1.6.0
	github.com/theb20/FILTER/backend/pkg v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.59.0
	google.golang.org/grpc v1.85.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261110000943-09e21b237f2c // indirect
)

// Module partagé du dépôt (le workspace go.work le résout aussi ; `replace` sert aux outils hors workspace).
replace github.com/theb20/FILTER/backend/pkg => ../../pkg
