// Copyright 2018 Google Inc. All rights reserved.
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
	"android/soong/android"
	"android/soong/java/config"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

var (
	nonStdDroiddoc = pctx.AndroidStaticRule("nonStdDroiddoc",
		blueprint.RuleParams{
			Command: `rm -rf "$outDir" "$srcJarDir" "$stubsDir" && mkdir -p "$outDir" "$srcJarDir" "$stubsDir" && ` +
				`${config.ExtractSrcJarsCmd} $srcJarDir $srcJarDir/list $srcJars && ` +
				`${config.JavadocCmd} -encoding UTF-8 -source 1.8 @$out.rsp @$srcJarDir/list ` +
				`-J-Xmx1600m -J-XX:-OmitStackTraceInFastThrow -XDignore.symbol.file ` +
				`-quiet -doclet com.google.doclava.Doclava -docletpath ${config.JsilverJar}:${config.DoclavaJar} ` +
				`-templatedir $customTemplateDir $htmlDirArgs $htmlDir2Args ` +
				`$bootclasspathArgs $classpathArgs -sourcepath $sourcepath ` +
				`-d $outDir $currentBuild $currentTime $droiddocOptions -stubs $stubsDir && ` +
				`${config.SoongZipCmd} -d -o $docZip -C $outDir -D $outDir && ` +
				`${config.SoongZipCmd} -jar -o $out.tmp -C $stubsDir -D $stubsDir && ` +
				`${config.Ziptime} $out.tmp && ` +
				`(if cmp -s $out.tmp $out ; then rm $out.tmp ; else mv $out.tmp $out ; fi )`,
			CommandDeps: []string{
				"${config.ExtractSrcJarsCmd}",
				"${config.JavadocCmd}",
				"${config.JsilverJar}",
				"${config.DoclavaJar}",
				"${config.SoongZipCmd}",
				"${config.Ziptime}",
			},
			Rspfile:        "$out.rsp",
			RspfileContent: "$in",
			Restat:         true,
		},
		"outDir", "srcJarDir", "stubsDir", "srcJars", "customTemplateDir",
		"htmlDirArgs", "htmlDir2Args", "bootclasspathArgs", "classpathArgs", "sourcepath",
		"currentBuild", "currentTime", "droiddocOptions", "docZip")

	stdDroiddoc = pctx.AndroidStaticRule("stdDroiddoc",
		blueprint.RuleParams{
			Command: `rm -rf "$outDir" "$srcJarDir" "$stubsDir" && mkdir -p "$outDir" "$srcJarDir" "$stubsDir" && ` +
				`${config.ExtractSrcJarsCmd} $srcJarDir $srcJarDir/list $srcJars && ` +
				`${config.JavadocCmd} -encoding UTF-8 @$out.rsp @$srcJarDir/list ` +
				`-J-Xmx1024m -XDignore.symbol.file -Xdoclint:none ` +
				`$bootClasspathArgs $classpathArgs -sourcepath $sourcepath ` +
				`-d $outDir -quiet && ` +
				`${config.SoongZipCmd} -d -o $docZip -C $outDir -D $outDir && ` +
				`${config.SoongZipCmd} -jar -o $out -C $stubsDir -D $stubsDir`,
			CommandDeps: []string{
				"${config.ExtractSrcJarsCmd}",
				"${config.JavadocCmd}",
				"${config.SoongZipCmd}",
			},
			Rspfile:        "$out.rsp",
			RspfileContent: "$in",
			Restat:         true,
		},
		"outDir", "srcJarDir", "stubsDir", "srcJars", "bootClasspathArgs", "classpathArgs", "sourcepath", "docZip")
)

func init() {
	android.RegisterModuleType("droiddoc", DroiddocFactory)
	android.RegisterModuleType("droiddoc_host", DroiddocHostFactory)
}

type DroiddocProperties struct {
	// list of source files used to compile the Java module.  May be .java, .logtags, .proto,
	// or .aidl files.
	Srcs []string `android:"arch_variant"`

	// list of directories rooted at the Android.bp file that will
	// be added to the search paths for finding source files when passing package names.
	Local_sourcepaths []string `android:"arch_variant"`

	// directory relative to ANDROID_BUILD_TOP that contains doc templates files.
	Custom_template_dir string `android:"arch_variant"`

	// list of source files that should not be used to build the Java module.
	// This is most useful in the arch/multilib variants to remove non-common files
	// filegroup or genrule can be included within this property.
	Exclude_srcs []string `android:"arch_variant"`

	// directory relative to ANDROID_BUILD_TOP is passed with "-htmldir" when compiling with
	// non-std doclet.
	Html_dir string `android:"arch_variant"`

	// directory relative to ANDROID_BUILD_TOP is passed with "-htmldir2" when compiling with
	// non-std doclet.
	Html_dir2 string `android:"arch_variant"`

	// list of module-specific flags that will be used for droiddoc compiles.
	Droiddoc_options []string `android:"arch_variant"`

	// list of of java libraries that will be in the classpath.
	Libs []string `android:"arch_variant"`

	// a list of strings is passed with "-hdf" when compiling with non-std doclet.
	Hdf []string `android:"arch_variant"`

	// filename is passed with "-proofread" when compiling with non-std doclet.
	// the file will be generated within out dir.
	Proofread_file string `android:"arch_variant"`

	// filename is passed with "-proofread" when compiling with non-std doclet.
	// the file will be generated relative to "-o" dir when compiling with non-std doclet.
	Todo_file string `android:"arch_variant"`

	// filename is passed with "-api" when compiling ahat with non-std doclet.
	// the file will be generated within out dir.
	Ahat_api string `android:"arch_variant"`

	// filename is passed with "-removeApi" when compiling ahat with non-std doclet.
	// the file will be generated within out dir.
	Ahat_removed_api string `android:"arch_variant"`

	// a list of files relative to current module source directory, and is passed with
	// "-knowntags" when compiling with non-std doclet.
	// filegroup or genrule can be included within this property.
	Knowntags []string `android:"arch_variant"`

	// If set to false, don't allow this module(-docs.zip) to be exported. Defaults to true.
	Installable *bool `android:"arch_variant"`

	// Defaults to false.
	Use_standard_doclet *bool `android:"arch_variant"`

	// if not blank, set to the version of the sdk to compile against
	Sdk_version *string `android:"arch_variant"`
}

type Droiddoc struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties DroiddocProperties

	docZip   android.Path
	stubsJar android.Path
}

func InitDroiddocModule(module android.DefaultableModule, hod android.HostOrDeviceSupported) {
	android.InitAndroidArchModule(module, hod, android.MultilibCommon)
	android.InitDefaultableModule(module)
}

func DroiddocFactory() android.Module {
	module := &Droiddoc{}

	module.AddProperties(&module.properties)

	InitDroiddocModule(module, android.HostAndDeviceSupported)
	return module
}

func DroiddocHostFactory() android.Module {
	module := &Droiddoc{}

	module.AddProperties(&module.properties)

	InitDroiddocModule(module, android.HostSupported)
	return module
}

func (d *Droiddoc) DepsMutator(ctx android.BottomUpMutatorContext) {
	if ctx.Device() {
		sdkDep := decodeSdkDep(ctx, String(d.properties.Sdk_version))
		if sdkDep.useDefaultLibs {
			ctx.AddDependency(ctx.Module(), bootClasspathTag, config.DefaultBootclasspathLibraries...)
			ctx.AddDependency(ctx.Module(), libTag, []string{"ext", "framework"}...)
		} else if sdkDep.useModule {
			ctx.AddDependency(ctx.Module(), bootClasspathTag, sdkDep.module)
		}
	}

	ctx.AddDependency(ctx.Module(), libTag, d.properties.Libs...)

	android.ExtractSourcesDeps(ctx, d.properties.Srcs)

	// exclude_srcs may contain filegroup or genrule.
	android.ExtractSourcesDeps(ctx, d.properties.Exclude_srcs)

	// knowntags  may contain filegroup or genrule.
	android.ExtractSourcesDeps(ctx, d.properties.Knowntags)
}

// glob the directory relative to ANDROID_BUILD_TOP.
func (d *Droiddoc) globDir(ctx android.ModuleContext, pattern string, excludes []string) []android.Path {
	paths, err := ctx.GlobWithDeps(pattern, excludes)
	if err != nil {
		ctx.ModuleErrorf("glob: %s", err.Error())
	}
	var ret []android.Path
	for _, p := range paths {
		ret = append(ret, android.PathForSource(ctx, p))
	}
	return ret
}

func (d *Droiddoc) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	var deps deps

	sdkDep := decodeSdkDep(ctx, String(d.properties.Sdk_version))
	if sdkDep.invalidVersion {
		ctx.AddMissingDependencies([]string{sdkDep.module})
	} else if sdkDep.useFiles {
		deps.bootClasspath = append(deps.bootClasspath, sdkDep.jar)
	}

	ctx.VisitDirectDeps(func(module android.Module) {
		otherName := ctx.OtherModuleName(module)
		tag := ctx.OtherModuleDependencyTag(module)

		switch dep := module.(type) {
		case Dependency:
			switch tag {
			case bootClasspathTag:
				deps.bootClasspath = append(deps.bootClasspath, dep.ImplementationJars()...)
			case libTag:
				java8Home := ctx.Config().Getenv("ANDROID_JAVA8_HOME")
				deps.classpath = append(deps.classpath, android.PathForSource(ctx, java8Home, "lib/tools.jar"))
				deps.classpath = append(deps.classpath, dep.ImplementationJars()...)
			default:
				panic(fmt.Errorf("unknown dependency %q for %q", otherName, ctx.ModuleName()))
			}
		case android.SourceFileProducer:
			switch tag {
			case libTag:
				checkProducesJars(ctx, dep)
				deps.classpath = append(deps.classpath, dep.Srcs()...)
			case android.DefaultsDepTag, android.SourceDepTag:
				// Nothing to do
			default:
				ctx.ModuleErrorf("dependency on genrule %q may only be in srcs, libs", otherName)
			}
		default:
			switch tag {
			case android.DefaultsDepTag, android.SourceDepTag:
				// Nothing to do
			default:
				ctx.ModuleErrorf("depends on non-java module %q", otherName)
			}
		}
	})
	// do not pass exclude_srcs directly when expanding srcFiles since exclude_srcs
	// may contain filegroup or genrule.
	srcFiles := ctx.ExpandSources(d.properties.Srcs, nil)

	// filter out excludes from srcsFiles.
	excludes := ctx.ExpandSources(d.properties.Exclude_srcs, nil)
	for _, e := range excludes {
		for j, s := range srcFiles {
			if s.String() == e.String() {
				srcFiles = append(srcFiles[:j], srcFiles[j+1:]...)
			}
		}
	}

	// srcs may depend on some genrule output.
	srcJars := srcFiles.FilterByExt(".srcjar")
	deps.etc = append(deps.etc, srcJars...)
	srcFiles = srcFiles.FilterOutByExt(".srcjar")

	docZip := android.PathForModuleOut(ctx, ctx.ModuleName()+"-"+"docs.zip")
	stubsJar := android.PathForModuleOut(ctx, ctx.ModuleName()+"-"+"stubs.jar")

	// droiddoc_options should have nothing to do with any file path within source tree.
	droiddocOptions := d.properties.Droiddoc_options

	if d.properties.Local_sourcepaths == nil {
		d.properties.Local_sourcepaths = append(d.properties.Local_sourcepaths, ".")
	}
	sourcepaths := android.PathsForModuleSrc(ctx, d.properties.Local_sourcepaths).Strings()
	sourcepaths = append(sourcepaths, deps.bootClasspath.Strings()...)
	sourcepaths = append(sourcepaths, deps.classpath.Strings()...)

	var implicits android.Paths
	implicits = append(implicits, deps.bootClasspath...)
	implicits = append(implicits, deps.classpath...)

	if Bool(d.properties.Use_standard_doclet) {
		var bootClasspathArgs, classpathArgs string
		if ctx.Config().UseOpenJDK9() {
			if len(deps.bootClasspath.Strings()) > 0 {
				// For OpenJDK 9 we use --patch-module to define the core libraries code.
				// TODO(tobiast): Reorganize this when adding proper support for OpenJDK 9
				// modules. Here we treat all code in core libraries as being in java.base
				// to work around the OpenJDK 9 module system. http://b/62049770
				bootClasspathArgs = "--patch-module=java.base=" + strings.Join(deps.bootClasspath.Strings(), ":")
			}
		} else {
			if len(deps.bootClasspath.Strings()) > 0 {
				// For OpenJDK 8 we can use -bootclasspath to define the core libraries code.
				bootClasspathArgs = "-bootclasspath " + strings.Join(deps.bootClasspath.Strings(), ":")
			}
		}
		if len(deps.classpath.Strings()) > 0 {
			classpathArgs = "-classpath " + strings.Join(deps.classpath.Strings(), ":")
		}

		implicits = append(implicits, deps.etc...)

		ctx.Build(pctx, android.BuildParams{
			Rule:           stdDroiddoc,
			Description:    "Droiddoc with standard doclet",
			Output:         stubsJar,
			ImplicitOutput: docZip,
			Inputs:         srcFiles,
			Implicits:      implicits,
			Args: map[string]string{
				"outDir":            android.PathForModuleOut(ctx, "docs", "out").String(),
				"srcJarDir":         android.PathForModuleOut(ctx, "docs", "srcjars").String(),
				"stubsDir":          android.PathForModuleOut(ctx, "docs", "stubsDir").String(),
				"bootClasspathArgs": bootClasspathArgs,
				"classpathArgs":     classpathArgs,
				"sourcepath":        strings.Join(sourcepaths, ":"),
				"docZip":            docZip.String(),
			},
		})
	} else {
		var bootClasspathArgs, classpathArgs string
		if len(deps.bootClasspath.Strings()) > 0 {
			bootClasspathArgs = "-bootclasspath " + strings.Join(deps.bootClasspath.Strings(), ":")
		}
		if len(deps.classpath.Strings()) > 0 {
			classpathArgs = "-classpath " + strings.Join(deps.classpath.Strings(), ":")
		}

		// templateDir is relative to ANDROID_BUILD_TOP instead of current module.
		templateDir := android.PathForSource(ctx, d.properties.Custom_template_dir).String()
		deps.etc = append(deps.etc, d.globDir(ctx, filepath.Join(templateDir, "**/*"), nil)...)

		// htmlDir is relative to ANDROID_BUILD_TOP instead of current module.
		var htmlDirArgs string
		if d.properties.Html_dir != "" {
			htmlDir := android.PathForSource(ctx, d.properties.Html_dir).String()
			deps.etc = append(deps.etc, d.globDir(ctx, filepath.Join(htmlDir, "**/*"), nil)...)
			htmlDirArgs = "-htmldir " + htmlDir
		}

		// htmlDir2 is relative to ANDROID_BUILD_TOP instead of current module.
		var htmlDir2Args string
		if d.properties.Html_dir2 != "" {
			htmlDir2 := android.PathForSource(ctx, d.properties.Html_dir2).String()
			deps.etc = append(deps.etc, d.globDir(ctx, filepath.Join(htmlDir2, "**/*"), nil)...)
			htmlDirArgs = "-htmldir2 " + htmlDir2
		}

		var implicitOutputs android.WritablePaths

		knownTags := ctx.ExpandSources(d.properties.Knowntags, nil)
		deps.etc = append(deps.etc, knownTags...)

		for _, kt := range knownTags {
			droiddocOptions = append(droiddocOptions, "-knowntags "+kt.String())
		}
		for _, hdf := range d.properties.Hdf {
			droiddocOptions = append(droiddocOptions, "-hdf "+hdf)
		}

		if d.properties.Proofread_file != "" {
			proofreadFile := android.PathForModuleOut(ctx, d.properties.Proofread_file)
			droiddocOptions = append(droiddocOptions, "-proofread "+proofreadFile.String())
		}
		if d.properties.Todo_file != "" {
			// tricky part:
			// we should not compute full path for todo_file through PathForModuleOut().
			// Its doclet will get the full path relative to "-o" when compiling with non-std
			// doclet.
			droiddocOptions = append(droiddocOptions, "-todo "+d.properties.Todo_file)
		}
		if d.properties.Ahat_api != "" {
			ahatApi := android.PathForModuleOut(ctx, d.properties.Ahat_api)
			droiddocOptions = append(droiddocOptions, "-api "+ahatApi.String())
			implicitOutputs = append(implicitOutputs, ahatApi)
		}
		if d.properties.Ahat_removed_api != "" {
			ahatRemovedApi := android.PathForModuleOut(ctx, d.properties.Ahat_removed_api)
			droiddocOptions = append(droiddocOptions, "-removedApi "+ahatRemovedApi.String())
		}

		implicits = append(implicits, deps.etc...)

		ctx.Build(pctx, android.BuildParams{
			Rule:            nonStdDroiddoc,
			Description:     "Droiddoc with non-standard doclet",
			Output:          stubsJar,
			ImplicitOutput:  docZip,
			Inputs:          srcFiles,
			Implicits:       implicits,
			ImplicitOutputs: implicitOutputs,
			Args: map[string]string{
				"outDir":            android.PathForModuleOut(ctx, "docs", "out").String(),
				"srcJarDir":         android.PathForModuleOut(ctx, "docs", "srcjars").String(),
				"stubsDir":          android.PathForModuleOut(ctx, "docs", "stubsDir").String(),
				"srcJars":           strings.Join(srcJars.Strings(), " "),
				"customTemplateDir": templateDir,
				"htmlDirArgs":       htmlDirArgs,
				"htmlDir2Args":      htmlDir2Args,
				"bootclasspathArgs": bootClasspathArgs,
				"classpathArgs":     classpathArgs,
				"sourcepath":        strings.Join(sourcepaths, ":"),
				"currentBuild":      "-hdf page.build " + ctx.Config().BuildId() + "-" + ctx.Config().BuildNumberFromFile(),
				"currentTime":       "-hdf page.now " + `"$$(` + ctx.Config().DateFromFile() + ` "+%d %b %Y %k:%M")"`,
				"droiddocOptions":   strings.Join(droiddocOptions, " "),
				"docZip":            docZip.String(),
			},
		})
	}
	d.stubsJar = stubsJar
	d.docZip = docZip
}
