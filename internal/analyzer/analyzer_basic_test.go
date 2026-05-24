package analyzer

import (
	"testing"

	"github.com/usuginus/go-rpcatlas/internal/rules"
)

func TestAnalyzeDetectsGRPCHandlerTrail(t *testing.T) {
	flows, err := Analyze([]string{"testdata/basic_flow"}, Options{})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if flow.Name != "GetFoo" {
		t.Fatalf("flow.Name = %q, want GetFoo", flow.Name)
	}
	if flow.Entrypoint.File != "internal/analyzer/testdata/basic_flow/handler.go" {
		t.Fatalf("entrypoint file = %q", flow.Entrypoint.File)
	}
	if flow.Request.Type != "*pb.GetFooRequest" {
		t.Fatalf("request type = %q", flow.Request.Type)
	}
	usecases := flow.Trail.LayerCalls("usecase")
	if len(usecases) != 1 {
		t.Fatalf("usecases = %d, want 1", len(usecases))
	}
	if usecases[0].Symbol != "s.fooUsecase.GetFoo" {
		t.Fatalf("usecase symbol = %q", usecases[0].Symbol)
	}
	if len(flow.Errors.GRPCCodes) != 1 || flow.Errors.GRPCCodes[0] != "Internal" {
		t.Fatalf("grpc codes = %#v, want [Internal]", flow.Errors.GRPCCodes)
	}
}

func TestAnalyzeDepthTwoFollowsInterfaceImplementationCandidate(t *testing.T) {
	flows, err := Analyze([]string{"testdata/basic_flow"}, Options{Depth: 2})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	usecases := flow.Trail.LayerCalls("usecase")
	if len(usecases) != 2 {
		t.Fatalf("usecases = %d, want 2", len(usecases))
	}
	if usecases[1].Symbol != "fooUsecase.GetFoo" {
		t.Fatalf("depth-2 usecase symbol = %q", usecases[1].Symbol)
	}
	if usecases[1].Depth != 2 {
		t.Fatalf("depth-2 usecase depth = %d", usecases[1].Depth)
	}
	repositories := flow.Trail.LayerCalls("repository")
	if len(repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repositories))
	}
	if repositories[0].Symbol != "f.repos.Foo.FindFoo" {
		t.Fatalf("repository symbol = %q", repositories[0].Symbol)
	}
	if repositories[0].Via != "fooUsecase.GetFoo" {
		t.Fatalf("repository via = %q", repositories[0].Via)
	}
	if len(flow.Trail.InterfaceCalls) != 1 || len(flow.Trail.InterfaceCalls[0].Implementations) != 1 {
		t.Fatalf("interface calls = %#v, want one implementation", flow.Trail.InterfaceCalls)
	}
	if !flow.Trail.InterfaceCalls[0].Implementations[0].Expanded {
		t.Fatal("interface implementation expanded = false, want true")
	}
	if hasCall(flow.Trail.Unknown, "stdstrings.TrimSpace") {
		t.Fatal("standard library alias call was not ignored")
	}
}

func TestAnalyzeRecordsInterfaceCallCandidates(t *testing.T) {
	flows, err := Analyze([]string{"testdata/basic_flow"}, Options{Depth: 1})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if len(flow.Trail.InterfaceCalls) != 1 {
		t.Fatalf("interface calls = %#v, want 1", flow.Trail.InterfaceCalls)
	}
	trace := flow.Trail.InterfaceCalls[0]
	if trace.Call.Symbol != "s.fooUsecase.GetFoo" {
		t.Fatalf("interface call symbol = %q, want s.fooUsecase.GetFoo", trace.Call.Symbol)
	}
	if trace.Interface != "FooUsecase" {
		t.Fatalf("interface = %q, want FooUsecase", trace.Interface)
	}
	if len(trace.Implementations) != 1 {
		t.Fatalf("implementations = %#v, want 1", trace.Implementations)
	}
	implementation := trace.Implementations[0]
	if implementation.Call.Symbol != "fooUsecase.GetFoo" {
		t.Fatalf("implementation = %q, want fooUsecase.GetFoo", implementation.Call.Symbol)
	}
	if implementation.Expanded {
		t.Fatal("implementation expanded at depth 1, want false")
	}
}

func TestAnalyzeMissingRPCReturnsNoFlows(t *testing.T) {
	flows, err := Analyze([]string{"testdata/basic_flow"}, Options{RPC: "MissingRPC"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 0 {
		t.Fatalf("len(flows) = %d, want 0", len(flows))
	}
}

func TestAnalyzeDepthThreeFollowsNestedStructFieldCandidate(t *testing.T) {
	flows, err := Analyze([]string{"testdata/basic_flow"}, Options{Depth: 3})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	repositories := flow.Trail.LayerCalls("repository")
	if len(repositories) != 2 {
		t.Fatalf("repositories = %d, want 2", len(repositories))
	}
	if repositories[1].Symbol != "fooRepository.FindFoo" {
		t.Fatalf("depth-3 repository symbol = %q", repositories[1].Symbol)
	}
	if repositories[1].Depth != 3 {
		t.Fatalf("depth-3 repository depth = %d", repositories[1].Depth)
	}
	if repositories[1].Via != "f.repos.Foo.FindFoo" {
		t.Fatalf("depth-3 repository via = %q", repositories[1].Via)
	}
}

func TestAnalyzeUsesConfiguredLayerNameInTrail(t *testing.T) {
	ruleSet, err := rules.Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	for i := range ruleSet.Layers {
		if ruleSet.Layers[i].Name == "usecase" {
			ruleSet.Layers[i].Name = "application"
		}
	}

	flows, err := Analyze([]string{"testdata/basic_flow"}, Options{Rules: ruleSet})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if len(flow.Trail.LayerCalls("usecase")) != 0 {
		t.Fatalf("usecase layer = %#v, want empty", flow.Trail.LayerCalls("usecase"))
	}
	if got := flow.Trail.LayerCalls("application"); len(got) != 1 || got[0].Symbol != "s.fooUsecase.GetFoo" {
		t.Fatalf("application layer = %#v, want s.fooUsecase.GetFoo", got)
	}
}
