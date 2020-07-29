// Copyright (C) 2020 The Android Open Source Project
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
	"path"
	"sync"

	"github.com/google/blueprint"
)

func init() {
	RegisterModuleType("source_tree", SourceTreeFactory)
}

type SourceTreeProperties struct {
	// The release version of the current source tree.
	// It sets the "release_version" properties of all modules in this directory and the
	// subdirectories unless the modules have their own "release_version".
	Release_version *string
}

type SourceTreeModule struct {
	ModuleBase

	properties SourceTreeProperties
}

type ReleaseVersionInterface interface {
	DefaultReleaseVersion() *string
}

func (s *SourceTreeModule) GenerateAndroidBuildActions(ctx ModuleContext) {
}

func (s *SourceTreeModule) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

// "source_tree" module defines the properties of the directory and its subdirectories.
func SourceTreeFactory() Module {
	module := &SourceTreeModule{}

	// source_tree module does not need to have a name property.
	// The names of the modules are assigned automatically.
	name := fmt.Sprintf("source_tree-%d", getUniqueId())
	module.nameProperties.Name = &name

	module.AddProperties(&module.properties)

	return module
}

func RegisterSourceTreeMutators(ctx RegisterMutatorsContext) {
	ctx.BottomUp("source_tree", sourceTreeMutator).Parallel()
	ctx.BottomUp("release_version", releaseVersionMutator).Parallel()
}

var sourceTreeIdLock sync.Mutex
var sourceTreeId = 0

func getUniqueId() int {
	sourceTreeIdLock.Lock()
	defer sourceTreeIdLock.Unlock()
	sourceTreeId++
	return sourceTreeId
}

var sourceTreeVersionLock sync.Mutex
var sourceTreeVersionKey = NewOnceKey("sourceTreeVersion")

func sourceTreeVersions(config Config) map[string]*string {
	return config.Once(sourceTreeVersionKey, func() interface{} {
		return make(map[string]*string)
	}).(map[string]*string)
}

// sourceTreeMutator reads the source_tree modules and update the sourceTreeVersions map
// that has directory strings of the source_tree modules as keys and the release_version
// string pointers as values.
func sourceTreeMutator(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(*SourceTreeModule)
	if !ok {
		return
	}
	dir := ctx.ModuleDir()
	if m.properties.Release_version == nil {
		ctx.ModuleErrorf("in %s must have \"release_version\" property", dir)
	}
	sourceTreeVersionMap := sourceTreeVersions(ctx.Config())
	if _, ok := sourceTreeVersionMap[dir]; ok {
		ctx.ModuleErrorf("in %s duplicates with the other \"source_tree\" module in the same directory", dir)
		return
	}

	sourceTreeVersionLock.Lock()
	defer sourceTreeVersionLock.Unlock()
	sourceTreeVersionMap[dir] = m.properties.Release_version
}

// releaseVersionMutator assigns the "release_version" property of each module that does
// not have the property with it. releaseVersionMutator searches the "release_version"
// value from the "source_tree" modules in the same directory and the parent directories
// and assigns the value to the "release_version" property of the current module.
func releaseVersionMutator(ctx BottomUpMutatorContext) {
	if ctx.Module().base().commonProperties.Release_version != nil {
		return
	}

	if m, ok := ctx.Module().(ReleaseVersionInterface); ok {
		if defaultVersion := m.DefaultReleaseVersion(); defaultVersion != nil {
			ctx.Module().base().commonProperties.Release_version = defaultVersion
			return
		}
	}

	dir := ctx.ModuleDir()
	sourceTreeVersionMap := sourceTreeVersions(ctx.Config())

	for dir != "." {
		if versionPtr, ok := sourceTreeVersionMap[dir]; ok {
			ctx.Module().base().commonProperties.Release_version = versionPtr
			return
		}
		dir = path.Dir(dir)
	}
}
