package goel

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
)

type literalCompiledExpression struct {
	nopExpression
	value interface{}
	typ   reflect.Type
}

func (lce *literalCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	return lce.value, nil
}

func (lce *literalCompiledExpression) Error() error {
	return nil
}

func (lce *literalCompiledExpression) ReturnType() (reflect.Type, error) {
	return lce.typ, nil
}

func literal(v interface{}, t reflect.Type) CompiledExpression {
	return &literalCompiledExpression{nopExpression{}, v, t}
}

func evalLiteralExpr(ctx context.Context, exp *ast.BasicLit) CompiledExpression {
	switch exp.Kind {
	case token.INT:
		i, _ := strconv.Atoi(exp.Value)
		return literal(i, IntType)
	case token.FLOAT:
		var f float64
		fmt.Sscanf(exp.Value, "%f", &f)
		return literal(f, DoubleType)
	case token.STRING, token.CHAR:
		return literal(exp.Value[1:len(exp.Value)-1], StringType)
	default:
		return newErrorExpression(errors.Errorf("%d: unknown literal type: %s with value %s", exp.Pos(), exp.Kind, exp.Value))
	}
}
