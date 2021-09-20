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

package bazel

import (
	"github.com/google/blueprint"
)

// moduleInfo implements blueprint.Module interface with sufficient information to mock a subset of
// a blueprint ModuleContext
type moduleInfo struct {
	name string
	typ  string
	dir  string
}

// Name returns name for moduleInfo -- required to implement blueprint.Module
func (mi moduleInfo) Name() string {
	return mi.name
}

// GenerateBuildActions unused, but required to implmeent blueprint.Module
func (mi moduleInfo) GenerateBuildActions(blueprint.ModuleContext) {}

func (mi moduleInfo) equals(other moduleInfo) bool {
	return mi.name == other.name && mi.typ == other.typ && mi.dir == other.dir
}

// ensure moduleInfo implements blueprint.Module
var _ blueprint.Module = moduleInfo{}

// otherModuleContext is a mock context that implements OtherModuleContext
type otherModuleContext struct {
	modules []moduleInfo
}

// ModuleFromName retrieves the moduleInfo corresponding to name, if it exists
func (omc otherModuleContext) ModuleFromName(name string) (blueprint.Module, bool) {
	for _, m := range omc.modules {
		if m.name == name {
			return m, true
		}
	}
	return moduleInfo{}, false
}

// moduleInfo returns the moduleInfo corresponding to a blueprint.Module if it exists in omc
func (omc otherModuleContext) moduleInfo(m blueprint.Module) (moduleInfo, bool) {
	mi, ok := m.(moduleInfo)
	if !ok {
		return moduleInfo{}, false
	}
	for _, other := range omc.modules {
		if other.equals(mi) {
			return mi, true
		}
	}
	return moduleInfo{}, false
}

// OtherModuleType returns type of m if it exists in omc
func (omc otherModuleContext) OtherModuleType(m blueprint.Module) string {
	if mi, ok := omc.moduleInfo(m); ok {
		return mi.typ
	}
	return ""
}

// OtherModuleName returns name of m if it exists in omc
func (omc otherModuleContext) OtherModuleName(m blueprint.Module) string {
	if mi, ok := omc.moduleInfo(m); ok {
		return mi.name
	}
	return ""
}

// OtherModuleDir returns dir of m if it exists in omc
func (omc otherModuleContext) OtherModuleDir(m blueprint.Module) string {
	if mi, ok := omc.moduleInfo(m); ok {
		return mi.dir
	}
	return ""
}

// Ensure otherModuleContext implements OtherModuleContext
var _ OtherModuleContext = otherModuleContext{}
