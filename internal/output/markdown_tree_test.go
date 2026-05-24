package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usuginus/go-rpcatlas/internal/model"
)

func TestWriteMarkdownUsesConfiguredLayerNames(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name: "GetFoo",
			Kind: "grpc",
			Entrypoint: model.Entrypoint{
				Symbol: "Server.GetFoo",
				File:   "handler.go",
				Line:   10,
			},
			Request:  model.TypeRef{Type: "*pb.GetFooRequest"},
			Response: model.TypeRef{Type: "*pb.GetFooResponse"},
			Trail: model.Trail{
				Layers: []model.LayerCalls{
					{
						Name: "domain",
						Calls: []model.CallRef{
							{Symbol: "entity.Foo.Validate", Receiver: "entity.Foo", Method: "Validate", File: "entity.go", Line: 12, Depth: 1},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "### call tree\n\n- [handler] `Server.GetFoo` (handler.go:10)\n  - [domain] `entity.Foo.Validate` (entity.go:12)") {
		t.Fatalf("markdown output does not include configured layer:\n%s", got)
	}
	if !strings.Contains(got, "### function index\n\n#### domain\n\n| function | location | occurrences |") {
		t.Fatalf("markdown output does not include function index:\n%s", got)
	}
}

func TestWriteMarkdownSummarizesRepositoryOperations(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name: "CreateFoo",
			Kind: "grpc",
			Entrypoint: model.Entrypoint{
				Symbol: "Server.CreateFoo",
				File:   "handler.go",
				Line:   10,
			},
			Request:  model.TypeRef{Type: "*pb.CreateFooRequest"},
			Response: model.TypeRef{Type: "*pb.Foo"},
			Trail: model.Trail{
				Layers: []model.LayerCalls{
					{
						Name: "usecase",
						Calls: []model.CallRef{
							{Symbol: "s.fooUsecase.CreateFoo", Receiver: "s.fooUsecase", Method: "CreateFoo", File: "handler.go", Line: 12, Depth: 1},
							{Symbol: "fooUsecase.CreateFoo", Receiver: "fooUsecase", Method: "CreateFoo", File: "usecase.go", Line: 20, Depth: 2, Via: "s.fooUsecase.CreateFoo"},
						},
					},
					{
						Name: "repository",
						Calls: []model.CallRef{
							{Symbol: "u.repos.Foo.FindFoo", Receiver: "u.repos.Foo", Method: "FindFoo", File: "usecase.go", Line: 23, Depth: 2, Via: "s.fooUsecase.CreateFoo"},
							{Symbol: "FooRepository.FindFoo", Receiver: "FooRepository", Method: "FindFoo", File: "repository.go", Line: 30, Depth: 3, Via: "u.repos.Foo.FindFoo"},
							{Symbol: "repo.columns", Receiver: "repo", Method: "columns", File: "repository.go", Line: 31, Depth: 3, Via: "u.repos.Foo.FindFoo"},
							{Symbol: "u.repos.Foo.FindFoo", Receiver: "u.repos.Foo", Method: "FindFoo", File: "usecase.go", Line: 40, Depth: 2, Via: "s.fooUsecase.CreateFoo"},
							{Symbol: "FooRepository.FindFoo", Receiver: "FooRepository", Method: "FindFoo", File: "repository.go", Line: 30, Depth: 3, Via: "u.repos.Foo.FindFoo"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"- [usecase] `s.fooUsecase.CreateFoo` (handler.go:12)\n    - [usecase] `fooUsecase.CreateFoo` (usecase.go:20)",
		"- [repository] `u.repos.Foo.FindFoo` (usecase.go:23)\n      - [repository] `FooRepository.FindFoo` (repository.go:30)",
		"| `FooRepository.FindFoo` | `repository.go:30` | 1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "repo.columns") {
		t.Fatalf("markdown output includes internal repository helper:\n%s", got)
	}
}

func TestWriteMarkdownDisambiguatesSameSymbolHelpersByFile(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name: "GetFoo",
			Kind: "grpc",
			Entrypoint: model.Entrypoint{
				Symbol: "Server.GetFoo",
				File:   "handler.go",
				Line:   10,
			},
			Request:  model.TypeRef{Type: "*pb.GetFooRequest"},
			Response: model.TypeRef{Type: "*pb.GetFooResponse"},
			Trail: model.Trail{
				Layers: []model.LayerCalls{
					{
						Name: "repository",
						Calls: []model.CallRef{
							{Symbol: "s.fooRepo.Find", Receiver: "s.fooRepo", Method: "Find", File: "handler.go", Line: 12, Depth: 1},
							{Symbol: "repo.decode", Receiver: "repo", Method: "decode", File: "foo_repository.go", Line: 11, Depth: 2, Via: "s.fooRepo.Find"},
							{Symbol: "FooRepository.Decode", Receiver: "FooRepository", Method: "Decode", File: "foo_repository.go", Line: 20, Depth: 3, Via: "repo.decode"},
							{Symbol: "s.barRepo.Find", Receiver: "s.barRepo", Method: "Find", File: "handler.go", Line: 22, Depth: 1},
							{Symbol: "repo.decode", Receiver: "repo", Method: "decode", File: "bar_repository.go", Line: 11, Depth: 2, Via: "s.barRepo.Find"},
							{Symbol: "BarRepository.Decode", Receiver: "BarRepository", Method: "Decode", File: "bar_repository.go", Line: 20, Depth: 3, Via: "repo.decode"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"`repo.decode` (foo_repository.go:11)\n      - [repository] `FooRepository.Decode` (foo_repository.go:20)",
		"`repo.decode` (bar_repository.go:11)\n      - [repository] `BarRepository.Decode` (bar_repository.go:20)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{
		"`repo.decode` (foo_repository.go:11)\n      - [repository] `BarRepository.Decode`",
		"`repo.decode` (bar_repository.go:11)\n      - [repository] `FooRepository.Decode`",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("markdown output attaches helper child to wrong parent:\n%s", got)
		}
	}
}

func TestWriteMarkdownDoesNotTreatViaOnlyCallAsImplementation(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name: "CreateFoo",
			Kind: "grpc",
			Entrypoint: model.Entrypoint{
				Symbol: "Server.CreateFoo",
				File:   "handler.go",
				Line:   10,
			},
			Request:  model.TypeRef{Type: "*pb.CreateFooRequest"},
			Response: model.TypeRef{Type: "*pb.Foo"},
			Trail: model.Trail{
				Layers: []model.LayerCalls{
					{
						Name: "usecase",
						Calls: []model.CallRef{
							{Symbol: "fooUsecase.CreateFoo", Receiver: "fooUsecase", Method: "CreateFoo", File: "usecase.go", Line: 18, Depth: 2, Via: "s.fooUsecase.CreateFoo"},
						},
					},
					{
						Name: "repository",
						Calls: []model.CallRef{
							{Symbol: "repository.IsNotFoundError", Receiver: "repository", Method: "IsNotFoundError", File: "usecase.go", Line: 40, Depth: 2, Via: "fooUsecase.CreateFoo"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"| `fooUsecase.CreateFoo` | `usecase.go:18` | 1 |",
		"| `repository.IsNotFoundError` | `usecase.go:40` | 1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "implementation: usecase.go:40") {
		t.Fatalf("markdown output treats callsite as implementation:\n%s", got)
	}
}

func TestWriteMarkdownUsesConfiguredLayerForExternalCalls(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "GetFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Service.GetFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*GetFooRequest"},
			Response:   model.TypeRef{Type: "*Foo"},
			Trail: model.Trail{
				Layers: []model.LayerCalls{
					{
						Name: "external_client",
						Calls: []model.CallRef{
							{Symbol: "s.clients.Alpha.GetFoo", Receiver: "s.clients.Alpha", Method: "GetFoo", File: "usecase.go", Line: 20, Depth: 2},
							{Symbol: "s.clients.Beta.GetFoo", Receiver: "s.clients.Beta", Method: "GetFoo", File: "usecase.go", Line: 22, Depth: 2},
							{Symbol: "gammaClient.GetFoo", Receiver: "gammaClient", Method: "GetFoo", File: "gamma_client.go", Line: 30, Depth: 3, Via: "clients.Gamma.GetFoo"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"- [external_client] `s.clients.Alpha.GetFoo` (usecase.go:20)",
		"- [external_client] `s.clients.Beta.GetFoo` (usecase.go:22)",
		"| `gammaClient.GetFoo` | `gamma_client.go:30` | 1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
}

func TestWriteMarkdownDoesNotCollapseAmbiguousImplementations(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name: "ProcessFoo",
			Kind: "grpc",
			Entrypoint: model.Entrypoint{
				Symbol: "Service.ProcessFoo",
				File:   "handler.go",
				Line:   10,
			},
			Request:  model.TypeRef{Type: "*ProcessFooRequest"},
			Response: model.TypeRef{Type: "*ProcessFooResponse"},
			Trail: model.Trail{
				Layers: []model.LayerCalls{
					{
						Name: "domain",
						Calls: []model.CallRef{
							{Symbol: "payload.Normalize", Receiver: "payload", Method: "Normalize", File: "application.go", Line: 20, Depth: 2, Via: "fooApplication.ProcessFoo"},
							{Symbol: "AlphaPayload.Normalize", Receiver: "AlphaPayload", Method: "Normalize", File: "application.go", Line: 8, Depth: 3, Via: "payload.Normalize"},
							{Symbol: "payload.Normalize", Receiver: "payload", Method: "Normalize", File: "application.go", Line: 24, Depth: 2, Via: "fooApplication.ProcessFoo"},
							{Symbol: "BetaPayload.Normalize", Receiver: "BetaPayload", Method: "Normalize", File: "application.go", Line: 14, Depth: 3, Via: "payload.Normalize"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"| `payload.Normalize` | `application.go:20` | 1 |",
		"| `payload.Normalize` | `application.go:24` | 1 |",
		"| `AlphaPayload.Normalize` | `application.go:8` | 1 |",
		"| `BetaPayload.Normalize` | `application.go:14` | 1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "implementation: application.go:8") || strings.Contains(got, "related internal call") {
		t.Fatalf("markdown output collapsed ambiguous implementations:\n%s", got)
	}
}
