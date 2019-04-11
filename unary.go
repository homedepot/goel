package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
)

type unaryCompiledExpression struct {
	nopExpression
	exp      *ast.UnaryExpr
	xexp     CompiledExpression
	xtyp     reflect.Type
	operator func(interface{}) interface{}
}

func (uce *unaryCompiledExpression) ReturnType() (reflect.Type, error) {
	return uce.xtyp, nil
}

func (uce *unaryCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	expValue, err := uce.xexp.Execute(ectx)
	if err != nil {
		return nil, err
	}
	if reflect.TypeOf(expValue).AssignableTo(uce.xtyp) {
		return uce.operator(expValue), nil
	}
	return nil, errors.Errorf("%d: type mismatch.  expected %s, found %T", uce.exp.Pos(), uce.xtyp.Name(), expValue)
}

func negateBool(v interface{}) interface{} {
	return !v.(bool)
}

func negateInt(v interface{}) interface{} {
	return -v.(int)
}

func negateFloat(v interface{}) interface{} {
	return -v.(float64)
}

func evalUnaryExpr(pctx context.Context, exp *ast.UnaryExpr) CompiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	expTyp, _ := xexp.ReturnType()
	switch {
	case expTyp.AssignableTo(BoolType):
		if exp.Op == token.NOT {
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, negateBool}
		}
	case expTyp.AssignableTo(IntType):
		switch exp.Op {
		case token.SUB:
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, negateInt}
		}
	case expTyp.AssignableTo(DoubleType):
		switch exp.Op {
		case token.SUB:
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, negateFloat}
		}
	}
	return newErrorExpression(errors.Errorf("%d: unsupported unary expression: %s%s", exp.OpPos, exp.Op.String(), expTyp.Name()))
}
