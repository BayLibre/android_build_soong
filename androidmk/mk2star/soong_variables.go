package mk2star

import (
	mkparser "android/soong/androidmk/parser"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var callFuncRex = regexp.MustCompile("^call +add_json_(str|val|bool|csv|list|map) *,")
var (
	knownBoolCalls  = regexp.MustCompile("^(call +invert_bool|(filter|filter-out) +(true|false)),$")
	knownOtherCalls = regexp.MustCompile("^(filter|filter-out) +.*,$")
)

// Scans the makefile Soong uses to generate soong.variables file,
// collecting variable names and types from the lines that look like this:
//    $(call add_json_XXX,  <...>,             $(VAR))
//
func FindSoongVariables(mkFile string, includeFileScope mkparser.Scope, registrar VariableRegistrar) error {
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
			handleVariable(registrar, t)
		case *mkparser.Directive:
			handleInclude(registrar, t, includeFileScope)
		}
	}
	return nil
}

func handleInclude(registrar VariableRegistrar, t *mkparser.Directive, scope mkparser.Scope) {
	if t.Name != "include" && t.Name != "-include" {
		return
	}
	includedPath := t.Args.Value(scope)
	err := FindSoongVariables(includedPath, scope, registrar)
	if err != nil && t.Name == "include" {
		fmt.Fprintf(os.Stderr, "cannot include %s: %s", includedPath, err)
	}
}

func handleVariable(registrar VariableRegistrar, t *mkparser.Variable) {
	name := t.Name
	if !strings.HasPrefix(name.Strings[0], "call add_json") {
		return
	}
	match := callFuncRex.FindAllStringSubmatch(name.Strings[0], -1)
	if match == nil {
		panic(fmt.Errorf("cannot match the call: %s", name.Strings[0]))
	}
	flavor := match[0][1]
	// the name is a sequence constants and variables <C0> <V0> <C1> <V1> ...
	// Find <Ci> containing the second ',', then V<i> will be third arg
	var v *mkparser.Variable
	for i, cc := 0, 0; i < len(name.Variables); i++ {
		if cc += strings.Count(name.Strings[i], ","); cc >= 2 {
			v = &name.Variables[i]
			break
		}
	}
	if v == nil {
		return
	}
	if len(v.Name.Strings) == 1 {
		registrar.NewVariable(v.Name.Strings[0], flavor)
		return
	}
	v, inferredFlavor := unwrapKnownCalls(v)
	if len(v.Name.Strings) != 1 {
		fmt.Fprintf(os.Stderr, "Found %s: %s\n", v.Dump(), flavor)
		return
	}
	switch inferredFlavor {
	case "=":
	case "bool":
		if flavor != "bool" {
			fmt.Fprintf(os.Stderr, "Type mismatch for %s\n", v.Name.Strings[0])
		}
	default:
		flavor = ""
	}
	registrar.NewVariable(v.Name.Strings[0], flavor)
}

// Strips known call layers and returns a variable and its inferred type
// E.g.:
//    $(filter true, $(FOO)) returns [$(FOO), "bool"]
//    $(FOO) returns [$(FOO), "="]
//    $(filter eng, $(FOO)) returns [$(FOO), "?"
func unwrapKnownCalls(v *mkparser.Variable) (*mkparser.Variable, string) {
	switch len(v.Name.Strings) {
	case 1:
		return v, "="
	case 2:
		if v.Name.Strings[1] == "" {
			call := v.Name.Strings[0]
			if knownBoolCalls.MatchString(call) {
				subV, subT := unwrapKnownCalls(&v.Name.Variables[0])
				if subT == "bool" {
					return subV, subT
				} else {
					return subV, ""
				}
			} else if knownOtherCalls.MatchString(call) {
				subV, _ := unwrapKnownCalls(&v.Name.Variables[0])
				return subV, ""
			}
		}
		return v, ""
	default:
		return v, ""
	}
}
