package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

type typeAssertionCompiledExpression struct {
	nopExpression
	exp  *ast.TypeAssertExpr
	xexp CompiledExpression
	texp CompiledExpression
}

func (tace *typeAssertionCompiledExpression) ReturnType() (reflect.Type, error) {
	return InterfaceType, nil
}

func (tace *typeAssertionCompiledExpression) Execute(executionContext context.Context) (interface{}, error) {
	x, err := tace.xexp.Execute(executionContext)
	if err != nil {
		return nil, err
	}
	t, err := tace.texp.Execute(executionContext)
	if err != nil {
		return nil, err
	}
	if typ, ok := t.(reflect.Type); ok {
		xvalue := reflect.ValueOf(x)
		xtyp := xvalue.Type()
		if xtyp.AssignableTo(typ) {
			return xvalue.Convert(typ).Interface(), nil
		}
		return nil, errors.Errorf("%d: %s is not assignable to %s.", tace.exp.Type.Pos(), xtyp.Name(), typ.Name())
	} else {
		return nil, errors.Errorf("%d: type expression is not a type.", tace.exp.Type.Pos())
	}
}

func evalTypeAssertionExpr(pctx context.Context, exp *ast.TypeAssertExpr) CompiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	texp := compile(pctx, exp.Type)
	if texp.Error() != nil {
		return texp
	}
	if ttyp, _ := texp.ReturnType(); !ttyp.AssignableTo(reflect.TypeOf(IntType)) {
		return newErrorExpression(errors.Errorf("%d: expected a reflect.Type but found %s", exp.Lparen, ttyp.Name()))
	}
	return &typeAssertionCompiledExpression{nopExpression{}, exp, xexp, texp}
}
