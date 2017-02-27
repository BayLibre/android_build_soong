package python

// This file contains the base module types for compiling Python library/binary.

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.PostDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.TopDown("version_requirements", versionRequirementsDepsMutator())
		ctx.BottomUp("version_variation", versionVariationMutator()).Parallel()
	})
}

type pythonBaseModuleProperties struct {
	// list of source files used to compile the Python module. Must be .py files.
	// srcs may reference the outputs of other modules that produce source files like genrule
	// or filegroup using the syntax ":module".
	// Srcs has to be non-empty.
	Srcs []string

	// the package path containing all the source/data files of current module, which can be
	// referenced by other Python modules/packages.
	// eg. PkgPath = "a/b/c"; Other packages can reference this module by using
	// (from a.b.c import ...) statement.
	// if left unspecified, all the source/data files of current module are copied to "runfiles/"
	// tree directory directly.
	PkgPath string

	// list of files or filegroup modules that provide data that should be installed alongside
	// the test
	Data []string

	// list of the Python libraries/modules this module depends on.
	Py_libs []string

	// the path version of .py source files listed in the srcs attribute of this module.
	// valid values now are "PY2ONLY", "PY2ANDPY3", or "PY3ONLY" (default).
	// TODO support "PY2TO3"
	Srcs_version string
}

type pythonBaseModule struct {
	android.ModuleBase
	subModule PythonSubModule

	properties pythonBaseModuleProperties

	// cache all the src paths after expanding the "srcs" attribute.
	expandedSrcs android.Paths

	// cache all the data paths after expanding the "data" attribute.
	expandedData android.Paths

	// store all the versions required from interpreter or its parents(binaries).
	versionsRequired map[string]bool

	// store the py version after module split.
	versionSplit string
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

var pythonDependencyTag blueprint.BaseDependencyTag

func (p *pythonBaseModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	android.ExtractSourcesDeps(ctx, p.properties.Srcs)
	android.ExtractSourcesDeps(ctx, p.properties.Data)

	ctx.AddDependency(ctx.Module(), pythonDependencyTag, p.properties.Py_libs...)
}

func (p *pythonBaseModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.subModule.GeneratePythonBuildActions(ctx)
}

func (p *pythonBaseModule) GeneratePythonBuildActions(
	ctx android.ModuleContext) (srcs, data map[string]string) {
	p.expandedSrcs = ctx.ExpandSources(p.properties.Srcs, nil)
	p.expandedData = ctx.ExpandSources(p.properties.Data, nil)

	processSrcs(ctx, p.properties.PkgPath, p.expandedSrcs, &srcs)
	processData(ctx, p.properties.PkgPath, p.expandedData, &data)

	ctx.VisitDepsDepthFirst(func(module blueprint.Module) {
		if d, ok := module.(*pythonBaseModule); ok {
			processSrcs(ctx, d.properties.PkgPath, d.expandedSrcs, &srcs)
			processData(ctx, d.properties.PkgPath, d.expandedData, &data)
		}
	})

	return
}

func processSrcs(ctx android.ModuleContext, pkgPath string, srcs android.Paths,
	dest *map[string]string) {
	for _, s := range srcs {
		if s.Ext() != ".py" {
			ctx.PropertyErrorf(
				"srcs", "attribute: srcs must not have any source except .py file!")
		}
		runfilesPath := filepath.Clean(filepath.Join(pkgPath, s.Rel()))
		substrs := strings.Split(runfilesPath, ".py")
		if len(substrs) != 2 || substrs[1] != "" || strings.Contains(substrs[0], ".") ||
			strings.HasPrefix(substrs[0], "/") {
			ctx.PropertyErrorf("srcs", "the path %q is not a valid format for package %q",
				runfilesPath, pkgPath)
		}
		identifiers := strings.Split(substrs[0], "/")
		for _, token := range identifiers {
			if match, _ := regexp.MatchString(
				"([a-z]|[A-Z]|_)([a-z]|[A-Z][0-9]|_)*", token); !match {
				ctx.PropertyErrorf("srcs", "the path %q contains invalid token %q",
					runfilesPath, token)
			}
		}
		if _, found := (*dest)[runfilesPath]; found {
			ctx.PropertyErrorf("srcs", "attribute: srcs must not have package path duplicates!")
		} else {
			(*dest)[runfilesPath] = s.String()
		}
	}
}

func processData(ctx android.ModuleContext, pkgPath string, data android.Paths,
	dest *map[string]string) {
	for _, d := range data {
		if d.Ext() == ".py" {
			ctx.PropertyErrorf(
				"data", "attribute: data must not have any .py file!")
		}
		runfilesPath := filepath.Clean(filepath.Join(pkgPath, d.Rel()))
		if runfilesPath == "." || runfilesPath == ".." || strings.HasPrefix(runfilesPath, "../") ||
			strings.HasPrefix(runfilesPath, "/") {
			ctx.PropertyErrorf("data", "the path %q is not a valid format for package %q",
				runfilesPath, pkgPath)
		}
		if _, found := (*dest)[runfilesPath]; found {
			ctx.PropertyErrorf("data", "attribute: data must not have package path duplicates!")
		} else {
			(*dest)[runfilesPath] = d.String()
		}
	}
}

var (
	interpreter_version3 = []string{"", "PY3"}
	interpreter_version2 = []string{"PY2"}
	source_version3      = []string{"", "PY3ONLY", "PY2ANDPY3"}
	source_version2      = []string{"PY2ONLY", "PY2ANDPY3"}
)

func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// Propagate Python version requirements down from roots(binaries).
func versionRequirementsDepsMutator() func(android.TopDownMutatorContext) {
	return func(mctx android.TopDownMutatorContext) {
		if base, ok := mctx.Module().(*pythonBaseModule); ok {
			srcs_version := base.properties.Srcs_version
			if !stringInSlice(srcs_version, source_version2) &&
				!stringInSlice(srcs_version, source_version3) {
				mctx.PropertyErrorf("the srcs_version %q in %q is not valid",
					srcs_version, mctx.ModuleName())
			}
			if bin, ok := base.subModule.(*PythonBinary); ok {
				interp_version := bin.binaryProperties.Py_interpreter_version
				if !stringInSlice(interp_version, interpreter_version2) &&
					!stringInSlice(interp_version, interpreter_version3) {
					mctx.PropertyErrorf(
						"the py_interpreter_version %q in %q is not valid",
						interp_version, mctx.ModuleName())
				}
				if len(base.versionsRequired) > 0 {
					mctx.ModuleErrorf(
						"the binary %q cannot be depended by other modules",
						interp_version, mctx.ModuleName())
				}
				if interp_version == "" {
					interp_version = "PY3"
				}
				base.versionsRequired[interp_version] = true
				mctx.VisitDepsDepthFirst(func(module blueprint.Module) {
					if d, ok := module.(*pythonBaseModule); ok {
						d.versionsRequired[interp_version] = true
					}
				})
			}
		}
	}
}

// Create version variants for modules that need them
func versionVariationMutator() func(android.BottomUpMutatorContext) {
	return func(mctx android.BottomUpMutatorContext) {
		if base, ok := mctx.Module().(*pythonBaseModule); ok &&
			len(base.versionsRequired) > 0 {
			if len(base.versionsRequired) > 1 {
				if _, ok := base.subModule.(*PythonBinary); ok {
					mctx.ModuleErrorf(
						"the binary %q cannot be built with "+
							"multiple py versions",
						mctx.ModuleName())
				}
			}
			for required, _ := range base.versionsRequired {
				if (stringInSlice(required, interpreter_version2) &&
					!stringInSlice(base.properties.Srcs_version, source_version2)) ||
					(stringInSlice(required, interpreter_version3) &&
						!stringInSlice(base.properties.Srcs_version, source_version3)) {
					mctx.ModuleErrorf("the required version %q is not compatible with %q",
						required, mctx.ModuleName())
				}
			}
			required := []string{}
			for k, _ := range base.versionsRequired {
				required = append(required, k)
			}
			modules := mctx.CreateVariations(required...)
			for i, r := range required {
				modules[i].(*PythonBinary).versionSplit = r
			}
		}
	}
}
