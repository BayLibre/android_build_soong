// Copyright 2020 Google Inc. All rights reserved.
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
	"regexp"
	"sync"
)

// Patterns for the values that can be specified in license property.
const (
	licenseRulePattern = `^(?:` + packagePattern + `)?(?:` + namePattern + `)$`
)

var licenseRuleRegexp = regexp.MustCompile(licenseRulePattern)

// An applicable licenses rule is associated with a module and describes the license terms that apply to the module.
type applicableLicensesRule interface {
	getLicenses() []string
}

// Describes the properties provided by a module that contains applicable licenses rules.
type applicableLicensesPropertyImpl struct {
	name             string
	licensesProperty *[]string
}

type applicableLicensesProperty interface {
	getName() string
	getLicenses() []string
}

func newApplicableLicensesProperty(name string, licensesProperty *[]string) applicableLicensesProperty {
	return applicableLicensesPropertyImpl{
		name: name,
		licensesProperty: licensesProperty,
	}
}

func (p applicableLicensesPropertyImpl) getName() string {
	return p.name
}

func (p applicableLicensesPropertyImpl) getLicenses() []string {
	return *p.licensesProperty
}

type licensesRule struct {
	licenses []string
}

func newLicensesSet(s map[string]bool) licensesRule {
	var r licensesRule
	r.licenses = make([]string,0,len(s))
	for l := range s {
		r.licenses = append(r.licenses, l)
	}
	return r
}

func (r licensesRule) getLicenses() []string {
	return r.licenses
}

var applicableLicensesRuleMap = NewOnceKey("applicableLicensesRuleMap")

// The map from qualifiedModuleName to licenseRules.
func moduleToApplicableLicensesRuleMap(config Config) *sync.Map {
	return config.Once(applicableLicensesRuleMap, func() interface{} {
		return &sync.Map{}
	}).(*sync.Map)
}

var inheritedLicensesRuleMap = NewOnceKey("inheritedLicensesRuleMap")

// The map from qualifiedModuleName to licenseRules from dependencies.
func moduleToInheritedLicensesRuleMap(config Config) *sync.Map {
	return config.Once(inheritedLicensesRuleMap, func() interface{} {
		return &sync.Map{}
	}).(*sync.Map)
}

// The rule checker needs to be registered before defaults expansion to correctly check that
// //visibility:xxx isn't combined with other packages in the same list in any one module.
func RegisterLicenseRuleChecker(ctx RegisterMutatorsContext) {
	ctx.BottomUp("licensesRuleChecker", licensesRuleChecker).Parallel()
}

// Registers the function that gathers the applicable licenses rules for each module.
//
// License is not dependent on arch so this must be registered before the arch phase to avoid
// having to process multiple variants for each module. This goes after defaults expansion to gather
// the complete license rule lists from flat lists and after the package info is gathered to ensure
// that default_applicable_licenses is available.
func RegisterLicenseRuleGatherer(ctx RegisterMutatorsContext) {
	ctx.BottomUp("licensesRuleGatherer", licensesRuleGatherer).Parallel()
}

// This must be registered after the deps have been resolved.
func RegisterLicensesRuleEnforcer(ctx RegisterMutatorsContext) {
	ctx.BottomUp("licensesRuleEnforcer", licensesRuleEnforcer).Parallel()
}

// Checks the per-module license rule lists before defaults expansion.
func licensesRuleChecker(ctx BottomUpMutatorContext) {
	qualified := createQualifiedModuleName(ctx)
	if m, ok := ctx.Module().(Module); ok {
		applicableLicensesProperties := m.applicableLicensesProperties()
		for _, p := range applicableLicensesProperties {
			if licenses := p.getLicenses(); licenses != nil {
				checkLicensesRules(ctx, qualified.pkg, p.getName(), licenses)
			}
		}
	}
}

func checkLicensesRules(ctx BaseModuleContext, currentPkg, property string, licenses []string) {
	ruleCount := len(licenses)
	if ruleCount == 0 {
		// This prohibits an empty list as its meaning is unclear, e.g. it could mean no license and
		// it could mean default license. Requiring at least one rule makes the owner's intent
		// clearer.
		ctx.PropertyErrorf(property, "must contain at least one applicable licenses rule")
		return
	}

	for _, l := range licenses {
		ok, _, _ := splitLicensePath(ctx, l, currentPkg, property)
		if !ok {
			continue
		}
	}
}

// Gathers the flattened applicable licenses rules after defaults expansion, parses the license
// properties, stores them in a map by qualifiedModuleName for retrieval during enforcement.
func licensesRuleGatherer(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	qualifiedModuleId := m.qualifiedModuleId(ctx)
	currentPkg := qualifiedModuleId.pkg

	// Parse the licenses rules that apply to the module and store them by id
	// for use when enforcing the rules.
	primaryProperty := m.base().primaryLicensesProperty
	if primaryProperty != nil {
		if licenses := primaryProperty.getLicenses(); licenses != nil {
			rule := parseLicensesRules(ctx, currentPkg, primaryProperty.getName(), licenses)
			moduleToApplicableLicensesRuleMap(ctx.Config()).Store(qualifiedModuleId, rule)
		}
	}
}

func parseLicensesRules(ctx BaseModuleContext, currentPkg, property string, propVals []string) licensesRule {
	licenses := make([]string, 0, len(propVals))
	for _, l := range propVals {
		ok, pkg, name := splitLicensePath(ctx, l, currentPkg, property)
		if !ok {
			continue
		}

		licenses = append(licenses, "//" + pkg + ":" + name)
	}

	return licensesRule{licenses}
}

func splitLicensePath(ctx BaseModuleContext, ruleExpression string, currentPkg, property string) (bool, string, string) {
	// Make sure that the rule is of the correct format.
	matches := licenseRuleRegexp.FindStringSubmatch(ruleExpression)
	if ruleExpression == "" || matches == nil {
		// License rule is invalid so ignore it. Keep going rather than aborting straight away to
		// ensure all the rules on this module are checked.
		ctx.PropertyErrorf(property,
			"invalid license pattern %q must match"+
				" //<package>:<module> or :<module>",
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

	return true, pkg, name
}

func licensesRuleEnforcer(ctx BottomUpMutatorContext) {
	if _, ok := ctx.Module().(Module); !ok {
		return
	}

	qualified := createQualifiedModuleName(ctx)
	s := make(map[string]bool)

	// Visit all the dependencies making sure that this module has access to them all.
	ctx.VisitDirectDeps(func(dep Module) {
		depName := ctx.OtherModuleName(dep)
		depDir := ctx.OtherModuleDir(dep)
		depQualified := qualifiedModuleName{depDir, depName}

		rule := effectiveApplicableLicensesRule(ctx.Config(), depQualified)
		if rule != nil {
			for _, l := range rule.getLicenses() {
				s[l] = true
			}
		}
	})
	moduleToInheritedLicensesRuleMap(ctx.Config()).Store(qualified, newLicensesSet(s))
}

func effectiveApplicableLicensesRule(config Config, qualified qualifiedModuleName) applicableLicensesRule {
	moduleToApplicableLicensesRule := moduleToApplicableLicensesRuleMap(config)
	value, ok := moduleToApplicableLicensesRule.Load(qualified)
	var rule applicableLicensesRule
	if ok {
		rule = value.(applicableLicensesRule)
	} else {
		rule = packageDefaultApplicableLicenses(config, qualified)
	}
	return rule
}

func packageDefaultApplicableLicenses(config Config, moduleId qualifiedModuleName) applicableLicensesRule {
	moduleToApplicableLicensesRule := moduleToApplicableLicensesRuleMap(config)
	packageQualifiedId := moduleId.getContainingPackageId()
	for {
		value, ok := moduleToApplicableLicensesRule.Load(packageQualifiedId)
		if ok {
			return value.(applicableLicensesRule)
		}

		if packageQualifiedId.isRootPackage() {
			return nil
		}

		packageQualifiedId = packageQualifiedId.getContainingPackageId()
	}
}

// Get the effective license rules, i.e. the actual rules that apply to the module
// property irrespective of where they are defined.
//
// Includes applicable licenses rules specified by package default_applicable_licenses and/or on defaults.
// Short hand forms, e.g. //:my_license are replaced with their full form, e.g.
// //package/containing/rule:my_license.
func EffectiveApplicableLicenses(ctx BaseModuleContext, module Module) []string {
	moduleName := ctx.OtherModuleName(module)
	dir := ctx.OtherModuleDir(module)
	qualified := qualifiedModuleName{dir, moduleName}

	s := make(map[string]bool)
	rule := effectiveApplicableLicensesRule(ctx.Config(), qualified)
	if rule != nil {
		for _, l := range rule.getLicenses() {
			s[l] = true
		}
	}

	moduleToInheritedLicensesRule := moduleToInheritedLicensesRuleMap(ctx.Config())
	value, ok := moduleToInheritedLicensesRule.Load(qualified)
	if ok {
		rule = value.(applicableLicensesRule)
		for _, l := range rule.getLicenses() {
			s[l] = true
		}
	}
	return newLicensesSet(s).getLicenses()
}

// Clear the default applicable licenses properties so they can be replaced.
func clearApplicableLicensesProperties(module Module) {
	module.base().applicableLicensesPropertyInfo = nil
}

// Add a property that contains applicable licenses rules so that they are checked for
// correctness.
func AddApplicableLicensesProperty(module Module, name string, licensesProperty *[]string) {
	addApplicableLicensesProperty(module, name, licensesProperty)
}

func addApplicableLicensesProperty(module Module, name string, licensesProperty *[]string) applicableLicensesProperty {
	base := module.base()
	property := newApplicableLicensesProperty(name, licensesProperty)
	base.applicableLicensesPropertyInfo = append(base.applicableLicensesPropertyInfo, property)
	return property
}

// Set the primary applicable licenses property.
//
// Also adds the property to the list of properties to be validated.
func setPrimaryLicensesProperty(module Module, name string, licensesProperty *[]string) {
	module.base().primaryLicensesProperty = addApplicableLicensesProperty(module, name, licensesProperty)
}
