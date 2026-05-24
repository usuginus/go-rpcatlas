package analyzer

import (
	"github.com/usuginus/go-rpcatlas/internal/model"
)

func hasCall(calls []model.CallRef, symbol string) bool {
	for _, call := range calls {
		if call.Symbol == symbol {
			return true
		}
	}
	return false
}

func hasCallImplementation(candidates []model.ImplementationCandidate, symbol string) bool {
	for _, candidate := range candidates {
		if candidate.Call.Symbol == symbol {
			return true
		}
	}
	return false
}

func findInterfaceCall(calls []model.InterfaceCallTrace, symbol string) *model.InterfaceCallTrace {
	for i := range calls {
		if calls[i].Call.Symbol == symbol {
			return &calls[i]
		}
	}
	return nil
}

func findFunctionValue(calls []model.FunctionValueTrace, wrapper string, function string) *model.FunctionValueTrace {
	for i := range calls {
		if calls[i].Wrapper.Symbol == wrapper && calls[i].Function.Symbol == function {
			return &calls[i]
		}
	}
	return nil
}

func findBranch(branches []model.BranchTrace, kind string, expr string) *model.BranchTrace {
	for i := range branches {
		if branches[i].Kind == kind && branches[i].Expr == expr {
			return &branches[i]
		}
	}
	return nil
}

func findCase(cases []model.BranchCase, label string) *model.BranchCase {
	for i := range cases {
		for _, got := range cases[i].Labels {
			if got == label {
				return &cases[i]
			}
		}
	}
	return nil
}

func findDefaultCase(cases []model.BranchCase) *model.BranchCase {
	for i := range cases {
		if cases[i].Default {
			return &cases[i]
		}
	}
	return nil
}

func findDispatch(dispatches []model.DispatchTrace, symbol string, table string, key string) *model.DispatchTrace {
	for i := range dispatches {
		if dispatches[i].Call.Symbol == symbol && dispatches[i].Table == table && dispatches[i].Key == key {
			return &dispatches[i]
		}
	}
	return nil
}

func findDispatchCase(cases []model.DispatchCase, label string) *model.DispatchCase {
	for i := range cases {
		for _, got := range cases[i].Labels {
			if got == label {
				return &cases[i]
			}
		}
	}
	return nil
}
