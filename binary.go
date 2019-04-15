package goel

import (
	"context"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
)

type binaryCompiledExpression struct {
	nopExpression
	returnType reflect.Type
	left       CompiledExpression
	right      CompiledExpression
	operate    func(l, r interface{}) interface{}
	lpos, rpos token.Pos
}

func newBinaryCompiledExpression(rt reflect.Type, left CompiledExpression, right CompiledExpression, exp *ast.BinaryExpr, op func(l, r interface{}) interface{}) *binaryCompiledExpression {
	return &binaryCompiledExpression{nopExpression{}, rt, left, right, op, exp.X.Pos(), exp.Y.Pos()}
}

func addint(l, r interface{}) interface{} {
	return l.(int) + r.(int)
}

func addfloat(l, r interface{}) interface{} {
	return l.(float64) + r.(float64)
}

func addstring(l, r interface{}) interface{} {
	return l.(string) + r.(string)
}

func subfloat(l, r interface{}) interface{} {
	return l.(float64) - r.(float64)
}

func subint(l, r interface{}) interface{} {
	return l.(int) - r.(int)
}

func mulfloat(l, r interface{}) interface{} {
	return l.(float64) * r.(float64)
}

func mulint(l, r interface{}) interface{} {
	return l.(int) * r.(int)
}

func divfloat(l, r interface{}) interface{} {
	return l.(float64) / r.(float64)
}

func divint(l, r interface{}) interface{} {
	return l.(int) / r.(int)
}

func gtrint(l, r interface{}) interface{} {
	return l.(int) > r.(int)
}

func gtrfloat(l, r interface{}) interface{} {
	return l.(float64) > r.(float64)
}

func gtrstring(l, r interface{}) interface{} {
	return l.(string) > r.(string)
}

func geqint(l, r interface{}) interface{} {
	return l.(int) >= r.(int)
}

func geqfloat(l, r interface{}) interface{} {
	return l.(float64) >= r.(float64)
}

func geqstring(l, r interface{}) interface{} {
	return l.(string) >= r.(string)
}

func lssint(l, r interface{}) interface{} {
	return l.(int) < r.(int)
}

func lssfloat(l, r interface{}) interface{} {
	return l.(float64) < r.(float64)
}

func lssstring(l, r interface{}) interface{} {
	return l.(string) < r.(string)
}

func leqint(l, r interface{}) interface{} {
	return l.(int) <= r.(int)
}

func leqfloat(l, r interface{}) interface{} {
	return l.(float64) <= r.(float64)
}

func leqstring(l, r interface{}) interface{} {
	return l.(string) <= r.(string)
}

func and(l, r interface{}) interface{} {
	return l.(bool) && r.(bool)
}

func or(l, r interface{}) interface{} {
	return l.(bool) || r.(bool)
}

func eq(l, r interface{}) interface{} {
	return l == r
}

func neq(l, r interface{}) interface{} {
	return l != r
}

func (bce *binaryCompiledExpression) Execute(ectx context.Context) (interface{}, error) {
	l, err := bce.left.Execute(ectx)
	if err != nil {
		return nil, err
	}
	r, err := bce.right.Execute(ectx)
	if err != nil {
		return nil, err
	}
	lt, _ := bce.left.ReturnType()
	if !reflect.TypeOf(r).AssignableTo(lt) {
		return nil, errors.Errorf("type mismatch expected %s but found %T", bce.returnType.Name(), r)
	}
	return bce.operate(l, r), nil
}

func (bce *binaryCompiledExpression) Error() error {
	return nil
}

func (bce *binaryCompiledExpression) ReturnType() (reflect.Type, error) {
	return bce.returnType, nil
}

func evalBinaryExpr(pctx context.Context, exp *ast.BinaryExpr) CompiledExpression {
	left := compile(pctx, exp.X)
	if left.Error() != nil {
		return left
	}
	lt, _ := left.ReturnType()
	right := compile(pctx, exp.Y)
	if right.Error() != nil {
		return right
	}
	rt, _ := right.ReturnType()
	if !lt.AssignableTo(rt) {
		return newErrorExpression(errors.Errorf("%d: type mismatch in binary expression", exp.OpPos))
	}
	if !(lt.AssignableTo(StringType) || lt.AssignableTo(IntType) || lt.AssignableTo(DoubleType) || lt.AssignableTo(BoolType)) {
		return newErrorExpression(errors.Errorf("%d: unsupported binary expression type: %s", exp.OpPos, lt.String()))
	}
	switch exp.Op {
	case token.ADD:
		switch {
		case lt.AssignableTo(StringType):
			return newBinaryCompiledExpression(lt, left, right, exp, addstring)
		case lt.AssignableTo(IntType):
			return newBinaryCompiledExpression(lt, left, right, exp, addint)
		case lt.AssignableTo(DoubleType):
			return newBinaryCompiledExpression(lt, left, right, exp, addfloat)
		default:
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.SUB:
		switch {
		case lt.AssignableTo(IntType):
			return newBinaryCompiledExpression(lt, left, right, exp, subint)
		case lt.AssignableTo(DoubleType):
			return newBinaryCompiledExpression(lt, left, right, exp, subfloat)
		default:
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.MUL:
		switch {
		case lt.AssignableTo(IntType):
			return newBinaryCompiledExpression(lt, left, right, exp, mulint)
		case lt.AssignableTo(DoubleType):
			return newBinaryCompiledExpression(lt, left, right, exp, mulfloat)
		default:
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.QUO:
		switch {
		case lt.AssignableTo(IntType):
			return newBinaryCompiledExpression(lt, left, right, exp, divint)
		case lt.AssignableTo(DoubleType):
			return newBinaryCompiledExpression(lt, left, right, exp, divfloat)
		default:
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.LAND:
		if lt.AssignableTo(BoolType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, and)
		} else {
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.LOR:
		if lt.AssignableTo(BoolType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, or)
		} else {
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.EQL:
		return newBinaryCompiledExpression(BoolType, left, right, exp, eq)
	case token.NEQ:
		return newBinaryCompiledExpression(BoolType, left, right, exp, neq)
	case token.GTR:
		if lt.AssignableTo(IntType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, gtrint)
		} else if lt.AssignableTo(DoubleType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, gtrfloat)
		} else if lt.AssignableTo(StringType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, gtrstring)
		} else {
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.GEQ:
		if lt.AssignableTo(IntType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, geqint)
		} else if lt.AssignableTo(DoubleType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, geqfloat)
		} else if lt.AssignableTo(StringType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, geqstring)
		} else {
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.LSS:
		if lt.AssignableTo(IntType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, lssint)
		} else if lt.AssignableTo(DoubleType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, lssfloat)
		} else if lt.AssignableTo(StringType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, lssstring)
		} else {
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	case token.LEQ:
		if lt.AssignableTo(IntType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, leqint)
		} else if lt.AssignableTo(DoubleType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, leqfloat)
		} else if lt.AssignableTo(StringType) {
			return newBinaryCompiledExpression(BoolType, left, right, exp, leqstring)
		} else {
			return newErrorExpression(errors.Errorf("%d: unsupported type %s", exp.X.Pos(), lt.Name()))
		}
	default:
		return newErrorExpression(errors.Errorf("%d: unsupported binary operation %s", exp.OpPos, exp.Op))
	}
}
