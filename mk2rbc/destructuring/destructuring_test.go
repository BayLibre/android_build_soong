package destructuring

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

type AnyStruct interface {
	isAStruct()
}

type StructA struct {
	nested  AnyStruct
	aString string
	aBool   bool
}

func (*StructA) isAStruct() {
}

type StructB struct {
	aString string
	Aint    int
}

func (*StructB) isAStruct() {
}

type StructB2 struct {
	aString string
	Aint    int
}

func (*StructB2) isAStruct() {
}

type StructC struct {
	aFloat float32
}

func (*StructC) isAStruct() {
}

type StructD struct {
	aBool bool
}

func (*StructD) isAStruct() {
}

func TestBasic(t *testing.T) {
	input := "foo"
	var result string
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	assertEqual(t, input, result, "Expected result to be equal to input")
}

func TestSimpleStruct(t *testing.T) {
	input := StructB{
		aString: "foo",
		Aint:    5,
	}
	result := StructB {
		aString: "",
		Aint:    0,
	}
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	assertTrue(t, reflect.DeepEqual(input, result), "Input was not equal to result")
}

func TestNestedStructs(t *testing.T) {
	input := StructA{
		nested: &StructB{
			aString: "bar",
			Aint:    7,
		},
		aString: "foo",
		aBool: false,
	}
	result_nested := &StructB{}
	result := StructA{
		nested: result_nested,
		aString: "foo",
		aBool: false,
	}
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	assertTrue(t, reflect.DeepEqual(input, result), "Input was not equal to result")
	assertEqual(t, result_nested.Aint, 7, "result_nested.Aint wasn't updated")
}

func TestNestedStructsWithCustomType(t *testing.T) {
	input := StructA{
		nested: &StructB{
			aString: "bar",
			Aint:    7,
		},
		aString: "foo",
		aBool: true,
	}
	result := struct{
		nested *struct{
			aString string
			Aint int
		}
		aString string
		aBool bool
	}{}
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	// We can no longer use reflect.DeepEqual because there's no interface wrapping nested in result
	assertEqual(t, result.nested.aString, "bar", "Result.nested.aString wasn't updated")
	assertEqual(t, result.nested.Aint, 7, "Result.nested.Aint wasn't updated")
	assertEqual(t, result.aString, input.aString, "Result.aString wasn't updated")
	assertEqual(t, result.aBool, input.aBool, "Result.aBool wasn't updated")
}

func TestNestedStructsIncompatibleType(t *testing.T) {
	input := StructA{
		nested: &StructB{
			aString: "bar",
			Aint:    7,
		},
		aString: "foo",
		aBool: true,
	}
	result := struct{
		nested *struct{
			aBool bool
			Aint int
		}
		aString string
		aBool bool
	}{}
	assertFalse(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring succeeded when it shouldn't have")
}

func TestNestedStructsWrongRequiredType(t *testing.T) {
	input := StructA{
		nested: &StructB{
			aString: "bar",
			Aint:    7,
		},
		aString: "foo",
		aBool: true,
	}
	result := struct{
		nested *struct{
			StructType string `required:"StructB2"`
			aString string
			Aint int
		}
		aString string
		aBool bool
	}{}
	assertFalse(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring succeeded when it shouldn't have")
}

func TestNestedStructsCorrectRequiredType(t *testing.T) {
	input := StructA{
		nested: &StructB{
			aString: "bar",
			Aint:    7,
		},
		aString: "foo",
		aBool: true,
	}
	result := struct{
		nested *struct{
			StructType string `required:"StructB"`
			aString string
			Aint int
		}
		aString string
		aBool bool
	}{}
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	// We can no longer use reflect.DeepEqual because there's no interface wrapping nested in result
	assertEqual(t, result.nested.aString, "bar", "Result.nested.aString wasn't updated")
	assertEqual(t, result.nested.Aint, 7, "Result.nested.Aint wasn't updated")
	assertEqual(t, result.aString, input.aString, "Result.aString wasn't updated")
	assertEqual(t, result.aBool, input.aBool, "Result.aBool wasn't updated")
}

func TestStructCorrectRequired(t *testing.T) {
	input := StructD{
		aBool: true,
	}
	result := struct{
		aBool bool `required:"true"`
	}{}
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	assertEqual(t, input.aBool, result.aBool, "input not equal to result")
}

func TestStructInCorrectRequired(t *testing.T) {
	fmt.Fprintf(os.Stderr, "Beginning of latest test\n")
	input := StructD{
		aBool: true,
	}
	result := struct{
		aBool bool `required:"false"`
	}{}
	assertFalse(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring succeeded when it should've failed")
}

func TestSlice(t *testing.T) {
	input := []AnyStruct {
		&StructC{aFloat: 3},
		&StructC{aFloat: -5},
	}
	result := []*StructC{{}, {},}
	assertTrue(t, Destructure(reflect.ValueOf(&input), reflect.ValueOf(&result)), "Destructuring failed")
	if !reflect.DeepEqual(input[0], result[0]) || !reflect.DeepEqual(input[1], result[1]){
		t.Errorf("Input was not equal to result: %v, %v", input, result)
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a != b {
		t.Fatal(fmt.Sprintf("%s: %v != %v", message, a, b))
	}
}

func assertTrue(t *testing.T, b bool, message string) {
	if !b {
		t.Fatal(message)
	}
}

func assertFalse(t *testing.T, b bool, message string) {
	if b {
		t.Fatal(message)
	}
}
