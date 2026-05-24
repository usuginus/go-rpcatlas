package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/usuginus/go-rpcatlas/internal/model"
	"github.com/usuginus/go-rpcatlas/internal/rules"
)

type dispatchTableInfo struct {
	OwnerType string
	FieldName string
	ValueType string
	Cases     []dispatchTableCase
}

type dispatchTableCase struct {
	Label     string
	ValueType string
}

type dispatchLookupInfo struct {
	Table string
	Key   string
	Info  dispatchTableInfo
}

func collectDispatchTables(fset *token.FileSet, packageName string, file *ast.File, index *projectIndex) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ValueSpec:
			collectPackageDispatchTables(fset, packageName, n, index)
		case *ast.CompositeLit:
			collectStructDispatchTables(fset, packageName, n, index)
		}
		return true
	})
}

func collectPackageDispatchTables(fset *token.FileSet, packageName string, spec *ast.ValueSpec, index *projectIndex) {
	for i, name := range spec.Names {
		if i >= len(spec.Values) {
			continue
		}
		mapLit, ok := spec.Values[i].(*ast.CompositeLit)
		if !ok {
			continue
		}
		table := dispatchTableFromMapLiteral(fset, mapLit, *index)
		if len(table.Cases) == 0 {
			continue
		}
		table.FieldName = name.Name
		addPackageDispatchTable(index, packageName, name.Name, table)
	}
}

func collectStructDispatchTables(fset *token.FileSet, packageName string, lit *ast.CompositeLit, index *projectIndex) {
	ownerType := typeKey(packageName, typeString(lit.Type))
	if ownerType == "" {
		return
	}
	for _, elt := range lit.Elts {
		field, mapLit, ok := dispatchFieldMapLiteral(elt)
		if !ok {
			continue
		}
		table := dispatchTableFromMapLiteral(fset, mapLit, *index)
		if len(table.Cases) == 0 {
			continue
		}
		table.OwnerType = ownerType
		table.FieldName = field
		addDispatchTable(index, table)
	}
}

func dispatchFieldMapLiteral(expr ast.Expr) (string, *ast.CompositeLit, bool) {
	kv, ok := expr.(*ast.KeyValueExpr)
	if !ok {
		return "", nil, false
	}
	field, ok := kv.Key.(*ast.Ident)
	if !ok {
		return "", nil, false
	}
	mapLit, ok := kv.Value.(*ast.CompositeLit)
	if !ok {
		return "", nil, false
	}
	if _, ok := mapLit.Type.(*ast.MapType); !ok {
		return "", nil, false
	}
	return field.Name, mapLit, true
}

func dispatchTableFromMapLiteral(fset *token.FileSet, mapLit *ast.CompositeLit, index projectIndex) dispatchTableInfo {
	table := dispatchTableInfo{ValueType: mapLiteralValueType(mapLit)}
	if table.ValueType == "" {
		return dispatchTableInfo{}
	}
	for _, mapElt := range mapLit.Elts {
		tableCase, ok := dispatchMapCase(fset, mapElt, index)
		if !ok {
			continue
		}
		table.Cases = append(table.Cases, tableCase)
	}
	return table
}

func mapLiteralValueType(lit *ast.CompositeLit) string {
	mapType, ok := lit.Type.(*ast.MapType)
	if !ok {
		return ""
	}
	return typeString(mapType.Value)
}

func dispatchMapCase(fset *token.FileSet, expr ast.Expr, index projectIndex) (dispatchTableCase, bool) {
	kv, ok := expr.(*ast.KeyValueExpr)
	if !ok {
		return dispatchTableCase{}, false
	}
	label := nodeString(fset, kv.Key)
	valueType := dispatchValueType(kv.Value, index)
	if label == "" || valueType == "" {
		return dispatchTableCase{}, false
	}
	return dispatchTableCase{Label: label, ValueType: baseType(valueType)}, true
}

func dispatchValueType(expr ast.Expr, index projectIndex) string {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return dispatchCallReturnType(e, index)
	case *ast.CompositeLit:
		return typeString(e.Type)
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return dispatchValueType(e.X, index)
		}
	}
	return ""
}

func dispatchCallReturnType(call *ast.CallExpr, index projectIndex) string {
	candidates := callReturnTypeCandidates(call, index)
	if concreteType := commonConcreteReturnType(candidates, index); concreteType != "" {
		return concreteType
	}
	return commonReturnType(candidates)
}

func callReturnTypeCandidates(call *ast.CallExpr, index projectIndex) []functionInfo {
	ref := callTarget(call.Fun, index, scopeInfo{})
	if ref.Method == "" {
		return nil
	}
	if ref.Receiver != "" {
		var matches []functionInfo
		for _, candidate := range index.functionsByName[ref.Method] {
			if candidate.packageName == ref.Receiver {
				matches = append(matches, candidate)
			}
		}
		return matches
	}
	return index.functionsByName[ref.Method]
}

func commonConcreteReturnType(candidates []functionInfo, index projectIndex) string {
	var typ string
	for _, candidate := range candidates {
		concreteType := concreteReturnType(candidate, index)
		if concreteType == "" {
			return ""
		}
		if typ == "" {
			typ = concreteType
			continue
		}
		if typ != concreteType {
			return ""
		}
	}
	return typ
}

func concreteReturnType(info functionInfo, index projectIndex) string {
	if info.fn == nil || info.fn.Body == nil {
		return ""
	}
	declared := baseType(info.returnType)
	if declared != "" && len(lookupTypeMembers(index.interfaces, typeKey(info.packageName, info.returnType), declared)) == 0 {
		return baseType(info.returnType)
	}
	var typ string
	for _, stmt := range info.fn.Body.List {
		returnStmt, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(returnStmt.Results) == 0 {
			continue
		}
		concreteType := dispatchValueType(returnStmt.Results[0], index)
		if concreteType == "" {
			return ""
		}
		if typ == "" {
			typ = concreteType
			continue
		}
		if typ != concreteType {
			return ""
		}
	}
	return typ
}

func addDispatchTable(index *projectIndex, table dispatchTableInfo) {
	key := dispatchTableKey(table.OwnerType, table.FieldName)
	addDispatchTableByKey(index, key, table)
	fallbackKey := dispatchTableKey(baseType(table.OwnerType), table.FieldName)
	if fallbackKey != key {
		addDispatchTableByKey(index, fallbackKey, table)
	}
}

func addPackageDispatchTable(index *projectIndex, packageName string, name string, table dispatchTableInfo) {
	addDispatchTableByKey(index, packageDispatchTableKey(packageName, name), table)
	if _, exists := index.dispatchTables[name]; !exists {
		addDispatchTableByKey(index, name, table)
	}
}

func addDispatchTableByKey(index *projectIndex, key string, table dispatchTableInfo) {
	existing := index.dispatchTables[key]
	if existing.OwnerType == "" {
		existing = table
	} else {
		if existing.ValueType == "" {
			existing.ValueType = table.ValueType
		}
		existing.Cases = appendDispatchTableCases(existing.Cases, table.Cases...)
	}
	existing.Cases = appendDispatchTableCases(nil, existing.Cases...)
	index.dispatchTables[key] = existing
}

func appendDispatchTableCases(cases []dispatchTableCase, more ...dispatchTableCase) []dispatchTableCase {
	seen := make(map[string]bool, len(cases)+len(more))
	for _, tableCase := range cases {
		seen[dispatchTableCaseKey(tableCase)] = true
	}
	for _, tableCase := range more {
		key := dispatchTableCaseKey(tableCase)
		if seen[key] {
			continue
		}
		seen[key] = true
		cases = append(cases, tableCase)
	}
	return cases
}

func dispatchTableCaseKey(tableCase dispatchTableCase) string {
	return tableCase.Label + "\x00" + tableCase.ValueType
}

func dispatchTableKey(ownerType string, fieldName string) string {
	return baseType(ownerType) + "." + fieldName
}

func packageDispatchTableKey(packageName string, name string) string {
	if packageName == "" {
		return name
	}
	return packageName + "." + name
}

func collectLocalDispatchTables(fset *token.FileSet, body *ast.BlockStmt, index projectIndex) map[string]dispatchTableInfo {
	out := make(map[string]dispatchTableInfo)
	if body == nil {
		return out
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			if len(n.Rhs) == 1 {
				bindLocalDispatchTable(fset, out, n.Lhs[:1], n.Rhs[0], index)
				return true
			}
			for i, rhs := range n.Rhs {
				if i >= len(n.Lhs) {
					continue
				}
				bindLocalDispatchTable(fset, out, n.Lhs[i:i+1], rhs, index)
			}
		case *ast.ValueSpec:
			for i, value := range n.Values {
				if i >= len(n.Names) {
					continue
				}
				bindLocalDispatchTable(fset, out, []ast.Expr{n.Names[i]}, value, index)
			}
		}
		return true
	})
	return out
}

func bindLocalDispatchTable(
	fset *token.FileSet,
	out map[string]dispatchTableInfo,
	lhs []ast.Expr,
	rhs ast.Expr,
	index projectIndex,
) {
	mapLit, ok := rhs.(*ast.CompositeLit)
	if !ok || len(lhs) == 0 {
		return
	}
	name, ok := lhs[0].(*ast.Ident)
	if !ok || name.Name == "_" {
		return
	}
	table := dispatchTableFromMapLiteral(fset, mapLit, index)
	if len(table.Cases) == 0 {
		return
	}
	table.FieldName = name.Name
	out[name.Name] = table
}

func collectLocalDispatches(fset *token.FileSet, body *ast.BlockStmt, index projectIndex, scope scopeInfo) map[string]dispatchLookupInfo {
	out := make(map[string]dispatchLookupInfo)
	if body == nil {
		return out
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			if len(n.Rhs) == 1 {
				bindDispatchLookup(fset, out, n.Lhs[:1], n.Rhs[0], index, scope)
				return true
			}
			for i, rhs := range n.Rhs {
				if i >= len(n.Lhs) {
					continue
				}
				bindDispatchLookup(fset, out, n.Lhs[i:i+1], rhs, index, scope)
			}
		case *ast.ValueSpec:
			for i, value := range n.Values {
				if i >= len(n.Names) {
					continue
				}
				bindDispatchLookup(fset, out, []ast.Expr{n.Names[i]}, value, index, scope)
			}
		}
		return true
	})
	return out
}

func bindDispatchLookup(
	fset *token.FileSet,
	out map[string]dispatchLookupInfo,
	lhs []ast.Expr,
	rhs ast.Expr,
	index projectIndex,
	scope scopeInfo,
) {
	lookup, ok := dispatchLookupForExpr(fset, rhs, index, scope)
	if !ok || len(lhs) == 0 {
		return
	}
	name, ok := lhs[0].(*ast.Ident)
	if !ok || name.Name == "_" {
		return
	}
	out[name.Name] = lookup
}

func dispatchLookupForExpr(fset *token.FileSet, expr ast.Expr, index projectIndex, scope scopeInfo) (dispatchLookupInfo, bool) {
	indexExpr, ok := expr.(*ast.IndexExpr)
	if !ok {
		return dispatchLookupInfo{}, false
	}
	if table, ok := dispatchLookupTable(indexExpr.X, index, scope); ok {
		return dispatchLookupInfo{
			Table: nodeString(fset, indexExpr.X),
			Key:   nodeString(fset, indexExpr.Index),
			Info:  table,
		}, true
	}
	return dispatchLookupInfo{}, false
}

func dispatchLookupTable(expr ast.Expr, index projectIndex, scope scopeInfo) (dispatchTableInfo, bool) {
	if ident, ok := expr.(*ast.Ident); ok {
		if table, ok := scope.localDispatchTables[ident.Name]; ok && len(table.Cases) > 0 {
			return table, true
		}
		for _, key := range []string{packageDispatchTableKey(scope.packageName, ident.Name), ident.Name} {
			table, ok := index.dispatchTables[key]
			if ok && len(table.Cases) > 0 {
				return table, true
			}
		}
		return dispatchTableInfo{}, false
	}
	tableKey, ok := dispatchLookupTableKey(expr, scope)
	if !ok {
		return dispatchTableInfo{}, false
	}
	table, ok := index.dispatchTables[tableKey]
	if !ok || len(table.Cases) == 0 {
		return dispatchTableInfo{}, false
	}
	return table, true
}

func dispatchLookupTableKey(expr ast.Expr, scope scopeInfo) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ownerType := baseType(resolveReceiverType(typeString(selector.X), scope))
	if ownerType == "" {
		return "", false
	}
	return dispatchTableKey(ownerType, selector.Sel.Name), true
}

func recordDispatchCall(
	fset *token.FileSet,
	flow *model.APIFlow,
	call *ast.CallExpr,
	ref model.CallRef,
	scope scopeInfo,
	index projectIndex,
	candidateDepth int,
	maxDepth int,
	ruleSet rules.RuleSet,
) {
	if candidateDepth > maxDepth {
		return
	}
	if directRef, lookup, ok := directDispatchLookupCall(fset, call, ref, index, scope); ok {
		recordDispatchTrace(fset, flow, directRef, lookup, index, candidateDepth, maxDepth, ruleSet)
		return
	}
	if ref.Receiver == "" || ref.Method == "" {
		return
	}
	lookup, ok := scope.localDispatches[ref.Receiver]
	if !ok {
		return
	}
	recordDispatchTrace(fset, flow, ref, lookup, index, candidateDepth, maxDepth, ruleSet)
}

func directDispatchLookupCall(
	fset *token.FileSet,
	call *ast.CallExpr,
	ref model.CallRef,
	index projectIndex,
	scope scopeInfo,
) (model.CallRef, dispatchLookupInfo, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return model.CallRef{}, dispatchLookupInfo{}, false
	}
	lookup, ok := dispatchLookupForExpr(fset, selector.X, index, scope)
	if !ok {
		return model.CallRef{}, dispatchLookupInfo{}, false
	}
	if receiver := nodeString(fset, selector.X); receiver != "" {
		ref.Receiver = receiver
		ref.Symbol = receiver + "." + selector.Sel.Name
	}
	ref.Method = selector.Sel.Name
	return ref, lookup, true
}

func recordDispatchTrace(
	fset *token.FileSet,
	flow *model.APIFlow,
	ref model.CallRef,
	lookup dispatchLookupInfo,
	index projectIndex,
	candidateDepth int,
	maxDepth int,
	ruleSet rules.RuleSet,
) {
	trace := model.DispatchTrace{
		Table:     lookup.Table,
		Key:       lookup.Key,
		Call:      ref,
		Interface: baseType(lookup.Info.ValueType),
	}
	for _, tableCase := range lookup.Info.Cases {
		dispatchCase := model.DispatchCase{Labels: []string{tableCase.Label}}
		for _, candidate := range dispatchCaseCandidates(tableCase, ref.Method, index, ruleSet) {
			recordDispatchImplementation(fset, &dispatchCase, candidate, ref.Symbol, candidateDepth, ruleSet)
			if candidateDepth < maxDepth {
				traceFunctionCallsForDispatchCase(fset, &dispatchCase, candidate, index, candidateDepth, implementationSymbol(candidate), maxDepth, ruleSet)
			}
		}
		if dispatchCaseHasCalls(dispatchCase) {
			trace.Cases = append(trace.Cases, dispatchCase)
		}
	}
	appendDispatchTrace(flow, trace)
}

func dispatchCaseCandidates(tableCase dispatchTableCase, method string, index projectIndex, ruleSet rules.RuleSet) []functionInfo {
	var candidates []functionInfo
	for _, candidate := range index.methodsByName[method] {
		if candidate.receiverType != tableCase.ValueType {
			continue
		}
		if isMockCandidate(candidate, ruleSet.Resolution) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func traceFunctionCallsForDispatchCase(
	fset *token.FileSet,
	dispatchCase *model.DispatchCase,
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
			ref, added := recordDispatchCaseCall(fset, info.file, dispatchCase, n, scope, index, currentDepth, via, ruleSet, info.stdlibPackageAliases)
			traceFunctionValueArgsForDispatchCase(fset, info.file, dispatchCase, n, scope, index, currentDepth, via, maxDepth, ruleSet, info.stdlibPackageAliases)
			if !added || currentDepth >= maxDepth {
				return true
			}
			for _, candidate := range resolveCandidates(ref, scope, index, ruleSet) {
				candidateDepth := currentDepth + 1
				recordDispatchImplementation(fset, dispatchCase, candidate, ref.Symbol, candidateDepth, ruleSet)
				if candidateDepth < maxDepth {
					traceFunctionCallsForDispatchCase(fset, dispatchCase, candidate, index, candidateDepth, implementationSymbol(candidate), maxDepth, ruleSet)
				}
			}
		}
		return true
	})
}

func traceFunctionValueArgsForDispatchCase(
	fset *token.FileSet,
	file string,
	dispatchCase *model.DispatchCase,
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
			dispatchCase.FunctionValues = appendFunctionValueTrace(dispatchCase.FunctionValues, trace)
		},
		recordImplementation: func(info functionInfo, via string, depth int) {
			recordDispatchImplementation(fset, dispatchCase, info, via, depth, ruleSet)
		},
		traceImplementation: func(info functionInfo, depth int, via string) {
			traceFunctionCallsForDispatchCase(fset, dispatchCase, info, index, depth, via, maxDepth, ruleSet)
		},
	})
}

func recordDispatchCaseCall(
	fset *token.FileSet,
	file string,
	dispatchCase *model.DispatchCase,
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
	appendDispatchCaseCall(dispatchCase, decision.Ref, classify(decision.Ref, scope, ruleSet.Layers), ruleSet)
	return decision.Ref, true
}

func recordDispatchImplementation(fset *token.FileSet, dispatchCase *model.DispatchCase, info functionInfo, via string, depth int, ruleSet rules.RuleSet) {
	ref := implementationRef(fset, info, via, depth)
	appendDispatchCaseCall(dispatchCase, ref, classifyByFile(ref, info.file, ruleSet.Layers), ruleSet)
}

func appendDispatchCaseCall(dispatchCase *model.DispatchCase, ref model.CallRef, layer string, ruleSet rules.RuleSet) {
	if layer == "unknown" {
		dispatchCase.Unknown = append(dispatchCase.Unknown, ref)
		return
	}
	dispatchCase.AppendLayerCall(layer, ref, layerNames(ruleSet.Layers))
}

func dispatchCaseHasCalls(dispatchCase model.DispatchCase) bool {
	return len(dispatchCase.Layers) > 0 || len(dispatchCase.FunctionValues) > 0 || len(dispatchCase.Unknown) > 0
}

func appendDispatchTrace(flow *model.APIFlow, trace model.DispatchTrace) {
	if len(trace.Cases) == 0 {
		return
	}
	for i := range flow.Trail.Dispatches {
		if dispatchTraceKey(flow.Trail.Dispatches[i]) != dispatchTraceKey(trace) {
			continue
		}
		flow.Trail.Dispatches[i].Cases = appendDispatchCases(flow.Trail.Dispatches[i].Cases, trace.Cases...)
		return
	}
	flow.Trail.Dispatches = append(flow.Trail.Dispatches, trace)
}

func appendDispatchCases(cases []model.DispatchCase, more ...model.DispatchCase) []model.DispatchCase {
	seen := make(map[string]int, len(cases)+len(more))
	for i, dispatchCase := range cases {
		seen[dispatchCaseKey(dispatchCase)] = i
	}
	for _, dispatchCase := range more {
		key := dispatchCaseKey(dispatchCase)
		if existingIndex, ok := seen[key]; ok {
			cases[existingIndex].Layers = append(cases[existingIndex].Layers, dispatchCase.Layers...)
			for _, functionValue := range dispatchCase.FunctionValues {
				cases[existingIndex].FunctionValues = appendFunctionValueTrace(cases[existingIndex].FunctionValues, functionValue)
			}
			cases[existingIndex].Unknown = append(cases[existingIndex].Unknown, dispatchCase.Unknown...)
			continue
		}
		seen[key] = len(cases)
		cases = append(cases, dispatchCase)
	}
	return cases
}

func dispatchTraceKey(trace model.DispatchTrace) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d", trace.Table, trace.Key, trace.Call.Symbol, trace.Call.File, trace.Call.Line)
}

func dispatchCaseKey(dispatchCase model.DispatchCase) string {
	return strings.Join(dispatchCase.Labels, "\x00")
}
