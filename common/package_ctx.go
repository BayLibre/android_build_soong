// Copyright 2015 Google Inc. All rights reserved.
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

package common

import (
	"github.com/google/blueprint"
)

// AndroidPackageContext is a wrapper for blueprint.PackageContext that adds
// some android-specific helper functions.
type AndroidPackageContext struct {
	blueprint.PackageContext
}

func NewPackageContext(pkgPath string) AndroidPackageContext {
	return AndroidPackageContext{blueprint.NewPackageContext(pkgPath)}
}

// SourcePathVariable returns a Variable whose value is the source directory
// appended with the supplied path. It may only be called during a Go package's
// initialization - either from the init() function or as part of a
// package-scoped variable's initialization.
func (p AndroidPackageContext) SourcePathVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(config interface{}) (string, error) {
		p, err := safePathForSourceConfig(config.(Config), path)
		if err != nil {
			return "", err
		}
		return p.String(), nil
	})
}

// HostBinVariable returns a Variable whose value is the result of
// Config.HostBinTool(path). It may only be called during a Go package's
// initialization - either from the init() function or as part of a
// package-scoped variable's initialization.
func (p AndroidPackageContext) HostBinToolVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(config interface{}) (string, error) {
		p, err := config.(Config).HostBinTool(path)
		if err != nil {
			return "", err
		}
		return p.String(), nil
	})
}

// HostJavaToolVariable returns a Variable whose value is the result of
// Config.HostJavaTool(path). It may only be called during a Go package's
// initialization - either from the init() function or as part of a
// package-scoped variable's initialization.
func (p AndroidPackageContext) HostJavaToolVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(config interface{}) (string, error) {
		p, err := config.(Config).HostJavaTool(path)
		if err != nil {
			return "", err
		}
		return p.String(), nil
	})
}

// HostJavaToolVariable returns a Variable whose value is the intermediate
// directory appended with the supplied path. It may only be called during a Go
// package's initialization - either from the init() function or as part of a
// package-scoped variable's initialization.
func (p AndroidPackageContext) IntermediatesPathVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(config interface{}) (string, error) {
		p, err := PathForIntermediatesConfig(config.(Config), path)
		if err != nil {
			return "", err
		}
		return p.String(), err
	})
}
