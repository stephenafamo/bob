// Package bobcheck checks for discarded Bob query results.
package bobcheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// New returns an analyzer for direct calls to Apply methods that return a
// single bob.Query. Apply methods that modify a query in place return nothing.
func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "bobcheck",
		Doc:      "report discarded results of query-returning Apply methods",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      run,
	}
}

func run(pass *analysis.Pass) (any, error) {
	if query := queryInterface(pass.Pkg); query != nil {
		checkCalls(pass, query)
	}
	return nil, nil //nolint:nilnil // This analyzer reports diagnostics and has no result.
}

func checkCalls(pass *analysis.Pass, query *types.Interface) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	for n := range insp.PreorderSeq((*ast.ExprStmt)(nil), (*ast.GoStmt)(nil), (*ast.DeferStmt)(nil)) {
		var call *ast.CallExpr
		switch s := n.(type) {
		case *ast.ExprStmt:
			call, _ = ast.Unparen(s.X).(*ast.CallExpr)
		case *ast.GoStmt:
			call = s.Call
		case *ast.DeferStmt:
			call = s.Call
		}
		if call == nil {
			continue
		}
		sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Apply" {
			continue
		}
		method := pass.TypesInfo.Selections[sel]
		if method == nil || method.Kind() == types.FieldVal {
			continue // Package function or function-valued field.
		}
		// Use the signature at the call site so generic results are instantiated.
		results := method.Type().(*types.Signature).Results()
		if results.Len() == 1 && types.Implements(results.At(0).Type(), query) {
			pass.ReportRangef(sel.Sel, "result of Apply is discarded; use the returned query")
		}
	}
}

func queryInterface(pkg *types.Package) *types.Interface {
	// A caller can use generated queries without importing Bob directly.
	const bob = "github.com/stephenafamo/bob"
	deps := typeutil.Dependencies(pkg)
	lookup := func(path, name string) types.Type {
		for _, dep := range deps {
			if dep.Path() == path {
				if obj := dep.Scope().Lookup(name); obj != nil {
					return obj.Type()
				}
				break
			}
		}
		return nil
	}
	if t := lookup(bob, "Query"); t != nil {
		iface, _ := t.Underlying().(*types.Interface)
		return iface
	}

	// go vet's shallow export data may omit bob.Query itself for an indirect
	// import. Its method signatures' types, however, must be present if a
	// returned type implements the interface. Use their actual type identities.
	kind, dialect := lookup(bob, "QueryType"), lookup(bob, "Dialect")
	ctx, writer := lookup("context", "Context"), lookup("io", "StringWriter")
	if kind == nil || dialect == nil || ctx == nil || writer == nil {
		return nil
	}
	tuple := func(ts ...types.Type) *types.Tuple {
		vars := make([]*types.Var, len(ts))
		for i, t := range ts {
			vars[i] = types.NewVar(token.NoPos, nil, "", t)
		}
		return types.NewTuple(vars...)
	}
	method := func(name string, params, results *types.Tuple) *types.Func {
		return types.NewFunc(token.NoPos, nil, name, types.NewSignatureType(nil, nil, nil, params, results, false))
	}
	index := types.Typ[types.Int]
	argsErr := tuple(types.NewSlice(types.Universe.Lookup("any").Type()), types.Universe.Lookup("error").Type())
	return types.NewInterfaceType([]*types.Func{
		method("Type", nil, tuple(kind)),
		method("WriteQuery", tuple(ctx, writer, index), argsErr),
		method("WriteSQL", tuple(ctx, writer, dialect, index), argsErr),
	}, nil).Complete()
}
