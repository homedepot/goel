package goel

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
)

var (
	StringType = reflect.TypeOf("")
	IntType    = reflect.TypeOf(0)
	DoubleType = reflect.TypeOf(1.0)
	BoolType   = reflect.TypeOf(true)
	ErrorType  = reflect.TypeOf((*error)(nil)).Elem()
)

type ExprFunction func(context.Context) (interface{}, error)

func literal(v interface{}) ExprFunction {
	return ExprFunction(func(ectx context.Context) (interface{}, error) {
		return v, nil
	})
}

func Compile(ctx context.Context, exp ast.Expr) (ExprFunction, reflect.Type, error) {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return evalBinaryExpr(ctx, exp)
	case *ast.UnaryExpr:
		return evalUnaryExpr(ctx, exp)
	case *ast.Ident:
		switch exp.Name {
		case "true":
			return literal(true), BoolType, nil
		case "false":
			return literal(false), BoolType, nil
		default:
			_vtype := ctx.Value(exp.Name)
			if _vtype != nil {
				vtype, ok := _vtype.(reflect.Type)
				if !ok {
					return nil, nil, errors.Errorf("%d: identifier type is not a reflect.Type: %s(%T)", exp.NamePos, exp.Name, vtype)
				}
				return ExprFunction(func(ectx context.Context) (interface{}, error) {
					_v := ectx.Value(exp.Name)
					if _v == nil {
						return nil, errors.Errorf("%d: undefined identifier: %s", exp.NamePos, exp.Name)
					}
					v, ok := _v.(reflect.Value)
					if ok && v.IsValid() && vtype.AssignableTo(v.Type()) {
						return v.Interface(), nil
					} else {
						return nil, errors.Errorf("%d: value type mismatch: %s with type %v", exp.NamePos, exp.Name, v)
					}
				}), vtype, nil
			} else {
				return nil, nil, errors.Errorf("%d: unknown identifier: %s", exp.NamePos, exp.Name)
			}
		}
	case *ast.BasicLit:
		switch exp.Kind {
		case token.INT:
			i, _ := strconv.Atoi(exp.Value)
			return literal(i), IntType, nil
		case token.FLOAT:
			var f float64
			fmt.Sscanf(exp.Value, "%f", &f)
			return literal(f), DoubleType, nil
		case token.STRING, token.CHAR:
			return literal(exp.Value[1 : len(exp.Value)-1]), StringType, nil
		default:
			return nil, nil, errors.Errorf("%d: unknown literal type: %s with value %s", exp.Pos(), exp.Kind, exp.Value)
		}
	case *ast.ParenExpr:
		return Compile(ctx, exp.X)
	case *ast.CallExpr:
		return evalCallExpr(ctx, exp)
	case *ast.SelectorExpr:
		return evalSelectorExpr(ctx, exp)
	case *ast.IndexExpr:
		return evalInnerExpr(ctx,exp)
	default:
		return nil, nil, errors.Errorf("%d: unknown expression type", exp.Pos())
	}
}

func evalSelectorExpr(pctx context.Context, exp *ast.SelectorExpr) (ExprFunction, reflect.Type, error) {
	xfn, xtyp, err := Compile(pctx, exp.X)
	if err != nil {
		return nil, nil, err
	}
	if xtyp.Kind() == reflect.Ptr {
		xtyp = xtyp.Elem()
	}
	var selTyp *reflect.Type
	var method reflect.Value
	if xtyp.Kind() == reflect.Struct {
		sf, ok := xtyp.FieldByName(exp.Sel.Name)
		if ok {
			selTyp = &sf.Type
		}
	}
	if selTyp == nil {
		mf, ok := xtyp.MethodByName(exp.Sel.Name)
		if !ok {
			return nil, nil, errors.Errorf("%d: unknown selector %s for %s", exp.Sel.NamePos, exp.Sel.Name, xtyp.String())
		}
		selTyp = &mf.Type
		method = mf.Func
	}
	return ExprFunction(func(ectx context.Context) (interface{}, error) {
		x, err := xfn(ectx)
		if err != nil {
			return nil, err
		}
		if x == nil {
			return nil, errors.Errorf("%d: dereferencing a nil value", exp.Pos())
		}
		xValue := reflect.ValueOf(x)
		if xValue.IsValid() && xValue.Type().Kind() == reflect.Ptr {
			xValue = xValue.Elem()
		}
		if !xValue.IsValid() {
			return nil, errors.Errorf("%d: value is invalid!", exp.Pos())
		}
		if method.IsValid() {
			fValue := xValue.MethodByName(exp.Sel.Name)
			if fValue.IsValid() {
				return fValue.Interface(), nil
			}
		} else {
			if xValue.Kind() == reflect.Struct {
				fValue := xValue.FieldByName(exp.Sel.Name)
				if fValue.IsValid() {
					return fValue.Interface(), nil
				}
			}
		}
		return nil, errors.Errorf("%d: unknown selector %s for %T", exp.Sel.NamePos, exp.Sel.Name, x)
	}), *selTyp, nil
}

func callFunction(ectx context.Context, exp *ast.CallExpr, fn reflect.Value, argFns []ExprFunction, argTyps []reflect.Type, returnsError bool) (interface{}, error) {
	args := make([]reflect.Value, 0, len(argFns))
	for i, argFn := range argFns {
		v, err := argFn(ectx)
		if err != nil {
			return nil, err
		}
		if !reflect.TypeOf(v).AssignableTo(argTyps[i]) {
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

func functionArgs(pctx context.Context, isMember bool, fnType reflect.Type, exp *ast.CallExpr) ([]ExprFunction, []reflect.Type, error) {
	expectedNumberofArgs := fnType.NumIn()
	argOffset := 0
	if isMember {
		expectedNumberofArgs -= 1
		argOffset = 1
	}
	if expectedNumberofArgs > len(exp.Args) {
		return nil, nil, errors.Errorf("%d: to few parameters to function call, expected %d, found %d", exp.Pos(), expectedNumberofArgs, len(exp.Args))
	}
	if expectedNumberofArgs < len(exp.Args) {
		return nil, nil, errors.Errorf("%d: to many parameters to function call, expected %d, found %d", exp.Pos(), expectedNumberofArgs, len(exp.Args))
	}
	argFns := make([]ExprFunction, 0, len(exp.Args))
	argTyps := make([]reflect.Type, 0, len(exp.Args))
	for i, argExpr := range exp.Args {
		argFn, argTyp, err := Compile(pctx, argExpr)
		if err != nil {
			return nil, nil, err
		}
		if !argTyp.AssignableTo(fnType.In(i + argOffset)) {
			return nil, nil, errors.Errorf("%d: type mismatch in argument %d", argExpr.Pos(), i)
		}
		argFns = append(argFns, argFn)
		argTyps = append(argTyps, argTyp)
	}
	if expectedNumberofArgs != len(argFns) {
		panic("failed to build argFns array")
	}
	return argFns, argTyps, nil
}

func evalCallExpr(pctx context.Context, exp *ast.CallExpr) (ExprFunction, reflect.Type, error) {
	switch fnExp := exp.Fun.(type) {
	case *ast.Ident:
		_fnType := pctx.Value(fnExp.Name)
		if _fnType == nil {
			return nil, nil, errors.Errorf("%d: unknown function %s", fnExp.NamePos, fnExp.Name)
		}
		fnType, ok := _fnType.(reflect.Type)
		if !ok {
			return nil, nil, errors.Errorf("%d: not a function %s", fnExp.NamePos, fnExp.Name)
		}
		if fnType.IsVariadic() {
			return nil, nil, errors.Errorf("%d: variadic functions are not supported: %s", fnExp.NamePos, fnExp.Name)
		}
		returnsError := functionReturnsError(fnType)
		argFns, argTyps, err := functionArgs(pctx, false, fnType, exp)
		if err != nil {
			return nil, nil, err
		}
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			_fn := ectx.Value(fnExp.Name)
			if _fn == nil {
				return nil, errors.Errorf("%d: function %s not found", fnExp.NamePos, fnExp.Name)
			}
			fn, ok := _fn.(reflect.Value)
			if !ok {
				return nil, errors.Errorf("%d: %s not a function", fnExp.NamePos, fnExp.Name)
			}
			return callFunction(ectx, exp, fn, argFns, argTyps, returnsError)
		}), fnType.Out(0), nil
	case *ast.SelectorExpr:
		fnFn, fnType, err := evalSelectorExpr(pctx, fnExp)
		if err != nil {
			return nil, nil, err
		}
		argFns, argTyps, err := functionArgs(pctx, true, fnType, exp)
		if err != nil {
			return nil, nil, err
		}
		returnsError := functionReturnsError(fnType)
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			_fn, err := fnFn(ectx)
			if err != nil {
				return nil, err
			}
			fn := reflect.ValueOf(_fn)
			if err != nil {
				return nil, err
			}
			return callFunction(ectx, exp, fn, argFns, argTyps, returnsError)
		}), fnType.Out(0), nil
	default:
		return nil, nil, errors.Errorf("%d: unknown expression type: %T", exp.Pos(), exp.Fun)
	}
}

func evalUnaryExpr(pctx context.Context, exp *ast.UnaryExpr) (ExprFunction, reflect.Type, error) {
	exprFn, expTyp, err := Compile(pctx, exp.X)
	if err != nil {
		return nil, nil, err
	}
	if expTyp.AssignableTo(BoolType) {
		if exp.Op == token.NOT {
			return ExprFunction(func(ectx context.Context) (interface{}, error) {
				expValue, err := exprFn(ectx)
				if err != nil {
					return nil, err
				}
				b, ok := expValue.(bool)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch.  %s requires a boolean expression, found %T", exp.X.Pos(), exp.Op.String(), expValue)
				}
				return !b, nil
			}), BoolType, nil
		}
	} else if expTyp.AssignableTo(IntType) || expTyp.AssignableTo(DoubleType) {
		if exp.Op == token.SUB {
			return ExprFunction(func(ectx context.Context) (interface{}, error) {
				expValue, err := exprFn(ectx)
				if err != nil {
					return nil, err
				}
				switch v := expValue.(type) {
				case int:
					return -v, nil
				case float64:
					return -v, nil
				default:
					return nil, errors.Errorf("%d: type mismatch.  %s requires a number expression, found %T", exp.X.Pos(), exp.Op.String(), expValue)
				}
			}), expTyp, nil
		}
	}
	return nil, nil, errors.Errorf("%d: unsupported unary operator: %s", exp.OpPos, exp.Op.String())
}

func evalBinaryExpr(pctx context.Context, exp *ast.BinaryExpr) (ExprFunction, reflect.Type, error) {
	left, lt, err := Compile(pctx, exp.X)
	if err != nil {
		return nil, nil, err
	}
	right, rt, err := Compile(pctx, exp.Y)
	if err != nil {
		return nil, nil, err
	}
	if !lt.AssignableTo(rt) {
		return nil, nil, errors.Errorf("%d: type mismatch in binary expression", exp.OpPos)
	}
	if !(lt.AssignableTo(StringType) || lt.AssignableTo(IntType) || lt.AssignableTo(DoubleType) || lt.AssignableTo(BoolType)) {
		return nil, nil, errors.Errorf("%d: unsupported binary expression type: %s", exp.OpPos, lt.String())
	}

	switch exp.Op {
	case token.ADD:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			switch lv := l.(type) {
			case string:
				rv, ok := r.(string)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				s := lv + rv
				return s, nil
			case int:
				rv, ok := r.(int)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv + rv, nil
			case float64:
				rv, ok := r.(float64)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv + rv, nil
			default:
				return nil, errors.Errorf("%d: unsupported type %T", exp.X.Pos(), l)
			}
		}), lt, nil
	case token.SUB:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			switch lv := l.(type) {
			case int:
				rv, ok := r.(int)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv - rv, nil
			case float64:
				rv, ok := r.(float64)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv - rv, nil
			default:
				return nil, errors.Errorf("%d: unsupported type %T", exp.X.Pos(), l)
			}
		}), lt, nil
	case token.MUL:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			switch lv := l.(type) {
			case int:
				rv, ok := r.(int)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv * rv, nil
			case float64:
				rv, ok := r.(float64)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv * rv, nil
			default:
				return nil, errors.Errorf("%d: unsupported type %T", exp.X.Pos(), l)
			}
		}), lt, nil
	case token.QUO:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			switch lv := l.(type) {
			case int:
				rv, ok := r.(int)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv / rv, nil
			case float64:
				rv, ok := r.(float64)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv / rv, nil
			default:
				return nil, errors.Errorf("%d: unsupported type %T", exp.X.Pos(), l)
			}
		}), lt, nil
	case token.LAND:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			switch lv := l.(type) {
			case bool:
				rv, ok := r.(bool)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv && rv, nil
			default:
				return nil, errors.Errorf("%d: unsupported type %T", exp.X.Pos(), l)
			}
		}), BoolType, nil
	case token.LOR:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			switch lv := l.(type) {
			case bool:
				rv, ok := r.(bool)
				if !ok {
					return nil, errors.Errorf("%d: type mismatch expected string but found %T", exp.Y.Pos(), r)
				}
				return lv || rv, nil
			default:
				return nil, errors.Errorf("%d: unsupported type %T", exp.X.Pos(), l)
			}
		}), BoolType, nil
	case token.EQL:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			return l == r, nil
		}), BoolType, nil
	case token.NEQ:
		return ExprFunction(func(ectx context.Context) (interface{}, error) {
			l, err := left(ectx)
			if err != nil {
				return nil, err
			}
			r, err := right(ectx)
			if err != nil {
				return nil, err
			}
			return l != r, nil
		}), BoolType, nil
	default:
		return nil, nil, errors.Errorf("%d: unsupported binary operation %s", exp.OpPos, exp.Op)
	}
}
