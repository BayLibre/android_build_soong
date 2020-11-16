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

// This file contains the "Base" module type for building Python program.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func init() {
	android.PreDepsMutators(RegisterPythonPreDepsMutators)
}

// Register Python pre-deps mutators in the given mutators context.
func RegisterPythonPreDepsMutators(ctx android.RegisterMutatorsContext) {
	ctx.BottomUp("version_split", versionSplitMutator()).Parallel()
}

// the version properties that apply to python libraries and binaries.
type VersionProperties struct {
	// true, if the module is required to be built with this version.
	Enabled *bool `android:"arch_variant"`

	// non-empty list of .py files under this strict Python version.
	// srcs may reference the outputs of other modules that produce source files like genrule
	// or filegroup using the syntax ":module".
	Srcs []string `android:"path,arch_variant"`

	// list of source files that should not be used to build the Python module.
	// This is most useful in the arch/multilib variants to remove non-common files
	Exclude_srcs []string `android:"path,arch_variant"`

	// list of the Python libraries under this Python version.
	Libs []string `android:"arch_variant"`

	// true, if the binary is required to be built with embedded launcher.
	// TODO(nanzhang): Remove this flag when embedded Python3 is supported later.
	Embedded_launcher *bool `android:"arch_variant"`
}

// properties that apply to python libraries and binaries.
type BaseProperties struct {
	// the package path prefix within the output artifact at which to place the source/data
	// files of the current module.
	// eg. Pkg_path = "a/b/c"; Other packages can reference this module by using
	// (from a.b.c import ...) statement.
	// if left unspecified, all the source/data files path is unchanged within zip file.
	Pkg_path *string `android:"arch_variant"`

	// true, if the Python module is used internally, eg, Python std libs.
	Is_internal *bool `android:"arch_variant"`

	// list of source (.py) files compatible both with Python2 and Python3 used to compile the
	// Python module.
	// srcs may reference the outputs of other modules that produce source files like genrule
	// or filegroup using the syntax ":module".
	// Srcs has to be non-empty.
	Srcs []string `android:"path,arch_variant"`

	// list of source files that should not be used to build the C/C++ module.
	// This is most useful in the arch/multilib variants to remove non-common files
	Exclude_srcs []string `android:"path,arch_variant"`

	// list of files or filegroup modules that provide data that should be installed alongside
	// the test. the file extension can be arbitrary except for (.py).
	Data []string `android:"path,arch_variant"`

	// list of the Python libraries compatible both with Python2 and Python3.
	Libs []string `android:"arch_variant"`

	Version struct {
		// all the "srcs" or Python dependencies that are to be used only for Python2.
		Py2 VersionProperties `android:"arch_variant"`

		// all the "srcs" or Python dependencies that are to be used only for Python3.
		Py3 VersionProperties `android:"arch_variant"`
	} `android:"arch_variant"`

	// the actual version each module uses after variations created.
	// this property name is hidden from users' perspectives, and soong will populate it during
	// runtime.
	Actual_version string `blueprint:"mutated"`

	// true, if the module is required to be built with actual_version.
	Enabled *bool `blueprint:"mutated"`

	// true, if the binary is required to be built with embedded launcher.
	// TODO(nanzhang): Remove this flag when embedded Python3 is supported later.
	Embedded_launcher *bool `blueprint:"mutated"`
}

// Used to store files of current module after expanding dependencies
// Typically used as: <dest: runfile_path, src: source_path>
type pathMapping struct {
	dest string
	src  android.Path
}

type Module struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties      BaseProperties
	protoProperties android.ProtoProperties

	// initialize before calling Init
	hod      android.HostOrDeviceSupported
	multilib android.Multilib

	bootstrapper bootstrapper
	// may be nil if module cannot be installed (e.g. library)
	installer installer

	// the Python files of current module after expanding source dependencies.
	// pathMapping: <dest: runfile_path, src: source_path>
	srcsPathMappings []pathMapping

	// the data files of current module after expanding source dependencies.
	// pathMapping: <dest: runfile_path, src: source_path>
	dataPathMappings []pathMapping

	// the zip filepath for zipping current module source/data files.
	srcsZip android.Path

	// dependency modules' zip filepath for zipping current module source/data files.
	depsSrcsZips android.Paths

	// (.intermediate) module output path as installation source.
	installSource android.OptionalPath

	// Map to ensure sub-part of the AndroidMk for this module is only added once
	subAndroidMkOnce map[subAndroidMkProvider]bool
}

// generate new Python base module
func newModule(hod android.HostOrDeviceSupported, multilib android.Multilib) *Module {
	return &Module{
		hod:      hod,
		multilib: multilib,
	}
}

//
type bootstrapper interface {
	bootstrapperProps() []interface{}
	bootstrap(ctx android.ModuleContext, ActualVersion string, embeddedLauncher bool,
		srcsPathMappings []pathMapping, srcsZip android.Path,
		depsSrcsZips android.Paths) android.OptionalPath

	autorun() bool
}

// installer interface, for installable
type installer interface {
	install(ctx android.ModuleContext, path android.Path)
	setAndroidMkSharedLibs(sharedLibs []string)
}

// interface implemented by Python modules to provide source and data zip/mappings to modules that depend on it
type pythonDependency interface {
	GetSrcsPathMappings() []pathMapping
	GetDataPathMappings() []pathMapping
	GetSrcsZip() android.Path
}

// Get path mapping of src : runfiles destination
func (p *Module) GetSrcsPathMappings() []pathMapping {
	return p.srcsPathMappings
}

// Get path mapping of data : runfiles destination
func (p *Module) GetDataPathMappings() []pathMapping {
	return p.dataPathMappings
}

// The filepath where the current module's source/data files are zipped.
func (p *Module) GetSrcsZip() android.Path {
	return p.srcsZip
}

var _ pythonDependency = (*Module)(nil)

var _ android.AndroidMkDataProvider = (*Module)(nil)

func (p *Module) init(additionalProps ...interface{}) android.Module {
	p.AddProperties(&p.properties, &p.protoProperties)

	// Add additional properties for bootstrapping/installation
	// This is currently tied to the bootstrapper interface;
	// however, these are a combination of properties for the installation and bootstrapping of a module
	if p.bootstrapper != nil {
		p.AddProperties(p.bootstrapper.bootstrapperProps()...)
	}

	android.InitAndroidArchModule(p, p.hod, p.multilib)
	android.InitDefaultableModule(p)

	return p
}

// Python-specific tag to transfer information on the purpose of a dependency.
// This is used when adding a dependency on a module, which can later be accessed when visiting dependencies.
type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var (
	pythonLibTag         = dependencyTag{name: "pythonLib"}
	launcherTag          = dependencyTag{name: "launcher"}
	launcherSharedLibTag = dependencyTag{name: "launcherSharedLib"}
	subPathRegexp        = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)
	pyExt                = ".py"
	protoExt             = ".proto"
	pyVersion2           = "PY2"
	pyVersion3           = "PY3"
	initFileName         = "__init__.py"
	mainFileName         = "__main__.py"
	entryPointFile       = "entry_point.txt"
	parFileExt           = ".zip"
	internalPath         = "internal"
)

// create version variants for modules.
// Append version-specific properties within a variant to properties in the appropriate module
func versionSplitMutator() func(android.BottomUpMutatorContext) {
	return func(mctx android.BottomUpMutatorContext) {
		if base, ok := mctx.Module().(*Module); ok {
			versionNames := []string{}
			versionProps := []VersionProperties{}
			// PY3 is first so that we alias the PY3 variant rather than PY2 if both
			// are available
			if !(base.properties.Version.Py3.Enabled != nil &&
				*(base.properties.Version.Py3.Enabled) == false) {
				versionNames = append(versionNames, pyVersion3)
				versionProps = append(versionProps, base.properties.Version.Py3)
			}
			if base.properties.Version.Py2.Enabled != nil &&
				*(base.properties.Version.Py2.Enabled) == true {
				versionNames = append(versionNames, pyVersion2)
				versionProps = append(versionProps, base.properties.Version.Py2)
			}
			modules := mctx.CreateVariations(versionNames...)
			// Alias module to the first variant
			if len(versionNames) > 0 {
				mctx.AliasVariation(versionNames[0])
			}
			for i, v := range versionNames {
				// set the actual version for Python module.
				modules[i].(*Module).properties.Actual_version = v
				// append versioned properties for the Python module
				err := proptools.AppendMatchingProperties([]interface{}{&modules[i].(*Module).properties}, &versionProps[i], nil)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func (p *Module) HostToolPath() android.OptionalPath {
	if p.installer == nil {
		// python_library is just meta module, and doesn't have any installer.
		return android.OptionalPath{}
	}
	return android.OptionalPathForPath(p.installer.(*binaryDecorator).path)
}

// Return output files if available.
func (p *Module) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "":
		if outputFile := p.installSource; outputFile.Valid() {
			return android.Paths{outputFile.Path()}, nil
		}
		return android.Paths{}, nil
	default:
		return nil, fmt.Errorf("unsupported module reference tag %q", tag)
	}
}

// Whether embedded launcher is enabled.
func (p *Module) isEmbeddedLauncherEnabled() bool {
	return p.installer != nil && Bool(p.properties.Embedded_launcher)
}

func anyHasExt(paths []string, ext string) bool {
	for _, p := range paths {
		if filepath.Ext(p) == ext {
			return true
		}
	}

	return false
}

func (p *Module) anySrcHasExt(ctx android.BottomUpMutatorContext, ext string) bool {
	return anyHasExt(p.properties.Srcs, ext)
}

func (p *Module) DepsMutator(ctx android.BottomUpMutatorContext) {
	android.ProtoDeps(ctx, &p.protoProperties)

	// If sources contain a proto file, add dependency on libprotobuf-python
	if p.anySrcHasExt(ctx, protoExt) && p.Name() != "libprotobuf-python" {
		ctx.AddVariationDependencies(nil, pythonLibTag, "libprotobuf-python")
	}

	// Add dependencies on uniquely listed libraries
	ctx.AddVariationDependencies(nil, pythonLibTag, android.LastUniqueStrings(p.properties.Libs)...)

	// If this module will be installed and has an embedded launcher, we need to add dependencies for:
	//   * standard library
	//   * launcher
	//   * shared dependencies of the launcher
	if p.installer != nil && p.isEmbeddedLauncherEnabled() {
		var stdLib string
		var launcherModule string
		// Add launcher shared lib dependencies. Ideally, these should be
		// derived from the `shared_libs` property of the launcher. However, we
		// cannot read the property at this stage and it will be too late to add
		// dependencies later.
		launcherSharedLibDeps := []string{
			"libsqlite",
		}
		if ctx.Target().Os.Bionic() {
			launcherSharedLibDeps = append(launcherSharedLibDeps, "libc", "libdl", "libm")
		}

		switch p.properties.Actual_version {
		case pyVersion2:
			stdLib = "py2-stdlib"

			launcherModule = "py2-launcher"
			if p.bootstrapper.autorun() {
				launcherModule = "py2-launcher-autorun"
			}

		case pyVersion3:
			stdLib = "py3-stdlib"

			launcherModule = "py3-launcher"
			if p.bootstrapper.autorun() {
				launcherModule = "py3-launcher-autorun"
			}

			if ctx.Device() {
				launcherSharedLibDeps = append(launcherSharedLibDeps, "liblog")
			}
		default:
			panic(fmt.Errorf("unknown Python Actual_version: %q for module: %q.",
				p.properties.Actual_version, ctx.ModuleName()))
		}
		ctx.AddVariationDependencies(nil, pythonLibTag, stdLib)
		ctx.AddFarVariationDependencies(ctx.Target().Variations(), launcherTag, launcherModule)
		ctx.AddFarVariationDependencies(ctx.Target().Variations(), launcherSharedLibTag, launcherSharedLibDeps...)
	}
}

func (p *Module) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.generatePythonBuildActions(ctx)

	// Only Python binaries and test has non-empty bootstrapper.
	if p.bootstrapper != nil {
		p.collectPathsFromTransitiveDeps(ctx)
		p.installSource = p.bootstrapper.bootstrap(ctx, p.properties.Actual_version,
			p.isEmbeddedLauncherEnabled(), p.srcsPathMappings, p.srcsZip, p.depsSrcsZips)
	}

	if p.installer != nil {
		var sharedLibs []string
		ctx.VisitDirectDeps(func(dep android.Module) {
			if ctx.OtherModuleDependencyTag(dep) == launcherSharedLibTag {
				sharedLibs = append(sharedLibs, ctx.OtherModuleName(dep))
			}
		})
		p.installer.setAndroidMkSharedLibs(sharedLibs)

		if p.installSource.Valid() {
			p.installer.install(ctx, p.installSource.Path())
		}
	}
}

func (p *Module) generatePythonBuildActions(ctx android.ModuleContext) {
	expandedSrcs := android.PathsForModuleSrcExcludes(ctx, p.properties.Srcs, p.properties.Exclude_srcs)
	requiresSrcs := true
	if p.bootstrapper != nil && !p.bootstrapper.autorun() {
		requiresSrcs = false
	}
	if len(expandedSrcs) == 0 && requiresSrcs {
		ctx.ModuleErrorf("doesn't have any source files!")
	}

	// expand data files from "data" property.
	expandedData := android.PathsForModuleSrc(ctx, p.properties.Data)

	// Validate pkg_path property
	pkgPath := String(p.properties.Pkg_path)
	if pkgPath != "" {
		// TODO: export validation from android paths handling to replace this functionality
		pkgPath = filepath.Clean(String(p.properties.Pkg_path))
		if pkgPath == ".." || strings.HasPrefix(pkgPath, "../") ||
			strings.HasPrefix(pkgPath, "/") {
			ctx.PropertyErrorf("pkg_path",
				"%q must be a relative path contained in par file.",
				String(p.properties.Pkg_path))
			return
		}
	}
	// If property Is_internal is set, prepend pkgPath with internalPath
	if proptools.BoolDefault(p.properties.Is_internal, false) {
		pkgPath = filepath.Join(internalPath, pkgPath)
	}

	// generate src:destination path mappings for this module
	p.genModulePathMappings(ctx, pkgPath, expandedSrcs, expandedData)

	p.srcsZip = p.createSrcsZip(ctx, pkgPath)
}

func isValidPythonPath(path string) error {
	identifiers := strings.Split(strings.TrimSuffix(path, filepath.Ext(path)), "/")
	for _, token := range identifiers {
		if !subPathRegexp.MatchString(token) {
			return fmt.Errorf("the path %q contains invalid subpath %q. "+
				"Subpaths must be at least one character long. "+
				"The first character must an underscore or letter. "+
				"Following characters may be any of: letter, digit, underscore, hyphen.",
				path, token)
		}
	}
	return nil
}

// For this module, generate unique pathMappings: <dest: runfiles_path, src: source_path>
// for python/data files expanded from properties.
func (p *Module) genModulePathMappings(ctx android.ModuleContext, pkgPath string,
	expandedSrcs, expandedData android.Paths) {
	// fetch <runfiles_path, source_path> pairs from "src" and "data" properties to
	// check current module duplicates.
	destToPySrcs := make(map[string]string)
	destToPyData := make(map[string]string)

	for _, s := range expandedSrcs {
		if s.Ext() != pyExt && s.Ext() != protoExt {
			ctx.PropertyErrorf("srcs", "found non (.py|.proto) file: %q!", s.String())
			continue
		}
		runfilesPath := filepath.Join(pkgPath, s.Rel())
		if err := isValidPythonPath(runfilesPath); err != nil {
			ctx.PropertyErrorf("srcs", err.Error())
		}
		if !checkForDuplicateOutputPath(ctx, destToPySrcs, runfilesPath, s.String(), p.Name(), p.Name()) {
			p.srcsPathMappings = append(p.srcsPathMappings, pathMapping{dest: runfilesPath, src: s})
		}
	}

	for _, d := range expandedData {
		if d.Ext() == pyExt || d.Ext() == protoExt {
			ctx.PropertyErrorf("data", "found (.py|.proto) file: %q!", d.String())
			continue
		}
		runfilesPath := filepath.Join(pkgPath, d.Rel())
		if !checkForDuplicateOutputPath(ctx, destToPyData, runfilesPath, d.String(), p.Name(), p.Name()) {
			p.dataPathMappings = append(p.dataPathMappings,
				pathMapping{dest: runfilesPath, src: d})
		}
	}
}

// register build actions to zip current module's sources.
func (p *Module) createSrcsZip(ctx android.ModuleContext, pkgPath string) android.Path {
	relativeRootMap := make(map[string]android.Paths)
	pathMappings := append(p.srcsPathMappings, p.dataPathMappings...)

	var protoSrcs android.Paths
	// "srcs" or "data" properties may have filegroup so it might happen that
	// the relative root for each source path is different.
	for _, path := range pathMappings {
		if path.src.Ext() == protoExt {
			protoSrcs = append(protoSrcs, path.src)
		} else {
			var relativeRoot string
			relativeRoot = strings.TrimSuffix(path.src.String(), path.src.Rel())
			if v, found := relativeRootMap[relativeRoot]; found {
				relativeRootMap[relativeRoot] = append(v, path.src)
			} else {
				relativeRootMap[relativeRoot] = android.Paths{path.src}
			}
		}
	}
	var zips android.Paths
	if len(protoSrcs) > 0 {
		protoFlags := android.GetProtoFlags(ctx, &p.protoProperties)
		protoFlags.OutTypeFlag = "--python_out"

		for _, srcFile := range protoSrcs {
			zip := genProto(ctx, srcFile, protoFlags, pkgPath)
			zips = append(zips, zip)
		}
	}

	if len(relativeRootMap) > 0 {
		// in order to keep stable order of soong_zip params, we sort the keys here.
		keys := android.SortedStringKeys(relativeRootMap)

		parArgs := []string{}
		if pkgPath != "" {
			parArgs = append(parArgs, `-P `+pkgPath)
		}
		implicits := android.Paths{}
		for _, k := range keys {
			parArgs = append(parArgs, `-C `+k)
			for _, path := range relativeRootMap[k] {
				parArgs = append(parArgs, `-f `+path.String())
				implicits = append(implicits, path)
			}
		}

		origSrcsZip := android.PathForModuleOut(ctx, ctx.ModuleName()+".py.srcszip")
		ctx.Build(pctx, android.BuildParams{
			Rule:        zip,
			Description: "python library archive",
			Output:      origSrcsZip,
			Implicits:   implicits,
			Args: map[string]string{
				"args": strings.Join(parArgs, " "),
			},
		})
		zips = append(zips, origSrcsZip)
	}
	if len(zips) == 1 {
		return zips[0]
	} else {
		combinedSrcsZip := android.PathForModuleOut(ctx, ctx.ModuleName()+".srcszip")
		ctx.Build(pctx, android.BuildParams{
			Rule:        combineZip,
			Description: "combine python library archive",
			Output:      combinedSrcsZip,
			Inputs:      zips,
		})
		return combinedSrcsZip
	}
}

// Whether Python Module is a library module or not
// Python libraries are not installable, while other modules are
func isPythonLibModule(module blueprint.Module) bool {
	if m, ok := module.(*Module); ok {
		// Python library has no installer
		if m.installer != nil {
			return false
		}
		return true
	}
	return false
}

// check Python source/data files duplicates for whole runfiles tree since Python binary/test
// need collect and zip all srcs of whole transitive dependencies to a final par file.
func (p *Module) collectPathsFromTransitiveDeps(ctx android.ModuleContext) {
	// fetch <runfiles_path, source_path> pairs from "src" and "data" properties to
	// check duplicates.
	destToPySrcs := make(map[string]string)
	destToPyData := make(map[string]string)
	for _, path := range p.srcsPathMappings {
		destToPySrcs[path.dest] = path.src.String()
	}
	for _, path := range p.dataPathMappings {
		destToPyData[path.dest] = path.src.String()
	}

	seen := make(map[android.Module]bool)

	// visit all its dependencies in depth first.
	ctx.WalkDeps(func(child, parent android.Module) bool {
		// we only collect dependencies tagged as python library deps
		if ctx.OtherModuleDependencyTag(child) != pythonLibTag {
			return false
		}
		if seen[child] {
			return false
		}
		seen[child] = true
		// Python modules only can depend on Python libraries.
		if !isPythonLibModule(child) {
			ctx.ModuleErrorf(
				"the dependency %q of module %q is not Python library!",
				ctx.ModuleName(), ctx.OtherModuleName(child))
		}
		// collect source and data paths, checking that there are no duplicate output file conflicts
		if dep, ok := child.(pythonDependency); ok {
			srcs := dep.GetSrcsPathMappings()
			for _, path := range srcs {
				checkForDuplicateOutputPath(ctx, destToPySrcs,
					path.dest, path.src.String(), ctx.ModuleName(), ctx.OtherModuleName(child))
			}
			data := dep.GetDataPathMappings()
			for _, path := range data {
				checkForDuplicateOutputPath(ctx, destToPyData,
					path.dest, path.src.String(), ctx.ModuleName(), ctx.OtherModuleName(child))
			}
			p.depsSrcsZips = append(p.depsSrcsZips, dep.GetSrcsZip())
		}
		return true
	})
}

// Check whether an path has already been encountered and would result in two files being placed in the same location.
func checkForDuplicateOutputPath(ctx android.ModuleContext, m map[string]string, outputPath, srcPath, curModule, otherModule string) bool {
	if oldSrcPath, found := m[outputPath]; found {
		ctx.ModuleErrorf("found two files to be placed at the same location within zip %q."+
			" First file: in module %s at path %q."+
			" Second file: in module %s at path %q.",
			outputPath, curModule, oldSrcPath, otherModule, srcPath)
		return true
	}
	m[outputPath] = srcPath

	return false
}

// Python is not supported in the system partition
func (p *Module) InstallInData() bool {
	return true
}

var Bool = proptools.Bool
var BoolDefault = proptools.BoolDefault
var String = proptools.String
