package goel_test

import (
	"context"
	"fmt"
	"github.com/homedepot/goel"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go/parser"
	"net/http"
	"reflect"
	"regexp"
	"testing"
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

type test struct {
	name                   string
	expression             string
	expectedValue          reflect.Value
	expectedParsingError   error
	expectedBuildingError  error
	expectedExecutionError error
	parsingContext         map[string]interface{}
	executionContext       map[string]interface{}
}

var testRequest *http.Request
var tests []test
var x int
var c chan int

func bar() interface{} {
	return "bar"
}

func init() {
	var testArray [6]int
	testArray[0] = 1
	testArray[1] = 2
	testArray[2] = 4
	testArray[3] = 8
	testArray[4] = 16
	testArray[5] = 32
	var err error
	testRequest, err = http.NewRequest("GET", "http://localhost/foobar", nil)
	if err != nil {
		panic(err.Error())
	}
	testRequest.Header.Add("Content-Type", "application/json")
	tests = []test{
		{
			name:          "boolean literal true",
			expression:    "true",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "boolean literal true",
			expression:    "false",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "boolean not true",
			expression:    "!true",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "boolean not false",
			expression:    "!false",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "simple integer literal",
			expression:    "5",
			expectedValue: reflect.ValueOf(5),
		},
		{
			name:          "simple integer literal negation",
			expression:    "-5",
			expectedValue: reflect.ValueOf(-5),
		},
		{
			name:          "simple double literal negation",
			expression:    "-5.7",
			expectedValue: reflect.ValueOf(-5.7),
		},
		{
			name:                  "type mismatch negation",
			expression:            `-"5.7"`,
			expectedBuildingError: errors.New("1: unsupported unary expression: -string"),
		},
		{
			name:                  "unsupported unary operator (bitwise complement)",
			expression:            "^5",
			expectedBuildingError: errors.New("1: unsupported unary expression: ^int"),
		},
		{
			name:          "simple integer literal add",
			expression:    "+5",
			expectedValue: reflect.ValueOf(5),
		},
		{
			name:          "simple double literal add",
			expression:    "+5.7",
			expectedValue: reflect.ValueOf(5.7),
		},
		{
			name:                  "type mismatch negation",
			expression:            `+"5.7"`,
			expectedBuildingError: errors.New("1: unsupported unary expression: +string"),
		},
		{
			name:                  "unsupported unary operator (pointer deref)",
			expression:            "*x",
			expectedBuildingError: errors.New("1: unknown expression type"),
			parsingContext: map[string]interface{}{
				"x": reflect.TypeOf(&x),
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(&x),
			},
		},
		{
			name:                  "unsupported unary operator (pointer to)",
			expression:            "&x",
			expectedBuildingError: errors.New("1: unsupported unary expression: &int"),
			parsingContext: map[string]interface{}{
				"x": reflect.TypeOf(x),
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(x),
			},
		},
		{
			name:                  "unsupported unary operator (channel input)",
			expression:            "<-c",
			expectedBuildingError: errors.New("1: unsupported unary expression: <-"),
			parsingContext: map[string]interface{}{
				"c": reflect.TypeOf(c),
			},
			executionContext: map[string]interface{}{
				"c": reflect.ValueOf(c),
			},
		},
		{
			name:          "simple float literal",
			expression:    "5.6",
			expectedValue: reflect.ValueOf(5.6),
		},
		{
			name:          "simple string literal",
			expression:    `"fubar"`,
			expectedValue: reflect.ValueOf("fubar"),
		},
		{
			name:                 "invalid string literal",
			expression:           `"fubar`,
			expectedParsingError: errors.Errorf("1:1: string literal not terminated"),
		},
		{
			name:          "simple char literal",
			expression:    `'f'`,
			expectedValue: reflect.ValueOf("f"),
		},
		{
			name:                  "unsupported operator (xor)",
			expression:            "5 ^ 2",
			expectedBuildingError: errors.Errorf("3: unsupported binary operation ^"),
		},
		{
			name:          "less than false",
			expression:    "5 < 2",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than false equal",
			expression:    "5 < 5",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than true",
			expression:    "2 < 5",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than or equal false",
			expression:    "5 <= 2",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than or equal true equal",
			expression:    "2 <= 2",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than or equal true less than",
			expression:    "2 <= 5",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than false (float)",
			expression:    "5.0 < 2.0",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than false equal (float)",
			expression:    "5.0 < 5.0",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than true (float)",
			expression:    "2.0 < 5.0",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than or equal false (float)",
			expression:    "5.0 <= 2.0",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than or equal true equal (float)",
			expression:    "2.0 <= 2.0",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than or equal true less than (float)",
			expression:    "2.0 <= 5.0",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than false (string)",
			expression:    `"5.0" < "2.0"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than false equal (string)",
			expression:    `"5.0" < "5.0"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than true (string)",
			expression:    `"2.0" < "5.0"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than or equal false (string)",
			expression:    `"5.0" <= "2.0"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "less than or equal true equal (string)",
			expression:    `"2.0" <= "2.0"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "less than or equal true less than (string)",
			expression:    `"2.0" <= "5.0"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than false",
			expression:    "2 > 5",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than false equal",
			expression:    "5 > 5",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than true",
			expression:    "5 > 2",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than false (float)",
			expression:    "2.0 > 5.0",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than false equal (float)",
			expression:    "5.0 > 5.0",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than true (float)",
			expression:    "5.0 > 2.0",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than false (string)",
			expression:    `"a" > "b"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than false equal (string)",
			expression:    `"b" > "b"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than true (string)",
			expression:    `"b" > "a"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than or equal false",
			expression:    "2 >= 5",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than or equal true equal",
			expression:    "2 >= 2",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than or equal true less than",
			expression:    "5 >= 2",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than or equal false (float)",
			expression:    "2.0 >= 5.0",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than or equal true equal (float)",
			expression:    "2.0 >= 2.0",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than or equal true less than (float)",
			expression:    "5.0 >= 2.0",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than or equal false (string)",
			expression:    `"2" >= "5"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "greater than or equal true equal (string)",
			expression:    `"2" >= "2"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "greater than or equal true less than (string)",
			expression:    `"5" >= "2"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:                  "unsupported operator (bitwise or)",
			expression:            "5 | 2",
			expectedBuildingError: errors.Errorf("3: unsupported binary operation |"),
		},
		{
			name:          "modulo",
			expression:    "5 % 2",
			expectedValue: reflect.ValueOf(1),
		},
		{
			name:                  "unsupported operator (shift left)",
			expression:            "5 << 2",
			expectedBuildingError: errors.Errorf("3: unsupported binary operation <<"),
		},
		{
			name:                  "unsupported operator (shift right)",
			expression:            "5 >> 2",
			expectedBuildingError: errors.Errorf("3: unsupported binary operation >>"),
		},
		{
			name:                  "unsupported operator (bitwise and)",
			expression:            "5 & 2",
			expectedBuildingError: errors.Errorf("3: unsupported binary operation &"),
		},
		{
			name:                  "unsupported operator (bit clear)",
			expression:            "5 &^ 2",
			expectedBuildingError: errors.Errorf("3: unsupported binary operation &^"),
		},
		{
			name:          "simple literal addition",
			expression:    "5 + 2",
			expectedValue: reflect.ValueOf(7),
		},
		{
			name:          "simple literal addition (float)",
			expression:    "5.0 + 2.0",
			expectedValue: reflect.ValueOf(7.0),
		},
		{
			name:                  "type mismatch literal addition",
			expression:            "'f' + 2",
			expectedBuildingError: errors.Errorf("5: type mismatch in binary expression"),
		},
		{
			name:                  "type mismatch literal subtraction",
			expression:            "3.14 - 2",
			expectedBuildingError: errors.Errorf("6: type mismatch in binary expression"),
		},
		{
			name:                  "type mismatch literal multiplication",
			expression:            "6.7 * 2",
			expectedBuildingError: errors.Errorf("5: type mismatch in binary expression"),
		},
		{
			name:                  "type mismatch literal division",
			expression:            "3.5 / 2",
			expectedBuildingError: errors.Errorf("5: type mismatch in binary expression"),
		},
		{
			name:                  "unsupported type subtraction",
			expression:            "'f' - 2",
			expectedBuildingError: errors.Errorf("5: type mismatch in binary expression"),
		},
		{
			name:                  "unsupported type  multiplication",
			expression:            "'f' * 2",
			expectedBuildingError: errors.Errorf("5: type mismatch in binary expression"),
		},
		{
			name:                  "type mismatch literal division",
			expression:            "'f' / 2",
			expectedBuildingError: errors.Errorf("5: type mismatch in binary expression"),
		},
		{
			name:          "string literal addition",
			expression:    `"foo" + "bar"`,
			expectedValue: reflect.ValueOf("foobar"),
		},
		{
			name:          "simple literal subtraction",
			expression:    "5.3 - 2.7",
			expectedValue: reflect.ValueOf(2.6),
		},
		{
			name:          "simple literal multiplication",
			expression:    "5.3 * 2.7",
			expectedValue: reflect.ValueOf(14.31),
		},
		{
			name:          "simple literal division",
			expression:    "5.3 / 2.7",
			expectedValue: reflect.ValueOf(1.962962),
		},
		{
			name:          "simple literal division (int)",
			expression:    "5 / 2",
			expectedValue: reflect.ValueOf(2),
		},
		{
			name:          "simple literal equality",
			expression:    "5 == 5",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "simple literal inequality",
			expression:    "5 != 3",
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "simple literal equality negation",
			expression:    "5 == 3",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "simple literal inequality negation",
			expression:    "5 != 5",
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "simple literal string equality",
			expression:    `"foo" == "foo"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "simple literal string inequality",
			expression:    `"foo" != "bar"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "simple literal string equality negation",
			expression:    `"foo" == "bar"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "simple literal string inequality negation",
			expression:    `"foo" != "foo"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "logical AND (true)",
			expression:    `5 == 5 && "foo" != "bar"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "logical AND (false)",
			expression:    `5 == 5 && "foo" != "foo"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "logical OR (true)",
			expression:    `5 == 5 || "foo" == "bar"`,
			expectedValue: reflect.ValueOf(true),
		},
		{
			name:          "logical OR (false)",
			expression:    `5 == 4 || "foo" != "foo"`,
			expectedValue: reflect.ValueOf(false),
		},
		{
			name:          "parenthesized literal expression",
			expression:    "(5 + 2) * 3",
			expectedValue: reflect.ValueOf(21),
		},
		{
			name:                   "binary expression left execution error",
			expression:             `returnsError() == 6`,
			expectedExecutionError: errors.New("Boo!"),
			parsingContext: map[string]interface{}{
				"returnsError": reflect.TypeOf(returnsError),
			},
			executionContext: map[string]interface{}{
				"returnsError": reflect.ValueOf(returnsError),
			},
		},
		{
			name:                   "binary expression right execution error",
			expression:             `6 == returnsError()`,
			expectedExecutionError: errors.New("Boo!"),
			parsingContext: map[string]interface{}{
				"returnsError": reflect.TypeOf(returnsError),
			},
			executionContext: map[string]interface{}{
				"returnsError": reflect.ValueOf(returnsError),
			},
		},
		{
			name:                  "binary expression left build error",
			expression:            `*3 == 6`,
			expectedBuildingError: errors.New("1: unknown expression type"),
		},
		{
			name:                  "binary expression right build error",
			expression:            `6 == *3`,
			expectedBuildingError: errors.New("6: unknown expression type"),
		},
		{
			name:                  "binary expression unsupported type",
			expression:            `x == y`,
			expectedBuildingError: errors.New("3: unsupported binary expression type: *http.Request"),
			parsingContext: map[string]interface{}{
				"x": reflect.TypeOf(testRequest),
				"y": reflect.TypeOf(testRequest),
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(testRequest),
				"y": reflect.ValueOf(testRequest),
			},
		},
		{
			name:                  "add expression unsupported type (bool)",
			expression:            `true + true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "sub expression unsupported type (bool)",
			expression:            `true - true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "mul expression unsupported type (bool)",
			expression:            `true * true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "quo expression unsupported type (bool)",
			expression:            `true / true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "gtr expression unsupported type (bool)",
			expression:            `true > true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "geq expression unsupported type (bool)",
			expression:            `true >= true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "ltr expression unsupported type (bool)",
			expression:            `true < true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "leq expression unsupported type (bool)",
			expression:            `true <= true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "rem expression unsupported type (bool)",
			expression:            `true % true`,
			expectedBuildingError: errors.New("1: unsupported type bool"),
		},
		{
			name:                  "LAND expression unsupported type (int)",
			expression:            `5 && 1`,
			expectedBuildingError: errors.New("1: unsupported type int"),
		},
		{
			name:                  "LOR expression unsupported type (int)",
			expression:            `5 || 1`,
			expectedBuildingError: errors.New("1: unsupported type int"),
		},
		{
			name:          "simple variable addition",
			expression:    "5 + x",
			expectedValue: reflect.ValueOf(7),
			parsingContext: map[string]interface{}{
				"x": goel.IntType,
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(2),
			},
		},
		{
			name:          "simple variable subtraction",
			expression:    "y - x",
			expectedValue: reflect.ValueOf(3),
			parsingContext: map[string]interface{}{
				"x": goel.IntType,
				"y": goel.IntType,
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(2),
				"y": reflect.ValueOf(5),
			},
		},
		{
			name:          "regexp matching",
			expression:    `matches("[0-9]{3}", x)`,
			expectedValue: reflect.ValueOf(true),
			parsingContext: map[string]interface{}{
				"x":       goel.StringType,
				"matches": reflect.TypeOf(matchesRegex),
			},
			executionContext: map[string]interface{}{
				"x":       reflect.ValueOf("321"),
				"matches": reflect.ValueOf(matchesRegex),
			},
		},
		{
			name:          "regexp matching negation",
			expression:    `matches("[a-z]{3}", x)`,
			expectedValue: reflect.ValueOf(false),
			parsingContext: map[string]interface{}{
				"x":       goel.StringType,
				"matches": reflect.TypeOf(matchesRegex),
			},
			executionContext: map[string]interface{}{
				"x":       reflect.ValueOf("321"),
				"matches": reflect.ValueOf(matchesRegex),
			},
		},
		{
			name:                   "function invocation returns error",
			expression:             `returnsError()`,
			expectedExecutionError: errors.New("Boo!"),
			parsingContext: map[string]interface{}{
				"returnsError": reflect.TypeOf(returnsError),
			},
			executionContext: map[string]interface{}{
				"returnsError": reflect.ValueOf(returnsError),
			},
		},
		{
			name:          "struct member access",
			expression:    "req.Method",
			expectedValue: reflect.ValueOf("GET"),
			parsingContext: map[string]interface{}{
				"req": reflect.TypeOf(testRequest),
			},
			executionContext: map[string]interface{}{
				"req": reflect.ValueOf(testRequest),
			},
		},
		{
			name:          "struct member Call",
			expression:    `req.Header.Get("Content-Type")`,
			expectedValue: reflect.ValueOf("application/json"),
			parsingContext: map[string]interface{}{
				"req": reflect.TypeOf(testRequest),
			},
			executionContext: map[string]interface{}{
				"req": reflect.ValueOf(testRequest),
			},
		},
		{
			name:          "struct member Call comparison",
			expression:    `req.Header.Get("Content-Type") == "application/json"`,
			expectedValue: reflect.ValueOf(true),
			parsingContext: map[string]interface{}{
				"req": reflect.TypeOf(testRequest),
			},
			executionContext: map[string]interface{}{
				"req": reflect.ValueOf(testRequest),
			},
		},
		{
			name:                 "simple invalid literal",
			expression:           "5x",
			expectedParsingError: errors.Errorf("1:2: expected 'EOF', found x"),
		},
		{
			name:                  "simple invalid identifier",
			expression:            "x",
			expectedBuildingError: errors.Errorf("1: unknown identifier: x"),
		},
		{
			name:                  "simple invalid selector",
			expression:            "x.Foo",
			expectedBuildingError: errors.Errorf("3: unknown selector Foo for int"),
			parsingContext: map[string]interface{}{
				"x": goel.IntType,
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(2),
			},
		},
		{
			name:                   "type assertion failure",
			expression:             "x.(string)",
			expectedExecutionError: errors.Errorf("4: int is not assignable to string."),
			parsingContext: map[string]interface{}{
				"x": goel.IntType,
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(2),
			},
		},
		{
			name:          "type assertion success",
			expression:    "bar().(string)",
			expectedValue: reflect.ValueOf("bar"),
			parsingContext: map[string]interface{}{
				"bar": reflect.TypeOf(bar),
			},
			executionContext: map[string]interface{}{
				"bar": reflect.ValueOf(bar),
			},
		},
		{
			name:          "type assertion success custom type",
			expression:    "request().(Request).Header.Get(\"Content-Type\")",
			expectedValue: reflect.ValueOf("application/json"),
			parsingContext: map[string]interface{}{
				"request": reflect.TypeOf(returnsRequestAsInterface),
				"Request": reflect.TypeOf(testRequest),
			},
			executionContext: map[string]interface{}{
				"request": reflect.ValueOf(returnsRequestAsInterface),
				"Request": reflect.TypeOf(testRequest),
			},
		},
		{
			name:       "type mismatch in call",
			expression: `matches("[0-9]{3}", x)`,
			parsingContext: map[string]interface{}{
				"x":       goel.IntType,
				"matches": reflect.TypeOf(matchesRegex),
			},
			executionContext: map[string]interface{}{
				"x":       reflect.ValueOf(321),
				"matches": reflect.ValueOf(matchesRegex),
			},
			expectedBuildingError: errors.Errorf("21: type mismatch in argument 1"),
		},
		{
			name:                  "type mismatch in operator",
			expression:            `req.Header.Get("Content-Type") == 37`,
			expectedBuildingError: errors.Errorf("32: type mismatch in binary expression"),
			parsingContext: map[string]interface{}{
				"req": reflect.TypeOf(testRequest),
			},
			executionContext: map[string]interface{}{
				"req": reflect.ValueOf(testRequest),
			},
		},
		{
			name:                  "unknown expression (type conversion)",
			expression:            "float64(x)",
			expectedBuildingError: errors.Errorf("1: unknown identifier: float64"),
			parsingContext: map[string]interface{}{
				"x": goel.IntType,
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(2),
			},
		},
		{
			name:                  "unknown expression (function literal)",
			expression:            "func(i int)(x)",
			expectedBuildingError: errors.Errorf("1: unknown expression type"),
			parsingContext: map[string]interface{}{
				"x": goel.IntType,
			},
			executionContext: map[string]interface{}{
				"x": reflect.ValueOf(2),
			},
		},
		{
			name:          "unknown expression (inner expression)",
			expression:    "a[0]",
			expectedValue: reflect.ValueOf(5),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{5}),
			},
		},
		{
			name:          "slice expression on string",
			expression:    "a[0:1]",
			expectedValue: reflect.ValueOf("a"),
			parsingContext: map[string]interface{}{
				"a": goel.StringType,
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf("abcdef"),
			},
		},
		{
			name:                  "slice3 expression on string not allowed",
			expression:            "a[0:1:2]",
			expectedBuildingError: errors.New("1: type mismatch expected a slice but found string"),
			parsingContext: map[string]interface{}{
				"a": goel.StringType,
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf("abcdef"),
			},
		},
		{
			name:                  "slice expression on array",
			expression:            "a[0:1]",
			expectedBuildingError: errors.New("1: type mismatch expected a slice or string but found [6]int"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf(&testArray).Elem(),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf(&testArray).Elem(),
			},
		},
		{
			name:          "slice expression",
			expression:    "a[0:1]",
			expectedValue: reflect.ValueOf([]int{1}),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:          "slice expression no lower",
			expression:    "a[:1]",
			expectedValue: reflect.ValueOf([]int{1}),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:          "slice expression no high",
			expression:    "a[5:]",
			expectedValue: reflect.ValueOf([]int{32}),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:          "slice expression w/max",
			expression:    "a[0:2:3]",
			expectedValue: reflect.ValueOf([]int{1, 2}),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:                 "invalid slice expression no max",
			expression:           "a[0:2:]",
			expectedParsingError: errors.New("1:6: 3rd index required in 3-index slice"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:                   "invalid slice expression low greater than high",
			expression:             "a[3:2]",
			expectedExecutionError: errors.New("5: index out of range: 2"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:                   "invalid slice expression max less than than high",
			expression:             "a[2:3:1]",
			expectedExecutionError: errors.New("7: index out of range: 1"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:                   "invalid slice expression low greater than or equal to length",
			expression:             "a[6:]",
			expectedExecutionError: errors.New("3: index out of range: 6"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:                   "invalid slice expression high greater than or equal to length",
			expression:             "a[:7]",
			expectedExecutionError: errors.New("4: index out of range: 7"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:                   "invalid slice expression max greater than to capacity",
			expression:             "a[:6:7]",
			expectedExecutionError: errors.New("6: index out of range: 7"),
			parsingContext: map[string]interface{}{
				"a": reflect.TypeOf([]int{}),
			},
			executionContext: map[string]interface{}{
				"a": reflect.ValueOf([]int{1, 2, 4, 8, 16, 32}),
			},
		},
		{
			name:          "map expression not nil",
			expression:    `m["foo"]`,
			expectedValue: reflect.ValueOf(5),
			parsingContext: map[string]interface{}{
				"m": reflect.TypeOf(map[string]int{}),
			},
			executionContext: map[string]interface{}{
				"m": reflect.ValueOf(map[string]int{"foo": 5, "bar": 6}),
			},
		},
		{
			name:          "map expression nil",
			expression:    `m["snafu"]`,
			expectedValue: reflect.ValueOf(0),
			parsingContext: map[string]interface{}{
				"m": reflect.TypeOf(map[string]int{}),
			},
			executionContext: map[string]interface{}{
				"m": reflect.ValueOf(map[string]int{"foo": 5, "bar": 6}),
			},
		},
		{
			name:                  "unknown expression (variadic function call)",
			expression:            "f(1,2,3)",
			expectedBuildingError: errors.Errorf("2: variadic functions are not supported."),
			parsingContext: map[string]interface{}{
				"f": reflect.TypeOf(variadicSum),
			},
			executionContext: map[string]interface{}{
				"f": reflect.ValueOf(variadicSum),
			},
		},
	}
}

func variadicSum(values ...int) int {
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum
}

func returnsRequestAsInterface() interface{} {
	return testRequest
}

func returnsError() (int, error) {
	return 0, errors.New("Boo!")
}

func matchesRegex(regex, x string) (bool, error) {
	return regexp.MatchString(regex, x)
}

func TestCompile(t *testing.T) {
	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			pctx := contextFromMap(tst.parsingContext)
			ectx := contextFromMap(tst.executionContext)
			exp, err := parser.ParseExpr(tst.expression)
			if tst.expectedParsingError == nil {
				if assert.NoError(t, err) {
					cexp := goel.NewCompiledExpression(pctx, exp)
					if tst.expectedBuildingError == nil {
						if assert.NoError(t, cexp.Error()) {
							actual, err := cexp.Execute(ectx)
							if tst.expectedExecutionError == nil {
								if assert.NoError(t, err) {
									fnTyp := reflect.TypeOf(actual)
									assert.True(t, fnTyp.AssignableTo(tst.expectedValue.Type()))
									if tst.expectedValue.Type().AssignableTo(goel.DoubleType) {
										assert.InDelta(t, tst.expectedValue.Float(), actual, 0.0001)
									} else {
										assert.EqualValues(t, tst.expectedValue.Interface(), actual)
									}
								}
							} else if err != nil {
								assert.Equal(t, tst.expectedExecutionError.Error(), err.Error())
							} else {
								assert.Failf(t, "expected an execution error but got none", "%s", tst.expectedExecutionError.Error())
							}
						}
					} else if cexp.Error() != nil {
						assert.Equal(t, tst.expectedBuildingError.Error(), cexp.Error().Error())
					} else {
						assert.Failf(t, "expected a building error but got none", "%s", tst.expectedBuildingError.Error())
					}
				}
			} else if err != nil {
				assert.Equal(t, tst.expectedParsingError.Error(), err.Error())
			} else {
				assert.Failf(t, "expected a parsing error but got none", "%s", tst.expectedParsingError.Error())
			}
		})
	}
}

func contextFromMap(contextMap map[string]interface{}) context.Context {
	pctx := context.Background()
	for k, v := range contextMap {
		pctx = context.WithValue(pctx, k, v)
	}
	return pctx
}
