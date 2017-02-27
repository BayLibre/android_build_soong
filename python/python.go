package python

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.PostDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.TopDown("version_check", versionCheckDepsMutator())
		ctx.BottomUp("version_split", versionSplitMutator()).Parallel()
	})
}

// the common properties the user can specify about Python library and binary module.
type pythonBaseModuleProperties struct {
	// list of source files used to compile the Python module. Must be .py files.
	// srcs may reference the outputs of other modules that produce source files like genrule
	// or filegroup using the syntax ":module".
	// Srcs has to be non-empty.
	Srcs []string

	// the package path prefix within the output artifact at which to place the source/data
	// files of the current module.
	// eg. PkgPath = "a/b/c"; Other packages can reference this module by using
	// (from a.b.c import ...) statement.
	// if left unspecified, all the source/data files of current module are copied to
	// "runfiles/" tree directory directly.
	PkgPath string

	// list of files or filegroup modules that provide data that should be installed alongside
	// the test
	Data []string

	// list of the Python libraries/modules this module depends on.
	Py_libs []string

	// the path version of .py source files listed in the srcs attribute of this module.
	// valid values now are "PY2ONLY", "PY2ANDPY3", or "PY3ONLY" (default).
	// TODO(nanzhang): support "PY2TO3"
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

	// store all the versions supported based on properties.Srcs_version
	versionsSupported []string

	// denote the actual Python version this module runs on.
	actualVersion string
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
var pythonDependencyTag blueprint.BaseDependencyTag

func (p *pythonBaseModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// deps from "srcs".
	android.ExtractSourcesDeps(ctx, p.properties.Srcs)
	// deps from "data".
	android.ExtractSourcesDeps(ctx, p.properties.Data)

	// deps from "py_libs".
	ctx.AddDependency(ctx.Module(), pythonDependencyTag, p.properties.Py_libs...)

	if !stringInSlice(p.properties.Srcs_version, source_versions) {
		ctx.PropertyErrorf("the srcs_version %q in %q is not valid",
			p.properties.Srcs_version, ctx.ModuleName())
	}
	switch p.properties.Srcs_version {
	case "PY3ONLY", "":
		p.versionsSupported = append(p.versionsSupported, "PY3")
	case "PY2ONLY":
		p.versionsSupported = append(p.versionsSupported, "PY2")
	case "PY2ANDPY3":
		p.versionsSupported = append(p.versionsSupported, "PY3", "PY2")
	}

	if bin, ok := p.subModule.(*PythonBinary); ok {
		if !stringInSlice(bin.binaryProperties.Py_interpreter_version,
			interpreter_versions) {
			ctx.PropertyErrorf("the py_interpreter_version %q in %q is not valid",
				bin.binaryProperties.Py_interpreter_version, ctx.ModuleName())
		}
		if !stringInSlice(bin.binaryProperties.Py_interpreter_version,
			p.versionsSupported) {
			ctx.PropertyErrorf("the srcs_version %q in %q is not compatible"+
				"with py_interpreter_version %q",
				p.properties.Srcs_version, ctx.ModuleName(),
				bin.binaryProperties.Py_interpreter_version)
		}
		if bin.binaryProperties.Py_interpreter_version == "" {
			p.actualVersion = "PY3"
		} else {
			p.actualVersion = bin.binaryProperties.Py_interpreter_version
		}
	}
}

func (p *pythonBaseModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.subModule.GeneratePythonBuildActions(ctx)
}

func (p *pythonBaseModule) GeneratePythonBuildActions(
	ctx android.ModuleContext) (srcs, data map[string]string) {

	p.expandedSrcs = ctx.ExpandSources(p.properties.Srcs, nil)
	p.expandedData = ctx.ExpandSources(p.properties.Data, nil)

	// process the "srcs" and "data" attribute for current module.
	processSrcs(ctx, p.properties.PkgPath, p.expandedSrcs, &srcs)
	processData(ctx, p.properties.PkgPath, p.expandedData, &data)

	// process the "srcs" and "data" attribute from its all upstream deps.
	ctx.VisitDepsDepthFirst(func(module blueprint.Module) {
		if d, ok := module.(*pythonBaseModule); ok {
			processSrcs(ctx, d.properties.PkgPath, d.expandedSrcs, &srcs)
			processData(ctx, d.properties.PkgPath, d.expandedData, &data)
		}
	})

	return
}

// check package path format correctness for each element within "srcs" attribute, and detect
// full package path duplicates.
func processSrcs(ctx android.ModuleContext, pkgPath string, srcs android.Paths,
	dest *map[string]string) {
	for _, s := range srcs {
		if s.Ext() != ".py" {
			ctx.PropertyErrorf(
				"srcs", "attribute: srcs must not have any files except .py file!")
		}
		runfilesPath := filepath.Clean(filepath.Join(pkgPath, s.Rel()))
		substrs := strings.Split(runfilesPath, ".py")
		if len(substrs) != 2 || substrs[1] != "" || strings.Contains(substrs[0], ".") ||
			strings.HasPrefix(substrs[0], "/") {
			ctx.PropertyErrorf("srcs", "path %q is not a valid format for package %q",
				runfilesPath, pkgPath)
		}
		identifiers := strings.Split(substrs[0], "/")
		for _, token := range identifiers {
			if match, _ := regexp.MatchString(
				"([a-z]|[A-Z]|_)([a-z]|[A-Z]|[0-9]|_)*", token); !match {
				ctx.PropertyErrorf("srcs", "the path %q contains invalid token %q",
					runfilesPath, token)
			}
		}
		if _, found := (*dest)[runfilesPath]; found {
			ctx.PropertyErrorf(
				"srcs", "attribute: srcs must not have package path duplicates!")
		} else {
			(*dest)[runfilesPath] = s.String()
		}
	}
}

// check package path format correctness for each element within "data" attribute, and detect
// full package path duplicates.
func processData(ctx android.ModuleContext, pkgPath string, data android.Paths,
	dest *map[string]string) {
	for _, d := range data {
		if d.Ext() == ".py" {
			ctx.PropertyErrorf(
				"data", "attribute: data must not have any .py file!")
		}
		runfilesPath := filepath.Clean(filepath.Join(pkgPath, d.Rel()))
		if runfilesPath == "." || runfilesPath == ".." ||
			strings.HasPrefix(runfilesPath, "../") ||
			strings.HasPrefix(runfilesPath, "/") {
			ctx.PropertyErrorf("data", "path %q is not a valid format for package %q",
				runfilesPath, pkgPath)
		}
		if _, found := (*dest)[runfilesPath]; found {
			ctx.PropertyErrorf(
				"data", "attribute: data must not have package path duplicates!")
		} else {
			(*dest)[runfilesPath] = d.String()
		}
	}
}

var (
	interpreter_versions = []string{"", "PY3", "PY2"}
	source_versions      = []string{"", "PY3ONLY", "PY2ANDPY3", "PY2ONLY"}
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
func versionCheckDepsMutator() func(android.TopDownMutatorContext) {
	return func(mctx android.TopDownMutatorContext) {
		if base, ok := mctx.Module().(*pythonBaseModule); ok {
			// the module is a binary module.
			if _, ok := base.subModule.(*PythonBinary); ok {
				if base.actualVersion != "PY3" && base.actualVersion != "PY2" {
					mctx.ModuleErrorf(
						"the actualVersion %q of %q is not valid",
						base.actualVersion, mctx.ModuleName())
				}
				// check the acutalVersion of this binary can be supported by its
				// all upstream dependencies.
				mctx.VisitDepsDepthFirst(func(module blueprint.Module) {
					if d, ok := module.(*pythonBaseModule); ok {
						if _, ok := d.subModule.(*PythonBinary); ok {
							mctx.ModuleErrorf(
								"%q depends on other binary %q"+
									"which is not allowed",
								mctx.ModuleName(),
								mctx.OtherModuleName(d))
						}
						if len(d.versionsSupported) == 0 {
							mctx.ModuleErrorf(
								"versionSupported of %q is empty",
								mctx.OtherModuleName(d))
						}
						if !stringInSlice(base.actualVersion,
							d.versionsSupported) {
							mctx.ModuleErrorf(
								"the version %q of %q is not"+
									"supported by"+
									"dependency %q",
								base.actualVersion,
								mctx.ModuleName(),
								mctx.OtherModuleName(d))
						}
					}
				})
			}
		}
	}
}

// Create version variants for modules that need them
func versionSplitMutator() func(android.BottomUpMutatorContext) {
	return func(mctx android.BottomUpMutatorContext) {
		if base, ok := mctx.Module().(*pythonBaseModule); ok {
			if _, ok := base.subModule.(*PythonBinary); ok {
				mctx.CreateVariations(base.actualVersion)
			} else {
				modules := mctx.CreateVariations(base.versionsSupported...)
				for i, v := range base.versionsSupported {
					// set the actual version for Python library module.
					modules[i].(*pythonBaseModule).actualVersion = v
				}
			}
		}
	}
}
