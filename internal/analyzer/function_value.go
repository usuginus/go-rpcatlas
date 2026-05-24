package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/usuginus/go-rpcatlas/internal/model"
	"github.com/usuginus/go-rpcatlas/internal/rules"
)

type functionValueTraceSink struct {
	recordFunctionValue  func(model.FunctionValueTrace)
	recordImplementation func(functionInfo, string, int)
	traceImplementation  func(functionInfo, int, string)
}

func traceFunctionValueArgs(
	fset *token.FileSet,
	file string,
	call *ast.CallExpr,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
	sink functionValueTraceSink,
) {
	for _, arg := range call.Args {
		traceFunctionValueArg(fset, file, call, arg, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases, sink)
	}
}

func traceFunctionValueArg(
	fset *token.FileSet,
	file string,
	call *ast.CallExpr,
	arg ast.Expr,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
	sink functionValueTraceSink,
) {
	ref, ok := functionValueCallRef(fset, file, arg, index, scope)
	if !ok || ref.Symbol == "" {
		return
	}
	candidates := resolveFunctionValueCandidates(ref, scope, index, ruleSet)
	if len(candidates) == 0 {
		return
	}

	wrapper := functionValueWrapperRef(fset, file, call, index, scope, depth, via)
	ref.Depth = depth
	ref.Via = wrapper.Symbol
	if ref.Via == "" {
		ref.Via = via
	}
	implementationDepth := depth
	traceDepth := depth + 1
	trace := model.FunctionValueTrace{
		Wrapper:  wrapper,
		Function: ref,
	}
	for _, candidate := range candidates {
		trace.Implementations = append(trace.Implementations, model.ImplementationCandidate{
			Call:     implementationRef(fset, candidate, ref.Symbol, implementationDepth),
			Expanded: traceDepth <= maxDepth,
		})
	}
	sink.recordFunctionValue(trace)

	if implementationDepth > maxDepth {
		return
	}
	for _, candidate := range candidates {
		sink.recordImplementation(candidate, via, implementationDepth)
		if traceDepth <= maxDepth {
			sink.traceImplementation(candidate, traceDepth, implementationSymbol(candidate))
		}
	}
}

func functionValueCallRef(
	fset *token.FileSet,
	file string,
	arg ast.Expr,
	index projectIndex,
	scope scopeInfo,
) (model.CallRef, bool) {
	arg = unwrapParenExpr(arg)
	switch arg.(type) {
	case *ast.Ident, *ast.SelectorExpr:
	default:
		return model.CallRef{}, false
	}

	target := callTarget(arg, index, scope)
	if target.Method == "" || target.Symbol == "" {
		return model.CallRef{}, false
	}
	pos := fset.Position(arg.Pos())
	return model.CallRef{
		Symbol:   target.Symbol,
		Receiver: target.Receiver,
		Method:   target.Method,
		File:     file,
		Line:     pos.Line,
	}, true
}

func functionValueWrapperRef(
	fset *token.FileSet,
	file string,
	call *ast.CallExpr,
	index projectIndex,
	scope scopeInfo,
	depth int,
	via string,
) model.CallRef {
	ref := callRef(fset, file, call, index, scope)
	ref.Depth = depth
	ref.Via = via
	return ref
}

func unwrapParenExpr(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

func resolveFunctionValueCandidates(ref model.CallRef, scope scopeInfo, index projectIndex, ruleSet rules.RuleSet) []functionInfo {
	if ref.Method == "" {
		return nil
	}
	seen := make(map[string]bool)
	var candidates []functionInfo
	add := func(candidate functionInfo) {
		if candidate.fn == nil || isMockCandidate(candidate, ruleSet.Resolution) {
			return
		}
		key := functionInfoKey(candidate)
		if seen[key] {
			return
		}
		seen[key] = true
		candidates = append(candidates, candidate)
	}

	if ref.Receiver != "" {
		for _, candidate := range packageFunctionsByName(ref, index) {
			add(candidate)
		}
		for _, candidate := range resolveCandidates(ref, scope, index, ruleSet) {
			add(candidate)
		}
		return candidates
	}

	if scope.localTypes[ref.Method] != "" {
		return nil
	}
	for _, candidate := range index.functionsByName[ref.Method] {
		if candidate.packageName == scope.packageName {
			add(candidate)
		}
	}
	return candidates
}

func appendFunctionValueTrace(traces []model.FunctionValueTrace, trace model.FunctionValueTrace) []model.FunctionValueTrace {
	key := functionValueTraceKey(trace)
	for i := range traces {
		if functionValueTraceKey(traces[i]) != key {
			continue
		}
		traces[i].Implementations = appendImplementationCandidates(
			traces[i].Implementations,
			trace.Implementations...,
		)
		return traces
	}
	trace.Implementations = appendImplementationCandidates(nil, trace.Implementations...)
	return append(traces, trace)
}

func functionValueTraceKey(trace model.FunctionValueTrace) string {
	return fmt.Sprintf(
		"%s\x00%s\x00%d\x00%s\x00%s\x00%d",
		trace.Wrapper.Symbol,
		trace.Wrapper.File,
		trace.Wrapper.Line,
		trace.Function.Symbol,
		trace.Function.File,
		trace.Function.Line,
	)
}
