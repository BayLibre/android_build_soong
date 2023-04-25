// Copyright 2023 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint/parser"
)

func expressionContainsAnyString(expr parser.Expression, needles ...string) bool {
	switch e := expr.(type) {
	case *parser.String:
		return listContains(needles, e.Value)
	case *parser.List:
		for _, subExpr := range e.Values {
			if expressionContainsAnyString(subExpr, needles...) {
				return true
			}
			return false
		}
	}
	return false
}

func listContains[T comparable](haystack []T, needle T) bool {
	for _, x := range haystack {
		if x == needle {
			return true
		}
	}
	return false
}

func hasModuleWithName(haystack []*parser.Module, needle string) bool {
	for _, x := range haystack {
		if x.Name() == needle {
			return true
		}
	}
	return false
}

func moduleHasDefaultsWithName(mod *parser.Module, defaultsNames ...string) bool {
	for _, prop := range mod.Properties {
		if prop.Name == "defaults" {
			if expressionContainsAnyString(prop.Value, defaultsNames...) {
				return true
			}
			break
		}
	}
	return false
}

func realMain() error {
	flag.Parse()
	var parsedFiles []*parser.File
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if d.Name() == "Android.bp" {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			scope := parser.NewScope(nil)
			parsedFile, errs := parser.Parse(d.Name(), file, scope)
			if len(errs) > 0 {
				return errs[0]
			}
			parsedFiles = append(parsedFiles, parsedFile)
		}
		return nil
	})
	if err != nil {
		return err
	}

	var nonDefaultsRdeps []*parser.Module
	needles := []string{"art_defaults"}
	newDefaults := true
	for newDefaults {
		newDefaults = false
		for _, file := range parsedFiles {
			for _, def := range file.Defs {
				if d, ok := def.(*parser.Module); ok {
					if moduleHasDefaultsWithName(d, needles...) {
						if strings.HasSuffix(d.Type, "_defaults") {
							if !listContains(needles, d.Name()) {
								needles = append(needles, d.Name())
								newDefaults = true
							}
						} else {
							if !hasModuleWithName(nonDefaultsRdeps, d.Name()) {
								nonDefaultsRdeps = append(nonDefaultsRdeps, d)
							}
						}
					}
				}
			}
		}
	}

	for _, d := range nonDefaultsRdeps {
		fmt.Printf("%s of type %s\n", d.Name(), d.Type)
	}

	return nil
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
