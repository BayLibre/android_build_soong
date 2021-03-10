// Copyright 2021 Google Inc. All rights reserved.
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
	"android/soong/bazel"
	"strings"
)

// BazelModuleBase contains the property structs with metadata for modules which can be converted to
// Bazel.
type BazelModuleBase struct {
	bazelProperties bazel.Properties
}

// Bazelable is specifies the interface for modules that can be converted to Bazel.
type Bazelable interface {
	bazelProps() *bazel.Properties
	GetBazelLabel() string
	ConvertWithBp2build(TopDownMutatorContext) bool
}

// BazelModule is a lightweight wrapper interface around Module for Bazel-convertible modules.
type BazelModule interface {
	Module
	Bazelable
}

// InitBazelModule is a wrapper function that decorates a BazelModule with Bazel-conversion
// properties.
func InitBazelModule(module BazelModule) {
	module.AddProperties(module.bazelProps())
}

// bazelProps returns the Bazel properties for the given BazelModuleBase.
func (b *BazelModuleBase) bazelProps() *bazel.Properties {
	return &b.bazelProperties
}

// GetBazelLabel returns the Bazel label for the given BazelModuleBase.
func (b *BazelModuleBase) GetBazelLabel() string {
	return b.bazelProperties.Bazel_module.Label
}

type Bp2BuildPackageConversionConfigEntry int
type Bp2BuildPackageConversionConfig map[string]Bp2BuildPackageConversionConfigEntry

const (
	// iota + 1 ensures that the int value is not 0 when used in the Bp2buildAllowlist map,
	// which can also mean that the key doesn't exist in a lookup.
	AllModules  Bp2BuildPackageConversionConfigEntry = iota + 1 // convert all modules in this package
	ModuleOptIn                                                 // rely on the bp2build_available property on the module.
)

func makePackageConversionConfig(config Config) Bp2BuildPackageConversionConfig {
	packageConversionConfig := Bp2BuildPackageConversionConfig{}

	for _, p := range config.productVariables.Bp2BuildAllModules {
		packageConversionConfig[p] = AllModules
	}

	for _, p := range config.productVariables.Bp2BuildOptInModules {
		packageConversionConfig[p] = ModuleOptIn
	}

	return packageConversionConfig
}

// ConvertWithBp2build returns whether the given BazelModuleBase should be converted with bp2build.
func (b *BazelModuleBase) ConvertWithBp2build(ctx TopDownMutatorContext) bool {
	// |             | bp2build_available: true | bp2build_available: false |
	// | AllModules  | convert                  | convert                   |
	// | ModuleOptIn | convert                  | don't convert             |
	packagePath := ctx.ModuleDir()
	if convertAllModulesInPackage(packagePath, makePackageConversionConfig(ctx.Config())) {
		// allowlisted by package.
		return true
	}
	// decide at the module level.
	return b.bazelProperties.Bazel_module.Bp2build_available
}

// convertAllModulesInPackage checks that the package is
// convertAllModulesInPackage in in the set of prefixes. That is, if the package
// is x/y/z, and the list allows x, x/y, or x/y/z, this function will return
// true.
//
// However, if the package is x/y, and the set contains just x/y/z, this
// function returns false since x/y/z is _not_ a prefix of x/y.
func convertAllModulesInPackage(packagePath string, packageConversionConfig Bp2BuildPackageConversionConfig) bool {
	ret := false

	if packageConversionConfig[packagePath] == ModuleOptIn {
		return false
	}

	packagePrefix := ""
	// e.g. for x/y/z, iterate over x, x/y, then x/y/z, taking the final value from the allowlist.
	for _, part := range strings.Split(packagePath, "/") {
		packagePrefix += part
		if packageConversionConfig[packagePrefix] == AllModules {
			// package contains this prefix and this prefix should convert all modules
			return true
		}
		// Continue to the next part of the package dir.
		packagePrefix += "/"
	}

	return ret
}
