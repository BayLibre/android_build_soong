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

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.PreDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.BottomUp("version_split", versionSplitMutator()).Parallel()
	})
}

// the version properties the user can specify about Python library and binary module.
type PythonVersionProperties struct {
	// true, if the module is required to be built with this version.
	Enabled bool

	// if specified, the "srcs" from PythonBaseModuleProperties will be converted with
	// converter tool.
	// valid values now are "2TO3" and "3TO2".
	Converter string

	// list of source files under this Python version, used to compile the Python module.
	// Must be .py files.
	// srcs may reference the outputs of other modules that produce source files like genrule
	// or filegroup using the syntax ":module".
	// Srcs has to be non-empty.
	Srcs []string

	// list of the Python modules(Py_libraries or Py_binaries) under this Python version
	// this module depends on.
	Py_libs []string
}

// the common properties the user can specify about Python library and binary module.
type PythonBaseModuleProperties struct {
	// the package path prefix within the output artifact at which to place the source/data
	// files of the current module.
	// eg. Pkg_path = "a/b/c"; Other packages can reference this module by using
	// (from a.b.c import ...) statement.
	// if left unspecified, all the source/data files of current module are copied to
	// "runfiles/" tree directory directly.
	Pkg_path string

	// list of source files compatible both with Python2 and Python3 used to compile the
	// Python module. Must be .py files.
	// srcs may reference the outputs of other modules that produce source files like genrule
	// or filegroup using the syntax ":module".
	// Srcs has to be non-empty.
	Srcs []string

	// list of files or filegroup modules that provide data that should be installed alongside
	// the test
	Data []string

	// list of the Python modules(Py_libraries or Py_binaries) compatible both with Python2 and
	// Python3 this module depends on.
	Py_libs []string

	// all the "srcs" or Python dependencies under this property has to be "only" compatible
	// with Python2.
	Version_py2 PythonVersionProperties

	// all the "srcs" or Python dependencies under this property has to be "only" compatible
	// with Python3.
	Version_py3 PythonVersionProperties

	// the actual version each module uses after variations created.
	// this property name is hided from users' perspectives, and soong will populate it during
	// runtime.
	ActualVersion string `blueprint:"mutated"`
}

type pythonBaseModule struct {
	android.ModuleBase
	subModule PythonSubModule

	properties PythonBaseModuleProperties

	// Key: runfiles tree path for src file; Value: current source tree path for src file
	destToPySrcs map[string]string

	// Key: runfiles tree path for data file; Value: current source tree path for data file
	destToPyData map[string]string
}

type PythonSubModule interface {
	GeneratePythonBuildActions(ctx android.ModuleContext)
}

func InitPythonBaseModule(baseModule *pythonBaseModule, subModule PythonSubModule,
	props ...interface{}) (blueprint.Module, []interface{}) {

	baseModule.subModule = subModule

	props = append(props, &baseModule.properties)

	return android.InitAndroidModule(baseModule, props...)
}

// the tag used to mark dependencies within "py_libs" attribute.
type pythonDependencyTag struct {
	blueprint.BaseDependencyTag
}

var pyDependencyTag pythonDependencyTag

var (
	pyIdentifierRegexp = regexp.MustCompile(`^([a-z]|[A-Z]|_)([a-z]|[A-Z]|[0-9]|_)*$`)
	pyExt              = ".py"
	pyVersion2         = "PY2"
	pyVersion3         = "PY3"
)

func (p *pythonBaseModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// deps from "data".
	android.ExtractSourcesDeps(ctx, p.properties.Data)
	// deps from "srcs".
	android.ExtractSourcesDeps(ctx, p.properties.Srcs)
	// add deps from "py_libs"
	//ctx.AddDependency(ctx.Module(), pyDependencyTag,
	//	p.properties.Py_libs...)
	ctx.AddVariationDependencies(nil, pyDependencyTag, p.properties.Py_libs...)
	if p.properties.ActualVersion == pyVersion2 {
		// deps from "srcs" of version property.
		android.ExtractSourcesDeps(ctx, p.properties.Version_py2.Srcs)
		// deps from "py_libs" of version property.
		//ctx.AddDependency(ctx.Module(), pyDependencyTag,
		//	p.properties.Version_py2.Py_libs...)
		ctx.AddVariationDependencies(nil, pyDependencyTag,
			p.properties.Version_py2.Py_libs...)
	} else if p.properties.ActualVersion == pyVersion3 {
		// deps from "srcs" of version property.
		android.ExtractSourcesDeps(ctx, p.properties.Version_py3.Srcs)
		// deps from "py_libs" of version property.
		///ctx.AddDependency(ctx.Module(), pyDependencyTag,
		///	p.properties.Version_py3.Py_libs...)
		ctx.AddVariationDependencies(nil, pyDependencyTag,
			p.properties.Version_py3.Py_libs...)
	} else {
		panic(fmt.Errorf("unknown Python actualVersion: %q for module: %q",
			p.properties.ActualVersion, ctx.ModuleName))
	}
}

func (p *pythonBaseModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.subModule.GeneratePythonBuildActions(ctx)
}

func (p *pythonBaseModule) GeneratePythonBuildActions(ctx android.ModuleContext) {
	// expand srcs & data from parent properties.
	expandedSrcs := ctx.ExpandSources(p.properties.Srcs, nil)
	if len(expandedSrcs) == 0 {
		ctx.ModuleErrorf("doesn't have any source files!")
	}
	expandedData := ctx.ExpandSources(p.properties.Data, nil)

	if p.properties.ActualVersion == pyVersion2 {
		expandedSrcs = append(expandedSrcs,
			ctx.ExpandSources(p.properties.Version_py2.Srcs, nil)...)
	} else if p.properties.ActualVersion == pyVersion3 {
		expandedSrcs = append(expandedSrcs,
			ctx.ExpandSources(p.properties.Version_py3.Srcs, nil)...)
	} else {
		panic(fmt.Errorf("unknown Python actualVersion: %q for module: %q",
			p.properties.ActualVersion, ctx.ModuleName))
	}

	pkg_path := filepath.Clean(p.properties.Pkg_path)
	if pkg_path == ".." || strings.HasPrefix(pkg_path, "../") ||
		strings.HasPrefix(pkg_path, "/") {
		ctx.PropertyErrorf("pkg_path", "%q is not a valid format", p.properties.Pkg_path)
		return
	}
	for _, s := range expandedSrcs {
		if s.Ext() != pyExt {
			ctx.PropertyErrorf("srcs", "must not have any files except .py file!")
			continue
		}
		runfilesPath := filepath.Join(pkg_path, s.Rel())
		identifiers := strings.Split(strings.TrimSuffix(runfilesPath, pyExt), "/")
		for _, token := range identifiers {
			if !pyIdentifierRegexp.MatchString(token) {
				ctx.PropertyErrorf("srcs", "the path %q contains invalid token %q",
					runfilesPath, token)
			}
		}
		fillInMap(ctx, &p.destToPySrcs, runfilesPath, s.String(), p.Name())
	}
	for _, d := range expandedData {
		if d.Ext() == pyExt {
			ctx.PropertyErrorf("data", "must not have any .py file!")
			continue
		}
		runfilesPath := filepath.Join(pkg_path, d.Rel())
		fillInMap(ctx, &p.destToPyData, runfilesPath, d.String(), p.Name())
	}

	// fetch "srcs" and "data" files from its direct upstream dependencies.
	ctx.VisitDirectDeps(func(module blueprint.Module) {
		if ctx.OtherModuleDependencyTag(module) == pyDependencyTag {
			if dep, ok := module.(*pythonBaseModule); ok {
				for k, v := range dep.destToPySrcs {
					fillInMap(ctx, &p.destToPySrcs,
						k, v, ctx.OtherModuleName(module))
				}
				for k, v := range dep.destToPyData {
					fillInMap(ctx, &p.destToPyData,
						k, v, ctx.OtherModuleName(module))
				}
			}
		}
	})
}

func fillInMap(ctx android.ModuleContext, m *map[string]string, key, value, otherModule string) {
	if _, found := (*m)[key]; found {
		ctx.ModuleErrorf("found duplicated (runfiles dir) map key: %q from module: %q!",
			key, otherModule)
	} else {
		(*m)[key] = value
	}
}

// Create version variants for modules that need them
func versionSplitMutator() func(android.BottomUpMutatorContext) {
	return func(mctx android.BottomUpMutatorContext) {
		if base, ok := mctx.Module().(*pythonBaseModule); ok {
			versionNames := []string{}
			if base.properties.Version_py2.Enabled {
				versionNames = append(versionNames, pyVersion2)
			}
			if !base.properties.Version_py2.Enabled ||
				base.properties.Version_py3.Enabled {
				versionNames = append(versionNames, pyVersion3)
			}
			modules := mctx.CreateVariations(versionNames...)
			for i, v := range versionNames {
				// set the actual version for Python module.
				modules[i].(*pythonBaseModule).properties.ActualVersion = v
			}
		}
	}
}
