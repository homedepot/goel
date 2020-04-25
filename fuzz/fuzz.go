package fuzz

import (
	"context"
	"go/parser"

	"github.com/homedepot/goel"
)

func Fuzz(data []byte) int {
	pctx := context.Background()
	ectx := context.Background()

	exp, err := parser.ParseExpr(string(data))
	if err != nil {
		return -1
	}

	cexp := goel.NewCompiledExpression(pctx, exp)

	res, err := cexp.Execute(ectx)
	if err != nil {
		return -1
	}
	if res != nil {
		return 1
	}
	return 0
}
