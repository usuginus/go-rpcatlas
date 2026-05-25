# go-rpcatlas

[![CI](https://github.com/usuginus/go-rpcatlas/actions/workflows/ci.yaml/badge.svg)](https://github.com/usuginus/go-rpcatlas/actions/workflows/ci.yaml)

AI-ready context for explaining Go RPC flows.

`go-rpcatlas` generates static RPC/API flow summaries that help AI agents and
reviewers understand Go code faster. It reads source code, finds handlers, and
produces deterministic Markdown or JSON for code explanation, review, onboarding,
and CI.

It does not boot the service, send traffic, or require runtime tracing.

https://github.com/user-attachments/assets/fb778bb9-be8a-4b7b-8bb1-4cdcd8bd7da3

## Quick Example

### 1. List RPC entry points

```bash
rpcatlas ./... --list
```

```markdown
| rpc | handler | location |
| --- | --- | --- |
| `GetFoo` | `Server.GetFoo` | `examples/grpc-basic/handler.go:19` |
```

Use this first when you do not know the exact handler name.

### 2. Inspect one RPC

```bash
rpcatlas ./... --rpc GetFoo --depth 5
```

The Markdown report starts with the entry point, then shows the call tree and
the places where static resolution mattered.

```markdown
## GetFoo

### execution summary

- kind: `grpc`
- handler: `Server.GetFoo` (examples/grpc-basic/handler.go:19)
- request: `*GetFooRequest`
- response: `*FooResponse`
- layers:
  - usecase: 1 call
  - repository: 1 call
  - converter: 1 call
- call resolution:
  - interface calls: 1
  - function values: 0
- control flow:
  - conditional paths: 0
  - keyed dispatches: 0

### call tree

- [handler] `Server.GetFoo` (examples/grpc-basic/handler.go:19)
  - [usecase] `s.fooUsecase.GetFoo` (examples/grpc-basic/handler.go:20)
    - [usecase] `fooUsecase.GetFoo` (examples/grpc-basic/usecase.go:19)
      - [repository] `u.repositories.Foos.FindFoo` (examples/grpc-basic/usecase.go:20)
        - [repository] `fooRepository.FindFoo` (examples/grpc-basic/repository.go:12)
  - [converter] `s.fooConverter.ToResponse` (examples/grpc-basic/handler.go:24)
    - [converter] `fooConverter.ToResponse` (examples/grpc-basic/converter.go:5)

### call resolution

#### interface calls

| call | interface | candidates | resolution |
| --- | --- | --- | --- |
| `s.fooUsecase.GetFoo` (examples/grpc-basic/handler.go:20) | `FooUsecase` | `fooUsecase.GetFoo` (examples/grpc-basic/usecase.go:19) expanded | single expanded |
```

### 3. Use the output

- Give it to an AI agent before asking it to explain, review, or modify a request flow.
- Paste Markdown into a pull request, issue, or review note.
- Upload it as a CI artifact.
- Use JSON when you want to build automation around the same analysis data.

## Why rpcatlas?

| Problem | How rpcatlas helps |
| --- | --- |
| "Where does this RPC go?" | Shows a handler-centered call tree with source locations. |
| "What should I read first?" | Groups calls by layer, such as `usecase`, `repository`, and `external_client`. |
| "Can AI explain this code?" | Produces deterministic context before the agent reads every file. |
| "Can AI review this flow?" | Fits naturally into review prompts and PR comments. |
| "Can I run it in CI?" | Runs from source without booting the service or attaching a tracer. |
| "Is it safe to add to CI?" | Has no third-party module dependencies and keeps the supply-chain surface minimal. |
| "Where is static resolution important?" | Surfaces interface calls, function values, branches, and keyed dispatches. |

## Lightweight by Design

`go-rpcatlas` is built with Go's standard AST and type-analysis packages. The
small dependency surface keeps installs simple and audits straightforward.

## Install

```bash
go install github.com/usuginus/go-rpcatlas/cmd/rpcatlas@latest
```

If `rpcatlas` is installed but your shell cannot find it, make sure your Go
binary directory is on `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

You can also use the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/usuginus/go-rpcatlas/main/install.sh | sh
```

Pin a release or install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/usuginus/go-rpcatlas/main/install.sh \
  | VERSION=v1.0.0 INSTALL_DIR=/usr/local/bin sh
```

## GitHub Action

This repository ships a composite action. It installs the CLI from the same ref
as the action and runs it in your workflow.

```yaml
name: rpcatlas

on:
  pull_request:

jobs:
  rpcatlas:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: usuginus/go-rpcatlas@v1.0.0
        with:
          path: ./...
          config: .rpcatlas.yaml
          rpc: GetFoo
          depth: "5"
          format: markdown
          output-file: rpcatlas.md

      - uses: actions/upload-artifact@v4
        with:
          name: rpcatlas
          path: rpcatlas.md
```

## AI Agent Skill

This repository includes an optional agent skill at
[skills/rpcatlas/SKILL.md](skills/rpcatlas/SKILL.md).

Use it when an AI agent needs to explain a Go RPC/API handler, review a request
flow, or prepare compact source context before reading a large codebase.

## Configuration

`go-rpcatlas` works without configuration, but `.rpcatlas.yaml` makes the output
more useful for real projects.

For most repositories, put `.rpcatlas.yaml` at the repository root. Local CLI
runs automatically discover it by walking up from the target path:

```bash
rpcatlas ./... --rpc GetFoo --depth 5
```

In CI, pass `config: .rpcatlas.yaml` explicitly, as shown in the GitHub Action
example above. For monorepos, point `path` and `config` at the same service.

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
  - name: application
    match:
      call_name_contains:
        - usecase
        - service
      receiver_type_contains:
        - usecase
        - service
      file_path_contains:
        - /usecase/
        - /service/

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

See [rpcatlas.example.yaml](rpcatlas.example.yaml) for a fuller starting point.

## Output Modes

- `--format markdown`: optimized for pull requests, issues, docs, and AI prompts.
- `--format json`: the same analysis data in a machine-readable shape.

## Public Sample

For a public, non-domain-specific target, try
[evrone/go-clean-template](https://github.com/evrone/go-clean-template):

```bash
git clone https://github.com/evrone/go-clean-template.git /tmp/go-clean-template
rpcatlas /tmp/go-clean-template --list
```

## Limits

`go-rpcatlas` is intentionally lightweight. It relies on Go AST and type
information plus project-configurable heuristics; it is not a full SSA analyzer
or runtime tracer.

That means it can produce a concise, deterministic summary quickly, but it may
miss calls that depend on complex runtime wiring, reflection, generated dynamic
registries, or build tags that are not active in the current environment.

Always verify critical behavior by reading the source code.

## Development

```bash
go test ./...
go build -o /tmp/rpcatlas ./cmd/rpcatlas
/tmp/rpcatlas ./examples/grpc-basic --rpc GetFoo --depth 4
```

The CI workflow checks formatting, unit tests, and coverage summary output.
