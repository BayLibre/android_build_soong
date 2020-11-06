package mk2rbc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	mkparser "android/soong/androidmk/parser"
)

var callFuncRex = regexp.MustCompile("^call +add_json_(str|val|bool|csv|list) *,")

// Scans the makefile Soong uses to generate soong.variables file,
// collecting variable names and types from the lines that look like this:
//    $(call add_json_XXX,  <...>,             $(VAR))
//
func FindSoongVariables(mkFile string, includeFileScope mkparser.Scope) error {
	mkContents, err := ioutil.ReadFile(mkFile)
	if err != nil {
		return err
	}
	parser := mkparser.NewParser(mkFile, bytes.NewBuffer(mkContents))
	nodes, errs := parser.Parse()
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "ERROR:", e)
		}
		return fmt.Errorf("cannot parse %s", mkFile)
	}
	for _, node := range nodes {
		switch t := node.(type) {
		case *mkparser.Variable:
			handleVariable(t)
		case *mkparser.Directive:
			handleInclude(t, includeFileScope)
		}
	}
	return nil
}

func NewSoongVariable(name, typeString string) {
	var valueType starType
	switch typeString {
	case "bool":
		valueType = starTypeBool
	case "csv":
		// Only PLATFORM_VERSION_ALL_CODENAMES, and it's a list
		valueType = starTypeList
	case "list":
		valueType = starTypeList
	case "str":
		valueType = starTypeString
	case "val":
		// Only PLATFORM_SDK_VERSION uses this, and it's integer
		valueType = starTypeInt
	default:
		panic(fmt.Errorf("unknown Soong variable type %s", typeString))
	}

	KnownVariables.NewVariable(name, VarClassSoong, valueType)
}

func handleInclude(t *mkparser.Directive, scope mkparser.Scope) {
	if t.Name != "include" && t.Name != "-include" {
		return
	}
	includedPath := t.Args.Value(scope)
	err := FindSoongVariables(includedPath, scope)
	if err != nil && t.Name == "include" {
		fmt.Fprintf(os.Stderr, "cannot include %s: %s", includedPath, err)
	}
}

func handleVariable(t *mkparser.Variable) {
	name := t.Name
	// From the variable reference looking as follows:
	//  $(call json_add_TYPE,arg1,$(VAR))
	// we infer that the type of $(VAR) is TYPE
	// VAR can be a simple variable name, or another call
	// (e.g., $(call invert_bool, $(X)), from which we can infer
	// that the type of X is bool
	if len(name.Strings) != 2 ||
		!strings.HasPrefix(name.Strings[0], "call add_json") || name.Strings[1] != "" {
		return
	}

	match := callFuncRex.FindAllStringSubmatch(name.Strings[0], -1)
	if match == nil {
		panic(fmt.Errorf("cannot match the call: %s", name.Strings[0]))
	}
	inferSoongVariableType(match[0][1], name.Variables[0].Name)
}

var (
	callInvertBoolRex = regexp.MustCompile("^call +invert_bool *, *$")
	callFilterBoolRex = regexp.MustCompile("^(call +(filter|filter-out) +(true|false)),$")
)

func inferSoongVariableType(vType string, n *mkparser.MakeString) {
	if len(n.Strings) == 1 {
		NewSoongVariable(n.Strings[0], vType)
		return
	}
	// Is it $(call invert_bool, $(VAR))?
	if len(n.Strings) == 2 && callInvertBoolRex.MatchString(n.Strings[0]) && n.Strings[1] == "" {
		inferSoongVariableType("bool", n.Variables[0].Name)
	}

	// Is it $(filter [false|true],$(VAR)) or $(filter-out [false|true],$(VAR))
	if len(n.Strings) == 2 && callFilterBoolRex.MatchString(n.Strings[0]) && n.Strings[1] == "" {
		inferSoongVariableType("bool", n.Variables[0].Name)
	}
	return
}
