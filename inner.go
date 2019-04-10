package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

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
	zero := reflect.Zero(etyp)
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
	return &compiledExpression{nopExpression{}, ExprFunction(func(ectx context.Context) (interface{}, error) {
		x, err := xexp.Execute(ectx)
		if err != nil {
			return nil, err
		}
		if x == nil {
			return nil, errors.Errorf("%d: expression evaluates to nil", exp.X.Pos())
		}
		if xxtyp := reflect.TypeOf(x); !xxtyp.AssignableTo(xtyp) {
			return nil, errors.Errorf("%d: expression evaluated to incorrect type. expected %s found %s", exp.X.Pos(), xtyp.Name(), xxtyp.Name())
		}
		i, err := iexp.Execute(ectx)
		if err != nil {
			return nil, err
		}
		if i == nil {
			return nil, errors.Errorf("%d: expression evaluates to nil", exp.Index.Pos())
		}
		if iityp := reflect.TypeOf(i); !iityp.AssignableTo(ktyp) {
			return nil, errors.Errorf("%d: expression evaluated to incorrect type. expected %s found %s", exp.Index.Pos(), ktyp.Name(), iityp.Name())
		}
		var vv reflect.Value
		xx := reflect.ValueOf(x)
		if xtyp.Kind() == reflect.Map {
			vv = xx.MapIndex(reflect.ValueOf(i))
			switch etyp.Kind() {
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
				return nil, errors.Errorf("%d: result of expression is not an int.", exp.Index.Pos())
			}
			if idx >= xx.Len() {
				return nil, errors.Errorf("%d: index out of bounds, len = %d index = %d", exp.Index.Pos(), xx.Len(), idx)
			}
			vv = xx.Index(idx)
		}
		v := vv.Interface()
		return v, nil
	}), etyp}
}
