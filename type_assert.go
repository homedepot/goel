package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

// Map of global builtin types defined by go.
var builtinTypeIdentifiers = map[string]reflect.Type{
	"string":     StringType,
	"int":        IntType,
	"uint":       reflect.TypeOf(uint(0)),
	"uint8":      reflect.TypeOf(uint8(0)),
	"uint16":     reflect.TypeOf(uint16(0)),
	"uint32":     reflect.TypeOf(uint32(0)),
	"uint64":     reflect.TypeOf(uint64(0)),
	"int8":       reflect.TypeOf(int8(0)),
	"int16":      reflect.TypeOf(int16(0)),
	"int32":      reflect.TypeOf(int32(0)),
	"int64":      reflect.TypeOf(int64(0)),
	"float32":    reflect.TypeOf(float32(0.1)),
	"float64":    DoubleType,
	"byte":       reflect.TypeOf(byte(0)),
	"char":       reflect.TypeOf('a'),
	"complex128": reflect.TypeOf(1 + 1.0i),
	"complex64":  reflect.TypeOf(complex64(1.0 + 1.0i)),
	"bool":       BoolType,
	"error":      ErrorType,
	"uintptr":    reflect.TypeOf(uintptr(0)),
}

type typeAssertionCompiledExpression struct {
	nopExpression
	exp        *ast.TypeAssertExpr
	xexp       CompiledExpression
	assertType reflect.Type
}

func (tace *typeAssertionCompiledExpression) ReturnType() (reflect.Type, error) {
	return tace.assertType, nil
}

func (tace *typeAssertionCompiledExpression) Execute(executionContext context.Context) (interface{}, error) {
	x, err := tace.xexp.Execute(executionContext)
	if err != nil {
		return nil, err
	}
	xvalue := reflect.ValueOf(x)
	xtyp := xvalue.Type()
	if xtyp.AssignableTo(tace.assertType) {
		return xvalue.Convert(tace.assertType).Interface(), nil
	}
	return nil, errors.Errorf("%d: %s is not assignable to %s.", tace.exp.Type.Pos(), xtyp.Name(), tace.assertType.Name())
}

func evalTypeAssertionExpr(pctx context.Context, exp *ast.TypeAssertExpr) compiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	if ident, ok := exp.Type.(*ast.Ident); ok {
		assertType, ok := builtinTypeIdentifiers[ident.Name]
		if !ok {
			_assertType := pctx.Value(ident.Name)
			if _assertType == nil {
				return newErrorExpression(errors.Errorf("%d: unknown type %s", ident.NamePos, ident.Name))
			}
			assertType, ok = _assertType.(reflect.Type)
			if !ok {
				return newErrorExpression(errors.Errorf("%d: expected a reflect.Type in the parsing context for %s but found %T", ident.NamePos, ident.Name, _assertType))
			}
		}
		return &typeAssertionCompiledExpression{nopExpression{}, exp, xexp, assertType}
	}
	return newErrorExpression(errors.Errorf("%d: expression not supported for type assertion: %s", exp.Type.Pos(), exp.Type))
}
