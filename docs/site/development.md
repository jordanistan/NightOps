# Development

The repository development guide contains deeper package rules, provider
contracts, data provenance, testing expectations, and local verification
details.

`app.theme` supports `mission-control` (the default) and `observatory`. Theme
selection is applied at startup and shown in Settings; unsupported values are
rejected during configuration validation.

The minimum contribution gate is:

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
go build -o /tmp/nightops-verify ./cmd/nightops
go build -o /tmp/nightopsctl-verify ./cmd/nightopsctl
```

Keep changes small, document major features, and never add fake provider data
or secrets to fixtures.
