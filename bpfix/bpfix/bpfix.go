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
	"fmt"
	"github.com/google/blueprint/parser"
)

// A FixRequest specifies the details of which fixes to apply to an individual file
// A FixRequest doesn't specify whether to do a dry run or where to write the results; that's in cmd/bpfix.go
type FixRequest struct {
	simplifyKnownRedundantVariables bool
	removeEmptyLists                bool
}

func NewFixRequest() FixRequest {
	return FixRequest{}
}

func (r FixRequest) AddAll() (result FixRequest) {
	result = r
	result.simplifyKnownRedundantVariables = true
	result.removeEmptyLists = true
	return result
}

// FixTree repeatedly applies the fixes listed in the given FixRequest to the given File
// until there is no fix that affects the tree
func FixTree(tree *parser.File, config FixRequest) (fixed *parser.File, errors []error) {
	prevIdentifier, err := fingerprint(tree)
	if err != nil {
		return nil, []error{err}
	}

	fixed = tree
	maxNumIterations := 20
	i := 0
	for {
		fixed, errors = fixTreeOnce(fixed, config)
		newIdentifier, err := fingerprint(tree)
		if err != nil {
			return nil, []error{err}
		}
		if newIdentifier == prevIdentifier {
			break
		}
		prevIdentifier = newIdentifier
		// any errors from a previous iteration generally get thrown away and overwritten by errors on the next iteration

		// detect infinite loop
		i++
		if i >= maxNumIterations {
			errors = append(errors, fmt.Errorf("Applied fixes %s times and yet the tree continued to change. Is there an infinite loop?", i))
			break
		}
	}
	return fixed, errors
}

// returns a unique identifier for the given tree that can be used to determine whether the tree changed
func fingerprint(tree *parser.File) (fingerprint string, err error) {
	bytes, err := parser.Print(tree)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func fixTreeOnce(tree *parser.File, config FixRequest) (fixed *parser.File, errors []error) {
	if config.simplifyKnownRedundantVariables {
		tree, errors = SimplifyKnownVariablesDuplicatingEachOther(tree)
		if len(errors) > 0 {
			return tree, errors
		}
	}
	if config.removeEmptyLists {
		tree, errors = RemoveVariablesHavingTheirDefaultValues(tree)
		if len(errors) > 0 {
			return tree, errors
		}
	}
	return tree, errors
}

func SimplifyKnownVariablesDuplicatingEachOther(tree *parser.File) (fixed *parser.File, errors []error) {
	// remove from local_include_dirs anything in export_include_dirs
	fixed, errors = removeMatchingModuleListProperties(tree, "export_include_dirs", "local_include_dirs")
	return fixed, errors
}

// returns a new list with all items in <removals> removed from <items>
func filterExpressionList(items []parser.Expression, removals []parser.Expression) (filtered []parser.Expression) {
	filtered = make([]parser.Expression, 0)
	for _, item := range items {
		included := true
		for _, removal := range removals {
			equal, err := parser.ExpressionsAreSame(item, removal)
			if err != nil {
				continue
			}
			if equal {
				included = false
				break
			}
		}
		if included {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// Remove each modules[i].Properties[<legacyName>][j] that matches a modules[i].Properties[<canonicalName>][k]
func removeMatchingModuleListProperties(tree *parser.File, canonicalName string, legacyName string) (fixed *parser.File, errors []error) {
	for _, def := range tree.Defs {
		mod, ok := def.(*parser.Module)
		if !ok {
			continue
		}
		legacy, ok := mod.GetProperty(legacyName)
		if !ok {
			continue
		}
		legacyList, ok := legacy.Value.(*parser.List)
		if !ok {
			continue
		}
		canonical, ok := mod.GetProperty(canonicalName)
		if !ok {
			continue
		}
		canonicalList, ok := canonical.Value.(*parser.List)
		if !ok {
			continue
		}
		newList := filterExpressionList(legacyList.Values, canonicalList.Values)
		if len(newList) == len(legacyList.Values) {
			// nothing was removed, so we don't have to replace the list
			continue
		}
		// replace the list
		legacyList.Values = newList
	}
	return tree, nil
}

func RemoveVariablesHavingTheirDefaultValues(tree *parser.File) (fixed *parser.File, errors []error) {
	for _, def := range tree.Defs {
		mod, ok := def.(*parser.Module)
		if !ok {
			continue
		}
		propsToKeep := make([]*parser.Property, 0)
		for _, prop := range mod.Properties {
			val := prop.Value
			keep := true
			switch val := val.(type) {
			case *parser.List:
				if len(val.Values) == 0 {
					keep = false
				}
				break
			default:
				keep = true
			}
			if keep {
				propsToKeep = append(propsToKeep, prop)
			}
		}
		// if at least one property was removed, then replace the list
		if len(mod.Properties) != len(propsToKeep) {
			mod.Properties = propsToKeep
		}
	}
	return tree, nil
}
