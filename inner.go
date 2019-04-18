package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

type innerCompiledExpression struct {
	nopExpression
	exp  *ast.IndexExpr
	xexp CompiledExpression
	iexp CompiledExpression
	xtyp reflect.Type
	ktyp reflect.Type
	etyp reflect.Type
}

func (ice *innerCompiledExpression) ReturnType() (reflect.Type, error) {
	return ice.etyp, nil
}

func (ice *innerCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	x, err := ice.xexp.Execute(ectx)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, errors.Errorf("%d: expression evaluates to nil", ice.exp.X.Pos())
	}
	if xxtyp := reflect.TypeOf(x); !xxtyp.AssignableTo(ice.xtyp) {
		return nil, errors.Errorf("%d: expression evaluated to incorrect type. expected %s found %s", ice.exp.X.Pos(), ice.xtyp.Name(), xxtyp.Name())
	}
	i, err := ice.iexp.Execute(ectx)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, errors.Errorf("%d: expression evaluates to nil", ice.exp.Index.Pos())
	}
	if iityp := reflect.TypeOf(i); !iityp.AssignableTo(ice.ktyp) {
		return nil, errors.Errorf("%d: expression evaluated to incorrect type. expected %s found %s", ice.exp.Index.Pos(), ice.ktyp.Name(), iityp.Name())
	}
	var vv reflect.Value
	xx := reflect.ValueOf(x)
	if ice.xtyp.Kind() == reflect.Map {
		vv = xx.MapIndex(reflect.ValueOf(i))
		zero := reflect.Zero(ice.etyp)
		switch ice.etyp.Kind() {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
			if !vv.IsValid() {
				return zero.Interface(), nil
			}
			if vv.IsNil() {
				return zero.Interface(), nil
			}
		default:
			if !vv.IsValid() {
				return zero.Interface(), nil
			}
		}
	} else {
		idx, ok := i.(int)
		if !ok {
			return nil, errors.Errorf("%d: result of expression is not an int.", ice.exp.Index.Pos())
		}
		if idx >= xx.Len() {
			return nil, errors.Errorf("%d: index out of bounds, len = %d index = %d", ice.exp.Index.Pos(), xx.Len(), idx)
		}
		vv = xx.Index(idx)
	}
	v := vv.Interface()
	return v, nil
}

func evalInnerExpr(pctx context.Context, exp *ast.IndexExpr) CompiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	xtyp, _ := xexp.ReturnType()
	isPtr := xtyp.Kind() == reflect.Ptr
	if isPtr {
		xtyp = xtyp.Elem()
	}
	iexp := compile(pctx, exp.Index)
	if iexp.Error() != nil {
		return iexp
	}
	ityp, _ := iexp.ReturnType()
	etyp := xtyp.Elem()

	var ktyp reflect.Type
	if xtyp.Kind() == reflect.Map {
		ktyp = xtyp.Key()
	} else if xtyp.Kind() == reflect.Array || xtyp.Kind() == reflect.Slice || xtyp.Kind() == reflect.String {
		ktyp = IntType
	} else {
		return newErrorExpression(errors.Errorf("%d: not an index type %s", exp.X.Pos(), xtyp.Name()))
	}
	if !ityp.AssignableTo(ktyp) {
		return newErrorExpression(errors.Errorf("%d: incorrect index type. expected %s, found %s", exp.Index.Pos(), ktyp.Name(), ityp.Name()))
	}
	return &innerCompiledExpression{nopExpression{exp}, exp, xexp, iexp, xtyp, ktyp, etyp}

}
