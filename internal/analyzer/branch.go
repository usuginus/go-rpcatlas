package analyzer

import (
	"go/ast"
	"go/token"

	"github.com/usuginus/go-rpcatlas/internal/model"
	"github.com/usuginus/go-rpcatlas/internal/rules"
)

func recordSwitchBranchTrace(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	stmt *ast.SwitchStmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	function string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	pos := fset.Position(stmt.Switch)
	trace := model.BranchTrace{
		Kind:     "switch",
		Function: function,
		Expr:     nodeString(fset, stmt.Tag),
		File:     file,
		Line:     pos.Line,
		Depth:    depth,
	}
	for _, child := range stmt.Body.List {
		clause, ok := child.(*ast.CaseClause)
		if !ok {
			continue
		}
		branchCase := model.BranchCase{
			Labels:  caseLabels(fset, clause.List),
			Default: len(clause.List) == 0,
		}
		traceBranchCaseCalls(fset, file, &branchCase, clause.Body, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
		if branchCaseHasCalls(branchCase) {
			trace.Cases = append(trace.Cases, branchCase)
		}
	}
	appendBranchTrace(flow, trace)
}

func recordTypeSwitchBranchTrace(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	stmt *ast.TypeSwitchStmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	function string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	pos := fset.Position(stmt.Switch)
	trace := model.BranchTrace{
		Kind:     "type_switch",
		Function: function,
		Expr:     nodeString(fset, stmt.Assign),
		File:     file,
		Line:     pos.Line,
		Depth:    depth,
	}
	for _, child := range stmt.Body.List {
		clause, ok := child.(*ast.CaseClause)
		if !ok {
			continue
		}
		branchCase := model.BranchCase{
			Labels:  caseLabels(fset, clause.List),
			Default: len(clause.List) == 0,
		}
		caseScope := typeSwitchCaseScope(fset, scope, stmt, clause)
		traceBranchCaseCalls(fset, file, &branchCase, clause.Body, caseScope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
		if branchCaseHasCalls(branchCase) {
			trace.Cases = append(trace.Cases, branchCase)
		}
	}
	appendBranchTrace(flow, trace)
}

func recordIfBranchTrace(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	stmt *ast.IfStmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	function string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	pos := fset.Position(stmt.If)
	trace := model.BranchTrace{
		Kind:     "if",
		Function: function,
		Expr:     nodeString(fset, stmt.Cond),
		File:     file,
		Line:     pos.Line,
		Depth:    depth,
	}
	appendIfBranchCases(fset, file, &trace, stmt, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
	appendBranchTrace(flow, trace)
}

func appendIfBranchCases(
	fset *token.FileSet,
	file string,
	trace *model.BranchTrace,
	stmt *ast.IfStmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	function string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	thenCase := model.BranchCase{Labels: []string{"then"}}
	traceBranchCaseCalls(fset, file, &thenCase, stmt.Body.List, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
	if branchCaseHasCalls(thenCase) {
		trace.Cases = append(trace.Cases, thenCase)
	}
	appendIfElseCases(fset, file, trace, stmt.Else, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
}

func appendIfElseCases(
	fset *token.FileSet,
	file string,
	trace *model.BranchTrace,
	elseNode ast.Stmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	function string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	switch node := elseNode.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		elseCase := model.BranchCase{Default: true}
		traceBranchCaseCalls(fset, file, &elseCase, node.List, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
		if branchCaseHasCalls(elseCase) {
			trace.Cases = append(trace.Cases, elseCase)
		}
	case *ast.IfStmt:
		elseIfCase := model.BranchCase{Labels: []string{"else if " + nodeString(fset, node.Cond)}}
		traceBranchCaseCalls(fset, file, &elseIfCase, node.Body.List, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
		if branchCaseHasCalls(elseIfCase) {
			trace.Cases = append(trace.Cases, elseIfCase)
		}
		appendIfElseCases(fset, file, trace, node.Else, scope, index, depth, function, maxDepth, ruleSet, stdlibPackageAliases)
	}
}

func traceBranchCaseCalls(
	fset *token.FileSet,
	file string,
	branchCase *model.BranchCase,
	body []ast.Stmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	for _, stmt := range body {
		ast.Inspect(stmt, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.SwitchStmt, *ast.TypeSwitchStmt:
				return false
			case *ast.CallExpr:
				ref, added := recordBranchCall(fset, file, branchCase, n, scope, index, depth, via, ruleSet, stdlibPackageAliases)
				traceFunctionValueArgsForBranchCase(fset, file, branchCase, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
				if !added || depth >= maxDepth {
					return true
				}
				for _, candidate := range resolveCandidates(ref, scope, index, ruleSet) {
					candidateDepth := depth + 1
					recordBranchImplementation(fset, branchCase, candidate, ref.Symbol, candidateDepth, ruleSet)
					if candidateDepth < maxDepth {
						traceFunctionCallsForBranchCase(fset, branchCase, candidate, index, candidateDepth, implementationSymbol(candidate), maxDepth, ruleSet)
					}
				}
			}
			return true
		})
	}
}

func traceFunctionCallsForBranchCase(
	fset *token.FileSet,
	branchCase *model.BranchCase,
	info functionInfo,
	index projectIndex,
	currentDepth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
) {
	if currentDepth > maxDepth {
		return
	}
	scope := newScope(fset, info.fn, index, info.packageName, info.receiverType, info.receiverVar)
	ast.Inspect(info.fn.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			return false
		case *ast.CallExpr:
			ref, added := recordBranchCall(fset, info.file, branchCase, n, scope, index, currentDepth, via, ruleSet, info.stdlibPackageAliases)
			traceFunctionValueArgsForBranchCase(fset, info.file, branchCase, n, scope, index, currentDepth, via, maxDepth, ruleSet, info.stdlibPackageAliases)
			if !added || currentDepth >= maxDepth {
				return true
			}
			for _, candidate := range resolveCandidates(ref, scope, index, ruleSet) {
				candidateDepth := currentDepth + 1
				recordBranchImplementation(fset, branchCase, candidate, ref.Symbol, candidateDepth, ruleSet)
				if candidateDepth < maxDepth {
					traceFunctionCallsForBranchCase(fset, branchCase, candidate, index, candidateDepth, implementationSymbol(candidate), maxDepth, ruleSet)
				}
			}
		}
		return true
	})
}

func traceIfCasesForFlow(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	stmt *ast.IfStmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	traceIfInitForFlow(fset, file, flow, stmt.Init, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
	traceCaseCallsForFlow(fset, file, flow, stmt.Body.List, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
	traceIfElseForFlow(fset, file, flow, stmt.Else, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
}

func traceIfInitForFlow(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	init ast.Stmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	if init == nil {
		return
	}
	traceCaseCallsForFlow(fset, file, flow, []ast.Stmt{init}, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
}

func traceIfElseForFlow(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	elseNode ast.Stmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	switch node := elseNode.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		traceCaseCallsForFlow(fset, file, flow, node.List, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
	case *ast.IfStmt:
		traceIfInitForFlow(fset, file, flow, node.Init, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
		traceCaseCallsForFlow(fset, file, flow, node.Body.List, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
		traceIfElseForFlow(fset, file, flow, node.Else, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
	}
}

func traceTypeSwitchCasesForFlow(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	stmt *ast.TypeSwitchStmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	for _, child := range stmt.Body.List {
		clause, ok := child.(*ast.CaseClause)
		if !ok {
			continue
		}
		caseScope := typeSwitchCaseScope(fset, scope, stmt, clause)
		traceCaseCallsForFlow(fset, file, flow, clause.Body, caseScope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
	}
}

func traceCaseCallsForFlow(
	fset *token.FileSet,
	file string,
	flow *model.APIFlow,
	body []ast.Stmt,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	for _, stmt := range body {
		ast.Inspect(stmt, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.IfStmt:
				recordIfBranchTrace(fset, file, flow, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
				traceIfCasesForFlow(fset, file, flow, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
				return false
			case *ast.SwitchStmt:
				recordSwitchBranchTrace(fset, file, flow, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
			case *ast.TypeSwitchStmt:
				recordTypeSwitchBranchTrace(fset, file, flow, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
				traceTypeSwitchCasesForFlow(fset, file, flow, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
				return false
			case *ast.CallExpr:
				ref, added := recordCall(fset, file, flow, n, scope, index, depth, via, ruleSet, stdlibPackageAliases)
				traceFunctionValueArgsForFlow(fset, file, flow, n, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases)
				if !added || depth >= maxDepth {
					return true
				}
				candidateDepth := depth + 1
				resolved := resolveCall(ref, scope, index, ruleSet)
				recordInterfaceCall(fset, flow, ref, resolved, candidateDepth, maxDepth)
				recordDispatchCall(fset, flow, n, ref, scope, index, candidateDepth, maxDepth, ruleSet)
				for _, candidate := range resolved.candidates {
					recordImplementation(fset, flow, candidate, ref.Symbol, candidateDepth, ruleSet)
					if candidateDepth < maxDepth {
						traceFunctionCalls(fset, flow, candidate, index, candidateDepth, implementationSymbol(candidate), maxDepth, ruleSet)
					}
				}
			}
			return true
		})
	}
}

func traceFunctionValueArgsForBranchCase(
	fset *token.FileSet,
	file string,
	branchCase *model.BranchCase,
	call *ast.CallExpr,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	maxDepth int,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) {
	traceFunctionValueArgs(fset, file, call, scope, index, depth, via, maxDepth, ruleSet, stdlibPackageAliases, functionValueTraceSink{
		recordFunctionValue: func(trace model.FunctionValueTrace) {
			branchCase.FunctionValues = appendFunctionValueTrace(branchCase.FunctionValues, trace)
		},
		recordImplementation: func(info functionInfo, via string, depth int) {
			recordBranchImplementation(fset, branchCase, info, via, depth, ruleSet)
		},
		traceImplementation: func(info functionInfo, depth int, via string) {
			traceFunctionCallsForBranchCase(fset, branchCase, info, index, depth, via, maxDepth, ruleSet)
		},
	})
}

func recordBranchCall(
	fset *token.FileSet,
	file string,
	branchCase *model.BranchCase,
	call *ast.CallExpr,
	scope scopeInfo,
	index projectIndex,
	depth int,
	via string,
	ruleSet rules.RuleSet,
	stdlibPackageAliases map[string]bool,
) (model.CallRef, bool) {
	decision := decideCallTrace(fset, file, call, scope, index, depth, via, ruleSet, stdlibPackageAliases)
	if !decision.Trace {
		return decision.Ref, false
	}
	appendBranchCall(branchCase, decision.Ref, classify(decision.Ref, scope, ruleSet.Layers), ruleSet)
	return decision.Ref, true
}

func recordBranchImplementation(fset *token.FileSet, branchCase *model.BranchCase, info functionInfo, via string, depth int, ruleSet rules.RuleSet) {
	ref := implementationRef(fset, info, via, depth)
	appendBranchCall(branchCase, ref, classifyByFile(ref, info.file, ruleSet.Layers), ruleSet)
}

func appendBranchCall(branchCase *model.BranchCase, ref model.CallRef, layer string, ruleSet rules.RuleSet) {
	if layer == "unknown" {
		branchCase.Unknown = append(branchCase.Unknown, ref)
		return
	}
	branchCase.AppendLayerCall(layer, ref, layerNames(ruleSet.Layers))
}

func caseLabels(fset *token.FileSet, expressions []ast.Expr) []string {
	labels := make([]string, 0, len(expressions))
	for _, expr := range expressions {
		if label := nodeString(fset, expr); label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

func branchCaseHasCalls(branchCase model.BranchCase) bool {
	return len(branchCase.Layers) > 0 || len(branchCase.FunctionValues) > 0 || len(branchCase.Unknown) > 0
}

func appendBranchTrace(flow *model.APIFlow, trace model.BranchTrace) {
	if len(trace.Cases) == 0 {
		return
	}
	for _, existing := range flow.Trail.Branches {
		if existing.Kind == trace.Kind &&
			existing.Function == trace.Function &&
			existing.File == trace.File &&
			existing.Line == trace.Line {
			return
		}
	}
	flow.Trail.Branches = append(flow.Trail.Branches, trace)
}

func typeSwitchCaseScope(fset *token.FileSet, scope scopeInfo, stmt *ast.TypeSwitchStmt, clause *ast.CaseClause) scopeInfo {
	name := typeSwitchVariableName(stmt.Assign)
	if name == "" || len(clause.List) != 1 {
		return scope
	}
	typeName := nodeString(fset, clause.List[0])
	if typeName == "" || typeName == "nil" {
		return scope
	}
	next := scope
	next.localTypes = make(map[string]string, len(scope.localTypes)+1)
	for name, typ := range scope.localTypes {
		next.localTypes[name] = typ
	}
	next.localTypes[name] = typeName
	return next
}

func typeSwitchVariableName(assign ast.Stmt) string {
	switch stmt := assign.(type) {
	case *ast.AssignStmt:
		if len(stmt.Lhs) != 1 {
			return ""
		}
		ident, ok := stmt.Lhs[0].(*ast.Ident)
		if !ok || ident.Name == "_" {
			return ""
		}
		return ident.Name
	default:
		return ""
	}
}
