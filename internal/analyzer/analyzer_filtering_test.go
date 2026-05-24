package analyzer

import (
	"testing"
)

func TestAnalyzeShortRPCFilterCanMatchMultipleHandlers(t *testing.T) {
	flows, err := Analyze([]string{"testdata/rpc_filtering"}, Options{RPC: "GetFoo"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("len(flows) = %d, want 2", len(flows))
	}
}

func TestAnalyzeAllowsReceiverQualifiedRPCFilter(t *testing.T) {
	flows, err := Analyze([]string{"testdata/rpc_filtering"}, Options{RPC: "userService.GetFoo"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}
	if flows[0].Entrypoint.Symbol != "userService.GetFoo" {
		t.Fatalf("entrypoint symbol = %q, want userService.GetFoo", flows[0].Entrypoint.Symbol)
	}
}

func TestDetectHandlersReturnsHandlerHeadersOnly(t *testing.T) {
	flows, err := DetectHandlers([]string{"testdata/basic_flow"}, Options{})
	if err != nil {
		t.Fatalf("DetectHandlers returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if flow.Name != "GetFoo" {
		t.Fatalf("flow.Name = %q, want GetFoo", flow.Name)
	}
	if flow.Entrypoint.Symbol != "Server.GetFoo" {
		t.Fatalf("entrypoint symbol = %q, want Server.GetFoo", flow.Entrypoint.Symbol)
	}
	if flow.Request.Type != "*pb.GetFooRequest" {
		t.Fatalf("request type = %q, want *pb.GetFooRequest", flow.Request.Type)
	}
	if len(flow.Trail.Layers) != 0 || len(flow.Trail.InterfaceCalls) != 0 || len(flow.Trail.Branches) != 0 {
		t.Fatalf("flow trail = %#v, want empty", flow.Trail)
	}
}

func TestDetectHandlersExcludesOutboundGRPCClients(t *testing.T) {
	flows, err := DetectHandlers([]string{"testdata/handler_detection"}, Options{})
	if err != nil {
		t.Fatalf("DetectHandlers returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want only inbound handler", len(flows))
	}
	if flows[0].Entrypoint.Symbol != "Server.Create" {
		t.Fatalf("entrypoint symbol = %q, want Server.Create", flows[0].Entrypoint.Symbol)
	}
}
