# Documentation

## What is GoEL?

GoEL is an interpreter for [golang expressions](https://golang.org/ref/spec#Expressions).
Expressions are handled in 3 steps.  First, the expression needs to be parsed by
the [go expression parsing API](https://golang.org/pkg/go/parser/#ParseExpr)
that produces an AST.  Second, the expression is "compiled" by the goel API that
takes the AST and an accompanying `Context` containing information about
available variables, types, and functions.  Third, the expression can be
executed, multiple times, by passing in an execution `Context` containing the
values of the variables and functions.

## The Parsing Context

The parsing context contains type information for variables, types and functions.
For each identifier that is not a built (e.g. true or int), the parsing context
is required to return the [reflect.Type](https://golang.org/pkg/reflect/#Type)
for the identifier.  For instance, if an `int` variable named `foo` needs to be
exposed to the expression being compiled, an entry should be added to the 
parsing context named "foo" and it should have the value of `reflect.Type` for
int.

## The Execution Context

The Execution Context contains the actual values that will be made available to
the expression when it is executed. There should be an entry in the execution 
context for each value in the parsing context. In the case of variables and 
functions the execution context must contain the [reflect.Value](https://golang.org/pkg/reflect/#Value)
for the variable or function.  In the case of types, the `reflect.Type` value
passed to the parsing context should be passed.

| Class of Identifier| Parsing Context Value | Execution Context Value |
|--------------------|-----------------------|-------------------------|
| type | reflect.TypeOf(&lt;the type>) | reflect.TypeOf(&lt;the type>) |
| variable | reflect.TypeOf(&lt;the variable>) | reflect.ValueOf(&lt;the variable>) |
| function | reflect.TypeOf(&lt;function name>) | reflect.ValueOf(&lt;function name>) |

## Safety

This API is designed to run a stateless expression but the rules of go do not
require that you make things stateless.  You could, for instance, have a
variable for a map that a function call can modify.  This is unsafe.  It
produces code that is likely to have bugs.  Also because structs and interfaces
can have functions attached and GoEL gives you access to those methods, you may
be able to modify the variables passed into the context.  In most cases it would
be best to pass readonly interfaces of your types.  For example the following
code passes a struct by reference to the context and does not prevent a call to
A.SetName:

``` go
package main

import (
	"context"
	"fmt"
	"github.com/homedepot/goel"
	"github.com/pkg/errors"
	"go/parser"
	"reflect"
)

type A struct {
	name string
}

func (a *A) Name() string {
	return a.name
}

func (a *A) SetName(newName string) string {
	oldName := a.name
	a.name = newName
	return oldName
}

type ReadA interface {
	Name() string
}

func evaluateExpressionOnA(a *A, expression string) (interface{}, error) {
	ast, err := parser.ParseExpr(expression)
	if err != nil {
		return nil, errors.Errorf("parsing error: %s", err.Error())
	}
	pctx := context.Background()
	pctx = context.WithValue(pctx, "a", reflect.TypeOf(a))
	exp := goel.NewCompiledExpression(pctx, ast)
	if exp.Error() != nil {
		return nil, errors.Errorf("building error: %s", exp.Error().Error())
	}
	ectx := context.Background()
	ectx = context.WithValue(pctx, "a", reflect.ValueOf(a))
	return exp.Execute(ectx)
}

func main() {
	a := &A{"joe"}
	v, err := evaluateExpressionOnA(a, `a.Name()`)
	if err != nil {
		fmt.Printf("error executing your expression: %s\n", err.Error())
	} else {
		fmt.Printf("a.Name() = %+v\n", v)
	}
	v, err = evaluateExpressionOnA(a, `a.SetName("jill")`)
	if err != nil {
		fmt.Printf("error executing your expression: %s\n", err.Error())
	} else {
		fmt.Printf("a.SetName(\"jill\") = %+v\n", v)
	}
	v, err = evaluateExpressionOnA(a, `a.Name()`)
	if err != nil {
		fmt.Printf("error executing your expression: %s\n", err.Error())
	} else {
		fmt.Printf("a.Name() = %+v\n", v)
	}
}
```
Will Output:
```
a.Name() = joe
a.SetName("jill") = joe
a.Name() = jill
```

After changing the evaluateExpressionOnA code to the following, the second 
expression will result in an error:

``` go
func evaluateExpressionOnA(a *A, expression string) (interface{}, error) {
	ast, err := parser.ParseExpr(expression)
	if err != nil {
		return nil, errors.Errorf("parsing error: %s", err.Error())
	}
	pctx := context.Background()
	var readA ReadA
	pctx = context.WithValue(pctx, "a", reflect.TypeOf(&readA).Elem())
	exp := goel.NewCompiledExpression(pctx, ast)
	if exp.Error() != nil {
		return nil, errors.Errorf("building error: %s", exp.Error().Error())
	}
	ectx := context.Background()
	ectx = context.WithValue(pctx, "a", reflect.ValueOf(a))
	return exp.Execute(ectx)
}
```

And it will output:

```
a.Name() = joe
error executing your expression: building error: 3: unknown selector SetName for main.ReadA
a.Name() = joe
```