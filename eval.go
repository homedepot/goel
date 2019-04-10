package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

var (
	StringType = reflect.TypeOf("")
	IntType    = reflect.TypeOf(0)
	DoubleType = reflect.TypeOf(1.0)
	BoolType   = reflect.TypeOf(true)
	ErrorType  = reflect.TypeOf((*error)(nil)).Elem()
)

type CompiledExpression interface {
	Execute(executionContext context.Context) (interface{}, error)
	ReturnType() (reflect.Type, error)
	Error() error
	HasOwner() bool
}

type ExprFunction func(context.Context) (interface{}, error)

type nopExpression struct {
}

func (nop *nopExpression) Execute() (interface{}, error) {
	return nil, nil
}

func (nop *nopExpression) ReturnType() (reflect.Type, error) {
	return nil, nil
}

func (nop *nopExpression) Error() error {
	return nil
}

func (nop *nopExpression) HasOwner() bool {
	return false
}

type errExpression struct {
	nopExpression
	err error
}

func newErrorExpression(err error) CompiledExpression {
	return &errExpression{nopExpression{},err}
}

func (ee *errExpression) Execute(executionContext context.Context) (interface{}, error) {
	return nil, ee.err
}

func (ee *errExpression) ReturnType() (reflect.Type, error) {
	return nil, ee.err
}

func (ee *errExpression) Error() error {
	return ee.err
}

type compiledExpression struct {
	nopExpression
	function   ExprFunction
	returnType reflect.Type
}

func (ce *compiledExpression) Execute(executionContext context.Context) (interface{}, error) {
	return ce.function(executionContext)
}

func (ce *compiledExpression) ReturnType() (reflect.Type, error) {
	return ce.returnType, nil
}

func NewCompiledExpression(parseContext context.Context, exp ast.Expr) CompiledExpression {
	return compile(parseContext, exp)
}

func compile(ctx context.Context, exp ast.Expr) CompiledExpression {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return evalBinaryExpr(ctx, exp)
	case *ast.UnaryExpr:
		return evalUnaryExpr(ctx, exp)
	case *ast.Ident:
		return evalIdentifierExpr(ctx, exp)
	case *ast.BasicLit:
		return evalLiteralExpr(ctx,exp)
	case *ast.ParenExpr:
		return compile(ctx, exp.X)
	case *ast.CallExpr:
		return evalCallExpr(ctx, exp)
	case *ast.SelectorExpr:
		return evalSelectorExpr(ctx, exp)
	case *ast.IndexExpr:
		return evalInnerExpr(ctx, exp)
	default:
		return newErrorExpression(errors.Errorf("%d: unknown expression type", exp.Pos()))
	}
}


