// Copyright 2019 Google Inc. All rights reserved.
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

package android

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Patterns for the values that can be specified in visibility property.
const (
	packagePattern        = namespacePrefix + `([^/:]+(?:/[^/:]+)*)`
	namePattern           = modulePrefix + `([^/:]+)`
	visibilityRulePattern = `(?:` + packagePattern + `)?(?:` + namePattern + `)?`
)

var visibilityRuleRegexp = regexp.MustCompile(visibilityRulePattern)

// Qualified id for a module
type qualifiedModule struct {
	// The package (i.e. directory) in which the module is defined, without trailing /
	pkg string

	// The name of the module.
	name string
}

func (q qualifiedModule) String() string {
	return fmt.Sprintf("//%s:%s", q.pkg, q.name)
}

// Visibility rule
type visibilityRule interface {
	// Check to see whether this rules matches m.
	// Returns true if it does, false otherwise.
	matches(m qualifiedModule) bool

	String() string
}

// A compositeRule is a visibility rule composed from other visibility rules.
type compositeRule struct {
	rules []visibilityRule
}

// A compositeRule matches if and only if any of its rules matches.
func (c compositeRule) matches(m qualifiedModule) bool {
	for _, r := range c.rules {
		if r.matches(m) {
			return true
		}
	}
	return false
}

func (r compositeRule) String() string {
	s := "["
	sep := ""
	for _, r := range r.rules {
		s += sep + r.String()
		sep = ", "
	}
	s += "]"
	return s
}

// A packageRule is a visibility rule that matches modules in a specific package (i.e. directory).
type packageRule struct {
	pkg string
}

func (r packageRule) matches(m qualifiedModule) bool {
	return m.pkg == r.pkg
}

func (r packageRule) String() string {
	return fmt.Sprintf("//%s:__pkg__", r.pkg)
}

// A subpackagesRule is a visibility rule that matches modules in a specific package (i.e.
// directory) or any of its subpackages (i.e. subdirectories).
type subpackagesRule struct {
	pkg string
}

func (r subpackagesRule) matches(m qualifiedModule) bool {
	mp := m.pkg + "/"
	rp := r.pkg + "/"
	return strings.HasPrefix(mp, rp)
}

func (r subpackagesRule) String() string {
	return fmt.Sprintf("//%s:__subpackages__", r.pkg)
}

// The map from qualifiedModule to visibilityRule.
var moduleToVisibilityRule = sync.Map{}

func registerVisibilityMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("visibilityRuleGatherer", visibilityRuleGatherer).Parallel()
	ctx.TopDown("visibilityRuleEnforcer", visibilityRuleEnforcer).Parallel()
}

// Gathers the visibility rules, parses the visibility properties, stores them in a map by
// qualifiedModule for retrieval during enforcement.
//
// A visibility rule is of one of the following formats: //<package>:<name>, //<package> or :<name>
//
// Where:
//     <package> is the name of the package (i.e. path from root directory) to which the rules gives access
//     <name> is the name of the target within the package.
//     //<package> is short hand for //<package>:__pkg__
//     :<name> is short hand for //<current package>:<name>
//
// <name> can be one of the following:
//    __pkg__         - which gives all the targets in the package access
//    __subpackages__ - which gives all the targets in the package and any of its subpackages access
//
// There are two special forms:
//    //visibility:private - which gives all the targets in the current package access
//    //visibility:public - which gives all targets access - the default
//
// The //visibility:private and //visibility:public rules cannot be mixed with other rules

func visibilityRuleGatherer(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	moduleName := m.Name()
	dir := ctx.ModuleDir()
	qualified := qualifiedModule{dir, moduleName}

	visibility := m.base().commonProperties.Visibility
	if visibility != nil {
		rule := parseRules(ctx, dir, visibility)
		if rule != nil {
			moduleToVisibilityRule.Store(qualified, rule)
		}
	}
}

func parseRules(ctx BottomUpMutatorContext, currentPkg string, visibility []string) visibilityRule {
	ruleCount := len(visibility)
	if ruleCount == 0 {
		ctx.PropertyErrorf("visibility", "must contain at least one visibility rule")
		return nil
	}

	rules := make([]visibilityRule, 0, ruleCount+1)
	for _, v := range visibility {
		ok, pkg, name := splitRule(ctx, v, currentPkg, ruleCount)
		if !ok {
			// Visibility rule was invalid so ignore it.
			continue
		}

		if pkg == "visibility" {
			if ruleCount != 1 {
				ctx.PropertyErrorf("visibility", "cannot mix %q with any other visibility rules", v)
			}
			switch name {
			case "private":
				return packageRule{currentPkg}
			case "public":
				return nil
			default:
				ctx.PropertyErrorf("visibility", "unrecognized visibility rule %q", v)
				return nil
			}
		}

		// Create the rule
		var r visibilityRule
		switch name {
		case "__pkg__":
			r = packageRule{pkg}
		case "__subpackages__":
			r = subpackagesRule{pkg}
		default:
			ctx.PropertyErrorf("visibility", "unrecognized visibility rule %q", v)
			return nil
		}

		if r != nil {
			rules = append(rules, r)
		}
	}

	// Targets are always visible to other targets in their own package.
	rules = append(rules, packageRule{currentPkg})

	return compositeRule{rules}
}

func splitRule(ctx BaseModuleContext, ruleExpression string, currentPkg string, ruleCount int) (bool, string, string) {
	// Make sure that the rule is of the correct format.
	if ruleExpression == "" || !visibilityRuleRegexp.MatchString(ruleExpression) {
		ctx.PropertyErrorf("visibility", "invalid visibility pattern %q should match %q",
			ruleExpression, visibilityRulePattern)
		return false, "", ""
	}

	// Extract the package and name.
	matches := visibilityRuleRegexp.FindStringSubmatch(ruleExpression)
	pkg := matches[1]
	name := matches[2]

	// Normalize the short hands
	if pkg == "" {
		pkg = currentPkg
	}
	if name == "" {
		name = "__pkg__"
	}

	return true, pkg, name
}

func visibilityRuleEnforcer(ctx TopDownMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	moduleName := m.Name()
	dir := ctx.ModuleDir()
	qualified := qualifiedModule{dir, moduleName}

	// Visit all the dependencies making sure that this module has access to them all.
	ctx.VisitDirectDeps(func(dep Module) {
		d, ok := dep.(Module)
		if !ok {
			return
		}

		depName := d.Name()
		depDir := ctx.OtherModuleDir(d)
		depQualified := qualifiedModule{depDir, depName}

		rule, ok := moduleToVisibilityRule.Load(depQualified)
		if ok {
			if !rule.(visibilityRule).matches(qualified) {
				ctx.ModuleErrorf("depends on %s which is not visible to this module", depQualified)
			}
		}
	})
}
