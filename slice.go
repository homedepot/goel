package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

type sliceCompiledExpression struct {
	nopExpression
	sliceExp               *ast.SliceExpr
	xexp, hexp, lexp, mexp compiledExpression
	returnType             reflect.Type
	slice3                 bool
}

func (sce *sliceCompiledExpression) ReturnType() (reflect.Type, error) {
	return sce.returnType, nil
}

func verifyIntExpression(executionContext context.Context, lexp compiledExpression, min, max int) (int, error) {
	_l, err := lexp.Execute(executionContext)
	if err != nil {
		return -1, err
	}
	l, ok := _l.(int)
	if !ok {
		return -1, errors.Errorf("%d: type mismatch expected an int but found %T", lexp.Pos(), _l)
	}
	if min <= l && l <= max {
		return l, nil
	}
	return -1, errors.Errorf("%d: index out of range: %d", lexp.Pos(), l)
}

func (sce *sliceCompiledExpression) Execute(executionContext context.Context) (interface{}, error) {
	x, err := sce.xexp.Execute(executionContext)
	if err != nil {
		return nil, err
	}
	xv := reflect.ValueOf(x)
	if xv.Kind() != reflect.Slice && xv.Kind() != reflect.String {
		return nil, errors.Errorf("%d: type mismatch expected a slice or string but found %T", sce.xexp.Pos(), x)
	}
	l, err := verifyIntExpression(executionContext, sce.lexp, 0, xv.Len()-1)
	if err != nil {
		return nil, err
	}
	h, err := verifyIntExpression(executionContext, sce.hexp, l, xv.Len())
	if err != nil {
		return nil, err
	}
	if sce.slice3 {
		if xv.Kind() != reflect.Slice {
			return nil, errors.Errorf("%d: type mismatch expected a slice but found %T", sce.xexp.Pos(), x)
		}
		m, err := verifyIntExpression(executionContext, sce.mexp, h, xv.Cap())
		if err != nil {
			return nil, err
		}
		return xv.Slice3(l, h, m).Interface(), nil
	}
	return xv.Slice(l, h).Interface(), nil
}

func newSliceCompiledExpression(sliceExp *ast.SliceExpr, returnType reflect.Type, xexp, hexp, lexp, mexp compiledExpression, slice3 bool) compiledExpression {
	return &sliceCompiledExpression{nopExpression{sliceExp}, sliceExp, xexp, hexp, lexp, mexp, returnType, slice3}
}

type lengthCompiledExpression struct {
	nopExpression
	xexp CompiledExpression
}

func (lce *lengthCompiledExpression) ReturnType() (reflect.Type, error) {
	return IntType, nil
}

func (lce *lengthCompiledExpression) Execute(executionContext context.Context) (interface{}, error) {
	s, err := lce.xexp.Execute(executionContext)
	if err != nil {
		return nil, err
	}
	vs := reflect.ValueOf(s)
	if vs.Kind() == reflect.Array || vs.Kind() == reflect.Slice || vs.Kind() == reflect.String {
		return vs.Len(), nil
	}
	return nil, errors.Errorf("%d: expected an array, slice, or string found %T", 0, s)
}

func newLengthCompiledExpression(xexp compiledExpression) compiledExpression {
	return &lengthCompiledExpression{nopExpression{}, xexp}
}

func evalSliceExpr(pctx context.Context, exp *ast.SliceExpr) compiledExpression {
	xexp := compile(pctx, exp.X)
	if xexp.Error() != nil {
		return xexp
	}
	xt, _ := xexp.ReturnType()
	if (xt.Kind() != reflect.Slice && xt.Kind() != reflect.String) || (exp.Slice3 && xt.Kind() == reflect.String) {
		if exp.Slice3 {
			return newErrorExpression(errors.Errorf("%d: type mismatch expected a slice but found %s", xexp.Pos(), xt))
		}
		return newErrorExpression(errors.Errorf("%d: type mismatch expected a slice or string but found %s", xexp.Pos(), xt))
	}
	returnType := xt
	var hexp, lexp, mexp compiledExpression
	if exp.Low != nil {
		lexp = compile(pctx, exp.Low)
	} else {
		lexp = literal(exp, 0, IntType)
	}
	if exp.High != nil {
		hexp = compile(pctx, exp.High)
	} else {
		hexp = newLengthCompiledExpression(xexp)
	}
	if exp.Slice3 {
		if xt.Kind() == reflect.String {
			return newErrorExpression(errors.Errorf("%d: type mismatch expected an array or slice but found %s", xexp.Pos(), xt.Name()))
		}
		mexp = compile(pctx, exp.Max)
	}
	return newSliceCompiledExpression(exp, returnType, xexp, hexp, lexp, mexp, exp.Slice3)
}
