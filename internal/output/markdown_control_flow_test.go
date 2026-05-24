package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usuginus/calltrail-go/internal/model"
)

func TestWriteMarkdownIncludesBranchSummary(t *testing.T) {
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
				Branches: []model.BranchTrace{
					{
						Kind:     "switch",
						Function: "fooApplication.ProcessFoo",
						Expr:     "cmd.Mode",
						File:     "application.go",
						Line:     24,
						Cases: []model.BranchCase{
							{
								Labels: []string{`"publish"`},
								Layers: []model.LayerCalls{
									{
										Name: "persistence",
										Calls: []model.CallRef{
											{Symbol: "a.store.Publish", Receiver: "a.store", Method: "Publish", File: "application.go", Line: 30, Depth: 2},
											{Symbol: "fooStore.Publish", Receiver: "fooStore", Method: "Publish", File: "persistence.go", Line: 12, Depth: 3, Via: "a.store.Publish"},
										},
									},
									{
										Name: "external_client",
										Calls: []model.CallRef{
											{Symbol: "a.index.Index", Receiver: "a.index", Method: "Index", File: "application.go", Line: 32, Depth: 2, Via: "fooApplication.ProcessFoo"},
											{Symbol: "previewClient.Index", Receiver: "previewClient", Method: "Index", File: "external_client.go", Line: 7, Depth: 3, Via: "a.index.Index"},
										},
									},
								},
							},
							{
								Default: true,
								Layers: []model.LayerCalls{
									{
										Name: "domain",
										Calls: []model.CallRef{
											{Symbol: "a.policy.RejectUnsupportedMode", Receiver: "a.policy", Method: "RejectUnsupportedMode", File: "application.go", Line: 42, Depth: 2, Via: "fooApplication.ProcessFoo"},
											{Symbol: "fooPolicy.RejectUnsupportedMode", Receiver: "fooPolicy", Method: "RejectUnsupportedMode", File: "domain.go", Line: 24, Depth: 3, Via: "a.policy.RejectUnsupportedMode"},
										},
									},
								},
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
		"### control flow\n\n#### conditional paths\n\n| function | condition | path | calls |",
		"| `fooApplication.ProcessFoo` (application.go:24) | switch `cmd.Mode` | case `\"publish\"` | persistence: `fooStore.Publish`<br>external_client: `previewClient.Index` |",
		"| `fooApplication.ProcessFoo` (application.go:24) | switch `cmd.Mode` | default | domain: `fooPolicy.RejectUnsupportedMode` |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
}

func TestWriteMarkdownBranchSummaryKeepsDirectDecisionCalls(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "GetFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Service.GetFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*GetFooRequest"},
			Response:   model.TypeRef{Type: "*Foo"},
			Trail: model.Trail{
				Branches: []model.BranchTrace{
					{
						Kind:     "type_switch",
						Function: "Service.GetFoo",
						Expr:     "req := in.GetRequest().(type)",
						File:     "handler.go",
						Line:     11,
						Cases: []model.BranchCase{
							{
								Labels: []string{"*GetFooRequest_V1"},
								Layers: []model.LayerCalls{
									{
										Name: "usecase",
										Calls: []model.CallRef{
											{Symbol: "s.foo.Get", Receiver: "s.foo", Method: "Get", File: "handler.go", Line: 14, Depth: 1, Via: "Service.GetFoo"},
											{Symbol: "fooUsecase.Get", Receiver: "fooUsecase", Method: "Get", File: "foo.go", Line: 20, Depth: 2, Via: "s.foo.Get"},
										},
									},
									{
										Name: "repository",
										Calls: []model.CallRef{
											{Symbol: "FooRepository.Find", Receiver: "FooRepository", Method: "Find", File: "repository.go", Line: 12, Depth: 3, Via: "fooUsecase.Get"},
										},
									},
								},
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
	want := "| `Service.GetFoo` (handler.go:11) | type switch `req := in.GetRequest().(type)` | case `*GetFooRequest_V1` | usecase: `fooUsecase.Get` |"
	if !strings.Contains(got, want) {
		t.Fatalf("markdown output does not contain direct branch summary %q:\n%s", want, got)
	}
	if !strings.Contains(got, "| `FooRepository.Find` | `repository.go:12` | 1 |") {
		t.Fatalf("markdown function index does not include transitive branch details:\n%s", got)
	}
	if strings.Contains(got, "| `Service.GetFoo` (handler.go:11) | type switch `req := in.GetRequest().(type)` | case `*GetFooRequest_V1` | usecase: `fooUsecase.Get`<br>repository: `FooRepository.Find` |") {
		t.Fatalf("markdown branch summary includes transitive implementation details:\n%s", got)
	}
}

func TestWriteMarkdownDecisionSummaryIncludesUnexportedHelpers(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "ProcessFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Service.ProcessFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*ProcessFooRequest"},
			Response:   model.TypeRef{Type: "*Foo"},
			Trail: model.Trail{
				Branches: []model.BranchTrace{
					{
						Kind:     "if",
						Function: "fooApplication.ProcessFoo",
						Expr:     "cmd.Fast",
						File:     "application.go",
						Line:     30,
						Cases: []model.BranchCase{
							{
								Labels: []string{"then"},
								Layers: []model.LayerCalls{
									{
										Name: "application",
										Calls: []model.CallRef{
											{Symbol: "a.fastPath", Receiver: "a", Method: "fastPath", File: "application.go", Line: 32, Depth: 2, Via: "fooApplication.ProcessFoo"},
											{Symbol: "fooApplication.fastPath", Receiver: "fooApplication", Method: "fastPath", File: "application.go", Line: 70, Depth: 3, Via: "a.fastPath"},
										},
									},
								},
								Unknown: []model.CallRef{
									{Symbol: "trace.Debug", Receiver: "trace", Method: "Debug", File: "application.go", Line: 33, Depth: 4, Via: "fooApplication.ProcessFoo"},
								},
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
	want := "| `fooApplication.ProcessFoo` (application.go:30) | if `cmd.Fast` | then | application: `fooApplication.fastPath`<br>other: `trace.Debug` |"
	if !strings.Contains(got, want) {
		t.Fatalf("markdown output hides unexported branch calls %q:\n%s", want, got)
	}
}

func TestWriteMarkdownIncludesDispatchSummary(t *testing.T) {
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
				Dispatches: []model.DispatchTrace{
					{
						Table:     "a.processors",
						Key:       "cmd.Kind",
						Call:      model.CallRef{Symbol: "processor.Process", Receiver: "processor", Method: "Process", File: "application.go", Line: 44, Depth: 2},
						Interface: "FooProcessor",
						Cases: []model.DispatchCase{
							{
								Labels: []string{"KindAlpha"},
								Layers: []model.LayerCalls{
									{
										Name: "application",
										Calls: []model.CallRef{
											{Symbol: "alphaProcessor.Process", Receiver: "alphaProcessor", Method: "Process", File: "application.go", Line: 56, Depth: 3, Via: "processor.Process"},
										},
									},
									{
										Name: "persistence",
										Calls: []model.CallRef{
											{Symbol: "fooStore.SaveAlpha", Receiver: "fooStore", Method: "SaveAlpha", File: "persistence.go", Line: 7, Depth: 4, Via: "alphaProcessor.Process"},
										},
									},
								},
							},
							{
								Labels: []string{"KindBeta"},
								Layers: []model.LayerCalls{
									{
										Name: "application",
										Calls: []model.CallRef{
											{Symbol: "betaProcessor.Process", Receiver: "betaProcessor", Method: "Process", File: "application.go", Line: 75, Depth: 3, Via: "processor.Process"},
										},
									},
									{
										Name: "external_client",
										Calls: []model.CallRef{
											{Symbol: "previewClient.RenderBeta", Receiver: "previewClient", Method: "RenderBeta", File: "external_client.go", Line: 7, Depth: 4, Via: "betaProcessor.Process"},
										},
									},
								},
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
		"### control flow\n\n#### keyed dispatches\n\n| lookup | case | calls |",
		"| `processor.Process` (application.go:44)<br>from `a.processors[cmd.Kind]`<br>interface: `FooProcessor` | case `KindAlpha` | application: `alphaProcessor.Process`<br>persistence: `fooStore.SaveAlpha` |",
		"| `processor.Process` (application.go:44)<br>from `a.processors[cmd.Kind]`<br>interface: `FooProcessor` | case `KindBeta` | application: `betaProcessor.Process`<br>external_client: `previewClient.RenderBeta` |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
}

func TestWriteMarkdownOrdersAnalysisSectionsAndOmitsErrorCodes(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "ProcessFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Service.ProcessFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*ProcessFooRequest"},
			Response:   model.TypeRef{Type: "*ProcessFooResponse"},
			Trail: model.Trail{
				InterfaceCalls: []model.InterfaceCallTrace{
					{Call: model.CallRef{Symbol: "s.processor.Process", File: "handler.go", Line: 12}, Interface: "Processor"},
				},
				Branches: []model.BranchTrace{
					{Kind: "switch", Function: "Service.ProcessFoo", Expr: "req.Kind", File: "handler.go", Line: 14},
				},
				Dispatches: []model.DispatchTrace{
					{Table: "processors", Key: "req.Kind", Call: model.CallRef{Symbol: "processor.Process", File: "handler.go", Line: 18}},
				},
			},
			Errors: model.ErrorSummary{GRPCCodes: []string{"InvalidArgument"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}

	got := buf.String()
	callResolutionIndex := strings.Index(got, "### call resolution")
	interfaceIndex := strings.Index(got, "#### interface calls")
	controlFlowIndex := strings.Index(got, "### control flow")
	conditionalPathsIndex := strings.Index(got, "#### conditional paths")
	keyedDispatchesIndex := strings.Index(got, "#### keyed dispatches")
	if callResolutionIndex < 0 || interfaceIndex < 0 || controlFlowIndex < 0 || conditionalPathsIndex < 0 || keyedDispatchesIndex < 0 {
		t.Fatalf("markdown output is missing analysis sections:\n%s", got)
	}
	if !(callResolutionIndex < interfaceIndex && interfaceIndex < controlFlowIndex && controlFlowIndex < conditionalPathsIndex && conditionalPathsIndex < keyedDispatchesIndex) {
		t.Fatalf("markdown analysis sections are not in stable order:\n%s", got)
	}
	if strings.Contains(got, "Error Codes") || strings.Contains(got, "InvalidArgument") {
		t.Fatalf("markdown output includes project-specific error summary:\n%s", got)
	}
}

func TestWriteMarkdownIncludesEntrypointTypeSwitchAsBranch(t *testing.T) {
	var buf bytes.Buffer
	err := WriteMarkdown(&buf, []model.APIFlow{
		{
			Name:       "GetFoo",
			Kind:       "grpc",
			Entrypoint: model.Entrypoint{Symbol: "Service.GetFoo", File: "handler.go", Line: 10},
			Request:    model.TypeRef{Type: "*GetFooRequest"},
			Response:   model.TypeRef{Type: "*Foo"},
			Trail: model.Trail{
				Branches: []model.BranchTrace{
					{
						Kind:     "type_switch",
						Function: "Service.GetFoo",
						Expr:     "payload := req.Payload.(type)",
						File:     "handler.go",
						Line:     12,
						Cases: []model.BranchCase{
							{
								Labels: []string{"*GetFooRequest_V1"},
								Layers: []model.LayerCalls{
									{
										Name: "usecase",
										Calls: []model.CallRef{
											{Symbol: "foo.Get", Receiver: "foo", Method: "Get", File: "usecase.go", Line: 20, Depth: 2},
										},
									},
								},
							},
							{
								Default: true,
								Unknown: []model.CallRef{
									{Symbol: "errors.NewInvalidArgumentErr", Receiver: "errors", Method: "NewInvalidArgumentErr", File: "handler.go", Line: 28, Depth: 1},
								},
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
		"### control flow\n\n#### conditional paths\n\n| function | condition | path | calls |",
		"| `Service.GetFoo` (handler.go:12) | type switch `payload := req.Payload.(type)` | case `*GetFooRequest_V1` | usecase: `foo.Get` |",
		"| `Service.GetFoo` (handler.go:12) | type switch `payload := req.Payload.(type)` | default | other: `errors.NewInvalidArgumentErr` |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown output does not contain %q:\n%s", want, got)
		}
	}
}
