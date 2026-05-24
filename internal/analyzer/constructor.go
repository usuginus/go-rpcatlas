package analyzer

import (
	"go/ast"
	"go/token"
)

type constructorScope struct {
	packageName        string
	localTypes         map[string]string
	localConcreteTypes map[string]map[string]bool
	localComposites    map[string]*ast.CompositeLit
	index              *projectIndex
}

func collectConcreteReturnTypes(sources []parsedSource, index *projectIndex) {
	for pass := 0; pass < 4; pass++ {
		changed := false
		for _, source := range sources {
			for _, decl := range source.file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				info := lookupFunctionInfo(source.packageName, fn, *index)
				if info.fn == nil {
					continue
				}
				for typ := range concreteReturnTypes(fn, source.packageName, index) {
					if addTypeSet(index.concreteReturnTypes, functionInfoKey(info), normalizeType(source.packageName, typ)) {
						changed = true
					}
				}
			}
		}
		if !changed {
			return
		}
	}
}

func collectConstructorFieldTypes(sources []parsedSource, index *projectIndex) {
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			scope := newConstructorScope(fn, source.packageName, index)
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				ret, ok := node.(*ast.ReturnStmt)
				if !ok || len(ret.Results) == 0 {
					return true
				}
				lit, ok := compositeLiteralFromExpr(ret.Results[0], scope)
				if !ok {
					return true
				}
				recordConstructorComposite(lit, scope)
				return true
			})
		}
	}
}

func collectParameterConcreteTypes(sources []parsedSource, index *projectIndex) {
	for pass := 0; pass < 8; pass++ {
		changed := false
		for _, source := range sources {
			for _, decl := range source.file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				scope := newConstructorScope(fn, source.packageName, index)
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					for _, callee := range constructorFunctionCandidates(call.Fun, source.packageName, index) {
						paramNames := functionParamNames(callee.fn)
						for i, arg := range call.Args {
							if i >= len(paramNames) || paramNames[i] == "" {
								continue
							}
							for typ := range concreteTypesFromExpr(arg, scope) {
								if addParameterConcreteType(index, functionInfoKey(callee), paramNames[i], normalizeType(source.packageName, typ)) {
									changed = true
								}
							}
						}
					}
					return true
				})
			}
		}
		if !changed {
			return
		}
	}
}

func lookupFunctionInfo(packageName string, fn *ast.FuncDecl, index projectIndex) functionInfo {
	if fn.Recv == nil {
		for _, candidate := range index.functionsByName[fn.Name.Name] {
			if candidate.packageName == packageName && candidate.fn == fn {
				return candidate
			}
		}
		return functionInfo{}
	}
	receiverType := receiverName(fn)
	for _, candidate := range index.methodsByName[fn.Name.Name] {
		if candidate.packageName == packageName && candidate.receiverType == receiverType && candidate.fn == fn {
			return candidate
		}
	}
	return functionInfo{}
}

func functionInfoKey(info functionInfo) string {
	if info.receiverTypeKey != "" {
		return info.receiverTypeKey + "." + info.fn.Name.Name
	}
	return info.packageName + "." + info.fn.Name.Name
}

func concreteReturnTypes(fn *ast.FuncDecl, packageName string, index *projectIndex) map[string]bool {
	out := make(map[string]bool)
	scope := newConstructorScope(fn, packageName, index)
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		ret, ok := node.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}
		for typ := range concreteTypesFromExpr(ret.Results[0], scope) {
			out[typ] = true
		}
		return true
	})
	return out
}

func newConstructorScope(fn *ast.FuncDecl, packageName string, index *projectIndex) constructorScope {
	scope := constructorScope{
		packageName:        packageName,
		localTypes:         make(map[string]string),
		localConcreteTypes: make(map[string]map[string]bool),
		localComposites:    make(map[string]*ast.CompositeLit),
		index:              index,
	}
	collectParamTypes(fn, scope)
	collectConstructorLocals(fn, scope)
	return scope
}

func collectParamTypes(fn *ast.FuncDecl, scope constructorScope) {
	if fn.Type.Params == nil {
		return
	}
	paramConcreteTypes := parameterConcreteTypesForFunction(fn, scope.packageName, scope.index)
	for _, field := range fn.Type.Params.List {
		typ := typeString(field.Type)
		for _, name := range field.Names {
			recordLocalType(scope, name.Name, typ)
			for concreteType := range paramConcreteTypes[name.Name] {
				addType(scope.localConcreteTypes, name.Name, concreteType)
			}
		}
	}
}

func collectConstructorLocals(fn *ast.FuncDecl, scope constructorScope) {
	if fn.Body == nil {
		return
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				name, ok := lhs.(*ast.Ident)
				if !ok || name.Name == "_" || i >= len(n.Rhs) {
					continue
				}
				recordLocalExpr(scope, name.Name, n.Rhs[i])
			}
		case *ast.ValueSpec:
			for i, name := range n.Names {
				if name.Name == "_" {
					continue
				}
				if n.Type != nil {
					recordLocalType(scope, name.Name, typeString(n.Type))
				}
				if i < len(n.Values) {
					recordLocalExpr(scope, name.Name, n.Values[i])
				}
			}
		}
		return true
	})
}

func recordLocalExpr(scope constructorScope, name string, expr ast.Expr) {
	if typ := declaredTypeFromExpr(expr, scope); typ != "" {
		recordLocalType(scope, name, typ)
	}
	if concrete := concreteTypesFromExpr(expr, scope); len(concrete) > 0 {
		scope.localConcreteTypes[name] = concrete
	}
	if lit, ok := compositeLiteralFromExpr(expr, scope); ok {
		scope.localComposites[name] = lit
	}
}

func recordLocalType(scope constructorScope, name string, typ string) {
	if name == "" || typ == "" {
		return
	}
	scope.localTypes[name] = typ
	if !isInterfaceType(scope.packageName, typ, *scope.index) {
		addType(scope.localConcreteTypes, name, normalizeType(scope.packageName, typ))
	}
}

func declaredTypeFromExpr(expr ast.Expr, scope constructorScope) string {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return typeString(e.Type)
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return declaredTypeFromExpr(e.X, scope)
		}
	case *ast.CallExpr:
		return constructorCallReturnType(e, scope)
	case *ast.Ident:
		return scope.localTypes[e.Name]
	}
	return ""
}

func concreteTypesFromExpr(expr ast.Expr, scope constructorScope) map[string]bool {
	out := make(map[string]bool)
	switch e := expr.(type) {
	case *ast.CompositeLit:
		addConcreteExprType(out, scope, typeString(e.Type))
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			for typ := range concreteTypesFromExpr(e.X, scope) {
				out[typ] = true
			}
		}
	case *ast.CallExpr:
		for typ := range constructorCallConcreteReturnTypes(e, scope) {
			out[typ] = true
		}
	case *ast.Ident:
		if concrete := scope.localConcreteTypes[e.Name]; len(concrete) > 0 {
			for typ := range concrete {
				out[typ] = true
			}
			break
		}
		addConcreteExprType(out, scope, scope.localTypes[e.Name])
	}
	return out
}

func addConcreteExprType(out map[string]bool, scope constructorScope, typ string) {
	if typ == "" || isInterfaceType(scope.packageName, typ, *scope.index) {
		return
	}
	out[normalizeType(scope.packageName, typ)] = true
}

func constructorCallReturnType(call *ast.CallExpr, scope constructorScope) string {
	for _, candidate := range constructorFunctionCandidates(call.Fun, scope.packageName, scope.index) {
		return candidate.returnType
	}
	return ""
}

func constructorCallConcreteReturnTypes(call *ast.CallExpr, scope constructorScope) map[string]bool {
	out := make(map[string]bool)
	for _, candidate := range constructorFunctionCandidates(call.Fun, scope.packageName, scope.index) {
		for typ := range scope.index.concreteReturnTypes[functionInfoKey(candidate)] {
			out[typ] = true
		}
	}
	return out
}

func constructorCallTarget(fun ast.Expr) (receiver string, name string) {
	switch fn := fun.(type) {
	case *ast.Ident:
		return "", fn.Name
	case *ast.SelectorExpr:
		return typeString(fn.X), fn.Sel.Name
	}
	return "", ""
}

func matchesConstructorFunction(receiver string, currentPackage string, candidate functionInfo) bool {
	if candidate.fn == nil || candidate.receiverType != "" {
		return false
	}
	if receiver == "" {
		return candidate.packageName == currentPackage
	}
	return candidate.packageName == receiver
}

func constructorFunctionCandidates(fun ast.Expr, currentPackage string, index *projectIndex) []functionInfo {
	receiver, name := constructorCallTarget(fun)
	if name == "" {
		return nil
	}
	var candidates []functionInfo
	for _, candidate := range index.functionsByName[name] {
		if matchesConstructorFunction(receiver, currentPackage, candidate) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func functionParamNames(fn *ast.FuncDecl) []string {
	if fn == nil || fn.Type.Params == nil {
		return nil
	}
	var names []string
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			names = append(names, "")
			continue
		}
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func parameterConcreteTypesForFunction(fn *ast.FuncDecl, packageName string, index *projectIndex) map[string]map[string]bool {
	if fn == nil || index == nil {
		return nil
	}
	info := lookupFunctionInfo(packageName, fn, *index)
	if info.fn == nil {
		return nil
	}
	return index.parameterConcreteTypes[functionInfoKey(info)]
}

func addParameterConcreteType(index *projectIndex, functionKey string, paramName string, concreteType string) bool {
	if functionKey == "" || paramName == "" || concreteType == "" {
		return false
	}
	if index.parameterConcreteTypes[functionKey] == nil {
		index.parameterConcreteTypes[functionKey] = make(map[string]map[string]bool)
	}
	return addTypeSet(index.parameterConcreteTypes[functionKey], paramName, concreteType)
}

func compositeLiteralFromExpr(expr ast.Expr, scope constructorScope) (*ast.CompositeLit, bool) {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return e, true
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return compositeLiteralFromExpr(e.X, scope)
		}
	case *ast.Ident:
		if lit := scope.localComposites[e.Name]; lit != nil {
			return lit, true
		}
	}
	return nil, false
}

func recordConstructorComposite(lit *ast.CompositeLit, scope constructorScope) {
	ownerType := typeString(lit.Type)
	if ownerType == "" {
		return
	}
	fieldOrder := lookupStructFieldOrder(scope.index.structFieldOrder, scope.packageName, ownerType)
	for i, elt := range lit.Elts {
		fieldName, value := constructorFieldValue(i, elt, fieldOrder)
		if fieldName == "" || value == nil {
			continue
		}
		for concreteType := range concreteTypesFromExpr(value, scope) {
			addConstructorFieldType(scope.index, scope.packageName, ownerType, fieldName, concreteType)
		}
	}
}

func constructorFieldValue(index int, expr ast.Expr, fieldOrder []string) (string, ast.Expr) {
	if kv, ok := expr.(*ast.KeyValueExpr); ok {
		name, ok := kv.Key.(*ast.Ident)
		if !ok {
			return "", nil
		}
		return name.Name, kv.Value
	}
	if index >= len(fieldOrder) {
		return "", nil
	}
	return fieldOrder[index], expr
}

func lookupStructFieldOrder(orders map[string][]string, packageName string, ownerType string) []string {
	for _, key := range []string{typeKey(packageName, ownerType), baseType(ownerType)} {
		if fields := orders[key]; fields != nil {
			return fields
		}
	}
	return nil
}

func addConstructorFieldType(index *projectIndex, packageName string, ownerType string, fieldName string, concreteType string) {
	if fieldName == "" || concreteType == "" {
		return
	}
	ownerKey := typeKey(packageName, ownerType)
	addNestedTypeSet(index.constructorFieldTypes, ownerKey, fieldName, normalizeType(packageName, concreteType))
	if base := baseType(ownerType); base != ownerKey {
		if _, exists := index.constructorFieldTypes[base]; !exists {
			addNestedTypeSet(index.constructorFieldTypes, base, fieldName, normalizeType(packageName, concreteType))
		}
	}
}

func addNestedTypeSet(out map[string]map[string]map[string]bool, ownerType string, fieldName string, concreteType string) {
	if ownerType == "" || fieldName == "" || concreteType == "" {
		return
	}
	if out[ownerType] == nil {
		out[ownerType] = make(map[string]map[string]bool)
	}
	addType(out[ownerType], fieldName, concreteType)
}

func addTypeSet(out map[string]map[string]bool, key string, typ string) bool {
	if key == "" || typ == "" {
		return false
	}
	if out[key] == nil {
		out[key] = make(map[string]bool)
	}
	if out[key][typ] {
		return false
	}
	out[key][typ] = true
	return true
}

func addType(out map[string]map[string]bool, key string, typ string) {
	_ = addTypeSet(out, key, typ)
}

func normalizeType(packageName string, typ string) string {
	return typeKey(packageName, typ)
}

func isInterfaceType(packageName string, typ string, index projectIndex) bool {
	if typ == "" {
		return false
	}
	return len(lookupInterfaceMethods(index.interfaces, packageName, typ)) > 0
}
