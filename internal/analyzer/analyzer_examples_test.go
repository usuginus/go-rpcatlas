package analyzer

import (
	"testing"

	"github.com/usuginus/calltrail-go/internal/rules"
)

func TestAnalyzeGRPCBasicExample(t *testing.T) {
	flows, err := Analyze([]string{"../../examples/grpc-basic"}, Options{Depth: 3})
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
	if !hasCall(flow.Trail.LayerCalls("usecase"), "fooUsecase.GetFoo") {
		t.Fatalf("usecase layer = %#v, want fooUsecase.GetFoo", flow.Trail.LayerCalls("usecase"))
	}
	if !hasCall(flow.Trail.LayerCalls("repository"), "fooRepository.FindFoo") {
		t.Fatalf("repository layer = %#v, want fooRepository.FindFoo", flow.Trail.LayerCalls("repository"))
	}
	if !hasCall(flow.Trail.LayerCalls("converter"), "fooConverter.ToResponse") {
		t.Fatalf("converter layer = %#v, want fooConverter.ToResponse", flow.Trail.LayerCalls("converter"))
	}
}

func TestAnalyzeCustomLayersExample(t *testing.T) {
	ruleSet, err := rules.Load("../../examples/custom-layers/.calltrail.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	flows, err := Analyze([]string{"../../examples/custom-layers"}, Options{
		Depth: 3,
		Rules: ruleSet,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if flow.Name != "ProcessFoo" {
		t.Fatalf("flow.Name = %q, want ProcessFoo", flow.Name)
	}
	for layer, symbol := range map[string]string{
		"application":     "fooApplication.ProcessFoo",
		"domain":          "fooPolicy.Validate",
		"persistence":     "fooStore.Insert",
		"external_client": "externalClient.Index",
	} {
		if !hasCall(flow.Trail.LayerCalls(layer), symbol) {
			t.Fatalf("%s layer = %#v, want %s", layer, flow.Trail.LayerCalls(layer), symbol)
		}
	}
}

func TestAnalyzeBranchDispatchExample(t *testing.T) {
	ruleSet, err := rules.Load("../../examples/branch-dispatch/.calltrail.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	flows, err := Analyze([]string{"../../examples/branch-dispatch"}, Options{
		Depth: 3,
		Rules: ruleSet,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if flow.Name != "ProcessFoo" {
		t.Fatalf("flow.Name = %q, want ProcessFoo", flow.Name)
	}
	if len(flow.Trail.Branches) != 2 {
		t.Fatalf("branches = %#v, want 2 branches", flow.Trail.Branches)
	}

	typeSwitch := findBranch(flow.Trail.Branches, "type_switch", "payload := cmd.Payload.(type)")
	if typeSwitch == nil {
		t.Fatalf("type switch branch not found: %#v", flow.Trail.Branches)
	}
	if got := len(typeSwitch.Cases); got != 3 {
		t.Fatalf("type switch cases = %d, want 3", got)
	}
	alphaCase := findCase(typeSwitch.Cases, "AlphaPayload")
	if alphaCase == nil || !hasCall(alphaCase.LayerCalls("domain"), "fooPolicy.ValidateAlpha") {
		t.Fatalf("AlphaPayload case = %#v, want fooPolicy.ValidateAlpha", alphaCase)
	}
	if !hasCall(alphaCase.LayerCalls("domain"), "AlphaPayload.Normalize") {
		t.Fatalf("AlphaPayload case = %#v, want AlphaPayload.Normalize", alphaCase)
	}
	betaCase := findCase(typeSwitch.Cases, "BetaPayload")
	if betaCase == nil || !hasCall(betaCase.LayerCalls("domain"), "BetaPayload.Normalize") {
		t.Fatalf("BetaPayload case = %#v, want BetaPayload.Normalize", betaCase)
	}
	defaultPayloadCase := findDefaultCase(typeSwitch.Cases)
	if defaultPayloadCase == nil || !hasCall(defaultPayloadCase.LayerCalls("domain"), "fooPolicy.RejectUnsupportedPayload") {
		t.Fatalf("type switch default case = %#v, want fooPolicy.RejectUnsupportedPayload", defaultPayloadCase)
	}

	valueSwitch := findBranch(flow.Trail.Branches, "switch", "cmd.Mode")
	if valueSwitch == nil {
		t.Fatalf("value switch branch not found: %#v", flow.Trail.Branches)
	}
	publishCase := findCase(valueSwitch.Cases, `"publish"`)
	if publishCase == nil {
		t.Fatalf("publish case not found: %#v", valueSwitch.Cases)
	}
	if !hasCall(publishCase.LayerCalls("persistence"), "fooStore.Publish") {
		t.Fatalf("publish case persistence = %#v, want fooStore.Publish", publishCase.LayerCalls("persistence"))
	}
	if !hasCall(publishCase.LayerCalls("external_client"), "previewClient.Index") {
		t.Fatalf("publish case external_client = %#v, want previewClient.Index", publishCase.LayerCalls("external_client"))
	}
	defaultModeCase := findDefaultCase(valueSwitch.Cases)
	if defaultModeCase == nil || !hasCall(defaultModeCase.LayerCalls("domain"), "fooPolicy.RejectUnsupportedMode") {
		t.Fatalf("mode switch default case = %#v, want fooPolicy.RejectUnsupportedMode", defaultModeCase)
	}
}

func TestAnalyzeMapDispatchExample(t *testing.T) {
	ruleSet, err := rules.Load("../../examples/map-dispatch/.calltrail.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	flows, err := Analyze([]string{"../../examples/map-dispatch"}, Options{
		Depth: 4,
		Rules: ruleSet,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	if flow.Name != "ProcessFoo" {
		t.Fatalf("flow.Name = %q, want ProcessFoo", flow.Name)
	}
	dispatch := findDispatch(flow.Trail.Dispatches, "processor.Process", "a.processors", "cmd.Kind")
	if dispatch == nil {
		t.Fatalf("dispatches = %#v, want processor.Process from a.processors[cmd.Kind]", flow.Trail.Dispatches)
	}
	if dispatch.Interface != "FooProcessor" {
		t.Fatalf("dispatch interface = %q, want FooProcessor", dispatch.Interface)
	}

	alphaCase := findDispatchCase(dispatch.Cases, "KindAlpha")
	if alphaCase == nil {
		t.Fatalf("KindAlpha case not found: %#v", dispatch.Cases)
	}
	for layer, symbol := range map[string]string{
		"application": "alphaProcessor.Process",
		"domain":      "fooPolicy.ValidateAlpha",
		"persistence": "fooStore.SaveAlpha",
	} {
		if !hasCall(alphaCase.LayerCalls(layer), symbol) {
			t.Fatalf("KindAlpha %s layer = %#v, want %s", layer, alphaCase.LayerCalls(layer), symbol)
		}
	}

	betaCase := findDispatchCase(dispatch.Cases, "KindBeta")
	if betaCase == nil {
		t.Fatalf("KindBeta case not found: %#v", dispatch.Cases)
	}
	for layer, symbol := range map[string]string{
		"application":     "betaProcessor.Process",
		"domain":          "fooPolicy.ValidateBeta",
		"external_client": "previewClient.RenderBeta",
	} {
		if !hasCall(betaCase.LayerCalls(layer), symbol) {
			t.Fatalf("KindBeta %s layer = %#v, want %s", layer, betaCase.LayerCalls(layer), symbol)
		}
	}
}
