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

// Enforces visibility rules between modules.
//
// Two stage process:
// * First stage works bottom up to extract visibility information from the modules, parse it,
//   create visibilityRule structures and store them in a map keyed by the module's qualifiedModule
//   instance, i.e. //<pkg>:<name>. The map is stored in the context rather than a global variable
//   for testing. Each test has its own Config so they do not share a map and so can be run in
//   parallel.
//
// * Second stage works top down and iterates over all the deps for each module. If the dep is in
//   the same package then it is automatically visible. Otherwise, for each dep it first extracts
//   its visibilityRule from the config map. If one could not be found then it assumes that it is
//   publicly visible. Otherwise, it calls the visibility rule to check that the module can see
//   the dependency. If it cannot then an error is reported.
//
// Prebuilts
//
// Prebuilts can be used to replace a non-prebuilt module. There are two separate ways that impacts
// visibility.
//
// Assume that alpha/bar -> beta/baz (so //beta/baz has //alpha:bar in its visibility rule).
//
// If a prebuilt is being used to replace beta/baz then it will need to have a visibility rule that
// allow alpha/bar to depend upon it. If beta/baz is present in the build then the prebuilt could
// potentially obtain the visibility rule from that module. However, if beta/baz is not present,
// e.g. because its git repo is not in the repo manifest then it cannot.
// todo - Come up with some concrete use cases that can be used to drive a solution.
//
// If a prebuilt is being used to replace alpha/bar then for checking visibility rules it would
// need to be as if it was the alpha/bar module. That requires that the prebuilt needs to know the
// package of the bar target. If the alpha/bar module is present in the build then that could be
// obtained by finding the module by name and getting its directory. However, if alpha/bar is not
// present then the prebuilt will need to explicitly specify the package/namespace for the module
// it is replacing. A similar issue applies when using a prebuilt to replace a module in a non-root
// namespace.
// todo - Support using prebuilts to replace modules in namespaces and then address this use case.

// Patterns for the values that can be specified in visibility property.
const (
	packagePattern        = namespacePrefix + `([^/:]+(?:/[^/:]+)*)`
	namePattern           = modulePrefix + `([^/:]+)`
	visibilityRulePattern = `^(?:` + packagePattern + `)?(?:` + namePattern + `)?$`
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

// A visibility rule is associated with a module and determines which other modules it is visible
// to, i.e. which other modules can depend on the rule's module.
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
	pkgPrefix string
}

func (r subpackagesRule) matches(m qualifiedModule) bool {
	return isAncestor(r.pkgPrefix, m.pkg)
}

func isAncestor(p1 string, p2 string) bool {
	return strings.HasPrefix(p2+"/", p1+"/")
}

func (r subpackagesRule) String() string {
	return fmt.Sprintf("//%s:__subpackages__", r.pkgPrefix)
}

var visibilityRuleMap = NewOnceKey("visibilityRuleMap")

// The map from qualifiedModule to visibilityRule.
func moduleToVisibilityRuleMap(ctx BaseModuleContext) *sync.Map {
	return ctx.Config().Once(visibilityRuleMap, func() interface{} {
		return &sync.Map{}
	}).(*sync.Map)
}

func registerVisibilityMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("visibilityRuleGatherer", visibilityRuleGatherer).Parallel()
	ctx.TopDown("visibilityRuleEnforcer", visibilityRuleEnforcer).Parallel()
}

// Gathers the visibility rules, parses the visibility properties, stores them in a map by
// qualifiedModule for retrieval during enforcement.
//
// See ../README.md#Visibility for information on the format of the visibility rules.

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
		rule := parseRules(ctx, dir, visibility, dir)
		if rule != nil {
			moduleToVisibilityRuleMap(ctx).Store(qualified, rule)
		}
	}
}

func parseRules(ctx BottomUpMutatorContext, currentPkg string, visibility []string,
	thisPackage string) visibilityRule {
	ruleCount := len(visibility)
	if ruleCount == 0 {
		ctx.PropertyErrorf("visibility", "must contain at least one visibility rule")
		return nil
	}

	rules := make([]visibilityRule, 0, ruleCount)
	for _, v := range visibility {
		ok, pkg, name := splitRule(ctx, v, currentPkg)
		if !ok {
			// Visibility rule was invalid so ignore it, an error has already been logged.
			continue
		}

		if pkg == "visibility" {
			if ruleCount != 1 {
				ctx.PropertyErrorf("visibility", "cannot mix %q with any other visibility rules", v)
				continue
			}
			switch name {
			case "private":
				return packageRule{currentPkg}
			case "public":
				return nil
			case "legacy_public":
				ctx.PropertyErrorf("visibility", "//visibility:legacy_public must not be used")
				return nil
			default:
				ctx.PropertyErrorf("visibility", "unrecognized visibility rule %q", v)
				continue
			}
		}

		// If the current directory is not in the vendor tree then there are some additional
		// restrictions on the rules.
		if !isAncestor("vendor", currentPkg) {
			if !isAllowedFromOutsideVendor(pkg, name) {
				ctx.PropertyErrorf("visibility",
					"%q is not allowed. Packages outside //vendor cannot make themselves visible to specific"+
						" targets within //vendor, they can only use //vendor:__subpackages__.", v)
				continue
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
			continue
		}

		rules = append(rules, r)
	}

	switch len(rules) {
	case 0:
		return nil
	case 1:
		return rules[0]
	default:
		return compositeRule{rules}
	}
}

func isAllowedFromOutsideVendor(pkg string, name string) bool {
	if pkg == "vendor" {
		if name == "__subpackages__" {
			return true
		}
		return false
	}

	return !isAncestor("vendor", pkg)
}

func splitRule(ctx BaseModuleContext, ruleExpression string, currentPkg string) (bool, string, string) {
	// Make sure that the rule is of the correct format.
	matches := visibilityRuleRegexp.FindStringSubmatch(ruleExpression)
	if ruleExpression == "" || matches == nil {
		ctx.PropertyErrorf("visibility",
			"invalid visibility pattern %q must match"+
				" //<namespace>:<module>, //<namespace> or :<module>",
			ruleExpression)
		return false, "", ""
	}

	// Extract the package and name.
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

	moduleToVisibilityRule := moduleToVisibilityRuleMap(ctx)

	// Visit all the dependencies making sure that this module has access to them all.
	ctx.VisitDirectDeps(func(dep Module) {
		depName := dep.Name()
		depDir := ctx.OtherModuleDir(dep)
		depQualified := qualifiedModule{depDir, depName}

		// Targets are always visible to other targets in their own package.
		if depDir == dir {
			return
		}

		rule, ok := moduleToVisibilityRule.Load(depQualified)
		if ok {
			if !rule.(visibilityRule).matches(qualified) {
				ctx.ModuleErrorf("depends on %s which is not visible to this module", depQualified)
			}
		}
	})
}
