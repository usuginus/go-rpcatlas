# go-rpcatlas

[![CI](https://github.com/usuginus/go-rpcatlas/actions/workflows/ci.yaml/badge.svg)](https://github.com/usuginus/go-rpcatlas/actions/workflows/ci.yaml)

`go-rpcatlas` generates a compact static RPC map for Go RPC/API handlers.

It is built for code review, onboarding, and AI-assisted code reading: point it at a
handler, then get a deterministic Markdown or JSON summary of the relevant calls,
layers, branches, dispatches, and interface/function-value edges.



https://github.com/user-attachments/assets/fb778bb9-be8a-4b7b-8bb1-4cdcd8bd7da3



## Features

- Find gRPC-style handlers and list available RPCs.
- Render a readable Markdown call tree for one handler.
- Group detected functions by configurable layers such as `repository`,
  `external_client`, or any project-specific name.
- Surface decision points: interface calls, function-value calls, branches, and
  keyed dispatches.
- Use AST-based static analysis only. It does not run the target service.
- Configure noise filtering and layer rules with `.rpcatlas.yaml`.

## Install

### Go install

```bash
go install github.com/usuginus/go-rpcatlas/cmd/rpcatlas@latest
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/usuginus/go-rpcatlas/main/install.sh | sh
```

Pin a release or install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/usuginus/go-rpcatlas/main/install.sh \
  | VERSION=v0.1.0 INSTALL_DIR=/usr/local/bin sh
```

## Quick Start

List handlers:

```bash
rpcatlas ./... --list
```

Generate a Markdown summary:

```bash
rpcatlas ./... --rpc GetFoo --depth 5 --format markdown
```

Generate JSON for automation:

```bash
rpcatlas ./... --rpc GetFoo --depth 5 --format json
```

Use a config file:

```bash
rpcatlas ./... --config .rpcatlas.yaml --rpc GetFoo --depth 5
```

## GitHub Action

This repository also ships a composite action. It installs the CLI from the same
ref as the action and runs it in your workflow.

```yaml
name: rpcatlas

on:
  pull_request:

jobs:
  rpcatlas:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: usuginus/go-rpcatlas@main
        with:
          path: ./...
          rpc: GetFoo
          depth: "5"
          format: markdown
          output-file: rpcatlas.md

      - uses: actions/upload-artifact@v4
        with:
          name: rpcatlas
          path: rpcatlas.md
```

For repeatable CI, pin the action to a release tag once releases are available.

## Configuration

`go-rpcatlas` works without configuration, but `.rpcatlas.yaml` makes the output
much more useful for real projects.

```yaml
# Optional package presets.
presets:
  - generic

# Calls that should be hidden from the call tree.
ignore_calls:
  packages:
    - fmt
    - log
  names:
    - Error

# Output sections. The name is emitted as-is.
layers:
  - name: repository
    match:
      call_name_contains:
        - repository
      receiver_type_contains:
        - repository
      file_path_contains:
        - /repository/

  - name: external_client
    match:
      call_name_contains:
        - client
      receiver_type_contains:
        - client
      file_path_contains:
        - /client/
        - /gateway/
```

See [rpcatlas.example.yaml](rpcatlas.example.yaml) for a fuller starting
point.

## Output

Markdown output is designed to be pasted into a pull request, issue, or AI
review prompt. It highlights the entry point, the important downstream calls, and
the places where static resolution matters.

```markdown
## GetFoo

### execution summary

- kind: `grpc`
- handler: `Service.GetFoo` (internal/handler/foo.go:24)
- request: `*foov1.GetFooRequest`
- response: `*foov1.GetFooResponse`
- layers:
  - application: 1 call
  - repository: 1 call
  - external_client: 1 call
- call resolution:
  - interface calls: 1
  - function values: 1
- control flow:
  - conditional paths: 1
  - keyed dispatches: 1

### call tree

- [handler] `Service.GetFoo` (internal/handler/foo.go:24)
  - [application] `s.fooService.GetFoo` (internal/handler/foo.go:31)
    - [repository] `fooRepo.Find` (internal/repository/foo.go:18)
    - [external_client] `profileClient.Get` (internal/client/profile.go:42)
  - [application] `workflow.Run` (internal/workflow/foo.go:15)
    - [function_value] `validateFoo` (internal/workflow/foo.go:28)

### function index

#### application

| function | location | occurrences |
| --- | --- | ---: |
| `s.fooService.GetFoo` | `internal/handler/foo.go:31` | 1 |
| `workflow.Run` | `internal/workflow/foo.go:15` | 1 |

#### repository

| function | location | occurrences |
| --- | --- | ---: |
| `fooRepo.Find` | `internal/repository/foo.go:18` | 1 |

#### external_client

| function | location | occurrences |
| --- | --- | ---: |
| `profileClient.Get` | `internal/client/profile.go:42` | 1 |

### call resolution

#### interface calls

| call | interface | candidates | resolution |
| --- | --- | --- | --- |
| `s.fooService.GetFoo` (internal/handler/foo.go:31) | `FooService` | `fooService.GetFoo` (internal/service/foo.go:12) expanded | single expanded |

#### function values

| wrapper | function value | resolved function | resolution |
| --- | --- | --- | --- |
| `workflow.Run` (internal/workflow/foo.go:15) | `validateFoo` | `validateFoo` (internal/workflow/foo.go:28) expanded | direct function argument |

### control flow

#### conditional paths

| function | condition | path | calls |
| --- | --- | --- |
| `fooService.GetFoo` | `req.IncludeProfile` | if | `profileClient.Get` |

#### keyed dispatches

| lookup | case | calls |
| --- | --- | --- |
| `processors[req.Kind]` | `FooKindStandard` | `standardProcessor.Process` |
```

JSON output contains the same analysis data in a machine-readable shape.

## Public Sample

For a public, non-domain-specific target, try
[evrone/go-clean-template](https://github.com/evrone/go-clean-template):

```bash
git clone https://github.com/evrone/go-clean-template.git /tmp/go-clean-template
rpcatlas /tmp/go-clean-template --list
```

## Limits

`go-rpcatlas` is intentionally lightweight. It relies on Go AST and type
information plus project-configurable heuristics; it is not a full SSA or runtime
tracer.

That means it can produce a concise, deterministic summary quickly, but it may
miss calls that depend on complex runtime wiring, reflection, generated dynamic
registries, or build tags that are not active in the current environment.

## Development

```bash
go test ./...
go build -o /tmp/rpcatlas ./cmd/rpcatlas
/tmp/rpcatlas ./examples/grpc-basic --rpc GetFoo --depth 4
```

The CI workflow also checks formatting, unit tests, and coverage summary output.
