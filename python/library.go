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

package python

// This file contains the module types for building Python library.

import (
	"fmt"

	"android/soong/android"
	"android/soong/bazel"
	"github.com/google/blueprint/proptools"
)

func init() {
	registerPythonLibraryComponents(android.InitRegistrationContext)
	//android.RegisterBp2BuildMutator("python_library_host", PythonLibraryHostBp2Build)
	android.RegisterBp2BuildMutator("python_library", PythonLibraryBp2Build)
}

func registerPythonLibraryComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("python_library_host", PythonLibraryHostFactory)
	ctx.RegisterModuleType("python_library", PythonLibraryFactory)
}

func PythonLibraryHostFactory() android.Module {
	module, _ := NewLibrary(android.HostSupported)

	//android.InitBazelModule(module)

	return module.init()
}

type bazelPythonLibraryAttributes struct {
	Srcs           bazel.LabelListAttribute
	Data           bazel.LabelListAttribute
	Python_version string
}

func PythonLibraryBp2Build(ctx android.TopDownMutatorContext) {
	m, ok := ctx.Module().(*Module)
	if !ok || !m.ConvertWithBp2build(ctx) {
		return
	}

	// a Module can be something other than a python_library
	if ctx.ModuleType() != "python_library" {
		return
	}

	// TODO(b/182306917): this doesn't fully handle all nested props versioned
	// by the python version, which would have been handled by the version split
	// mutator. This is sufficient for very simple python_library modules under
	// Bionic.
	py3Enabled := proptools.BoolDefault(m.properties.Version.Py3.Enabled, false)
	py2Enabled := proptools.BoolDefault(m.properties.Version.Py2.Enabled, false)
	var python_version string
	if py3Enabled && py2Enabled {
		panic(fmt.Errorf(
			"error for '%s' module: bp2build's python_library converter does not support "+
					"converting a module that is enabled for both Python 2 and 3 at the same time.", m.Name()))
	} else if py2Enabled {
		python_version = "PY2"
	} else {
		// do nothing, since python_version defaults to PY3.
	}

	srcs := android.BazelLabelForModuleSrcExcludes(ctx, m.properties.Srcs, m.properties.Exclude_srcs)
	data := android.BazelLabelForModuleSrc(ctx, m.properties.Data)

	attrs := &bazelPythonLibraryAttributes{
		Srcs:           bazel.MakeLabelListAttribute(srcs),
		Data:           bazel.MakeLabelListAttribute(data),
		Python_version: python_version,
	}

	props := bazel.BazelTargetModuleProperties{
		// Use the native py_library rule.
		Rule_class: "py_library",
	}

	ctx.CreateBazelTargetModule(m.Name(), props, attrs)
}

type LibraryProperties struct {
	// set the name of the output library.
	Stem *string `android:"arch_variant"`

	// append to the name of the output library.
	Suffix *string `android:"arch_variant"`

	// list of compatibility suites (for example "cts", "vts") that the module should be
	// installed into.
	Test_suites []string `android:"arch_variant"`

	// Flag to indicate whether or not to create test config automatically. If AndroidTest.xml
	// doesn't exist next to the Android.bp, this attribute doesn't need to be set to true
	// explicitly.
	Auto_gen_config *bool
}

// Currently superfluous, but for future expandability and modularity
type libraryDecorator struct {
	libraryProperties LibraryProperties
}

func NewLibrary(hod android.HostOrDeviceSupported) (*Module, *libraryDecorator) {
	module := newModule(hod, android.MultilibFirst)
	decorator := &libraryDecorator{}
	return module, decorator
}

func PythonLibraryFactory() android.Module {
	module, _ := NewLibrary(android.HostSupported)

	android.InitBazelModule(module)

	return module.init()
}

// get host interpreter name.
/*func (library *libraryDecorator) getHostInterpreterName(ctx android.ModuleContext,
		actualVersion string) string {
	var interp string
	switch actualVersion {
	case pyVersion2:
		interp = "python2.7"
	case pyVersion3:
		interp = "python3"
	default:
		panic(fmt.Errorf("unknown Python actualVersion: %q for module: %q.",
			actualVersion, ctx.ModuleName()))
	}

	return interp
}*/

/*func (library *libraryDecorator) getStem(ctx android.ModuleContext) string {
	stem := ctx.ModuleName()
	if String(library.libraryProperties.Stem) != "" {
		stem = String(library.libraryProperties.Stem)
	}

	return stem + String(library.libraryProperties.Suffix)
}*/