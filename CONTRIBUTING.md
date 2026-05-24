# Contributing

Thanks for helping improve `go-rpcatlas`.

## Development

```bash
go test ./...
go build -o /tmp/rpcatlas ./cmd/rpcatlas
/tmp/rpcatlas ./examples/grpc-basic --rpc GetFoo --depth 4
```

Before opening a pull request, run:

```bash
gofmt -w $(find . -name '*.go')
go test ./...
```

## Test Fixtures

Prefer neutral example names such as `GetFoo`, `CreateFoo`, `FooRepository`,
and `FooClient`. Fixtures should avoid domain-specific business terms so they
remain easy for contributors to reason about.

When adding a detector, include a fixture that demonstrates the smallest
representative Go pattern the detector is expected to support.

## Pull Requests

Please keep pull requests focused. A good PR usually contains:

- A short explanation of the analysis behavior being changed.
- Tests or fixtures for the new pattern.
- README or example updates when the user-facing output changes.

The project favors deterministic output and lightweight static analysis over
large framework-specific integrations.
