// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

type context struct {
	includeFileScope mkparser.Scope
	registrar        variableRegistrar
}

// Scans the makefile Soong uses to generate soong.variables file,
// collecting variable names and types from the lines that look like this:
//    $(call add_json_XXX,  <...>,             $(VAR))
//
func FindSoongVariables(mkFile string, includeFileScope mkparser.Scope, registrar variableRegistrar) error {
	ctx := context{includeFileScope, registrar}
	return ctx.doFind(mkFile)
}

func (ctx *context) doFind(mkFile string) error {
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
			ctx.handleVariable(t)
		case *mkparser.Directive:
			ctx.handleInclude(t)
		}
	}
	return nil
}

func (ctx context) NewSoongVariable(name, typeString string) {
	var valueType starlarkType
	switch typeString {
	case "bool":
		valueType = starlarkTypeBool
	case "csv":
		// Only PLATFORM_VERSION_ALL_CODENAMES, and it's a list
		valueType = starlarkTypeList
	case "list":
		valueType = starlarkTypeList
	case "str":
		valueType = starlarkTypeString
	case "val":
		// Only PLATFORM_SDK_VERSION uses this, and it's integer
		valueType = starlarkTypeInt
	default:
		panic(fmt.Errorf("unknown Soong variable type %s", typeString))
	}

	ctx.registrar.NewVariable(name, VarClassSoong, valueType)
}

func (ctx context) handleInclude(t *mkparser.Directive) {
	if t.Name != "include" && t.Name != "-include" {
		return
	}
	includedPath := t.Args.Value(ctx.includeFileScope)
	err := ctx.doFind(includedPath)
	if err != nil && t.Name == "include" {
		fmt.Fprintf(os.Stderr, "cannot include %s: %s", includedPath, err)
	}
}

func (ctx context) handleVariable(t *mkparser.Variable) {
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
	ctx.inferSoongVariableType(match[0][1], name.Variables[0].Name)
}

var (
	callInvertBoolRex = regexp.MustCompile("^call +invert_bool *, *$")
	callFilterBoolRex = regexp.MustCompile("^(call +(filter|filter-out) +(true|false)),$")
)

func (ctx context) inferSoongVariableType(vType string, n *mkparser.MakeString) {
	if n.Const() {
		ctx.NewSoongVariable(n.Strings[0], vType)
		return
	}
	// Is it $(call invert_bool, $(VAR))?
	if len(n.Strings) == 2 && callInvertBoolRex.MatchString(n.Strings[0]) && n.Strings[1] == "" {
		ctx.inferSoongVariableType("bool", n.Variables[0].Name)
	}

	// Is it $(filter [false|true],$(VAR)) or $(filter-out [false|true],$(VAR))
	if len(n.Strings) == 2 && callFilterBoolRex.MatchString(n.Strings[0]) && n.Strings[1] == "" {
		ctx.inferSoongVariableType("bool", n.Variables[0].Name)
	}
	return
}
