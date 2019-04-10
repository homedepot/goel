package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
)

func evalUnaryExpr(pctx context.Context, exp *ast.UnaryExpr) CompiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	expTyp, _ := xexp.ReturnType()
	if expTyp.AssignableTo(BoolType) {
		if exp.Op == token.NOT {
			return &compiledExpression{nopExpression{}, ExprFunction(func(ectx context.Context) (interface{}, error) {
				expValue, err := xexp.Execute(ectx)
				if err != nil {
					return nil, err
				}
				b, ok := expValue.(bool)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch.  %s requires a boolean expression, found %T", exp.X.Pos(), exp.Op.String(), expValue)
				}
				return !b, nil
			}), BoolType}
		}
	} else if expTyp.AssignableTo(IntType) || expTyp.AssignableTo(DoubleType) {
		if exp.Op == token.SUB {
			return &compiledExpression{nopExpression{}, ExprFunction(func(ectx context.Context) (interface{}, error) {
				expValue, err := xexp.Execute(ectx)
				if err != nil {
					return nil, err
				}
				switch v := expValue.(type) {
				case int:
					return -v, nil
				case float64:
					return -v, nil
				default:
					return nil, errors.Errorf("%d: type mismatch.  %s requires a number expression, found %T", exp.X.Pos(), exp.Op.String(), expValue)
				}
			}), expTyp}
		}
	}
	return newErrorExpression(errors.Errorf("%d: unsupported unary operator: %s", exp.OpPos, exp.Op.String()))
}

