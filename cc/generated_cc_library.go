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

package cc

import (
	"android/soong/android"
)

type GeneratedCcLibraryModule struct {

	callbacks  GeneratedCcLibraryCallbacks
	moduleName string
}

// Data type returned from GenerateSourceBuildActions
type GeneratedCc {
	// Which source files to be generated
	sources []android.Path

	// Non-exported headers to be generated
	headers []android.Path

	// Exported headers to be generated (this library will make these available to dependencies.
	exportHeaders []android.Path

	// Header dirs (-I) to be re-exported
	headerDirs []android.Path
}

type GeneratedCcLibraryCallbacks interface {
	// Called from inside DepsMutator, gives a chance to AddDependencies
	DepsMutator(module *GeneratedCcLibraryModule, ctx android.BottomUpMutatorContext)

	// Called from inside GenerateAndroidBuildActions. Add the build rules to
	// make the srcjar, and return the path to it.
	GenerateSourceBuildActions(module *GeneratedCcLibraryModule, ctx android.ModuleContext) GeneratedCcFiles
}

// GeneratedCCLibraryModuleFactory provides a utility for modules that are generated
// source code, including ones outside the cc package to build libraries from that generated source.
//
// To use GeneratedCcLibraryModule, call GeneratedCcLibraryModuleFactory with
// a callback interface and a properties object to add to the module.
//
// These modules will have some properties blocked, and it will be an error if
// modules attempt to set them. See the list of property names in GeneratedAndroidBuildActions
// for the list of those properties.
//
// For simplicity GeneratedCcLibraryModules are always static.
func GeneratedCcLibraryModuleFactory(moduleName string, hod android.HostOrDeviceSupported, callbacks GeneratedCcLibraryCallbacks, properties interface{}) android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyStatic()
	//	No direct export into sdk
//	module.sdkMemberTypes = []android.SdkMemberType{staticLibrarySdkMemberType}
	module.bazelable = false
//	module.bazelHandler = &ccLibraryBazelHandler{module: module}
	return module.Init()
}
