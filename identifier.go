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

func evalIdentifierExpr(pctx context.Context, exp *ast.Ident) CompiledExpression {
	switch exp.Name {
	case "true":
		return literal(true, BoolType)
	case "false":
		return literal(false, BoolType)
	default:
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
