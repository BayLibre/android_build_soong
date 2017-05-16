// Copyright 2017 Google Inc. All rights reserved.
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

// This file implements the logic of bpfix and also provides a programmatic interface

package bpfix

import (
	"github.com/google/blueprint/parser"
)

// A FixRequest specifies the details of which fixes to apply to an individual file
// A FixRequest doesn't specify whether to do a dry run or where to write the results; that's in bpfix.go
type FixRequest struct {
	removeKnownUnnecessaryVariables bool
}

func FixEverythingRequest() *FixRequest {
	return &FixRequest{true}
}

func FixTree(tree *parser.File, config *FixRequest) (fixed *parser.File, errors []error) {
	if config == nil {
		config = FixEverythingRequest()
	}
	if config.removeKnownUnnecessaryVariables {
		tree, errors = RemoveKnownUnnecessaryVariables(tree)
		if len(errors) > 0 {
			return tree, errors
		}
	}
	return tree, errors
}

func RemoveKnownUnnecessaryVariables(tree *parser.File) (fixed *parser.File, errors []error) {
	// remove local_include_dirs if it matches export_include_dirs
	fixed, errors = removeIfMatching(tree, "export_include_dirs", "local_include_dirs")
	return fixed, errors
}

// remove <legacyName> from each module that has the same value for <canonicalName>
func removeIfMatching(tree *parser.File, canonicalName string, legacyName string) (fixed *parser.File, errors []error) {
	for _, def := range tree.Defs {
		if mod, ok := def.(*parser.Module); ok {
			if legacy, ok := mod.GetProperty(legacyName); ok {
				if canonical, ok := mod.GetProperty(canonicalName); ok {
					areEqual, err := parser.ExpressionsHaveSameMeaning(legacy.Value, canonical.Value)
					if err == nil {
						if areEqual {
							mod.Remove(legacy.Name)
						}
					}
				}
			}
		}
	}
	return tree, nil
}
