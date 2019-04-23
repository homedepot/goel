package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
)

var (
	intr interface{}
	// StringType is a reflect.Type for strings
	StringType = reflect.TypeOf("")
	// IntType is a reflect.Type for int
	IntType = reflect.TypeOf(0)
	// DoubleType is a reflect.Type for float64
	DoubleType = reflect.TypeOf(1.0)
	// BoolType is a reflect.Type for bool
	BoolType = reflect.TypeOf(true)
	// ErrorType is a reflect.Type for error
	ErrorType = reflect.TypeOf((*error)(nil)).Elem()
	// TypeType is a reflect.Type for reflect.Type
	TypeType = reflect.TypeOf(reflect.TypeOf(IntType))
	// InterfaceType is a reflect.Type for interface{}
	InterfaceType = reflect.TypeOf(&intr).Elem()
)

// CompiledExpression represents a expression that has been compiled from a source string.
type CompiledExpression interface {
	// Execute will execute the expression with the given execution context and return the result or an error.
	Execute(executionContext context.Context) (interface{}, error)
	// ReturnType will return the type the expression is expected to return or an error if the expression did not
	// compile successfully
	ReturnType() (reflect.Type, error)
	// Error returns any building error that may have occurred.
	Error() error
}

type compiledExpression interface {
	CompiledExpression
	HasOwner() bool
	Pos() token.Pos
}

type nopExpression struct {
	exp ast.Expr
}

func (nop *nopExpression) Pos() token.Pos {
	return nop.exp.Pos()
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

func newErrorExpression(err error) compiledExpression {
	return &errExpression{nopExpression{}, err}
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

// NewCompiledExpression takes a parsing context and an expression AST and creates an executable CompiledExpression.
func NewCompiledExpression(parseContext context.Context, exp ast.Expr) CompiledExpression {
	return compile(parseContext, exp)
}

func compile(ctx context.Context, exp ast.Expr) compiledExpression {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return evalBinaryExpr(ctx, exp)
	case *ast.UnaryExpr:
		return evalUnaryExpr(ctx, exp)
	case *ast.Ident:
		return evalIdentifierExpr(ctx, exp)
	case *ast.BasicLit:
		return evalLiteralExpr(ctx, exp)
	case *ast.ParenExpr:
		return compile(ctx, exp.X)
	case *ast.CallExpr:
		return evalCallExpr(ctx, exp)
	case *ast.SelectorExpr:
		return evalSelectorExpr(ctx, exp)
	case *ast.IndexExpr:
		return evalInnerExpr(ctx, exp)
	case *ast.TypeAssertExpr:
		return evalTypeAssertionExpr(ctx, exp)
	case *ast.SliceExpr:
		return evalSliceExpr(ctx, exp)
	default:
		return newErrorExpression(errors.Errorf("%d: unknown expression type", exp.Pos()))
	}
}
