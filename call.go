package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

func callFunction(ectx context.Context, exp *ast.CallExpr, fn reflect.Value, argExps []CompiledExpression, returnsError bool) (interface{}, error) {
	args := make([]reflect.Value, 0, len(argExps))
	for i, argExp := range argExps {
		v, err := argExp.Execute(ectx)
		if err != nil {
			return nil, err
		}
		argTyp, _ := argExp.ReturnType()
		if !reflect.TypeOf(v).AssignableTo(argTyp) {
			return nil, errors.Errorf("%d: type mismatch", exp.Args[i].Pos())
		}
		args = append(args, reflect.ValueOf(v))
	}
	expectedNumberOfArgs := fn.Type().NumIn()
	if expectedNumberOfArgs != len(args) {
		howMany := "to few"
		if expectedNumberOfArgs < len(args) {
			howMany = "to many"
		}
		return nil, errors.Errorf("%d: %s arguments in call.  expected %d, found %d", exp.Pos(), howMany, expectedNumberOfArgs, len(args))
	}
	results := fn.Call(args)
	var outValues []reflect.Value
	var errValue *reflect.Value
	if returnsError {
		outValues = results[0 : fn.Type().NumOut()-1]
		errValue = &results[fn.Type().NumOut()-1]
	} else {
		outValues = results
	}
	var out interface{}
	if len(outValues) == 1 {
		out = outValues[0].Interface()
	} else {
		outs := make([]interface{}, 0, len(outValues))
		for _, o := range outValues {
			outs = append(outs, o.Interface())
		}
		out = outs
	}
	var err error
	if errValue != nil && errValue.CanInterface() && !errValue.IsNil() {
		err = errValue.Interface().(error)
	}
	return out, err
}

func functionReturnsError(fnType reflect.Type) bool {
	returnsError := fnType.Out(fnType.NumOut() - 1).Implements(ErrorType)
	return returnsError
}

func functionArgs(pctx context.Context, isMember bool, fnType reflect.Type, exp *ast.CallExpr) ([]CompiledExpression, error) {
	expectedNumberofArgs := fnType.NumIn()
	argOffset := 0
	if isMember {
		expectedNumberofArgs -= 1
		argOffset = 1
	}
	if expectedNumberofArgs > len(exp.Args) {
		return nil, errors.Errorf("%d: to few parameters to function call, expected %d, found %d", exp.Pos(), expectedNumberofArgs, len(exp.Args))
	}
	if expectedNumberofArgs < len(exp.Args) {
		return nil, errors.Errorf("%d: to many parameters to function call, expected %d, found %d", exp.Pos(), expectedNumberofArgs, len(exp.Args))
	}
	argExps := make([]CompiledExpression, 0, len(exp.Args))
	for i, argExpr := range exp.Args {
		argExp := compile(pctx, argExpr)
		if argExp.Error() != nil {
			return nil, argExp.Error()
		}
		argTyp, _ := argExp.ReturnType()
		if !argTyp.AssignableTo(fnType.In(i + argOffset)) {
			return nil, errors.Errorf("%d: type mismatch in argument %d", argExpr.Pos(), i)
		}
		argExps = append(argExps, argExp)
	}
	if expectedNumberofArgs != len(argExps) {
		panic("failed to build argFns array")
	}
	return argExps, nil
}

func evalCallExpr(pctx context.Context, exp *ast.CallExpr) CompiledExpression {
	switch fnExp := exp.Fun.(type) {
	case *ast.Ident:
		_fnType := pctx.Value(fnExp.Name)
		if _fnType == nil {
			return newErrorExpression(errors.Errorf("%d: unknown function %s", fnExp.NamePos, fnExp.Name))
		}
		fnType, ok := _fnType.(reflect.Type)
		if !ok {
			return newErrorExpression(errors.Errorf("%d: not a function %s", fnExp.NamePos, fnExp.Name))
		}
		if fnType.IsVariadic() {
			return newErrorExpression(errors.Errorf("%d: variadic functions are not supported: %s", fnExp.NamePos, fnExp.Name))
		}
		returnsError := functionReturnsError(fnType)
		argExps, err := functionArgs(pctx, false, fnType, exp)
		if err != nil {
			return newErrorExpression(err)
		}
		return &compiledExpression{nopExpression{}, ExprFunction(func(ectx context.Context) (interface{}, error) {
			_fn := ectx.Value(fnExp.Name)
			if _fn == nil {
				return nil, errors.Errorf("%d: function %s not found", fnExp.NamePos, fnExp.Name)
			}
			fn, ok := _fn.(reflect.Value)
			if !ok {
				return nil, errors.Errorf("%d: %s not a function", fnExp.NamePos, fnExp.Name)
			}
			return callFunction(ectx, exp, fn, argExps, returnsError)
		}), fnType.Out(0)}
	case *ast.SelectorExpr:
		selExp := evalSelectorExpr(pctx, fnExp)
		if selExp.Error() != nil {
			return selExp
		}
		funcTyp, _ := selExp.ReturnType()
		argExps, err := functionArgs(pctx, true, funcTyp, exp)
		if err != nil {
			return newErrorExpression(err)
		}
		returnsError := functionReturnsError(funcTyp)
		return &compiledExpression{nopExpression{}, ExprFunction(func(ectx context.Context) (interface{}, error) {
			_fn, err := selExp.Execute(ectx)
			if err != nil {
				return nil, err
			}
			fn := reflect.ValueOf(_fn)
			if err != nil {
				return nil, err
			}
			return callFunction(ectx, exp, fn, argExps, returnsError)
		}), funcTyp.Out(0)}
	default:
		return newErrorExpression(errors.Errorf("%d: unknown expression type: %T", exp.Pos(), exp.Fun))
	}
}
