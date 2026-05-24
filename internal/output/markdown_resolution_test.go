package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usuginus/go-rpcatlas/internal/model"
)

func TestWriteMarkdownShowsFunctionValuesAsCallResolution(t *testing.T) {
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
							{Symbol: "sampleUsecase.Run", Receiver: "sampleUsecase", Method: "Run", File: "usecase.go", Line: 20, Depth: 2, Via: "Server.CreateFoo"},
						},
					},
				},
				FunctionValues: []model.FunctionValueTrace{
					{
						Wrapper:  model.CallRef{Symbol: "InvokeWithToken", Method: "InvokeWithToken", File: "handler.go", Line: 12, Depth: 1, Via: "Server.CreateFoo"},
						Function: model.CallRef{Symbol: "s.processor.Run", Receiver: "s.processor", Method: "Run", File: "handler.go", Line: 12, Depth: 1, Via: "InvokeWithToken"},
						Implementations: []model.ImplementationCandidate{
							{Call: model.CallRef{Symbol: "sampleUsecase.Run", Receiver: "sampleUsecase", Method: "Run", File: "usecase.go", Line: 20, Depth: 2, Via: "s.processor.Run"}, Expanded: true},
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
		"- [handler] `Server.CreateFoo` (handler.go:10)\n  - [usecase] `sampleUsecase.Run` (usecase.go:20)",
		"### call resolution\n\n#### function values\n\n| wrapper | function value | resolved function | resolution |",
		"| `InvokeWithToken` (handler.go:12) | `s.processor.Run` (handler.go:12) | `sampleUsecase.Run` (usecase.go:20) expanded | single expanded |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "- [usecase] `s.processor.Run`") {
		t.Fatalf("markdown call tree includes function value callsite as a tree node:\n%s", got)
	}
}

func TestWriteMarkdownIncludesInterfaceCallSummary(t *testing.T) {
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
				InterfaceCalls: []model.InterfaceCallTrace{
					{
						Call:      model.CallRef{Symbol: "s.fooUsecase.GetFoo", File: "handler.go", Line: 12},
						Interface: "FooUsecase",
						Implementations: []model.ImplementationCandidate{
							{
								Call:     model.CallRef{Symbol: "fooUsecase.GetFoo", File: "usecase.go", Line: 20},
								Expanded: true,
							},
							{
								Call: model.CallRef{Symbol: "otherFooUsecase.GetFoo", File: "other_usecase.go", Line: 18},
							},
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
		"### call resolution\n\n#### interface calls\n\n| call | interface | candidates | resolution |",
		"| `s.fooUsecase.GetFoo` (handler.go:12) | `FooUsecase` | `fooUsecase.GetFoo` (usecase.go:20) expanded<br>`otherFooUsecase.GetFoo` (other_usecase.go:18) candidate | partial |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
}

func TestWriteMarkdownSplitsUnresolvedInterfaceCalls(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "GetFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Server.GetFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*pb.GetFooRequest"},
			Response:   model.TypeRef{Type: "*pb.GetFooResponse"},
			Trail: model.Trail{
				InterfaceCalls: []model.InterfaceCallTrace{
					{
						Call:      model.CallRef{Symbol: "s.fooUsecase.GetFoo", File: "handler.go", Line: 12},
						Interface: "FooUsecase",
						Implementations: []model.ImplementationCandidate{
							{Call: model.CallRef{Symbol: "fooUsecase.GetFoo", File: "usecase.go", Line: 20}, Expanded: true},
						},
					},
					{
						Call:      model.CallRef{Symbol: "s.alpha.List", File: "usecase.go", Line: 24},
						Interface: "AlphaClient",
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
		"| `s.fooUsecase.GetFoo` (handler.go:12) | `FooUsecase` | `fooUsecase.GetFoo` (usecase.go:20) expanded | single expanded |",
		"| `s.alpha.List` (usecase.go:24) | `AlphaClient` | - | unresolved |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
}

func TestWriteMarkdownOmitsInterfaceCallsWithOnlyInternalHelperImplementations(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "GetFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Server.GetFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*pb.GetFooRequest"},
			Response:   model.TypeRef{Type: "*pb.GetFooResponse"},
			Trail: model.Trail{
				InterfaceCalls: []model.InterfaceCallTrace{
					{
						Call:      model.CallRef{Symbol: "u.helper.normalize", Method: "normalize", File: "usecase.go", Line: 12},
						Interface: "FooNormalizer",
						Implementations: []model.ImplementationCandidate{
							{Call: model.CallRef{Symbol: "fooNormalizer.normalize", Method: "normalize", File: "normalizer.go", Line: 8, Depth: 2, Via: "u.helper.normalize"}, Expanded: true},
						},
					},
					{
						Call:      model.CallRef{Symbol: "s.fooUsecase.GetFoo", Method: "GetFoo", File: "handler.go", Line: 14},
						Interface: "FooUsecase",
						Implementations: []model.ImplementationCandidate{
							{Call: model.CallRef{Symbol: "fooUsecase.GetFoo", Method: "GetFoo", File: "usecase.go", Line: 20, Depth: 2, Via: "s.fooUsecase.GetFoo"}, Expanded: true},
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
	if strings.Contains(got, "u.helper.normalize") || strings.Contains(got, "fooNormalizer.normalize") {
		t.Fatalf("markdown output includes internal helper interface call:\n%s", got)
	}
	if !strings.Contains(got, "s.fooUsecase.GetFoo") {
		t.Fatalf("markdown output omitted useful interface call:\n%s", got)
	}
}
