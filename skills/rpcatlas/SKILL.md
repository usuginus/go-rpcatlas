---
name: rpcatlas
description: Use when analyzing Go RPC/API handlers, explaining request flows, preparing code review context, or generating AI-readable RPC flow summaries with the rpcatlas CLI.
---

# rpcatlas

Use `rpcatlas` before manually reading every file in a Go RPC/API codebase.

## Workflow

1. Check whether `rpcatlas` is available:
   `rpcatlas --help`
2. If it is not installed, suggest:
   `go install github.com/usuginus/go-rpcatlas/cmd/rpcatlas@latest`
3. List RPC/API entry points:
   `rpcatlas ./... --list`
4. Generate a Markdown flow summary:
   `rpcatlas ./... --rpc <RPCName> --depth 5 --format markdown`
5. Use the summary as the first-pass map.
6. Verify important behavior by reading the source code.

## Use Cases

- Explain what a Go RPC/API handler does.
- Review a PR that changes a request flow.
- Find downstream usecases, repositories, external clients, branches, dispatches, interface calls, and function-value calls.
- Prepare compact context for AI-assisted code review.

## Output Guidance

- Use Markdown for human review, PR comments, and AI prompts.
- Use JSON for automation.
- Prefer `--depth 5` for a useful first pass.
- Use `.rpcatlas.yaml` when project-specific layer names or noise filters are needed.

## Safety

`rpcatlas` is a static analysis helper. Do not treat its output as complete proof of runtime behavior.

Always verify critical paths, dynamic dispatch, configuration-dependent behavior, and side effects by reading the source code.
