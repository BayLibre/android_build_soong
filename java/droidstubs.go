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

package java

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/bazel"
	"android/soong/java/config"
	"android/soong/remoteexec"
)

func init() {
	RegisterStubsBuildComponents(android.InitRegistrationContext)
}

func RegisterStubsBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("stubs_defaults", StubsDefaultsFactory)

	ctx.RegisterModuleType("droidstubs", DroidstubsFactory)
	ctx.RegisterModuleType("droidstubs_host", DroidstubsHostFactory)

	ctx.RegisterModuleType("prebuilt_stubs_sources", PrebuiltStubsSourcesFactory)
}

func StubsDefaultsFactory() android.Module {
	module := &DocDefaults{}

	module.AddProperties(
		&JavadocProperties{},
		&DroidstubsProperties{},
	)

	android.InitDefaultsModule(module)

	return module
}

// Droidstubs
type Droidstubs struct {
	Javadoc

	properties DroidstubsProperties

	compoundModuleMultiplexer CompoundModuleMultiplexer

	metalavaPart                 metalavaPart
	apiCheckPart                 apiCheckPart
	updateApiPart                updateApiPart
	checkNullabilityWarningsPart checkNullabilityWarningsPart
}

type ApiToCheck struct {
	// path to the API txt file that the new API extracted from source code is checked
	// against. The path can be local to the module or from other module (via :module syntax).
	Api_file *string `android:"path"`

	// path to the API txt file that the new @removed API extractd from source code is
	// checked against. The path can be local to the module or from other module (via
	// :module syntax).
	Removed_api_file *string `android:"path"`

	// If not blank, path to the baseline txt file for approved API check violations.
	Baseline_file *string `android:"path"`

	// Arguments to the apicheck tool.
	Args *string
}

type DroidstubsProperties struct {
	// The generated public API filename by Metalava, defaults to <module>_api.txt
	Api_filename *string

	// the generated removed API filename by Metalava, defaults to <module>_removed.txt
	Removed_api_filename *string

	Check_api struct {
		Last_released ApiToCheck

		Current ApiToCheck

		Api_lint struct {
			Enabled *bool

			// If set, performs api_lint on any new APIs not found in the given signature file
			New_since *string `android:"path"`

			// If not blank, path to the baseline txt file for approved API lint violations.
			Baseline_file *string `android:"path"`
		}
	}

	// user can specify the version of previous released API file in order to do compatibility check.
	Previous_api *string `android:"path"`

	// is set to true, Metalava will allow framework SDK to contain annotations.
	Annotations_enabled *bool

	// a list of top-level directories containing files to merge qualifier annotations (i.e. those intended to be included in the stubs written) from.
	Merge_annotations_dirs []string

	// a list of top-level directories containing Java stub files to merge show/hide annotations from.
	Merge_inclusion_annotations_dirs []string

	// a file containing a list of classes to do nullability validation for.
	Validate_nullability_from_list *string

	// a file containing expected warnings produced by validation of nullability annotations.
	Check_nullability_warnings *string

	// if set to true, allow Metalava to generate doc_stubs source files. Defaults to false.
	Create_doc_stubs *bool

	// if set to true, cause Metalava to output Javadoc comments in the stubs source files. Defaults to false.
	// Has no effect if create_doc_stubs: true.
	Output_javadoc_comments *bool

	// if set to false then do not write out stubs. Defaults to true.
	//
	// TODO(b/146727827): Remove capability when we do not need to generate stubs and API separately.
	Generate_stubs *bool

	// if set to true, provides a hint to the build system that this rule uses a lot of memory,
	// whicih can be used for scheduling purposes
	High_mem *bool

	// if set to true, Metalava will allow framework SDK to contain API levels annotations.
	Api_levels_annotations_enabled *bool

	// Apply the api levels database created by this module rather than generating one in this droidstubs.
	Api_levels_module *string

	// the dirs which Metalava extracts API levels annotations from.
	Api_levels_annotations_dirs []string

	// the sdk kind which Metalava extracts API levels annotations from. Supports 'public', 'system', 'module-lib' and 'system-server'; defaults to public.
	Api_levels_sdk_type *string

	// the filename which Metalava extracts API levels annotations from. Defaults to android.jar.
	Api_levels_jar_filename *string

	// if set to true, collect the values used by the Dev tools and
	// write them in files packaged with the SDK. Defaults to false.
	Write_sdk_values *bool

	// path or filegroup to file defining extension an SDK name <-> numerical ID mapping and
	// what APIs exist in which SDKs; passed to metalava via --sdk-extensions-info
	Extensions_info_file *string `android:"path"`

	// API surface of this module. If set, the module contributes to an API surface.
	// For the full list of available API surfaces, refer to soong/android/sdk_version.go
	Api_surface *string
}

// Used by xsd_config
type ApiFilePath interface {
	ApiFilePath() android.Path
}

type ApiStubsSrcProvider interface {
	StubsSrcJar() android.Path
}

// Provider of information about API stubs, used by java_sdk_library.
type ApiStubsProvider interface {
	AnnotationsZip() android.Path
	ApiFilePath
	RemovedApiFilePath() android.Path

	ApiStubsSrcProvider
}

// droidstubs passes sources files through Metalava to generate stub .java files that only contain the API to be
// documented, filtering out hidden classes and methods.  The resulting .java files are intended to be passed to
// a droiddoc module to generate documentation.
func DroidstubsFactory() android.Module {
	module := &Droidstubs{}
	initDroidstubs(module)
	InitDroiddocModule(module, android.HostAndDeviceSupported)

	module.SetDefaultableHook(func(ctx android.DefaultableHookContext) {
		module.createApiContribution(ctx)
	})
	return module
}

// droidstubs_host passes sources files through Metalava to generate stub .java files that only contain the API
// to be documented, filtering out hidden classes and methods.  The resulting .java files are intended to be
// passed to a droiddoc_host module to generate documentation.  Use a droidstubs_host instead of a droidstubs
// module when symbols needed by the source files are provided by java_library_host modules.
func DroidstubsHostFactory() android.Module {
	module := &Droidstubs{}
	initDroidstubs(module)
	InitDroiddocModule(module, android.HostSupported)
	return module
}

// Common init implementation
func initDroidstubs(module *Droidstubs) {
	// Note that the ordering here is important.  The later "parts" use
	// variables computed in metalavaPart.
	module.compoundModuleMultiplexer.Parts = []interface{}{
		&module.metalavaPart,
		&module.apiCheckPart,
		&module.updateApiPart,
		&module.checkNullabilityWarningsPart,
	}

	// TODO: remove
	module.metalavaPart.droidstubs = module
	module.apiCheckPart.droidstubs = module
	module.updateApiPart.droidstubs = module
	module.checkNullabilityWarningsPart.droidstubs = module

	// These use the base Javadoc
	module.metalavaPart.javadoc = &module.Javadoc

	module.AddProperties(&module.properties,
		&module.Javadoc.properties)
}

// Use the OutputFiles from CompoundModuleMultiplexer, overriding the one from Javadoc
func (this *Droidstubs) OutputFiles(tag string) (android.Paths, error) {
	return this.compoundModuleMultiplexer.OutputFiles(tag)
}

func (d *Droidstubs) AnnotationsZip() android.Path {
	return d.metalavaPart.annotationsZip
}

func (d *Droidstubs) ApiFilePath() android.Path {
	return d.metalavaPart.apiFile
}

func (d *Droidstubs) RemovedApiFilePath() android.Path {
	return d.metalavaPart.removedApiFile
}

func (d *Droidstubs) StubsSrcJar() android.Path {
	return d.stubsSrcJar
}

var metalavaMergeAnnotationsDirTag = dependencyTag{name: "metalava-merge-annotations-dir"}
var metalavaMergeInclusionAnnotationsDirTag = dependencyTag{name: "metalava-merge-inclusion-annotations-dir"}
var metalavaAPILevelsAnnotationsDirTag = dependencyTag{name: "metalava-api-levels-annotations-dir"}
var metalavaAPILevelsModuleTag = dependencyTag{name: "metalava-api-levels-module-tag"}

func (d *Droidstubs) DepsMutator(ctx android.BottomUpMutatorContext) {
	d.Javadoc.addDeps(ctx)

	if len(d.properties.Merge_annotations_dirs) != 0 {
		for _, mergeAnnotationsDir := range d.properties.Merge_annotations_dirs {
			ctx.AddDependency(ctx.Module(), metalavaMergeAnnotationsDirTag, mergeAnnotationsDir)
		}
	}

	if len(d.properties.Merge_inclusion_annotations_dirs) != 0 {
		for _, mergeInclusionAnnotationsDir := range d.properties.Merge_inclusion_annotations_dirs {
			ctx.AddDependency(ctx.Module(), metalavaMergeInclusionAnnotationsDirTag, mergeInclusionAnnotationsDir)
		}
	}

	if len(d.properties.Api_levels_annotations_dirs) != 0 {
		for _, apiLevelsAnnotationsDir := range d.properties.Api_levels_annotations_dirs {
			ctx.AddDependency(ctx.Module(), metalavaAPILevelsAnnotationsDirTag, apiLevelsAnnotationsDir)
		}
	}

	if d.properties.Api_levels_module != nil {
		ctx.AddDependency(ctx.Module(), metalavaAPILevelsModuleTag, proptools.String(d.properties.Api_levels_module))
	}
}

func (d *Droidstubs) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Call to each of our parts
	d.compoundModuleMultiplexer.GenerateAndroidBuildActions(ctx)
}

func (dstubs *Droidstubs) AndroidMkEntries() []android.AndroidMkEntries {
	// If the stubsSrcJar is not generated (because generate_stubs is false) then
	// use the api file as the output file to ensure the relevant phony targets
	// are created in make if only the api txt file is being generated. This is
	// needed because an invalid output file would prevent the make entries from
	// being written.
	//
	// Note that dstubs.apiFile can be also be nil if WITHOUT_CHECKS_API is true.
	// TODO(b/146727827): Revert when we do not need to generate stubs and API separately.

	outputFile := android.OptionalPathForPath(dstubs.stubsSrcJar)
	if !outputFile.Valid() {
		outputFile = android.OptionalPathForPath(dstubs.metalavaPart.apiFile)
	}
	if !outputFile.Valid() {
		outputFile = android.OptionalPathForPath(dstubs.metalavaPart.apiVersionsXml)
	}
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "JAVA_LIBRARIES",
		OutputFile: outputFile,
		Include:    "$(BUILD_SYSTEM)/soong_droiddoc_prebuilt.mk",
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				if dstubs.Javadoc.stubsSrcJar != nil {
					entries.SetPath("LOCAL_DROIDDOC_STUBS_SRCJAR", dstubs.Javadoc.stubsSrcJar)
				}
				if dstubs.metalavaPart.apiVersionsXml != nil {
					entries.SetPath("LOCAL_DROIDDOC_API_VERSIONS_XML", dstubs.metalavaPart.apiVersionsXml)
				}
				if dstubs.metalavaPart.annotationsZip != nil {
					entries.SetPath("LOCAL_DROIDDOC_ANNOTATIONS_ZIP", dstubs.metalavaPart.annotationsZip)
				}
				if dstubs.metalavaPart.metadataZip != nil {
					entries.SetPath("LOCAL_DROIDDOC_METADATA_ZIP", dstubs.metalavaPart.metadataZip)
				}
			},
		},
		ExtraFooters: []android.AndroidMkExtraFootersFunc{
			func(w io.Writer, name, prefix, moduleDir string) {
				// Call to each of our parts
				dstubs.compoundModuleMultiplexer.GenerateMkExtraFooters(w, name, prefix, moduleDir)
			},
		},
	}}
}

//
// Metalava invocation
//

type metalavaPart struct {
	droidstubs *Droidstubs
	javadoc    *Javadoc

	apiFile        android.Path
	removedApiFile android.Path
	annotationsZip android.WritablePath
	apiVersionsXml android.WritablePath

	checkLastReleasedApiTimestamp android.WritablePath

	apiLintTimestamp android.WritablePath
	apiLintReport    android.WritablePath

	nullabilityWarningsFile android.WritablePath

	metadataZip android.WritablePath
	metadataDir android.WritablePath
}

func (this *metalavaPart) OutputFiles(tag string) (android.Paths, bool) {
	switch tag {
	case "":
		return android.Paths{this.javadoc.stubsSrcJar}, true
	case ".docs.zip":
		return android.Paths{this.javadoc.docZip}, true
	case ".api.txt":
		return android.Paths{this.apiFile}, true
	case ".removed-api.txt":
		return android.Paths{this.removedApiFile}, true
	case ".annotations.zip":
		return android.Paths{this.annotationsZip}, true
	case ".api_versions.xml":
		return android.Paths{this.apiVersionsXml}, true
	default:
		return nil, false
	}
}

// The values allowed for Droidstubs' Api_levels_sdk_type
var allowedApiLevelSdkTypes = []string{"public", "system", "module-lib", "system-server"}

func (this *metalavaPart) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	d := this.droidstubs

	deps := d.Javadoc.collectDeps(ctx)

	javaVersion := getJavaVersion(ctx, String(d.Javadoc.properties.Java_version), android.SdkContext(d))

	srcJarDir := android.PathForModuleOut(ctx, "metalava", "srcjars")

	rule := android.NewRuleBuilder(pctx, ctx)

	rule.Sbox(android.PathForModuleOut(ctx, "metalava"),
		android.PathForModuleOut(ctx, "metalava.sbox.textproto")).
		SandboxInputs()

	if BoolDefault(d.properties.High_mem, false) {
		// This metalava run uses lots of memory, restrict the number of metalava jobs that can run in parallel.
		rule.HighMem()
	}

	generateStubs := BoolDefault(d.properties.Generate_stubs, true)
	var stubsDir android.OptionalPath
	if generateStubs {
		d.Javadoc.stubsSrcJar = android.PathForModuleOut(ctx, "metalava", ctx.ModuleName()+"-"+"stubs.srcjar")
		stubsDir = android.OptionalPathForPath(android.PathForModuleOut(ctx, "metalava", "stubsDir"))
		rule.Command().Text("rm -rf").Text(stubsDir.String())
		rule.Command().Text("mkdir -p").Text(stubsDir.String())
	}

	srcJarList := zipSyncCmd(ctx, rule, srcJarDir, d.Javadoc.srcJars)

	homeDir := android.PathForModuleOut(ctx, "metalava", "home")

	rule.Command().Text("rm -rf").Flag(homeDir.String())
	rule.Command().Text("mkdir -p").Flag(homeDir.String())

	cmd := rule.Command()
	cmd.FlagWithArg("ANDROID_PREFS_ROOT=", homeDir.String())

	if metalavaUseRbe(ctx) {
		rule.Remoteable(android.RemoteRuleSupports{RBE: true})
		execStrategy := ctx.Config().GetenvWithDefault("RBE_METALAVA_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
		labels := map[string]string{"type": "tool", "name": "metalava"}
		// TODO: metalava pool rejects these jobs
		pool := ctx.Config().GetenvWithDefault("RBE_METALAVA_POOL", "java16")
		rule.Rewrapper(&remoteexec.REParams{
			Labels:          labels,
			ExecStrategy:    execStrategy,
			ToolchainInputs: []string{config.JavaCmd(ctx).String()},
			Platform:        map[string]string{remoteexec.PoolKey: pool},
		})
	}

	cmd.BuiltTool("metalava").ImplicitTool(ctx.Config().HostJavaToolPath(ctx, "metalava.jar")).
		Flag(config.JavacVmFlags).
		Flag("-J--add-opens=java.base/java.util=ALL-UNNAMED").
		FlagWithArg("-encoding ", "UTF-8").
		FlagWithArg("-source ", javaVersion.String()).
		FlagWithRspFileInputList("@", android.PathForModuleOut(ctx, "metalava.rsp"), d.Javadoc.srcFiles).
		FlagWithInput("@", srcJarList)

	if len(deps.bootClasspath) > 0 {
		cmd.FlagWithInputList("-bootclasspath ", deps.bootClasspath.Paths(), ":")
	}

	if len(deps.classpath) > 0 {
		cmd.FlagWithInputList("-classpath ", deps.classpath.Paths(), ":")
	}

	cmd.Flag("--no-banner").
		Flag("--color").
		Flag("--quiet").
		Flag("--format=v2").
		FlagWithArg("--repeat-errors-max ", "10").
		FlagWithArg("--hide ", "UnresolvedImport").
		FlagWithArg("--hide ", "InvalidNullabilityOverride").
		// b/223382732
		FlagWithArg("--hide ", "ChangedDefault")

	cmd.Implicits(d.Javadoc.implicits)

	if apiCheckEnabled(ctx, d.properties.Check_api.Current, "current") ||
		apiCheckEnabled(ctx, d.properties.Check_api.Last_released, "last_released") ||
		String(d.properties.Api_filename) != "" {
		filename := proptools.StringDefault(d.properties.Api_filename, ctx.ModuleName()+"_api.txt")
		uncheckedApiFile := android.PathForModuleOut(ctx, "metalava", filename)
		cmd.FlagWithOutput("--api ", uncheckedApiFile)
		this.apiFile = uncheckedApiFile
	} else if sourceApiFile := proptools.String(d.properties.Check_api.Current.Api_file); sourceApiFile != "" {
		// If check api is disabled then make the source file available for export.
		this.apiFile = android.PathForModuleSrc(ctx, sourceApiFile)
	}

	if apiCheckEnabled(ctx, d.properties.Check_api.Current, "current") ||
		apiCheckEnabled(ctx, d.properties.Check_api.Last_released, "last_released") ||
		String(d.properties.Removed_api_filename) != "" {
		filename := proptools.StringDefault(d.properties.Removed_api_filename, ctx.ModuleName()+"_removed.txt")
		uncheckedRemovedFile := android.PathForModuleOut(ctx, "metalava", filename)
		cmd.FlagWithOutput("--removed-api ", uncheckedRemovedFile)
		this.removedApiFile = uncheckedRemovedFile
	} else if sourceRemovedApiFile := proptools.String(d.properties.Check_api.Current.Removed_api_file); sourceRemovedApiFile != "" {
		// If check api is disabled then make the source removed api file available for export.
		this.removedApiFile = android.PathForModuleSrc(ctx, sourceRemovedApiFile)
	}

	if Bool(d.properties.Write_sdk_values) {
		this.metadataDir = android.PathForModuleOut(ctx, "metalava", "metadata")
		cmd.FlagWithArg("--sdk-values ", this.metadataDir.String())
	}

	if stubsDir.Valid() {
		if Bool(d.properties.Create_doc_stubs) {
			cmd.FlagWithArg("--doc-stubs ", stubsDir.String())
		} else {
			cmd.FlagWithArg("--stubs ", stubsDir.String())
			if !Bool(d.properties.Output_javadoc_comments) {
				cmd.Flag("--exclude-documentation-from-stubs")
			}
		}
	}

	if Bool(d.properties.Annotations_enabled) {
		cmd.Flag("--include-annotations")

		cmd.FlagWithArg("--exclude-annotation ", "androidx.annotation.RequiresApi")

		migratingNullability := String(d.properties.Previous_api) != ""
		if migratingNullability {
			previousApi := android.PathForModuleSrc(ctx, String(d.properties.Previous_api))
			cmd.FlagWithInput("--migrate-nullness ", previousApi)
		}

		if s := String(d.properties.Validate_nullability_from_list); s != "" {
			cmd.FlagWithInput("--validate-nullability-from-list ", android.PathForModuleSrc(ctx, s))
		}

		validatingNullability :=
			strings.Contains(String(d.Javadoc.properties.Args), "--validate-nullability-from-merged-stubs") ||
				String(d.properties.Validate_nullability_from_list) != ""
		if validatingNullability {
			this.nullabilityWarningsFile = android.PathForModuleOut(ctx, "metalava", ctx.ModuleName()+"_nullability_warnings.txt")
			cmd.FlagWithOutput("--nullability-warnings-txt ", this.nullabilityWarningsFile)
		}

		this.annotationsZip = android.PathForModuleOut(ctx, "metalava", ctx.ModuleName()+"_annotations.zip")
		cmd.FlagWithOutput("--extract-annotations ", this.annotationsZip)

		if len(d.properties.Merge_annotations_dirs) != 0 {
			ctx.VisitDirectDepsWithTag(metalavaMergeAnnotationsDirTag, func(m android.Module) {
				if t, ok := m.(*ExportedDroiddocDir); ok {
					cmd.FlagWithArg("--merge-qualifier-annotations ", t.dir.String()).Implicits(t.deps)
				} else {
					ctx.PropertyErrorf("merge_annotations_dirs",
						"module %q is not a metalava merge-annotations dir", ctx.OtherModuleName(m))
				}
			})
		}

		// TODO(tnorbye): find owners to fix these warnings when annotation was enabled.
		cmd.FlagWithArg("--hide ", "HiddenTypedefConstant").
			FlagWithArg("--hide ", "SuperfluousPrefix").
			FlagWithArg("--hide ", "AnnotationExtraction").
			// b/222738070
			FlagWithArg("--hide ", "BannedThrow").
			// b/223382732
			FlagWithArg("--hide ", "ChangedDefault")
	}

	ctx.VisitDirectDepsWithTag(metalavaMergeInclusionAnnotationsDirTag, func(m android.Module) {
		if t, ok := m.(*ExportedDroiddocDir); ok {
			cmd.FlagWithArg("--merge-inclusion-annotations ", t.dir.String()).Implicits(t.deps)
		} else {
			ctx.PropertyErrorf("merge_inclusion_annotations_dirs",
				"module %q is not a metalava merge-annotations dir", ctx.OtherModuleName(m))
		}
	})

	var apiVersions android.Path
	if proptools.Bool(d.properties.Api_levels_annotations_enabled) {
		if len(d.properties.Api_levels_annotations_dirs) == 0 {
			ctx.PropertyErrorf("api_levels_annotations_dirs",
				"has to be non-empty if api levels annotations was enabled!")
		}

		this.apiVersionsXml = android.PathForModuleOut(ctx, "metalava", "api-versions.xml")
		cmd.FlagWithOutput("--generate-api-levels ", this.apiVersionsXml)

		filename := proptools.StringDefault(d.properties.Api_levels_jar_filename, "android.jar")

		var dirs []string
		var extensions_dir string
		ctx.VisitDirectDepsWithTag(metalavaAPILevelsAnnotationsDirTag, func(m android.Module) {
			if t, ok := m.(*ExportedDroiddocDir); ok {
				extRegex := regexp.MustCompile(t.dir.String() + `/extensions/[0-9]+/public/.*\.jar`)

				// Grab the first extensions_dir and we find while scanning ExportedDroiddocDir.deps;
				// ideally this should be read from prebuiltApis.properties.Extensions_*
				for _, dep := range t.deps {
					if extRegex.MatchString(dep.String()) && d.properties.Extensions_info_file != nil {
						if extensions_dir == "" {
							extensions_dir = t.dir.String() + "/extensions"
						}
						cmd.Implicit(dep)
					}
					if dep.Base() == filename {
						cmd.Implicit(dep)
					}
					if filename != "android.jar" && dep.Base() == "android.jar" {
						// Metalava implicitly searches these patterns:
						//  prebuilts/tools/common/api-versions/android-%/android.jar
						//  prebuilts/sdk/%/public/android.jar
						// Add android.jar files from the api_levels_annotations_dirs directories to try
						// to satisfy these patterns.  If Metalava can't find a match for an API level
						// between 1 and 28 in at least one pattern it will fail.
						cmd.Implicit(dep)
					}
				}

				dirs = append(dirs, t.dir.String())
			} else {
				ctx.PropertyErrorf("api_levels_annotations_dirs",
					"module %q is not a metalava api-levels-annotations dir", ctx.OtherModuleName(m))
			}
		})

		// Add all relevant --android-jar-pattern patterns for Metalava.
		// When parsing a stub jar for a specific version, Metalava picks the first pattern that defines
		// an actual file present on disk (in the order the patterns were passed). For system APIs for
		// privileged apps that are only defined since API level 21 (Lollipop), fallback to public stubs
		// for older releases. Similarly, module-lib falls back to system API.
		var sdkDirs []string
		switch proptools.StringDefault(d.properties.Api_levels_sdk_type, "public") {
		case "system-server":
			sdkDirs = []string{"system-server", "module-lib", "system", "public"}
		case "module-lib":
			sdkDirs = []string{"module-lib", "system", "public"}
		case "system":
			sdkDirs = []string{"system", "public"}
		case "public":
			sdkDirs = []string{"public"}
		default:
			ctx.PropertyErrorf("api_levels_sdk_type", "needs to be one of %v", allowedApiLevelSdkTypes)
			return
		}

		for _, sdkDir := range sdkDirs {
			for _, dir := range dirs {
				cmd.FlagWithArg("--android-jar-pattern ", fmt.Sprintf("%s/%%/%s/%s", dir, sdkDir, filename))
			}
		}

		if d.properties.Extensions_info_file != nil {
			if extensions_dir == "" {
				ctx.ModuleErrorf("extensions_info_file set, but no SDK extension dirs found")
			}
			info_file := android.PathForModuleSrc(ctx, *d.properties.Extensions_info_file)
			cmd.Implicit(info_file)
			cmd.FlagWithArg("--sdk-extensions-root ", extensions_dir)
			cmd.FlagWithArg("--sdk-extensions-info ", info_file.String())
		}

		apiVersions = this.apiVersionsXml
	} else {
		ctx.VisitDirectDepsWithTag(metalavaAPILevelsModuleTag, func(m android.Module) {
			if s, ok := m.(*Droidstubs); ok {
				apiVersions = s.metalavaPart.apiVersionsXml
			} else {
				ctx.PropertyErrorf("api_levels_module",
					"module %q is not a droidstubs module", ctx.OtherModuleName(m))
			}
		})
	}
	if apiVersions != nil {
		cmd.FlagWithArg("--current-version ", ctx.Config().PlatformSdkVersion().String())
		cmd.FlagWithArg("--current-codename ", ctx.Config().PlatformSdkCodename())
		cmd.FlagWithInput("--apply-api-levels ", apiVersions)
	}

	d.expandArgs(ctx, cmd)

	for _, o := range d.Javadoc.properties.Out {
		cmd.ImplicitOutput(android.PathForModuleGen(ctx, o))
	}

	// Add options for the other optional tasks: API-lint and check-released.
	// We generate separate timestamp files for them.

	doApiLint := false
	doCheckReleased := false

	// Add API lint options.

	if BoolDefault(d.properties.Check_api.Api_lint.Enabled, false) {
		doApiLint = true

		newSince := android.OptionalPathForModuleSrc(ctx, d.properties.Check_api.Api_lint.New_since)
		if newSince.Valid() {
			cmd.FlagWithInput("--api-lint ", newSince.Path())
		} else {
			cmd.Flag("--api-lint")
		}
		this.apiLintReport = android.PathForModuleOut(ctx, "metalava", "api_lint_report.txt")
		cmd.FlagWithOutput("--report-even-if-suppressed ", this.apiLintReport) // TODO:  Change to ":api-lint"

		// TODO(b/154317059): Clean up this allowlist by baselining and/or checking in last-released.
		if d.Name() != "android.car-system-stubs-docs" &&
			d.Name() != "android.car-stubs-docs" {
			cmd.Flag("--lints-as-errors")
			cmd.Flag("--warnings-as-errors") // Most lints are actually warnings.
		}

		baselineFile := android.OptionalPathForModuleSrc(ctx, d.properties.Check_api.Api_lint.Baseline_file)
		updatedBaselineOutput := android.PathForModuleOut(ctx, "metalava", "api_lint_baseline.txt")
		this.apiLintTimestamp = android.PathForModuleOut(ctx, "metalava", "api_lint.timestamp")

		// Note this string includes a special shell quote $' ... ', which decodes the "\n"s.
		//
		// TODO: metalava also has a slightly different message hardcoded. Should we unify this
		// message and metalava's one?
		msg := `$'` + // Enclose with $' ... '
			`************************************************************\n` +
			`Your API changes are triggering API Lint warnings or errors.\n` +
			`To make these errors go away, fix the code according to the\n` +
			`error and/or warning messages above.\n` +
			`\n` +
			`If it is not possible to do so, there are workarounds:\n` +
			`\n` +
			`1. You can suppress the errors with @SuppressLint("<id>")\n` +
			`   where the <id> is given in brackets in the error message above.\n`

		if baselineFile.Valid() {
			cmd.FlagWithInput("--baseline:api-lint ", baselineFile.Path())
			cmd.FlagWithOutput("--update-baseline:api-lint ", updatedBaselineOutput)

			msg += fmt.Sprintf(``+
				`2. You can update the baseline by executing the following\n`+
				`   command:\n`+
				`       (cd $ANDROID_BUILD_TOP && cp \\\n`+
				`       "%s" \\\n`+
				`       "%s")\n`+
				`   To submit the revised baseline.txt to the main Android\n`+
				`   repository, you will need approval.\n`, updatedBaselineOutput, baselineFile.Path())
		} else {
			msg += fmt.Sprintf(``+
				`2. You can add a baseline file of existing lint failures\n`+
				`   to the build rule of %s.\n`, d.Name())
		}
		// Note the message ends with a ' (single quote), to close the $' ... ' .
		msg += `************************************************************\n'`

		cmd.FlagWithArg("--error-message:api-lint ", msg)
	}

	// Add "check released" options. (Detect incompatible API changes from the last public release)

	if apiCheckEnabled(ctx, d.properties.Check_api.Last_released, "last_released") {
		doCheckReleased = true

		if len(d.Javadoc.properties.Out) > 0 {
			ctx.PropertyErrorf("out", "out property may not be combined with check_api")
		}

		apiFile := android.PathForModuleSrc(ctx, String(d.properties.Check_api.Last_released.Api_file))
		removedApiFile := android.PathForModuleSrc(ctx, String(d.properties.Check_api.Last_released.Removed_api_file))
		baselineFile := android.OptionalPathForModuleSrc(ctx, d.properties.Check_api.Last_released.Baseline_file)
		updatedBaselineOutput := android.PathForModuleOut(ctx, "metalava", "last_released_baseline.txt")

		this.checkLastReleasedApiTimestamp = android.PathForModuleOut(ctx, "metalava", "check_last_released_api.timestamp")

		cmd.FlagWithInput("--check-compatibility:api:released ", apiFile)
		cmd.FlagWithInput("--check-compatibility:removed:released ", removedApiFile)

		if baselineFile.Valid() {
			cmd.FlagWithInput("--baseline:compatibility:released ", baselineFile.Path())
			cmd.FlagWithOutput("--update-baseline:compatibility:released ", updatedBaselineOutput)
		}

		// Note this string includes quote ($' ... '), which decodes the "\n"s.
		msg := `$'\n******************************\n` +
			`You have tried to change the API from what has been previously released in\n` +
			`an SDK.  Please fix the errors listed above.\n` +
			`******************************\n'`

		cmd.FlagWithArg("--error-message:compatibility:released ", msg)
	}

	if generateStubs {
		rule.Command().
			BuiltTool("soong_zip").
			Flag("-write_if_changed").
			Flag("-jar").
			FlagWithOutput("-o ", d.Javadoc.stubsSrcJar).
			FlagWithArg("-C ", stubsDir.String()).
			FlagWithArg("-D ", stubsDir.String())
	}

	if Bool(d.properties.Write_sdk_values) {
		this.metadataZip = android.PathForModuleOut(ctx, "metalava", ctx.ModuleName()+"-metadata.zip")
		rule.Command().
			BuiltTool("soong_zip").
			Flag("-write_if_changed").
			Flag("-d").
			FlagWithOutput("-o ", this.metadataZip).
			FlagWithArg("-C ", this.metadataDir.String()).
			FlagWithArg("-D ", this.metadataDir.String())
	}

	// TODO: We don't really need two separate API files, but this is a remnant of how
	// we used to run metalava separately for API lint and the "last_released" check. Unify them.
	if doApiLint {
		rule.Command().Text("touch").Output(this.apiLintTimestamp)
	}
	if doCheckReleased {
		rule.Command().Text("touch").Output(this.checkLastReleasedApiTimestamp)
	}

	// TODO(b/183630617): rewrapper doesn't support restat rules
	if !metalavaUseRbe(ctx) {
		rule.Restat()
	}

	zipSyncCleanupCmd(rule, srcJarDir)

	rule.Build("metalava", "metalava merged")
}

func (this *metalavaPart) GenerateMkExtraFooters(w io.Writer, name, prefix, moduleDir string) {
	moduleName := this.droidstubs.Name()

	if this.apiFile != nil {
		fmt.Fprintf(w, ".PHONY: %s %s.txt\n", moduleName, moduleName)
		fmt.Fprintf(w, "%s %s.txt: %s\n", moduleName, moduleName, this.apiFile)
	}
	if this.removedApiFile != nil {
		fmt.Fprintf(w, ".PHONY: %s %s.txt\n", moduleName, moduleName)
		fmt.Fprintf(w, "%s %s.txt: %s\n", moduleName, moduleName, this.removedApiFile)
	}
	if this.checkLastReleasedApiTimestamp != nil {
		fmt.Fprintln(w, ".PHONY:", moduleName+"-check-last-released-api")
		fmt.Fprintln(w, moduleName+"-check-last-released-api:",
			this.checkLastReleasedApiTimestamp.String())

		fmt.Fprintln(w, ".PHONY: checkapi")
		fmt.Fprintln(w, "checkapi:", this.checkLastReleasedApiTimestamp.String())

		fmt.Fprintln(w, ".PHONY: droidcore")
		fmt.Fprintln(w, "droidcore: checkapi")
	}
	if this.apiLintTimestamp != nil {
		fmt.Fprintln(w, ".PHONY:", moduleName+"-api-lint")
		fmt.Fprintln(w, moduleName+"-api-lint:", this.apiLintTimestamp.String())

		fmt.Fprintln(w, ".PHONY: checkapi")
		fmt.Fprintln(w, "checkapi:", moduleName+"-api-lint")

		fmt.Fprintln(w, ".PHONY: droidcore")
		fmt.Fprintln(w, "droidcore: checkapi")

		if this.apiLintReport != nil {
			fmt.Fprintf(w, "$(call dist-for-goals,%s,%s:%s)\n", moduleName+"-api-lint",
				this.apiLintReport.String(), "apilint/"+moduleName+"-lint-report.txt")
			fmt.Fprintf(w, "$(call declare-0p-target,%s)\n", this.apiLintReport.String())
		}
	}
}

//
// ApiCheck Invocation
//

type apiCheckPart struct {
	droidstubs               *Droidstubs
	checkCurrentApiTimestamp android.WritablePath
}

func (this *apiCheckPart) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	d := this.droidstubs

	if apiCheckEnabled(ctx, d.properties.Check_api.Current, "current") {

		if len(d.Javadoc.properties.Out) > 0 {
			ctx.PropertyErrorf("out", "out property may not be combined with check_api")
		}

		apiFile := android.PathForModuleSrc(ctx, String(d.properties.Check_api.Current.Api_file))
		removedApiFile := android.PathForModuleSrc(ctx, String(d.properties.Check_api.Current.Removed_api_file))
		baselineFile := android.OptionalPathForModuleSrc(ctx, d.properties.Check_api.Current.Baseline_file)

		if baselineFile.Valid() {
			ctx.PropertyErrorf("baseline_file", "current API check can't have a baseline file. (module %s)", ctx.ModuleName())
		}

		this.checkCurrentApiTimestamp = android.PathForModuleOut(ctx, "metalava", "check_current_api.timestamp")

		rule := android.NewRuleBuilder(pctx, ctx)

		// Diff command line.
		// -F matches the closest "opening" line, such as "package android {"
		// and "  public class Intent {".
		diff := `diff -u -F '{ *$'`

		rule.Command().Text("( true")
		rule.Command().
			Text(diff).
			Input(apiFile).Input(d.metalavaPart.apiFile)

		rule.Command().
			Text(diff).
			Input(removedApiFile).Input(d.metalavaPart.removedApiFile)

		msg := fmt.Sprintf(`\n******************************\n`+
			`You have tried to change the API from what has been previously approved.\n\n`+
			`To make these errors go away, you have two choices:\n`+
			`   1. You can add '@hide' javadoc comments (and remove @SystemApi/@TestApi/etc)\n`+
			`      to the new methods, etc. shown in the above diff.\n\n`+
			`   2. You can update current.txt and/or removed.txt by executing the following command:\n`+
			`         m %s-update-current-api\n\n`+
			`      To submit the revised current.txt to the main Android repository,\n`+
			`      you will need approval.\n`+
			`******************************\n`, ctx.ModuleName())

		rule.Command().
			Text("touch").Output(this.checkCurrentApiTimestamp).
			Text(") || (").
			Text("echo").Flag("-e").Flag(`"` + msg + `"`).
			Text("; exit 38").
			Text(")")

		rule.Build("metalavaCurrentApiCheck", "check current API")

	}
}

func (this *apiCheckPart) GenerateMkExtraFooters(w io.Writer, name, prefix, moduleDir string) {
	if this.checkCurrentApiTimestamp != nil {
		moduleName := this.droidstubs.Name()

		fmt.Fprintln(w, ".PHONY:", moduleName+"-check-current-api")
		fmt.Fprintln(w, moduleName+"-check-current-api:", this.checkCurrentApiTimestamp.String())

		fmt.Fprintln(w, ".PHONY: checkapi")
		fmt.Fprintln(w, "checkapi:", this.checkCurrentApiTimestamp.String())

		fmt.Fprintln(w, ".PHONY: droidcore")
		fmt.Fprintln(w, "droidcore: checkapi")
	}
}

//
// Update API Part
//

type updateApiPart struct {
	droidstubs                *Droidstubs
	updateCurrentApiTimestamp android.WritablePath
}

func (this *updateApiPart) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	d := this.droidstubs

	if apiCheckEnabled(ctx, d.properties.Check_api.Current, "current") {
		apiFile := android.PathForModuleSrc(ctx, String(d.properties.Check_api.Current.Api_file))
		removedApiFile := android.PathForModuleSrc(ctx, String(d.properties.Check_api.Current.Removed_api_file))

		this.updateCurrentApiTimestamp = android.PathForModuleOut(ctx, "metalava", "update_current_api.timestamp")
		rule := android.NewRuleBuilder(pctx, ctx)

		rule.Command().Text("( true")

		rule.Command().
			Text("cp").Flag("-f").
			Input(d.metalavaPart.apiFile).Flag(apiFile.String())

		rule.Command().
			Text("cp").Flag("-f").
			Input(d.metalavaPart.removedApiFile).Flag(removedApiFile.String())

		msg := "failed to update public API"

		rule.Command().
			Text("touch").Output(this.updateCurrentApiTimestamp).
			Text(") || (").
			Text("echo").Flag("-e").Flag(`"` + msg + `"`).
			Text("; exit 38").
			Text(")")

		rule.Build("metalavaCurrentApiUpdate", "update current API")
	}
}

func (this *updateApiPart) GenerateMkExtraFooters(w io.Writer, name, prefix, moduleDir string) {
	if this.updateCurrentApiTimestamp != nil {
		moduleName := this.droidstubs.Name()

		fmt.Fprintln(w, ".PHONY:", moduleName+"-update-current-api")
		fmt.Fprintln(w, moduleName+"-update-current-api:", this.updateCurrentApiTimestamp.String())

		fmt.Fprintln(w, ".PHONY: update-api")
		fmt.Fprintln(w, "update-api:", this.updateCurrentApiTimestamp.String())
	}
}

//
// Check Nullability Warnings invocation
//

type checkNullabilityWarningsPart struct {
	droidstubs                        *Droidstubs
	checkNullabilityWarningsTimestamp android.WritablePath
}

func (this *checkNullabilityWarningsPart) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	d := this.droidstubs

	if String(d.properties.Check_nullability_warnings) != "" {
		if d.metalavaPart.nullabilityWarningsFile == nil {
			ctx.PropertyErrorf("check_nullability_warnings",
				"Cannot specify check_nullability_warnings unless validating nullability")
		}

		checkNullabilityWarnings := android.PathForModuleSrc(ctx, String(d.properties.Check_nullability_warnings))

		this.checkNullabilityWarningsTimestamp = android.PathForModuleOut(ctx, "metalava", "check_nullability_warnings.timestamp")

		msg := fmt.Sprintf(`\n******************************\n`+
			`The warnings encountered during nullability annotation validation did\n`+
			`not match the checked in file of expected warnings. The diffs are shown\n`+
			`above. You have two options:\n`+
			`   1. Resolve the differences by editing the nullability annotations.\n`+
			`   2. Update the file of expected warnings by running:\n`+
			`         cp %s %s\n`+
			`       and submitting the updated file as part of your change.`,
			d.metalavaPart.nullabilityWarningsFile, checkNullabilityWarnings)

		rule := android.NewRuleBuilder(pctx, ctx)

		rule.Command().
			Text("(").
			Text("diff").Input(checkNullabilityWarnings).Input(d.metalavaPart.nullabilityWarningsFile).
			Text("&&").
			Text("touch").Output(this.checkNullabilityWarningsTimestamp).
			Text(") || (").
			Text("echo").Flag("-e").Flag(`"` + msg + `"`).
			Text("; exit 38").
			Text(")")

		rule.Build("nullabilityWarningsCheck", "nullability warnings check")
	}
}

var _ android.ApiProvider = (*Droidstubs)(nil)

type bazelJavaApiContributionAttributes struct {
	Api         bazel.LabelAttribute
	Api_surface *string
}

func (d *Droidstubs) ConvertWithApiBp2build(ctx android.TopDownMutatorContext) {
	props := bazel.BazelTargetModuleProperties{
		Rule_class:        "java_api_contribution",
		Bzl_load_location: "//build/bazel/rules/apis:java_api_contribution.bzl",
	}
	apiFile := d.properties.Check_api.Current.Api_file
	// Do not generate a target if check_api is not set
	if apiFile == nil {
		return
	}
	attrs := &bazelJavaApiContributionAttributes{
		Api: *bazel.MakeLabelAttribute(
			android.BazelLabelForModuleSrcSingle(ctx, proptools.String(apiFile)).Label,
		),
		Api_surface: proptools.StringPtr(bazelApiSurfaceName(d.Name())),
	}
	ctx.CreateBazelTargetModule(props, android.CommonAttributes{
		Name: android.ApiContributionTargetName(ctx.ModuleName()),
	}, attrs)
}

func (d *Droidstubs) createApiContribution(ctx android.DefaultableHookContext) {
	api_file := d.properties.Check_api.Current.Api_file
	api_surface := d.properties.Api_surface

	props := struct {
		Name        *string
		Api_surface *string
		Api_file    *string
		Visibility  []string
	}{}

	props.Name = proptools.StringPtr(d.Name() + ".api.contribution")
	props.Api_surface = api_surface
	props.Api_file = api_file
	props.Visibility = []string{"//visibility:override", "//visibility:public"}

	ctx.CreateModule(ApiContributionFactory, &props)
}

// TODO (b/262014796): Export the API contributions of CorePlatformApi
// A map to populate the api surface of a droidstub from a substring appearing in its name
// This map assumes that droidstubs (either checked-in or created by java_sdk_library)
// use a strict naming convention
var (
	droidstubsModuleNamingToSdkKind = map[string]android.SdkKind{
		//public is commented out since the core libraries use public in their java_sdk_library names
		"intracore":     android.SdkIntraCore,
		"intra.core":    android.SdkIntraCore,
		"system_server": android.SdkSystemServer,
		"system-server": android.SdkSystemServer,
		"system":        android.SdkSystem,
		"module_lib":    android.SdkModule,
		"module-lib":    android.SdkModule,
		"platform.api":  android.SdkCorePlatform,
		"test":          android.SdkTest,
		"toolchain":     android.SdkToolchain,
	}
)

// A helper function that returns the api surface of the corresponding java_api_contribution Bazel target
// The api_surface is populated using the naming convention of the droidstubs module.
func bazelApiSurfaceName(name string) string {
	// Sort the keys so that longer strings appear first
	// Otherwise substrings like system will match both system and system_server
	sortedKeys := make([]string, 0)
	for key := range droidstubsModuleNamingToSdkKind {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return len(sortedKeys[i]) > len(sortedKeys[j])
	})
	for _, sortedKey := range sortedKeys {
		if strings.Contains(name, sortedKey) {
			sdkKind := droidstubsModuleNamingToSdkKind[sortedKey]
			return sdkKind.String() + "api"
		}
	}
	// Default is publicapi
	return android.SdkPublic.String() + "api"
}

func (this *checkNullabilityWarningsPart) GenerateMkExtraFooters(w io.Writer, name, prefix, moduleDir string) {
	if this.checkNullabilityWarningsTimestamp != nil {
		moduleName := this.droidstubs.Name()

		fmt.Fprintln(w, ".PHONY:", moduleName+"-check-nullability-warnings")
		fmt.Fprintln(w, moduleName+"-check-nullability-warnings:", this.checkNullabilityWarningsTimestamp.String())

		fmt.Fprintln(w, ".PHONY:", "droidcore")
		fmt.Fprintln(w, "droidcore: ", moduleName+"-check-nullability-warnings")
	}
}

var _ android.PrebuiltInterface = (*PrebuiltStubsSources)(nil)

func metalavaUseRbe(ctx android.ModuleContext) bool {
	return ctx.Config().UseRBE() && ctx.Config().IsEnvTrue("RBE_METALAVA")
}

type PrebuiltStubsSourcesProperties struct {
	Srcs []string `android:"path"`
}

type PrebuiltStubsSources struct {
	android.ModuleBase
	android.DefaultableModuleBase
	prebuilt android.Prebuilt

	properties PrebuiltStubsSourcesProperties

	stubsSrcJar android.Path
}

func (p *PrebuiltStubsSources) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "":
		return android.Paths{p.stubsSrcJar}, nil
	default:
		return nil, fmt.Errorf("unsupported module reference tag %q", tag)
	}
}

func (d *PrebuiltStubsSources) StubsSrcJar() android.Path {
	return d.stubsSrcJar
}

func (p *PrebuiltStubsSources) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if len(p.properties.Srcs) != 1 {
		ctx.PropertyErrorf("srcs", "must only specify one directory path or srcjar, contains %d paths", len(p.properties.Srcs))
		return
	}

	src := p.properties.Srcs[0]
	if filepath.Ext(src) == ".srcjar" {
		// This is a srcjar. We can use it directly.
		p.stubsSrcJar = android.PathForModuleSrc(ctx, src)
	} else {
		outPath := android.PathForModuleOut(ctx, ctx.ModuleName()+"-"+"stubs.srcjar")

		// This is a directory. Glob the contents just in case the directory does not exist.
		srcGlob := src + "/**/*"
		srcPaths := android.PathsForModuleSrc(ctx, []string{srcGlob})

		// Although PathForModuleSrc can return nil if either the path doesn't exist or
		// the path components are invalid it won't in this case because no components
		// are specified and the module directory must exist in order to get this far.
		srcDir := android.PathForModuleSrc(ctx).(android.SourcePath).Join(ctx, src)

		rule := android.NewRuleBuilder(pctx, ctx)
		rule.Command().
			BuiltTool("soong_zip").
			Flag("-write_if_changed").
			Flag("-jar").
			FlagWithOutput("-o ", outPath).
			FlagWithArg("-C ", srcDir.String()).
			FlagWithRspFileInputList("-r ", outPath.ReplaceExtension(ctx, "rsp"), srcPaths)
		rule.Restat()
		rule.Build("zip src", "Create srcjar from prebuilt source")
		p.stubsSrcJar = outPath
	}
}

func (p *PrebuiltStubsSources) Prebuilt() *android.Prebuilt {
	return &p.prebuilt
}

func (p *PrebuiltStubsSources) Name() string {
	return p.prebuilt.Name(p.ModuleBase.Name())
}

// prebuilt_stubs_sources imports a set of java source files as if they were
// generated by droidstubs.
//
// By default, a prebuilt_stubs_sources has a single variant that expects a
// set of `.java` files generated by droidstubs.
//
// Specifying `host_supported: true` will produce two variants, one for use as a dependency of device modules and one
// for host modules.
//
// Intended only for use by sdk snapshots.
func PrebuiltStubsSourcesFactory() android.Module {
	module := &PrebuiltStubsSources{}

	module.AddProperties(&module.properties)

	android.InitPrebuiltModule(module, &module.properties.Srcs)
	InitDroiddocModule(module, android.HostAndDeviceSupported)
	return module
}

func apiCheckEnabled(ctx android.ModuleContext, apiToCheck ApiToCheck, apiVersionTag string) bool {
	if ctx.Config().IsEnvTrue("WITHOUT_CHECK_API") {
		return false
	} else if String(apiToCheck.Api_file) != "" && String(apiToCheck.Removed_api_file) != "" {
		return true
	} else if String(apiToCheck.Api_file) != "" {
		panic("for " + apiVersionTag + " removed_api_file has to be non-empty!")
	} else if String(apiToCheck.Removed_api_file) != "" {
		panic("for " + apiVersionTag + " api_file has to be non-empty!")
	}

	return false
}
