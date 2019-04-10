package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

func evalTypeAssertionExpr(pctx context.Context, exp *ast.TypeAssertExpr) (ExprFunction, reflect.Type, error) {
	//xfn, xtyp, err := compile(pctx, exp.X)
	//tfn, ttyp, err := compile(pctx, exp.Type)

	return nil, nil, errors.Errorf("%d: not yet implemented", exp.Lparen)
}
