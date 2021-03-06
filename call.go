package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"reflect"
)

type callCompiledExpression struct {
	nopExpression
	exp          *ast.CallExpr
	fnExp        compiledExpression
	args         []compiledExpression
	returnsError bool
	returnType   reflect.Type
}

func (cce *callCompiledExpression) ReturnType() (reflect.Type, error) {
	return cce.returnType, nil
}

func (cce *callCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	_fn, err := cce.fnExp.Execute(ectx)
	if err != nil {
		return nil, err
	}
	fn := reflect.ValueOf(_fn)
	if _fn == nil {
		return nil, errors.Errorf("%d: function expression returned nil", cce.exp.Fun.Pos())
	}
	if !fn.IsValid() || fn.Kind() != reflect.Func || fn.IsNil() {
		return nil, errors.Errorf("%d: not a function", cce.exp.Pos())
	}
	args, err := collectArgumentValues(ectx, fn, cce.args)
	if err != nil {
		return nil, err
	}
	results := fn.Call(args)
	var outValues []reflect.Value
	var errValue *reflect.Value
	if cce.returnsError {
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
	if errValue != nil && errValue.CanInterface() && !errValue.IsNil() {
		err = errValue.Interface().(error)
	}
	return out, err
}

func collectArgumentValues(ectx context.Context, fn reflect.Value, argExps []compiledExpression) ([]reflect.Value, error) {
	args := make([]reflect.Value, 0, len(argExps))
	for _, argExp := range argExps {
		v, err := argExp.Execute(ectx)
		if err != nil {
			return nil, err
		}
		argTyp, _ := argExp.ReturnType()
		if !reflect.TypeOf(v).AssignableTo(argTyp) {
			return nil, errors.Errorf("%d: type mismatch", argExp.Pos())
		}
		args = append(args, reflect.ValueOf(v))
	}
	expectedNumberOfArgs := fn.Type().NumIn()
	if expectedNumberOfArgs != len(args) {
		howMany := "too few"
		if expectedNumberOfArgs < len(args) {
			howMany = "too many"
		}
		return nil, errors.Errorf("%d: %s arguments in call.  expected %d, found %d", argExps[0].Pos(), howMany, expectedNumberOfArgs, len(args))
	}
	return args, nil
}

func functionReturnsError(fnType reflect.Type) bool {
	returnsError := fnType.Out(fnType.NumOut() - 1).Implements(ErrorType)
	return returnsError
}

func functionArgs(pctx context.Context, isMember bool, fnType reflect.Type, exp *ast.CallExpr) ([]compiledExpression, error) {
	expectedNumberofArgs := fnType.NumIn()
	argOffset := 0
	if isMember {
		expectedNumberofArgs--
		argOffset = 1
	}
	if expectedNumberofArgs > len(exp.Args) {
		return nil, errors.Errorf("%d: too few parameters to function call, expected %d, found %d", exp.Rparen, expectedNumberofArgs, len(exp.Args))
	}
	if expectedNumberofArgs < len(exp.Args) {
		return nil, errors.Errorf("%d: too many parameters to function call, expected %d, found %d", exp.Rparen, expectedNumberofArgs, len(exp.Args))
	}
	argExps := make([]compiledExpression, 0, len(exp.Args))
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

func evalCallExpr(pctx context.Context, exp *ast.CallExpr) compiledExpression {
	fnExp := compile(pctx, exp.Fun)
	if fnExp.Error() != nil {
		return fnExp
	}
	fnType, _ := fnExp.ReturnType()
	if fnType.Kind() != reflect.Func {
		if fnType.AssignableTo(TypeType) {
			return newErrorExpression(errors.Errorf("%d: type conversion not supported", exp.Lparen))
		}
		return newErrorExpression(errors.Errorf("%d: not a function", exp.Lparen))
	}
	if fnType.IsVariadic() {
		return newErrorExpression(errors.Errorf("%d: variadic functions are not supported.", exp.Lparen))
	}
	returnsError := functionReturnsError(fnType)
	argExps, err := functionArgs(pctx, fnExp.HasOwner(), fnType, exp)
	if err != nil {
		return newErrorExpression(err)
	}
	var returnType reflect.Type
	var thresholdArgs = 1
	if returnsError {
		thresholdArgs = 2
	}
	if fnType.NumOut() > thresholdArgs {
		returnType = reflect.TypeOf([]reflect.Value{})
	} else {
		returnType = fnType.Out(0)
	}
	return &callCompiledExpression{nopExpression{exp}, exp, fnExp, argExps, returnsError, returnType}
}
