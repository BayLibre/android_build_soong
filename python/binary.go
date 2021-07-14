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

// This file contains the module types for building Python binary.

import (
	"fmt"
	"os"
	"strings"

	"android/soong/android"
	"android/soong/bazel"

	"github.com/google/blueprint/proptools"
)

func init() {
	registerPythonBinaryComponents(android.InitRegistrationContext)
}

func registerPythonBinaryComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("python_binary_host", PythonBinaryHostFactory)
}

type bazelPythonBinaryAttributes struct {
	Main           *string
	Srcs           bazel.LabelListAttribute
	Deps           bazel.LabelListAttribute
	Python_version *string
}

func pythonBinaryBp2Build(ctx android.TopDownMutatorContext, m *Module) {
	var main *string
	for _, propIntf := range m.GetProperties() {
		if props, ok := propIntf.(*BinaryProperties); ok {
			// main is optional.
			if props.Main != nil {
				main = props.Main
				break
			}
		}
	}

	// TODO(b/182306917): this doesn't fully handle all nested props versioned
	// by the python version, which would have been handled by the version split
	// mutator. This is sufficient for very simple python_binary_host modules
	// under Bionic.
	py3Enabled := proptools.BoolDefault(m.properties.Version.Py3.Enabled, false)
	py2Enabled := proptools.BoolDefault(m.properties.Version.Py2.Enabled, false)
	var python_version *string
	if py3Enabled && py2Enabled {
		panic(fmt.Errorf(
			"error for '%s' module: bp2build's python_binary_host converter does not support "+
				"converting a module that is enabled for both Python 2 and 3 at the same time.", m.Name()))
	} else if py2Enabled {
		python_version = &pyVersion2
	} else {
		// do nothing, since python_version defaults to PY3.
	}

	baseAttrs := m.makeArchVariantBaseAttributes(ctx)
	attrs := &bazelPythonBinaryAttributes{
		Main:           main,
		Srcs:           baseAttrs.Srcs,
		Deps:           baseAttrs.Deps,
		Python_version: python_version,
	}

	props := bazel.BazelTargetModuleProperties{
		// Use the native py_binary rule.
		Rule_class: "py_binary",
	}

	ctx.CreateBazelTargetModule(props, android.CommonAttributes{
		Name: m.Name(),
		Data: baseAttrs.Data,
	}, attrs)
}

type BinaryProperties struct {
	// the name of the source file that is the main entry point of the program.
	// this file must also be listed in srcs.
	// If left unspecified, module name is used instead.
	// If name doesn’t match any filename in srcs, main must be specified.
	Main *string `android:"arch_variant"`

	// set the name of the output binary.
	Stem *string `android:"arch_variant"`

	// append to the name of the output binary.
	Suffix *string `android:"arch_variant"`

	// list of compatibility suites (for example "cts", "vts") that the module should be
	// installed into.
	Test_suites []string `android:"arch_variant"`

	// whether to use `main` when starting the executable. The default is true, when set to
	// false it will act much like the normal `python` executable, but with the sources and
	// libraries automatically included in the PYTHONPATH.
	Autorun *bool `android:"arch_variant"`

	// Flag to indicate whether or not to create test config automatically. If AndroidTest.xml
	// doesn't exist next to the Android.bp, this attribute doesn't need to be set to true
	// explicitly.
	Auto_gen_config *bool
}

type binaryDecorator struct {
	binaryProperties BinaryProperties

	*pythonInstaller
}

type IntermPathProvider interface {
	IntermPathForModuleOut() android.OptionalPath
}

var (
	StubTemplateHost = "build/soong/python/scripts/stub_template_host.txt"
)

type pythonBinaryBazelHandler struct {
	android.BazelHandler

	module *Module
}

func NewBinary(hod android.HostOrDeviceSupported) (*Module, *binaryDecorator) {
	module := newModule(hod, android.MultilibFirst)
	decorator := &binaryDecorator{pythonInstaller: NewPythonInstaller("bin", "bin64")}
	// NewPythonInstaller("bin", "bin")}

	module.bootstrapper = decorator
	module.installer = decorator
	module.bazelHandler = &pythonBinaryBazelHandler{module: module}

	return module, decorator
}

func PythonBinaryHostFactory() android.Module {
	module, _ := NewBinary(android.HostSupported)
	android.InitBazelModule(module)
	return module.init()
}

func (h *pythonBinaryBazelHandler) GenerateBazelBuildActions(
	ctx android.ModuleContext, label string) bool {
	// Catch if this module doesn't correspond to python_binary_host
	if ctx.ModuleType() != "python_binary_host" {
		return false
	}
	bazelCtx := ctx.Config().BazelContext

	info, ok, err := bazelCtx.GetPyBinInfo(label, android.GetConfigKey(ctx))
	if !ok {
		return false
	}
	if err != nil {
		ctx.ModuleErrorf("Failed to GetPyBinInfo for %s: %s", label, err.Error())
	}
	fmt.Fprintf(os.Stderr, "\nGot PyBinInfo for %s: %+v\n", label, info)

	//h.srcs = android.Paths{android.PathForBazelOut(ctx, info.Binary)}
	mod := h.module
	fmt.Fprintf(os.Stderr, "\n@PROPERTIES: %+v\n", mod.properties)
	// Our Bazel-ized mock version of GenerateAndroidBuildActions
	// TODO: Emulate the data property for java_data dependencies.

	// TODO: get pkgPath thru Bazel (once we've analog), not Soong properties
	pkgPath, ok := mod.validatePkgPath(ctx, mod.properties.Pkg_path)
	if !ok {
		return false
	}
	fmt.Fprintf(os.Stderr, "\n@pkgPath for %s: %s\n", label, pkgPath)

	binPath := android.PathForBazelOut(ctx, info.Binary)
	fmt.Fprintf(os.Stderr, "\n@binPath for %s: %s\n", label, binPath.String())

	libPaths := android.PathsForBazelOut(ctx, info.SharedLibs)
	fmt.Fprintf(os.Stderr, "\n@libPaths for %s: %+v\n", label, libPaths.Strings())

	// TODO: finish recreating generatePythonBuildActions
	// generate src:destination path mappings for this module
	//mod.genModulePathMappings(ctx, pkgPath, libPaths, android.Paths{binPath})

	// the zipfile of all source and data files
	zipPath := android.PathForBazelOut(ctx, info.SrcZip)
	fmt.Fprintf(os.Stderr, "\n@zipPath for %s: %s\n", label, zipPath.String())
	mod.srcsZip = zipPath

	// FIXME: libPaths doesn't give transitive deps, but e.g. FileProvider.files_to_build includes output files
	//mod.depsSrcsZips = android.PathsForModuleSrc(ctx, append(libPaths, info.Binary))
	mod.depsSrcsZips = android.Paths{zipPath} //append(libPaths, binPath)
	//mod.collectPathsFromTransitiveDeps(ctx)
	// TODO: recreate bootstrap
	// bootstrap the module, including resolving main file, getting launcher path, and
	// registering actions to build the par file
	// bootstrap returns the binary output path
	/*mod.installSource = mod.bootstrapper.bootstrap(ctx, mod.properties.Actual_version,
	mod.isEmbeddedLauncherEnabled(), mod.srcsPathMappings, mod.srcsZip, mod.depsSrcsZips)*/
	var launcherPath android.OptionalPath
	if mod.isEmbeddedLauncherEnabled() {
		fmt.Fprintf(os.Stderr, "@Embedded laucher enabled\n")
		ctx.VisitDirectDepsWithTag(launcherTag, func(m android.Module) {
			if provider, ok := m.(IntermPathProvider); ok {
				if launcherPath.Valid() {
					panic(fmt.Errorf("launcher path was found before: %q",
						launcherPath))
				}
				launcherPath = provider.IntermPathForModuleOut()
			}
		})
		fmt.Fprintf(os.Stderr, "\n@launcherPath for %s: %s\n", label, launcherPath.String())
	}

	d := mod.bootstrapper.(*binaryDecorator)
	stem := d.getStem(ctx)
	fmt.Fprintf(os.Stderr, "\n@stem for %s: %s\n", label, stem)
	interp := d.getHostInterpreterName(ctx, mod.properties.Actual_version)
	fmt.Fprintf(os.Stderr, "\n@interp for %s: %s\n", label, interp)
	_installSource := registerBuildActionForParFile(ctx, mod.isEmbeddedLauncherEnabled(), launcherPath,
		interp, binPath.String(), stem, android.Paths{zipPath})
	if _installSource.String() != binPath.String() {
		// Capture what Soong would've thought the binPath was going to be
		fmt.Fprintf(os.Stderr, "\n@_installSource: %s\n", _installSource)
	}
	// Recreate install
	mod.installSource = android.OptionalPathForPath(binPath) //zipPath)
	mod.installer.setAndroidMkSharedLibs(libPaths.Strings())
	mod.installer.install(ctx, binPath)
	//os.Exit(3)

	return true
}

func (binary *binaryDecorator) autorun() bool {
	return proptools.BoolDefault(binary.binaryProperties.Autorun, true)
}

func (binary *binaryDecorator) bootstrapperProps() []interface{} {
	return []interface{}{&binary.binaryProperties}
}

func (binary *binaryDecorator) bootstrap(ctx android.ModuleContext, actualVersion string,
	embeddedLauncher bool, srcsPathMappings []pathMapping, srcsZip android.Path,
	depsSrcsZips android.Paths) android.OptionalPath {
	main := ""
	if binary.autorun() {
		main = binary.getPyMainFile(ctx, srcsPathMappings)
	}

	var launcherPath android.OptionalPath
	if embeddedLauncher {
		ctx.VisitDirectDepsWithTag(launcherTag, func(m android.Module) {
			if provider, ok := m.(IntermPathProvider); ok {
				if launcherPath.Valid() {
					panic(fmt.Errorf("launcher path was found before: %q",
						launcherPath))
				}
				launcherPath = provider.IntermPathForModuleOut()
			}
		})
	}

	fmt.Fprintf(os.Stderr, "@BOOTSTRAPPING: %s\n", main)
	binFile := registerBuildActionForParFile(ctx, embeddedLauncher, launcherPath,
		binary.getHostInterpreterName(ctx, actualVersion),
		main, binary.getStem(ctx), append(android.Paths{srcsZip}, depsSrcsZips...))

	return android.OptionalPathForPath(binFile)
}

// get host interpreter name.
func (binary *binaryDecorator) getHostInterpreterName(ctx android.ModuleContext,
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
}

// find main program path within runfiles tree.
func (binary *binaryDecorator) getPyMainFile(ctx android.ModuleContext,
	srcsPathMappings []pathMapping) string {
	var main string
	_main := proptools.String(binary.binaryProperties.Main)
	if _main == "" {
		main = ctx.ModuleName() + pyExt
	} else {
		main = _main
	}
	fmt.Fprintf(os.Stderr, "\n@@MAIN: %s (%s)\n", main, _main)
	fmt.Fprintf(os.Stderr, "\n@@MAPPINGS: %+ v\n", srcsPathMappings)

	for _, path := range srcsPathMappings {
		rel := path.src.Rel()
		//if main == path.src.Rel() {
		if strings.HasSuffix(rel, "/"+main) || rel == main {
			fmt.Fprintf(os.Stderr, "\n@@@SUCCESS:\nrel: %s\ndest:%+ v\n", rel, path.dest)
			if main != rel {
				fmt.Fprintf(os.Stderr, "@@@WOULDFAIL:\nsrc: %+ v\n", path.src)
			}
			return path.dest
		}
	}
	//ctx.PropertyErrorf("main", "%q is not listed in srcs.", main)
	fmt.Fprintf(os.Stderr, "@@@FAIL: main %s not listed in srcs\n", main)

	return ""
}

func (binary *binaryDecorator) getStem(ctx android.ModuleContext) string {
	stem := ctx.ModuleName()
	if proptools.String(binary.binaryProperties.Stem) != "" {
		stem = proptools.String(binary.binaryProperties.Stem)
	}

	return stem + proptools.String(binary.binaryProperties.Suffix)
}
