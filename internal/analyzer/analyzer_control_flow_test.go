package analyzer

import (
	"testing"
)

func TestAnalyzeExpandsCommonConditionalAndDispatchShapes(t *testing.T) {
	flows, err := Analyze([]string{"testdata/control_flow"}, Options{Depth: 3})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("len(flows) = %d, want 1", len(flows))
	}

	flow := flows[0]
	ifBranch := findBranch(flow.Trail.Branches, "if", "cmd.Fast")
	if ifBranch == nil {
		t.Fatalf("branches = %#v, want if cmd.Fast", flow.Trail.Branches)
	}
	thenCase := findCase(ifBranch.Cases, "then")
	if thenCase == nil || !hasCall(thenCase.LayerCalls("usecase"), "u.fastPath") {
		t.Fatalf("then case = %#v, want u.fastPath", thenCase)
	}
	elseIfCase := findCase(ifBranch.Cases, "else if cmd.Kind == KindBeta")
	if elseIfCase == nil || !hasCall(elseIfCase.LayerCalls("usecase"), "u.betaPath") {
		t.Fatalf("else-if case = %#v, want u.betaPath", elseIfCase)
	}
	elseCase := findDefaultCase(ifBranch.Cases)
	if elseCase == nil || !hasCall(elseCase.LayerCalls("usecase"), "u.slowPath") {
		t.Fatalf("else case = %#v, want u.slowPath", elseCase)
	}

	for _, want := range []struct {
		symbol string
		table  string
		key    string
	}{
		{symbol: "packageProcessors[cmd.Kind].Process", table: "packageProcessors", key: "cmd.Kind"},
		{symbol: "localProcessors[cmd.Kind].Process", table: "localProcessors", key: "cmd.Kind"},
		{symbol: "u.processors[cmd.Kind].Process", table: "u.processors", key: "cmd.Kind"},
	} {
		dispatch := findDispatch(flow.Trail.Dispatches, want.symbol, want.table, want.key)
		if dispatch == nil {
			t.Fatalf("dispatches = %#v, want %s from %s[%s]", flow.Trail.Dispatches, want.symbol, want.table, want.key)
		}
		alphaCase := findDispatchCase(dispatch.Cases, "KindAlpha")
		if alphaCase == nil || !hasCall(alphaCase.LayerCalls("usecase"), "alphaUsecase.Process") {
			t.Fatalf("%s KindAlpha case = %#v, want alphaUsecase.Process", want.symbol, alphaCase)
		}
		betaCase := findDispatchCase(dispatch.Cases, "KindBeta")
		if betaCase == nil || !hasCall(betaCase.LayerCalls("usecase"), "betaUsecase.Process") {
			t.Fatalf("%s KindBeta case = %#v, want betaUsecase.Process", want.symbol, betaCase)
		}
	}
}
