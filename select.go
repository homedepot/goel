package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
)

type selectCompiledExpression struct {
	nopExpression
	x       CompiledExpression
	name    string
	selType reflect.Type
	method  reflect.Value
	pos     token.Pos
}

func (sce *selectCompiledExpression) HasOwner() bool {
	return true
}

func (sce *selectCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	x, err := sce.x.Execute(ectx)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, errors.Errorf("%d: dereferencing a nil value", sce.pos)
	}
	xValue := reflect.ValueOf(x)
	if xValue.IsValid() && xValue.Type().Kind() == reflect.Ptr {
		xValue = xValue.Elem()
	}
	if !xValue.IsValid() {
		return nil, errors.Errorf("%d: value is invalid!", sce.pos)
	}
	if sce.method.IsValid() {
		fValue := xValue.MethodByName(sce.name)
		if fValue.IsValid() {
			return fValue.Interface(), nil
		}
	} else {
		if xValue.Kind() == reflect.Struct {
			fValue := xValue.FieldByName(sce.name)
			if fValue.IsValid() {
				return fValue.Interface(), nil
			}
		}
	}
	return nil, errors.Errorf("%d: unknown selector %s for %T", sce.pos, sce.name, x)
}

func (sce *selectCompiledExpression) Error() error {
	return nil
}

func (sce *selectCompiledExpression) ReturnType() (reflect.Type, error) {
	return sce.selType, nil
}

func evalSelectorExpr(pctx context.Context, exp *ast.SelectorExpr) compiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	xtyp, _ := xexp.ReturnType()
	if xtyp.Kind() == reflect.Ptr {
		xtyp = xtyp.Elem()
	}
	var selTyp reflect.Type
	var method reflect.Value
	if xtyp.Kind() == reflect.Struct {
		sf, ok := xtyp.FieldByName(exp.Sel.Name)
		if ok {
			selTyp = sf.Type
		}
	}
	if selTyp == nil {
		mf, ok := xtyp.MethodByName(exp.Sel.Name)
		if !ok {
			return newErrorExpression(errors.Errorf("%d: unknown selector %s for %s", exp.Sel.NamePos, exp.Sel.Name, xtyp.String()))
		}
		selTyp = mf.Type
		method = mf.Func
	}
	return &selectCompiledExpression{nopExpression{exp}, xexp, exp.Sel.Name, selTyp, method, exp.Pos()}
}
