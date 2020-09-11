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

func plusInt(v interface{}) interface{} {
	return +(v.(int))
}

func plusFloat(v interface{}) interface{} {
	return +(v.(float64))
}

func evalUnaryExpr(pctx context.Context, exp *ast.UnaryExpr) compiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	expTyp, err := xexp.ReturnType()
	if err != nil {
		return newErrorExpression(errors.Errorf("unexpected return type: %v", err))
	}
	switch {
	case expTyp.AssignableTo(BoolType):
		if exp.Op == token.NOT {
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, negateBool}
		}
	case expTyp.AssignableTo(IntType):
		switch exp.Op {
		case token.SUB:
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, negateInt}
		case token.ADD:
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, plusInt}
		}
	case expTyp.AssignableTo(DoubleType):
		switch exp.Op {
		case token.SUB:
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, negateFloat}
		case token.ADD:
			return &unaryCompiledExpression{nopExpression{}, exp, xexp, expTyp, plusFloat}
		}
	}
	return newErrorExpression(errors.Errorf("%d: unsupported unary expression: %s%s", exp.OpPos, exp.Op.String(), expTyp.Name()))
}
