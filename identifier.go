package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

type lookUpIdentifierValueCompiledExpression struct {
	nopExpression
	exp *ast.Ident
	typ reflect.Type
}

func (luivce *lookUpIdentifierValueCompiledExpression) ReturnType() (reflect.Type, error) {
	return luivce.typ, nil
}

func (luivce *lookUpIdentifierValueCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	_v := ectx.Value(luivce.exp.Name)
	if _v == nil {
		return nil, errors.Errorf("%d: undefined identifier: %s", luivce.exp.NamePos, luivce.exp.Name)
	}
	v, ok := _v.(reflect.Value)
	if ok && v.IsValid() && luivce.typ.AssignableTo(v.Type()) {
		return v.Interface(), nil
	} else {
		return nil, errors.Errorf("%d: value type mismatch: %s with type %v", luivce.exp.NamePos, luivce.exp.Name, v)
	}
}

// Map of global types and constants that are defined in go.
var literalIdentifiers = map[string]interface{}{
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
	"true":       true,
	"false":      false,
}

func evalIdentifierExpr(pctx context.Context, exp *ast.Ident) CompiledExpression {
	if v, ok := literalIdentifiers[exp.Name]; ok {
		return literal(v, reflect.TypeOf(v))
	} else {
		_vtype := pctx.Value(exp.Name)
		if _vtype != nil {
			vtype, ok := _vtype.(reflect.Type)
			if !ok {
				return newErrorExpression(errors.Errorf("%d: identifier type is not a reflect.Type: %s(%T)", exp.NamePos, exp.Name, vtype))
			}
			return &lookUpIdentifierValueCompiledExpression{nopExpression{}, exp, vtype}
		} else {
			return newErrorExpression(errors.Errorf("%d: unknown identifier: %s", exp.NamePos, exp.Name))
		}
	}
}
