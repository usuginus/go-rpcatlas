package analyzer

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/usuginus/calltrail-go/internal/model"
	"github.com/usuginus/calltrail-go/internal/rules"
)

type scopeInfo struct {
	packageName           string
	receiverType          string
	receiverTypeKey       string
	receiverVar           string
	receiverFields        map[string]string
	localTypes            map[string]string
	localConcreteTypes    map[string]map[string]bool
	localDispatches       map[string]dispatchLookupInfo
	structFields          map[string]map[string]string
	interfaces            map[string]map[string]bool
	constructorFieldTypes map[string]map[string]map[string]bool
}

type resolvedCall struct {
	interfaceType string
	candidates    []functionInfo
}

func newScope(fset *token.FileSet, fn *ast.FuncDecl, index projectIndex, packageName string, receiverType string, receiverVar string) scopeInfo {
	receiverTypeKey := typeKey(packageName, receiverType)
	scope := scopeInfo{
		packageName:           packageName,
		receiverType:          receiverType,
		receiverTypeKey:       receiverTypeKey,
		receiverVar:           receiverVar,
		receiverFields:        lookupTypeMembers(index.structFields, receiverTypeKey, receiverType),
		structFields:          index.structFields,
		interfaces:            index.interfaces,
		constructorFieldTypes: index.constructorFieldTypes,
	}
	scope.localTypes, scope.localConcreteTypes = collectLocalTypes(fn, index, scope)
	scope.localDispatches = collectLocalDispatches(fset, fn.Body, index, scope)
	return scope
}

func resolveCandidates(ref model.CallRef, scope scopeInfo, index projectIndex, ruleSet rules.RuleSet) []functionInfo {
	return resolveCall(ref, scope, index, ruleSet).candidates
}

func resolveCall(ref model.CallRef, scope scopeInfo, index projectIndex, ruleSet rules.RuleSet) resolvedCall {
	resolvedType := resolveReceiverType(ref.Receiver, scope)
	concreteTypes := resolveReceiverConcreteTypes(ref.Receiver, scope)
	fieldType := baseType(resolvedType)
	fieldTypeKey := typeKey(scope.packageName, resolvedType)
	interfaceMethods := lookupInterfaceMethods(index.interfaces, scope.packageName, resolvedType)
	fieldTypeIsInterface := fieldType != "" && len(interfaceMethods) > 0
	if fieldTypeIsInterface && !strings.Contains(resolvedType, ".") && len(lookupTypeMembers(index.structFields, fieldTypeKey, fieldType)) > 0 {
		fieldTypeIsInterface = false
	}
	if fieldTypeIsInterface {
		if len(interfaceMethods) > 0 && !interfaceMethods[ref.Method] {
			return resolvedCall{interfaceType: fieldType}
		}
	}

	var candidates []functionInfo
	assertedImplementations := lookupTypeMembers(index.implementationAssertions, fieldTypeKey, fieldType)
	for _, candidate := range index.methodsByName[ref.Method] {
		if candidate.fn == nil || candidate.receiverType == "" {
			continue
		}
		if fieldTypeIsInterface && candidate.receiverType == fieldType && candidate.receiverTypeKey == fieldTypeKey {
			continue
		}
		if isMockCandidate(candidate, ruleSet.Resolution) {
			continue
		}
		if fieldTypeIsInterface {
			if len(concreteTypes) > 0 && !concreteTypesMatchCandidate(concreteTypes, candidate) {
				continue
			}
			if len(assertedImplementations) > 0 &&
				!assertedImplementations[candidate.receiverTypeKey] && !assertedImplementations[candidate.receiverType] {
				continue
			}
			if !implementsInterface(candidate, fieldTypeKey, fieldType, index) {
				continue
			}
		}
		if fieldType != "" && !fieldTypeIsInterface && candidate.receiverTypeKey != fieldTypeKey && candidate.receiverType != fieldType {
			continue
		}
		if fieldType == "" && candidate.receiverType != strings.TrimPrefix(ref.Receiver, "*") {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if fieldTypeIsInterface && len(candidates) == 0 && shouldUseInterfaceFallback(ref.Method, assertedImplementations, index, ruleSet) {
		candidates = fallbackInterfaceCandidates(ref, fieldType, concreteTypes, scope, index, ruleSet)
	}
	if fieldTypeIsInterface {
		candidates = preferNamedInterfaceImplementations(candidates, fieldType, fieldTypeKey)
	}
	out := resolvedCall{
		candidates: candidates,
	}
	if fieldTypeIsInterface {
		out.interfaceType = fieldType
	}
	return out
}

func preferNamedInterfaceImplementations(candidates []functionInfo, interfaceType string, interfaceTypeKey string) []functionInfo {
	var named []functionInfo
	for _, candidate := range candidates {
		if candidate.receiverType == interfaceType && candidate.receiverTypeKey != interfaceTypeKey {
			named = append(named, candidate)
		}
	}
	if len(named) == 0 {
		return candidates
	}
	return named
}

func implementsInterface(candidate functionInfo, interfaceTypeKey string, interfaceType string, index projectIndex) bool {
	interfaceMethods := lookupInterfaceMethods(index.interfaces, "", interfaceTypeKey)
	if len(interfaceMethods) == 0 {
		interfaceMethods = lookupInterfaceMethods(index.interfaces, "", interfaceType)
	}
	if len(interfaceMethods) == 0 {
		return true
	}
	receiverMethods := lookupTypeMembers(index.methodsByReceiver, candidate.receiverTypeKey, candidate.receiverType)
	if len(receiverMethods) == 0 {
		return false
	}
	for method := range interfaceMethods {
		if !receiverMethods[method] {
			return false
		}
	}
	return true
}

func isMockCandidate(candidate functionInfo, resolution rules.ResolutionRules) bool {
	if matchesAnyPrefix(candidate.receiverType, resolution.SkipImplementations.ReceiverNamePrefixes) {
		return true
	}
	return matchesAnyContains(strings.ToLower(candidate.file), resolution.SkipImplementations.FilePathContains)
}

func concreteTypesMatchCandidate(concreteTypes map[string]bool, candidate functionInfo) bool {
	return concreteTypes[candidate.receiverTypeKey] ||
		concreteTypes[typeKey(candidate.packageName, candidate.receiverType)] ||
		concreteTypes[candidate.receiverType]
}

func fallbackInterfaceCandidates(
	ref model.CallRef,
	interfaceType string,
	concreteTypes map[string]bool,
	scope scopeInfo,
	index projectIndex,
	ruleSet rules.RuleSet,
) []functionInfo {
	var candidates []functionInfo
	for _, candidate := range index.methodsByName[ref.Method] {
		if candidate.fn == nil || candidate.receiverType == "" || isMockCandidate(candidate, ruleSet.Resolution) {
			continue
		}
		if len(concreteTypes) > 0 {
			if concreteTypesMatchCandidate(concreteTypes, candidate) {
				candidates = append(candidates, candidate)
			}
			continue
		}
		if receiverNameMatchesInterface(candidate.receiverType, interfaceType) {
			candidates = append(candidates, candidate)
			continue
		}
		if receiverType := resolveReceiverType(ref.Receiver, scope); receiverNameMatchesInterface(candidate.receiverType, receiverType) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func shouldUseInterfaceFallback(
	methodName string,
	assertedImplementations map[string]bool,
	index projectIndex,
	ruleSet rules.RuleSet,
) bool {
	if len(assertedImplementations) == 0 {
		return true
	}
	for _, candidate := range index.methodsByName[methodName] {
		if !assertedImplementations[candidate.receiverTypeKey] && !assertedImplementations[candidate.receiverType] {
			continue
		}
		if !isMockCandidate(candidate, ruleSet.Resolution) {
			return false
		}
	}
	return true
}

func receiverNameMatchesInterface(receiverType string, interfaceType string) bool {
	receiver := strings.ToLower(baseType(receiverType))
	if receiver == "" {
		return false
	}
	iface := strings.ToLower(baseType(interfaceType))
	return iface != "" && (receiver == iface || strings.Contains(receiver, iface))
}

func callRef(fset *token.FileSet, file string, call *ast.CallExpr, index projectIndex, scope scopeInfo) model.CallRef {
	pos := fset.Position(call.Pos())
	ref := model.CallRef{File: file, Line: pos.Line}
	target := callTarget(call.Fun, index, scope)
	ref.Receiver = target.Receiver
	ref.Method = target.Method
	ref.Symbol = target.Symbol
	return ref
}

func callTarget(fun ast.Expr, index projectIndex, scope scopeInfo) model.CallRef {
	var ref model.CallRef
	switch fn := fun.(type) {
	case *ast.SelectorExpr:
		ref.Receiver = selectorReceiver(fn.X, index, scope)
		ref.Method = fn.Sel.Name
		if ref.Receiver == "" {
			return ref
		}
		if innerCall, ok := fn.X.(*ast.CallExpr); ok {
			if innerSymbol := callDisplaySymbol(innerCall, index, scope); innerSymbol != "" {
				ref.Symbol = innerSymbol + "." + ref.Method
				return ref
			}
		}
		ref.Symbol = ref.Receiver + "." + ref.Method
	case *ast.Ident:
		ref.Symbol = fn.Name
		ref.Method = fn.Name
	}
	return ref
}

func callDisplaySymbol(call *ast.CallExpr, index projectIndex, scope scopeInfo) string {
	ref := callTarget(call.Fun, index, scope)
	if ref.Symbol == "" {
		return ""
	}
	return ref.Symbol + "()"
}

func selectorReceiver(expr ast.Expr, index projectIndex, scope scopeInfo) string {
	switch fn := expr.(type) {
	case *ast.CallExpr:
		return baseType(callReturnType(fn, index, scope))
	default:
		return typeString(expr)
	}
}

func collectLocalTypes(fn *ast.FuncDecl, index projectIndex, scope scopeInfo) (map[string]string, map[string]map[string]bool) {
	out := make(map[string]string)
	concrete := make(map[string]map[string]bool)
	var body *ast.BlockStmt
	if fn != nil {
		body = fn.Body
	}
	if body == nil {
		return out, concrete
	}
	scope.localTypes = out
	scope.localConcreteTypes = concrete
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				name, ok := lhs.(*ast.Ident)
				if !ok || name.Name == "_" || i >= len(n.Rhs) {
					continue
				}
				if typ := inferExprType(n.Rhs[i], index, scope); typ != "" {
					out[name.Name] = typ
				}
				if types := inferExprConcreteTypes(n.Rhs[i], index, scope); len(types) > 0 {
					concrete[name.Name] = types
				}
			}
		case *ast.ValueSpec:
			for i, name := range n.Names {
				if name.Name == "_" {
					continue
				}
				if n.Type != nil {
					out[name.Name] = typeString(n.Type)
					if !isScopeInterfaceType(typeString(n.Type), scope) {
						addType(concrete, name.Name, normalizeType(scope.packageName, typeString(n.Type)))
					}
				}
				if i < len(n.Values) {
					if typ := inferExprType(n.Values[i], index, scope); typ != "" {
						if n.Type == nil {
							out[name.Name] = typ
						}
					}
					if types := inferExprConcreteTypes(n.Values[i], index, scope); len(types) > 0 {
						concrete[name.Name] = types
					}
				}
			}
		}
		return true
	})
	return out, concrete
}

func inferExprType(expr ast.Expr, index projectIndex, scope scopeInfo) string {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return callReturnType(e, index, scope)
	case *ast.CompositeLit:
		return typeString(e.Type)
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return inferExprType(e.X, index, scope)
		}
	case *ast.Ident:
		return scope.localTypes[e.Name]
	}
	return ""
}

func inferExprConcreteTypes(expr ast.Expr, index projectIndex, scope scopeInfo) map[string]bool {
	out := make(map[string]bool)
	switch e := expr.(type) {
	case *ast.CallExpr:
		for typ := range callReturnConcreteTypes(e, index, scope) {
			out[typ] = true
		}
	case *ast.CompositeLit:
		if typ := typeString(e.Type); typ != "" && !isScopeInterfaceType(typ, scope) {
			out[normalizeType(scope.packageName, typ)] = true
		}
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			for typ := range inferExprConcreteTypes(e.X, index, scope) {
				out[typ] = true
			}
		}
	case *ast.Ident:
		if concrete := scope.localConcreteTypes[e.Name]; len(concrete) > 0 {
			for typ := range concrete {
				out[typ] = true
			}
			break
		}
		if typ := scope.localTypes[e.Name]; typ != "" && !isScopeInterfaceType(typ, scope) {
			out[normalizeType(scope.packageName, typ)] = true
		}
	}
	return out
}

func callReturnType(call *ast.CallExpr, index projectIndex, scope scopeInfo) string {
	ref := callTarget(call.Fun, index, scope)
	if ref.Method == "" {
		return ""
	}
	if ref.Receiver != "" {
		if typ := lookupFunctionReturnType(ref, index); typ != "" {
			return typ
		}
		return commonReturnType(resolveCandidates(ref, scope, index, rules.RuleSet{}))
	}
	return commonReturnType(index.functionsByName[ref.Method])
}

func callReturnConcreteTypes(call *ast.CallExpr, index projectIndex, scope scopeInfo) map[string]bool {
	out := make(map[string]bool)
	ref := callTarget(call.Fun, index, scope)
	if ref.Method == "" {
		return out
	}
	var matches []functionInfo
	if ref.Receiver != "" {
		if functions := packageFunctionsByName(ref, index); len(functions) > 0 {
			matches = append(matches, functions...)
		} else {
			matches = append(matches, resolveCandidates(ref, scope, index, rules.RuleSet{})...)
		}
	} else {
		for _, candidate := range index.functionsByName[ref.Method] {
			if candidate.packageName == scope.packageName {
				matches = append(matches, candidate)
			}
		}
	}
	for _, candidate := range matches {
		for typ := range index.concreteReturnTypes[functionInfoKey(candidate)] {
			out[typ] = true
		}
	}
	return out
}

func lookupFunctionReturnType(ref model.CallRef, index projectIndex) string {
	return commonReturnType(packageFunctionsByName(ref, index))
}

func packageFunctionsByName(ref model.CallRef, index projectIndex) []functionInfo {
	var matches []functionInfo
	for _, candidate := range index.functionsByName[ref.Method] {
		if candidate.packageName == ref.Receiver {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func commonReturnType(candidates []functionInfo) string {
	var typ string
	for _, candidate := range candidates {
		if candidate.returnType == "" {
			continue
		}
		if typ == "" {
			typ = candidate.returnType
			continue
		}
		if typ != candidate.returnType {
			return ""
		}
	}
	return typ
}

func resolveReceiverType(receiver string, scope scopeInfo) string {
	if receiver == "" {
		return ""
	}
	if receiver == scope.receiverVar {
		return scope.receiverTypeKey
	}
	if typ := scope.localTypes[receiver]; typ != "" {
		return typ
	}
	parts := strings.Split(receiver, ".")
	if len(parts) == 0 {
		return ""
	}
	if parts[0] == scope.receiverVar {
		return resolveFieldChain(scope.receiverFields[parts[1]], parts[2:], scope)
	}
	if typ := scope.localTypes[parts[0]]; typ != "" {
		return resolveFieldChain(typ, parts[1:], scope)
	}
	return receiver
}

func resolveReceiverConcreteTypes(receiver string, scope scopeInfo) map[string]bool {
	if receiver == "" {
		return nil
	}
	if receiver == scope.receiverVar {
		return map[string]bool{scope.receiverTypeKey: true}
	}
	if concrete := scope.localConcreteTypes[receiver]; len(concrete) > 0 {
		return concrete
	}
	parts := strings.Split(receiver, ".")
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == scope.receiverVar {
		return resolveConcreteFieldChain(map[string]bool{scope.receiverTypeKey: true}, parts[1:], scope)
	}
	if concrete := scope.localConcreteTypes[parts[0]]; len(concrete) > 0 {
		return resolveConcreteFieldChain(concrete, parts[1:], scope)
	}
	if typ := scope.localTypes[parts[0]]; typ != "" && !isScopeInterfaceType(typ, scope) {
		return resolveConcreteFieldChain(map[string]bool{normalizeType(scope.packageName, typ): true}, parts[1:], scope)
	}
	return nil
}

func resolveConcreteFieldChain(currentTypes map[string]bool, fields []string, scope scopeInfo) map[string]bool {
	if len(fields) == 0 {
		return currentTypes
	}
	for _, field := range fields {
		nextTypes := make(map[string]bool)
		for currentType := range currentTypes {
			boundTypes := lookupConstructorFieldTypes(scope.constructorFieldTypes, scope.packageName, currentType, field)
			for typ := range boundTypes {
				nextTypes[typ] = true
			}
			if len(boundTypes) > 0 {
				continue
			}
			declaredType := lookupDeclaredFieldType(scope.structFields, scope.packageName, currentType, field)
			if declaredType != "" && !isScopeInterfaceType(declaredType, scope) {
				nextTypes[normalizeType(scope.packageName, declaredType)] = true
			}
		}
		if len(nextTypes) == 0 {
			return nil
		}
		currentTypes = nextTypes
	}
	return currentTypes
}

func lookupConstructorFieldTypes(bindings map[string]map[string]map[string]bool, packageName string, ownerType string, field string) map[string]bool {
	for _, key := range []string{typeKey(packageName, ownerType), baseType(ownerType)} {
		if fields := bindings[key]; fields != nil {
			if types := fields[field]; len(types) > 0 {
				return types
			}
		}
	}
	return nil
}

func lookupDeclaredFieldType(fields map[string]map[string]string, packageName string, ownerType string, field string) string {
	if members := lookupTypeMembers(fields, typeKey(packageName, ownerType), baseType(ownerType)); members != nil {
		return members[field]
	}
	return ""
}

func isScopeInterfaceType(typ string, scope scopeInfo) bool {
	return typ != "" && len(lookupInterfaceMethods(scope.interfaces, scope.packageName, typ)) > 0
}

func resolveFieldChain(currentType string, fields []string, scope scopeInfo) string {
	for _, field := range fields {
		typeFields := lookupTypeMembers(scope.structFields, typeKey(scope.packageName, currentType), baseType(currentType))
		if typeFields == nil {
			return currentType
		}
		nextType := typeFields[field]
		if nextType == "" {
			return currentType
		}
		currentType = nextType
	}
	return currentType
}

func lookupTypeMembers[T any](membersByType map[string]map[string]T, typeKeys ...string) map[string]T {
	for _, key := range typeKeys {
		if key == "" {
			continue
		}
		if members := membersByType[key]; members != nil {
			return members
		}
	}
	return nil
}

func lookupInterfaceMethods(interfaces map[string]map[string]bool, packageName string, typ string) map[string]bool {
	if typ == "" {
		return nil
	}
	typeName := strings.TrimPrefix(typ, "*")
	if members := interfaces[typeKey(packageName, typeName)]; members != nil {
		return members
	}
	if strings.Contains(typeName, ".") {
		return nil
	}
	return interfaces[baseType(typeName)]
}
