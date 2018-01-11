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
				`${config.JavadocCmd} -encoding UTF-8 @$out.rsp @$srcJarDir/list ` +
				`-source 1.8 -J-Xmx1600m -J-XX:-OmitStackTraceInFastThrow -XDignore.symbol.file ` +
				`-doclet com.google.doclava.Doclava -docletpath ${config.JsilverJar}:${config.DoclavaJar} ` +
				`-templatedir $customTemplateDir $htmlDirArgs $htmlDir2Args ` +
				`$currentBuild $currentTime $args -stubs $stubsDir ` +
				`$bootclasspathArgs $classpathArgs -sourcepath $sourcepath ` +
				`-d $outDir -quiet  && ` +
				`${config.SoongZipCmd} -write_if_changed -d -o $docZip -C $outDir -D $outDir && ` +
				`${config.SoongZipCmd} -write_if_changed -jar -o $out -C $stubsDir -D $stubsDir`,
			CommandDeps: []string{
				"${config.ExtractSrcJarsCmd}",
				"${config.JavadocCmd}",
				"${config.JsilverJar}",
				"${config.DoclavaJar}",
				"${config.SoongZipCmd}",
			},
			Rspfile:        "$out.rsp",
			RspfileContent: "$in",
			Restat:         true,
		},
		"outDir", "srcJarDir", "stubsDir", "srcJars", "customTemplateDir",
		"htmlDirArgs", "htmlDir2Args", "bootclasspathArgs", "classpathArgs", "sourcepath",
		"currentBuild", "currentTime", "args", "docZip")

	stdDroiddoc = pctx.AndroidStaticRule("stdDroiddoc",
		blueprint.RuleParams{
			Command: `rm -rf "$outDir" "$srcJarDir" "$stubsDir" && mkdir -p "$outDir" "$srcJarDir" "$stubsDir" && ` +
				`${config.ExtractSrcJarsCmd} $srcJarDir $srcJarDir/list $srcJars && ` +
				`${config.JavadocCmd} -encoding UTF-8 @$out.rsp @$srcJarDir/list ` +
				`-J-Xmx1024m -XDignore.symbol.file -Xdoclint:none ` +
				`$bootClasspathArgs $classpathArgs -sourcepath $sourcepath ` +
				`-d $outDir -quiet && ` +
				`${config.SoongZipCmd} -write_if_changed -d -o $docZip -C $outDir -D $outDir && ` +
				`${config.SoongZipCmd} -write_if_changed -jar -o $out -C $stubsDir -D $stubsDir`,
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
	android.RegisterModuleType("javadoc", JavadocFactory)
	android.RegisterModuleType("javadoc_host", JavadocHostFactory)
}

type JavadocProperties struct {
	// list of source files used to compile the Java module.  May be .java, .logtags, .proto,
	// or .aidl files.
	Srcs []string `android:"arch_variant"`

	// list of directories rooted at the Android.bp file that will
	// be added to the search paths for finding source files when passing package names.
	Local_sourcepaths []string `android:"arch_variant"`

	// list of source files that should not be used to build the Java module.
	// This is most useful in the arch/multilib variants to remove non-common files
	// filegroup or genrule can be included within this property.
	Exclude_srcs []string `android:"arch_variant"`

	// list of of java libraries that will be in the classpath.
	Libs []string `android:"arch_variant"`

	// If set to false, don't allow this module(-docs.zip) to be exported. Defaults to true.
	Installable *bool `android:"arch_variant"`

	// if not blank, set to the version of the sdk to compile against
	Sdk_version *string `android:"arch_variant"`
}

type DroiddocProperties struct {
	// directory relative to top of the source tree that contains doc templates files.
	Custom_template_dir string `android:"arch_variant"`

	// directory relative to top of the source tree is passed with "-htmldir" when compiling with
	// non-standard doclet.
	Html_dir string `android:"arch_variant"`

	// directory relative to top of the source tree is passed with "-htmldir2" when compiling with
	// non-standard doclet.
	Html_dir2 string `android:"arch_variant"`

	// list of module-specific flags that will be used for droiddoc compiles.
	//Droiddoc_options []string `android:"arch_variant"`

	// a list of strings is passed with "-hdf" when compiling with non-standard doclet.
	Hdf []string `android:"arch_variant"`

	// filename is passed with "-proofread" when compiling with non-standard doclet.
	// the file will be generated within out dir.
	Proofread_file string `android:"arch_variant"`

	// filename is passed with "-todo" when compiling with non-standard doclet.
	// the file will be generated relative to "-o" dir when compiling with non-standard doclet.
	Todo_file string `android:"arch_variant"`

	// local files that are used within user customized droiddoc options.
	Arg_files []string `android:"arch_variant"`

	// user customized droiddoc args.
	// Available variables for substitution:
	//
	//  $(location <label>): the path to the arg_files with name <label>
	Args string `android:"arch_variant"`

	// names of the output files used in args that will be generated
	Out []string `android:"arch_variant"`

	// a list of files relative to current module source directory, and is passed with
	// "-knowntags" when compiling with non-standard doclet.
	// filegroup or genrule can be included within this property.
	Knowntags []string `android:"arch_variant"`
}

type Doc struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties JavadocProperties

	srcJars     android.Paths
	srcFiles    android.Paths
	sourcepaths android.Paths

	docZip   android.WritablePath
	stubsJar android.WritablePath
}

type Javadoc struct {
	Doc
}

type Droiddoc struct {
	Doc

	properties DroiddocProperties
}

func InitDroiddocModule(module android.DefaultableModule, hod android.HostOrDeviceSupported) {
	android.InitAndroidArchModule(module, hod, android.MultilibCommon)
	android.InitDefaultableModule(module)
}

func JavadocFactory() android.Module {
	module := &Javadoc{}

	module.AddProperties(&module.properties)

	InitDroiddocModule(module, android.HostAndDeviceSupported)
	return module
}

func JavadocHostFactory() android.Module {
	module := &Javadoc{}

	module.AddProperties(&module.properties)

	InitDroiddocModule(module, android.HostSupported)
	return module
}

func DroiddocFactory() android.Module {
	module := &Droiddoc{}

	module.AddProperties(&module.properties,
		&module.Doc.properties)

	InitDroiddocModule(module, android.HostAndDeviceSupported)
	return module
}

func DroiddocHostFactory() android.Module {
	module := &Droiddoc{}

	module.AddProperties(&module.properties,
		&module.Doc.properties)

	InitDroiddocModule(module, android.HostSupported)
	return module
}

func (d *Doc) addDeps(ctx android.BottomUpMutatorContext) {
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
}

func (d *Doc) collectDeps(ctx android.ModuleContext) deps {
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
	srcFiles := ctx.ExpandSources(d.properties.Srcs, d.properties.Exclude_srcs)

	// srcs may depend on some genrule output.
	d.srcJars = srcFiles.FilterByExt(".srcjar")
	d.srcFiles = srcFiles.FilterOutByExt(".srcjar")

	d.docZip = android.PathForModuleOut(ctx, ctx.ModuleName()+"-"+"docs.zip")
	d.stubsJar = android.PathForModuleOut(ctx, ctx.ModuleName()+"-"+"stubs.jar")

	if d.properties.Local_sourcepaths == nil {
		d.properties.Local_sourcepaths = append(d.properties.Local_sourcepaths, ".")
	}
	d.sourcepaths = android.PathsForModuleSrc(ctx, d.properties.Local_sourcepaths)
	d.sourcepaths = append(d.sourcepaths, deps.bootClasspath...)
	d.sourcepaths = append(d.sourcepaths, deps.classpath...)

	return deps
}

func (j *Javadoc) DepsMutator(ctx android.BottomUpMutatorContext) {
	j.Doc.addDeps(ctx)
}

func (j *Javadoc) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	deps := j.collectDeps(ctx)

	var implicits android.Paths
	implicits = append(implicits, deps.bootClasspath...)
	implicits = append(implicits, deps.classpath...)

	var bootClasspathArgs, classpathArgs string
	if ctx.Config().UseOpenJDK9() {
		if len(deps.bootClasspath) > 0 {
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

	implicits = append(implicits, j.Doc.srcJars...)

	ctx.Build(pctx, android.BuildParams{
		Rule:           stdDroiddoc,
		Description:    "Droiddoc with standard doclet",
		Output:         j.Doc.stubsJar,
		ImplicitOutput: j.Doc.docZip,
		Inputs:         j.Doc.srcFiles,
		Implicits:      implicits,
		Args: map[string]string{
			"outDir":            android.PathForModuleOut(ctx, "docs", "out").String(),
			"srcJarDir":         android.PathForModuleOut(ctx, "docs", "srcjars").String(),
			"stubsDir":          android.PathForModuleOut(ctx, "docs", "stubsDir").String(),
			"bootClasspathArgs": bootClasspathArgs,
			"classpathArgs":     classpathArgs,
			"sourcepath":        strings.Join(j.Doc.sourcepaths.Strings(), ":"),
			"docZip":            j.Doc.docZip.String(),
		},
	})
}

func (d *Droiddoc) DepsMutator(ctx android.BottomUpMutatorContext) {
	d.Doc.addDeps(ctx)

	// extra_arg_files may contains filegroup or genrule.
	android.ExtractSourcesDeps(ctx, d.properties.Arg_files)

	// knowntags may contain filegroup or genrule.
	android.ExtractSourcesDeps(ctx, d.properties.Knowntags)
}

func (d *Droiddoc) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	deps := d.Doc.collectDeps(ctx)

	var implicits android.Paths
	implicits = append(implicits, deps.bootClasspath...)
	implicits = append(implicits, deps.classpath...)

	argFiles := ctx.ExpandSources(d.properties.Arg_files, nil)
	argFilesMap := map[string]android.Path{}

	for _, f := range argFiles {
		implicits = append(implicits, f)
		if _, exists := argFilesMap[f.Rel()]; !exists {
			argFilesMap[f.Rel()] = f
		} else {
			ctx.ModuleErrorf("multiple arg_files for %q, %q and %q",
				f, argFilesMap[f.Rel()], f.Rel())
		}
	}

	args, err := android.Expand(d.properties.Args, func(name string) (string, error) {
		if strings.HasPrefix(name, "location ") {
			label := strings.TrimSpace(strings.TrimPrefix(name, "location "))
			if f, ok := argFilesMap[label]; ok {
				return f.String(), nil
			} else {
				return "", fmt.Errorf("unknown location label %q", label)
			}
		} else if name == "genDir" {
			return android.PathForModuleGen(ctx).String(), nil
		}
		return "", fmt.Errorf("unknown variable '$(%s)'", name)
	})

	if err != nil {
		ctx.PropertyErrorf("extra_args", "%s", err.Error())
		return
	}

	var bootClasspathArgs, classpathArgs string
	if len(deps.bootClasspath.Strings()) > 0 {
		bootClasspathArgs = "-bootclasspath " + strings.Join(deps.bootClasspath.Strings(), ":")
	}
	if len(deps.classpath.Strings()) > 0 {
		classpathArgs = "-classpath " + strings.Join(deps.classpath.Strings(), ":")
	}

	// templateDir is relative to top of the source tree instead of current module.
	templateDir := android.PathForSource(ctx, d.properties.Custom_template_dir).String()
	implicits = append(implicits, globDir(ctx, filepath.Join(templateDir, "**/*"), nil)...)

	// htmlDir is relative to top of the source tree instead of current module.
	var htmlDirArgs string
	if d.properties.Html_dir != "" {
		htmlDir := android.PathForSource(ctx, d.properties.Html_dir).String()
		implicits = append(implicits, globDir(ctx, filepath.Join(htmlDir, "**/*"), nil)...)
		htmlDirArgs = "-htmldir " + htmlDir
	}

	// htmlDir2 is relative to top of the source tree instead of current module.
	var htmlDir2Args string
	if d.properties.Html_dir2 != "" {
		htmlDir2 := android.PathForSource(ctx, d.properties.Html_dir2).String()
		implicits = append(implicits, globDir(ctx, filepath.Join(htmlDir2, "**/*"), nil)...)
		htmlDirArgs = "-htmldir2 " + htmlDir2
	}

	var implicitOutputs android.WritablePaths

	knownTags := ctx.ExpandSources(d.properties.Knowntags, nil)
	implicits = append(implicits, knownTags...)

	for _, kt := range knownTags {
		args = args + " -knowntags " + kt.String()
	}
	for _, hdf := range d.properties.Hdf {
		args = args + " -hdf " + hdf
	}

	if d.properties.Proofread_file != "" {
		proofreadFile := android.PathForModuleOut(ctx, d.properties.Proofread_file)
		args = args + " -proofread " + proofreadFile.String()
	}
	if d.properties.Todo_file != "" {
		// tricky part:
		// we should not compute full path for todo_file through PathForModuleOut().
		// the non-standard doclet will get the full path relative to "-o".
		args = args + " -todo " + d.properties.Todo_file
	}

	implicits = append(implicits, d.Doc.srcJars...)
	ctx.Build(pctx, android.BuildParams{
		Rule:            nonStdDroiddoc,
		Description:     "Droiddoc with non-standard doclet",
		Output:          d.Doc.stubsJar,
		ImplicitOutput:  d.Doc.docZip,
		Inputs:          d.Doc.srcFiles,
		Implicits:       implicits,
		ImplicitOutputs: implicitOutputs,
		Args: map[string]string{
			"outDir":            android.PathForModuleOut(ctx, "docs", "out").String(),
			"srcJarDir":         android.PathForModuleOut(ctx, "docs", "srcjars").String(),
			"stubsDir":          android.PathForModuleOut(ctx, "docs", "stubsDir").String(),
			"srcJars":           strings.Join(d.Doc.srcJars.Strings(), " "),
			"customTemplateDir": templateDir,
			"htmlDirArgs":       htmlDirArgs,
			"htmlDir2Args":      htmlDir2Args,
			"bootclasspathArgs": bootClasspathArgs,
			"classpathArgs":     classpathArgs,
			"sourcepath":        strings.Join(d.Doc.sourcepaths.Strings(), ":"),
			"currentBuild":      "-hdf page.build " + ctx.Config().BuildId() + "-" + ctx.Config().BuildNumberFromFile(),
			"currentTime":       "-hdf page.now " + `"$$(` + ctx.Config().DateFromFile() + ` "+%d %b %Y %k:%M")"`,
			"args":              args,
			"docZip":            d.Doc.docZip.String(),
		},
	})
}

// glob the directory relative to top of the source tree.
func globDir(ctx android.ModuleContext, pattern string, excludes []string) []android.Path {
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
