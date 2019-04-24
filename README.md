
[![Site](https://img.shields.io/badge/goel-site-blue.svg?style=plastic)](https://homedepot.github.io/goel/)
[![Go Lang Version](https://img.shields.io/badge/go-1.12-00ADD8.svg?style=plastic)](http://golang.com)
[![Go Doc](https://img.shields.io/badge/godoc-reference-00ADD8.svg?style=plastic)](https://godoc.org/github.com/homedepot/goel)
[![Go Report Card](https://goreportcard.com/badge/github.com/homedepot/goel?style=plastic)](https://goreportcard.com/report/github.com/homedepot/goel)
[![codecov](https://img.shields.io/codecov/c/github/homedepot/goel.svg?style=plastic)](https://codecov.io/gh/homedepot/goel)
[![CircleCI](https://img.shields.io/circleci/project/github/homedepot/goel.svg?style=plastic)](https://circleci.com/gh/homedepot/goel/tree/master)

# Go EL
GoEL is an expression language parser that parses a go expression and
allows the execution of that expression with a context.  It currently
supports the following operations:

* Binary operators: `+` `-` `*` `/` `==` `!=` `&&` `||` `.` `%`
* Relation operators: `<` `>` `<=` `>=` 
* Unary operators: `-` `!` `+` 
* literals: `string`, `int`, `float64`.
  Note: `rune` literals are treated as strings.
* Types: `string`, `int`, `float64`, `bool`, `struct` types and 
  interfaces
* Function calls to both globally defined functions and functions 
  attached to types.
* inner expressions (e.g. `a[x]`)
* map expressions (e.g. `m["foo"]`)
* type assertion (e.g. `foo.(string)`
* slice expressions on slices (e.g. `a[x:y:m]`)

Not supported (in priority order):
1. variadic function calls
1. add operators: `^` `|`
1. type conversions
1. mul operators: `<<` `>>` `&` `&^`
1. unary operators: `^`

As time goes by, most of these expressions will be accepted.  Here is a
list of expressions that I doubt will ever be allowed:

* function literals
* composite literals
* unary operators: `*` `&` `<-`
* slice expressions on arrays (e.g. `a[x:y]`)

# Getting Started

## Contexts
There are two contexts that are used when evaluating expressions: The
parsing context and the execution context.  Each are used to provide
variables and functions to the two phases of execution.

### Parsing Context
This context should contain entries with string keys that are the type
of the referenced object.  For instance, the type of a structure must be
passed to enable the parser to understand the type of the variables and
know which are defined.  Variables referenced in the expression that do
not have a type in the context, will result in an error stating that the
variable is not defined.

### Execution Context
The execution context contains the actual values or functions associated
with the names used as keys.

## Function return values
If a function has multiple return values, it will return an 
`[]interface{}` containing the values instead.  For the most part it is
not well supported for a function to return multiple values.  This may
change in the future.

### Functions with Errors
One exception to the above rule is when a function returns an error as
its last output.  In that case, all the previous outputs will be treated
as described above but the error value will be checked against `nil`. If
the value is not nil, the evaluation will end and return the error.

## Example

```golang
package goel_test

import (
	"context"
	"fmt"
	"github.com/homedepot/goel"
	"go/parser"
	"reflect"
)

func ExampleCompile() {
	pctx := context.Background()
	ectx := context.Background()
	exp, _ := parser.ParseExpr("5 + 3")
	cexp := goel.NewCompiledExpression(pctx, exp)
	result, _ := cexp.Execute(ectx)
	fmt.Printf("%v\n", result)
	sum := func(x, y int) int {
		return x + y
	}

	pctx = context.WithValue(pctx, "sum", reflect.TypeOf(sum))
	ectx = context.WithValue(ectx, "sum", reflect.ValueOf(sum))
	exp, _ = parser.ParseExpr("sum(5,3)")
	cexp = goel.NewCompiledExpression(pctx, exp)
	result, _ = cexp.Execute(ectx)
	fmt.Printf("%v\n", result)

	x := 5
	y := 3
	pctx = context.WithValue(pctx, "x", reflect.TypeOf(x))
	ectx = context.WithValue(ectx, "x", reflect.ValueOf(x))
	pctx = context.WithValue(pctx, "y", reflect.TypeOf(y))
	ectx = context.WithValue(ectx, "y", reflect.ValueOf(y))
	exp, _ = parser.ParseExpr("sum(x,y)")
	cexp = goel.NewCompiledExpression(pctx, exp)
	result, _ = cexp.Execute(ectx)
	fmt.Printf("%v\n", result)
	// Output:
	// 8
	// 8
	// 8
}
```

For more details on how to use this API, see the
[go doc](https://godoc.org/github.com/homedepot/goel) and the
[pages site](https://homedepot.github.io/goel/)

# CI

Coming Soon...

# Maintainers

[Dana H. P'Simer](https://github.com/danapsimer)


# License

This project is released under the Apache 2.0 Open Source License.
Please see our [LICENSE](LICENSE) file for details.

# Contributing

Please see our [Contributing](CONTRIBUTING.md) document for details on
contributing. 