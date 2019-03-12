


# Go EL
GoEL is an expression language parser that parses a go expression and
allows the execution of that expression with a context.  It currently
supports the following operations:

* Binary operators: `+` `-` `*` `/` `==` `!=` `&&` `||` `.`
* Unary operators: `-` `!`
* literals: `string`, `int`, `float64`.
  Note: `rune` literals are treated as strings.
* Types: `string`, `int`, `float64`, `bool`, `struct` types and interfaces
* Function calls to both globally defined functions and functions 
  attached to types.

Notably not supported:
* type assertion
* type conversions
* function literals
* inner expressions (e.g. `a[x]`)
* slice expressions (e.g. `a[x:y]`)
* map expressions (e.g. `m["foo"]`)
* variadic function calls
* relation operators: `<` `>` `<=` `>=` 
* add operators: `^` `|`
* mul operators: `%` `<<` `>>` `&` `&^`
* unary operators: `+` `^` `*` `&` `<-`
* composite literals

As time goes by, most of these expressions will be accepted.  Here is a
list of expressions that I doubt will ever be allowed:

* function literals
* composite literals

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
	"github.homedepot.com/dhp236e/goel"
	"go/parser"
	"reflect"
)

func ExampleCompile() {
	pctx := context.Background()
	ectx := context.Background()
	exp, _ := parser.ParseExpr("5 + 3")
	fn, _, _ := goel.Compile(pctx, exp)
	result, _ := fn(ectx)
	fmt.Printf("%v\n", result)
	sum := func(x, y int) int {
		return x + y
	}

	pctx = context.WithValue(pctx, "sum", reflect.TypeOf(sum))
	ectx = context.WithValue(ectx, "sum", reflect.ValueOf(sum))
	exp, _ = parser.ParseExpr("sum(5,3)")
	fn, _, _ = goel.Compile(pctx, exp)
	result, _ = fn(ectx)
	fmt.Printf("%v\n", result)

	x := 5
	y := 3
	pctx = context.WithValue(pctx, "x", reflect.TypeOf(x))
	ectx = context.WithValue(ectx, "x", reflect.ValueOf(x))
	pctx = context.WithValue(pctx, "y", reflect.TypeOf(y))
	ectx = context.WithValue(ectx, "y", reflect.ValueOf(y))
	exp, _ = parser.ParseExpr("sum(x,y)")
	fn, _, _ = goel.Compile(pctx, exp)
	result, _ = fn(ectx)
	fmt.Printf("%v\n", result)
	// Output:
	// 8
	// 8
	// 8
}
```