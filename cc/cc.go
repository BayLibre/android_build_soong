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

// This file contains the module types for compiling C/C++ for Android, and converts the properties
// into the flags and filenames necessary to pass to the compiler.  The final creation of the rules
// is handled in builder.go

import (
	"blueprint"
	"blueprint/pathtools"
	"fmt"
	"path/filepath"
	"strings"

	"android/soong/common"
)

type Config interface {
	SrcDir() string
	PrebuiltOS() string
}

var (
	HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", Config.PrebuiltOS)
	SrcDir          = pctx.VariableConfigMethod("SrcDir", Config.SrcDir)

	LibcRoot = pctx.StaticVariable("LibcRoot", "${SrcDir}/bionic/libc")
	LibmRoot = pctx.StaticVariable("LibmRoot", "${SrcDir}/bionic/libm")
)

// Flags used by lots of devices.  Putting them in package static variables will save bytes in
// build.ninja so they aren't repeated for every file
var (
	commonGlobalCflags = []string{
		"-DANDROID",
		"-fmessage-length=0",
		"-W",
		"-Wall",
		"-Wno-unused",
		"-Winit-self",
		"-Wpointer-arith",

		// COMMON_RELEASE_CFLAGS
		"-DNDEBUG",
		"-UDEBUG",
	}

	deviceGlobalCflags = []string{
		// TARGET_ERROR_FLAGS
		"-Werror=return-type",
		"-Werror=non-virtual-dtor",
		"-Werror=address",
		"-Werror=sequence-point",
	}

	hostGlobalCflags = []string{}

	commonGlobalCppflags = []string{
		"-Wsign-promo",
		"-std=gnu++11",
	}
)

func init() {
	pctx.StaticVariable("commonGlobalCflags", strings.Join(commonGlobalCflags, " "))
	pctx.StaticVariable("deviceGlobalCflags", strings.Join(deviceGlobalCflags, " "))
	pctx.StaticVariable("hostGlobalCflags", strings.Join(hostGlobalCflags, " "))

	pctx.StaticVariable("commonGlobalCppflags", strings.Join(commonGlobalCppflags, " "))

	pctx.StaticVariable("commonClangGlobalCflags",
		strings.Join(clangFilterUnknownCflags(commonGlobalCflags), " "))
	pctx.StaticVariable("deviceClangGlobalCflags",
		strings.Join(clangFilterUnknownCflags(deviceGlobalCflags), " "))
	pctx.StaticVariable("hostClangGlobalCflags",
		strings.Join(clangFilterUnknownCflags(hostGlobalCflags), " "))
        pctx.StaticVariable("commonClangGlobalCppflags",
                strings.Join(clangFilterUnknownCflags(commonGlobalCppflags), " "))

	// Everything in this list is a crime against abstraction and dependency tracking.
	// Do not add anything to this list.
	pctx.StaticVariable("commonGlobalIncludes", strings.Join([]string{
		"-isystem ${SrcDir}/system/core/include",
		"-isystem ${SrcDir}/hardware/libhardware/include",
		"-isystem ${SrcDir}/hardware/libhardware_legacy/include",
		"-isystem ${SrcDir}/hardware/ril/include",
		"-isystem ${SrcDir}/libnativehelper/include",
		"-isystem ${SrcDir}/frameworks/native/include",
		"-isystem ${SrcDir}/frameworks/native/opengl/include",
		"-isystem ${SrcDir}/frameworks/av/include",
		"-isystem ${SrcDir}/frameworks/base/include",
	}, " "))

	pctx.StaticVariable("clangPath", "${SrcDir}/prebuilts/clang/${HostPrebuiltTag}/host/3.6/bin/")
}

// CcProperties describes properties used to compile all C or C++ modules
type ccProperties struct {
	// srcs: list of source files used to compile the C/C++ module.  May be .c, .cpp, or .S files.
	Srcs []string `android:"arch_variant,arch_subtract"`

	// cflags: list of module-specific flags that will be used for C and C++ compiles.
	Cflags []string `android:"arch_variant"`

	// cppflags: list of module-specific flags that will be used for C++ compiles
	Cppflags []string `android:"arch_variant"`

	// conlyflags: list of module-specific flags that will be used for C compiles
	Conlyflags []string `android:"arch_variant"`

	// asflags: list of module-specific flags that will be used for .S compiles
	Asflags []string `android:"arch_variant"`

	// ldflags: list of module-specific flags that will be used for all link steps
	Ldflags []string `android:"arch_variant"`

	// include_dirs: list of directories relative to the root of the source tree that will
	// be added to the include path using -I.
	// If possible, don't use this.  If adding paths from the current directory use
	// local_include_dirs, if adding paths from other modules use export_include_dirs in
	// that module.
	Include_dirs []string `android:"arch_variant"`

	// local_include_dirs: list of directories relative to the Blueprints file that will
	// be added to the include path using -I
	Local_include_dirs []string `android:"arch_variant"`

	// clang_cflags: list of module-specific flags that will be used for C and C++ compiles when
	// compiling with clang
	Clang_cflags []string `android:"arch_variant"`

	// clang_asflags: list of module-specific flags that will be used for .S compiles when
	// compiling with clang
	Clang_asflags []string `android:"arch_variant"`

	// system_shared_libs: list of system libraries that will be dynamically linked to
	// shared library and executable modules.  If unset, generally defaults to libc
	// and libm.  Set to [] to prevent linking against libc and libm.
	System_shared_libs []string

	// whole_static_libs: list of modules whose object files should be linked into this module
	// in their entirety.  For static library modules, all of the .o files from the intermediate
	// directory of the dependency will be linked into this modules .a file.  For a shared library,
	// the dependency's .a file will be linked into this module using -Wl,--whole-archive.
	Whole_static_libs []string `android:"arch_variant"`

	// static_libs: list of modules that should be statically linked into this module.
	Static_libs []string `android:"arch_variant"`

	// shared_libs: list of modules that should be dynamically linked into this module.
	Shared_libs []string `android:"arch_variant"`

	// allow_undefined_symbols: allow the module to contain undefined symbols.  By default,
	// modules cannot contain undefined symbols that are not satisified by their immediate
	// dependencies.  Set this flag to true to remove --no-undefined from the linker flags.
	// This flag should only be necessary for compiling low-level libraries like libc.
	Allow_undefined_symbols bool

	// nocrt: don't link in crt_begin and crt_end.  This flag should only be necessary for
	// compiling crt or libc.
	Nocrt bool `android:"arch_variant"`

	// no_default_compiler_flags: don't insert default compiler flags into asflags, cflags,
	// cppflags, conlyflags, ldflags, or include_dirs
	No_default_compiler_flags bool

	// clang: compile module with clang instead of gcc
	Clang bool `android:"arch_variant"`

	// rtti: pass -frtti instead of -fno-rtti
	Rtti bool

	// host_ldlibs: -l arguments to pass to linker for host-provided shared libraries
	Host_ldlibs []string `android:"arch_variant"`

	// stl: select the STL library to use.  Possible values are "libc++", "libc++_static",
	// "stlport", "stlport_static", "ndk", "libstdc++", or "none".  Leave blank to select the
	// default
	Stl string
}

type unusedProperties struct {
	Asan                bool
	Native_coverage     bool
	Strip               string
	Tags                []string
	Required            []string
	Export_include_dirs []string
}

type Producer interface {
	HostOrDevice() common.HostOrDevice
}

type StaticLibraryProducer interface {
	Producer
	StaticLibraryOutputFile() string
	StaticLibraryObjFiles() []string
}

type SharedLibraryProducer interface {
	Producer
	SharedLibraryOutputFile() string
}

type ccDepender interface {
	sharedDependencyOn(name string) bool
	staticDependencyOn(name string) bool
	wholeStaticDependencyOn(name string) bool
}

// ccBase contains the properties and members used by all C/C++ module types
type ccBase struct {
	common.AndroidModuleBase
	properties ccProperties
	unused     unusedProperties

	installPath string
	flags       compileFlags
	toolchain   toolchain
}

var _ common.AndroidDynamicDepender = (*ccBase)(nil)

func (c *ccBase) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	if c.DeviceSupported() {
		ctx.AddNamespacedDependencies(common.DeviceStaticLibrary,
			ctx.PropertyForAllArches("Whole_static_libs")...)
		ctx.AddNamespacedDependencies(common.DeviceStaticLibrary,
			ctx.PropertyForAllArches("Static_libs")...)
		ctx.AddNamespacedDependencies(common.DeviceSharedLibrary,
			ctx.PropertyForAllArches("Shared_libs")...)
	}

	if c.HostSupported() {
		ctx.AddNamespacedDependencies(common.HostStaticLibrary,
			ctx.PropertyForAllArches("Whole_static_libs")...)
		ctx.AddNamespacedDependencies(common.HostStaticLibrary,
			ctx.PropertyForAllArches("Static_libs")...)
		ctx.AddNamespacedDependencies(common.HostSharedLibrary,
			ctx.PropertyForAllArches("Shared_libs")...)
	}
	return nil
}

func (c *ccBase) setToolchain(ctx common.AndroidModuleContext) {
	arch := ctx.Arch()
	factory := toolchainFactories[arch.HostOrDevice][arch.ArchType]
	if factory == nil {
		panic(fmt.Sprintf("Toolchain not found for %s arch %q",
			arch.HostOrDevice.String(), arch.String()))
	}
	c.toolchain = factory(arch.ArchVariant, arch.CpuVariant)
}

func (c *ccBase) setCompilerFlags(ctx common.AndroidModuleContext, cflags []string, ldflags []string) {
	arch := ctx.Arch()

	flags := compileFlags{
		cFlags:     c.properties.Cflags,
		cppFlags:   c.properties.Cppflags,
		conlyFlags: c.properties.Conlyflags,
		ldFlags:    c.properties.Ldflags,
		asFlags:    c.properties.Asflags,
		nocrt:      c.properties.Nocrt,
		toolchain:  c.toolchain,
		clang:      c.properties.Clang,
	}

	if flags.clang {
		flags.cFlags = clangFilterUnknownCflags(flags.cFlags)
		flags.cFlags = append(flags.cFlags, c.properties.Clang_cflags...)
		flags.asFlags = append(flags.asFlags, c.properties.Clang_asflags...)
		flags.cppFlags = clangFilterUnknownCflags(flags.cppFlags)
		flags.conlyFlags = clangFilterUnknownCflags(flags.conlyFlags)
		flags.ldFlags = clangFilterUnknownCflags(flags.ldFlags)

		flags.cFlags = append(flags.cFlags, "-target "+c.toolchain.ClangTriple())
		flags.asFlags = append(flags.asFlags, "-target "+c.toolchain.ClangTriple())

		if arch.HostOrDevice.Device() {
			flags.cFlags = append(flags.cFlags, "-B"+c.toolchain.ToolchainRoot())
			flags.asFlags = append(flags.asFlags, "-B"+c.toolchain.ToolchainRoot())
		}
	}

	includeDirs := pathtools.PrefixPaths(c.properties.Include_dirs, ctx.Config().(Config).SrcDir())
	localIncludeDirs := pathtools.PrefixPaths(c.properties.Local_include_dirs, common.ModuleSrcDir(ctx))
	includeDirs = append(includeDirs, localIncludeDirs...)

	if !c.properties.No_default_compiler_flags {
		includeDirs = append(includeDirs, []string{
			common.ModuleSrcDir(ctx),
			common.ModuleOutDir(ctx),
			common.ModuleGenDir(ctx),
		}...)

		if arch.HostOrDevice.Device() && !c.properties.Allow_undefined_symbols {
			flags.ldFlags = append(flags.ldFlags, "-Wl,--no-undefined")
		}

		if flags.clang {
			flags.globalFlags = []string{
				"${commonGlobalIncludes}",
				c.toolchain.IncludeFlags(),
				c.toolchain.ClangCflags(),
				"${commonClangGlobalCflags}",
				fmt.Sprintf("${%sClangGlobalCflags}", arch.HostOrDevice),
			}
		} else {
			flags.globalFlags = []string{
				"${commonGlobalIncludes}",
				c.toolchain.IncludeFlags(),
				c.toolchain.Cflags(),
				"${commonGlobalCflags}",
				fmt.Sprintf("${%sGlobalCflags}", arch.HostOrDevice),
			}
		}

		if arch.HostOrDevice.Host() {
			flags.ldFlags = append(flags.ldFlags, c.properties.Host_ldlibs...)
		}

		if arch.HostOrDevice.Device() {
			if c.properties.Rtti {
				flags.cppFlags = append(flags.cppFlags, "-frtti")
			} else {
				flags.cppFlags = append(flags.cppFlags, "-fno-rtti")
			}
		}

		flags.asFlags = append(flags.asFlags, "-D__ASSEMBLY__")

		if flags.clang {
                        flags.cppFlags = append(flags.cppFlags, "${commonClangGlobalCppflags}")
			flags.cppFlags = append(flags.cppFlags, c.toolchain.ClangCppflags())
			flags.ldFlags = append(flags.ldFlags, c.toolchain.ClangLdflags())
		} else {
                        flags.cppFlags = append(flags.cppFlags, "${commonGlobalCppflags}")
			flags.cppFlags = append(flags.cppFlags, c.toolchain.Cppflags())
			flags.ldFlags = append(flags.ldFlags, c.toolchain.Ldflags())
		}

		stl := "libc++" // TODO: mingw needs libstdc++
		if c.properties.Stl != "" {
			stl = c.properties.Stl
		}

		stlStatic := false
		if strings.HasSuffix(stl, "_static") {
			stlStatic = true
		}

		switch stl {
		case "libc++", "libc++_static":
			flags.cFlags = append(flags.cFlags, "-D_USING_LIBCXX")
			includeDirs = append(includeDirs, "${SrcDir}/external/libcxx/include")
			if arch.HostOrDevice.Host() {
				flags.cppFlags = append(flags.cppFlags, "-nostdinc++")
				flags.ldFlags = append(flags.ldFlags, "-nodefaultlibs")
				flags.ldLibs = append(flags.ldLibs, "-lc", "-lm", "-lpthread")
			}
			if stlStatic {
				flags.extraStaticLibs = append(flags.extraStaticLibs, "libc++_static")
			} else {
				flags.extraSharedLibs = append(flags.extraSharedLibs, "libc++")
			}
		case "stlport", "stlport_static":
			if arch.HostOrDevice.Device() {
				includeDirs = append(includeDirs,
					"${SrcDir}/external/stlport/stlport",
					"${SrcDir}/bionic/libstdc++/include",
					"${SrcDir}/bionic",
				)
				if stlStatic {
					flags.extraStaticLibs = append(flags.extraStaticLibs, "libstdc++", "libstlport_static")
				} else {
					flags.extraSharedLibs = append(flags.extraSharedLibs, "libstdc++", "libstlport")
				}
			}
		case "ndk":
			panic("TODO")
		case "libstdc++":
			// Using bionic's basic libstdc++. Not actually an STL. Only around until the
			// tree is in good enough shape to not need it.
			// Host builds will use GNU libstdc++.
			if arch.HostOrDevice.Device() {
				includeDirs = append(includeDirs, "${SrcDir}/bionic/libstdc++/include")
				flags.extraSharedLibs = append(flags.extraSharedLibs, "libstdc++")
			}
		case "none":
			if arch.HostOrDevice.Host() {
				flags.cppFlags = append(flags.cppFlags, "-nostdinc++")
				flags.ldFlags = append(flags.ldFlags, "-nodefaultlibs")
				flags.ldLibs = append(flags.ldLibs, "-lc", "-lm")
			}
		default:
			ctx.ModuleErrorf("stl: %q is not a supported STL", stl)
		}
	}

	flags.cFlags = append(flags.cFlags, cflags...)
	flags.ldFlags = append(flags.ldFlags, ldflags...)

	flags.incFlags = includeDirsToFlags(includeDirs)

	// Optimization to reduce size of build.ninja
	// Replace the long list of flags for each file with a module-local variable
	ctx.Variable(pctx, "cflags", strings.Join(flags.cFlags, " "))
	ctx.Variable(pctx, "cppflags", strings.Join(flags.cppFlags, " "))
	ctx.Variable(pctx, "asflags", strings.Join(flags.asFlags, " "))
	ctx.Variable(pctx, "incflags", flags.incFlags)
	flags.cFlags = []string{"$cflags"}
	flags.cppFlags = []string{"$cppflags"}
	flags.asFlags = []string{"$asflags"}
	flags.incFlags = "$incflags"

	c.flags = flags
}

func (c *ccBase) generateCustomObjBuildActions(ctx common.AndroidModuleContext, subdir string,
	srcFiles []string, cflags []string) []string {
	srcFiles = pathtools.PrefixPaths(srcFiles, common.ModuleSrcDir(ctx))
	srcFiles = common.ExpandGlobs(ctx, srcFiles)

	flags := c.flags
	flags.cFlags = append(flags.cFlags, cflags...)

	return TransformSourceToObj(ctx, subdir, srcFiles, flags)
}

func (c *ccBase) generateObjBuildActions(ctx common.AndroidModuleContext) []string {
	srcFiles := pathtools.PrefixPaths(c.properties.Srcs, common.ModuleSrcDir(ctx))
	srcFiles = common.ExpandGlobs(ctx, srcFiles)

	return TransformSourceToObj(ctx, "", srcFiles, c.flags)
}

// ccDynamic contains the properties and members used by shared libraries and dynamic executables
type ccDynamic struct {
	*ccBase
}

const defaultSystemSharedLibraries = "__default__"

func (c *ccDynamic) possibleSystemSharedLibs(properties ccProperties) []string {
	if len(properties.System_shared_libs) == 1 &&
		properties.System_shared_libs[0] == defaultSystemSharedLibraries {

		if !c.DeviceSupported() {
			return []string{}
		} else {
			return []string{"libc", "libm"}
		}
	}
	return properties.System_shared_libs
}

func (c *ccDynamic) systemSharedLibs(properties ccProperties) []string {
	if len(properties.System_shared_libs) == 1 &&
		properties.System_shared_libs[0] == defaultSystemSharedLibraries {

		if c.HostOrDevice().Host() {
			return []string{}
		} else {
			return []string{"libc", "libm"}
		}
	}
	return properties.System_shared_libs
}

var (
	stlSharedLibs     = []string{"libc++", "libstlport", "libstdc++"}
	stlSharedHostLibs = []string{"libc++"}
	stlStaticLibs     = []string{"libc++_static", "libstlport_static", "libstdc++"}
	stlStaticHostLibs = []string{"libc++_static"}
)

func (c *ccDynamic) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	deps := c.ccBase.AndroidDynamicDependencies(ctx)
	if c.DeviceSupported() {
		ctx.AddNamespacedDependencies(common.DeviceSharedLibrary,
			c.possibleSystemSharedLibs(c.properties)...)
		ctx.AddNamespacedDependencies(common.DeviceStaticLibrary,
			"libcompiler_rt-extras",
			//"libgcov",
			"libatomic",
			"libgcc")
	}

	if c.HostSupported() {
		ctx.AddNamespacedDependencies(common.HostSharedLibrary,
			c.possibleSystemSharedLibs(c.properties)...)
	}

	if c.properties.Stl != "none" {
		if c.DeviceSupported() {
			ctx.AddNamespacedDependencies(common.DeviceSharedLibrary, stlSharedLibs...)
			ctx.AddNamespacedDependencies(common.DeviceStaticLibrary, stlStaticLibs...)
		}

		if c.HostSupported() {
			ctx.AddNamespacedDependencies(common.HostSharedLibrary, stlSharedHostLibs...)
			ctx.AddNamespacedDependencies(common.HostStaticLibrary, stlStaticHostLibs...)

		}
	}
	return deps
}

var _ common.AndroidDynamicDepender = (*ccDynamic)(nil)

var implicitStaticLibs = []string{"libcompiler_rt-extras", "libgcov", "libatomic", "libgcc"}

func (c *ccBase) isWholeStaticLibraryDependency(name string) bool {
	return inList(name, c.properties.Whole_static_libs)
}

func (c *ccBase) isStaticLibraryDependency(name string) bool {
	return inList(name, c.properties.Static_libs) || inList(name, implicitStaticLibs) ||
		inList(name, c.flags.extraStaticLibs)
}

func (c *ccDynamic) isSharedLibraryDependency(name string) bool {
	return inList(name, c.properties.Shared_libs) ||
		inList(name, c.systemSharedLibs(c.properties)) ||
		inList(name, c.flags.extraSharedLibs)
}

func (c *ccDynamic) collectDeps(ctx common.AndroidModuleContext) (staticLibs, sharedLibs,
	wholeStaticLibs []string, crtBegin, crtEnd string) {

	ctx.VisitDirectDeps(func(m blueprint.Module) {
		otherName := ctx.OtherModuleName(m)
		if producer, ok := m.(Producer); ok {
			if c.HostOrDevice() != producer.HostOrDevice() {
				ctx.ModuleErrorf("host/device mismatch between %q and %q", ctx.ModuleName(), otherName)
				return
			}
		}

		if c.isWholeStaticLibraryDependency(otherName) {
			if staticLib, ok := m.(StaticLibraryProducer); ok {
				wholeStaticLibs = append(wholeStaticLibs, staticLib.StaticLibraryOutputFile())
			} else {
				ctx.ModuleErrorf("module %q not a static library", otherName)
			}
		} else if c.isStaticLibraryDependency(otherName) {
			if staticLib, ok := m.(StaticLibraryProducer); ok {
				staticLibs = append(staticLibs, staticLib.StaticLibraryOutputFile())
			} else {
				ctx.ModuleErrorf("module %q not a static library", otherName)
			}
		} else if c.isSharedLibraryDependency(otherName) {
			if sharedLib, ok := m.(SharedLibraryProducer); ok {
				sharedLibs = append(sharedLibs, sharedLib.SharedLibraryOutputFile())
			} else {
				ctx.ModuleErrorf("module %q not a shared library", otherName)
			}
		} else if obj, ok := m.(*ccObject); ok {
			if strings.HasPrefix(otherName, "crtbegin") {
				if !c.properties.Nocrt {
					crtBegin = obj.outputFile
				}
			} else if strings.HasPrefix(otherName, "crtend") {
				if !c.properties.Nocrt {
					crtEnd = obj.outputFile
				}
			} else {
				ctx.ModuleErrorf("object module type only support for crtbegin and crtend, found %q",
					ctx.OtherModuleName(m))
			}
		} else {
			// This may happen if another variant depends on m, but this variant doesn't.  m has to
			// end up in the deps list, but will get ignored by the variants that don't use it.
			// TODO: add debug message?
			//fmt.Printf("unrecognized dependency type for %q on %q\n",
			//	ctx.ModuleName(), ctx.OtherModuleName(m))
			//ctx.ModuleErrorf("unrecognized dependency type for %q on %q",
			//	ctx.ModuleName(), ctx.OtherModuleName(m))
		}
	})

	return
}

//
// Static libraries
//

var _ StaticLibraryProducer = (*ccLibraryStatic)(nil)

type ccLibraryStatic struct {
	*ccBase

	outputFile string
	libName    string
	objFiles   []string
}

func NewCCLibraryStatic() (blueprint.Module, []interface{}) {
	module := &ccLibraryStatic{
		ccBase: &ccBase{},
	}
	return common.InitAndroidModule(module, common.HostAndDeviceSupported, "both",
		&module.properties, &module.unused)
}

func (c *ccLibraryStatic) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	c.setToolchain(ctx)

	cflags := []string{"-fPIC"}
	c.setCompilerFlags(ctx, cflags, nil)

	objFiles := c.generateObjBuildActions(ctx)

	c.generateLibraryBuildActions(ctx, objFiles)

}
func (c *ccLibraryStatic) generateLibraryBuildActions(ctx common.AndroidModuleContext, objFiles []string) {
	ctx.VisitDirectDeps(func(m blueprint.Module) {
		otherName := ctx.OtherModuleName(m)
		if producer, ok := m.(Producer); ok {
			if c.HostOrDevice() != producer.HostOrDevice() {
				ctx.ModuleErrorf("host/device mismatch between %q and %q", ctx.ModuleName(), otherName)
				return
			}
		}

		if c.isWholeStaticLibraryDependency(otherName) {
			if staticLib, ok := m.(StaticLibraryProducer); ok {
				objFiles = append(objFiles, staticLib.StaticLibraryObjFiles()...)
			} else {
				ctx.ModuleErrorf("module %q not a static library", otherName)
			}
		} else if obj, ok := m.(*ccObject); ok {
			if !strings.HasPrefix(otherName, "crtbegin") && !strings.HasPrefix(otherName, "crtend") {
				objFiles = append(objFiles, obj.outputFile)
			}
		}
	})

	outputFile := filepath.Join(common.ModuleOutDir(ctx), ctx.ModuleName()+staticLibraryExtension)

	TransformObjToStaticLib(ctx, objFiles, c.flags, outputFile)

	c.objFiles = objFiles
	c.outputFile = outputFile

	ctx.CheckbuildFile(outputFile)
}

func (c *ccLibraryStatic) StaticLibraryOutputFile() string {
	return c.outputFile
}

func (c *ccLibraryStatic) StaticLibraryObjFiles() []string {
	return c.objFiles
}

//
// Shared libraries
//

var _ SharedLibraryProducer = (*ccLibraryShared)(nil)

type ccLibraryShared struct {
	ccDynamic

	outputFile string
	libName    string
}

func NewCCLibraryShared() (blueprint.Module, []interface{}) {
	module := &ccLibraryShared{
		ccDynamic: ccDynamic{
			ccBase: &ccBase{},
		},
	}
	module.properties.System_shared_libs = []string{defaultSystemSharedLibraries}
	return common.InitAndroidModule(module, common.HostAndDeviceSupported, "both",
		&module.properties, &module.unused)
}

func (c *ccLibraryShared) ldflags(ctx common.AndroidModuleContext) []string {
	libName := ctx.ModuleName()
	// GCC for Android assumes that -shared means -Bsymbolic, use -Wl,-shared instead
	sharedFlag := "-Wl,-shared"
	if c.properties.Clang {
		sharedFlag = "-shared"
	}
	if c.HostOrDevice().Device() {
		return []string{
			"-nostdlib",
			"-Wl,--gc-sections",
			sharedFlag,
			"-Wl,-soname," + libName,
		}
	} else {
		return []string{
			"-Wl,--gc-sections",
			sharedFlag,
			"-Wl,-soname," + libName,
		}
	}
}

func (c *ccLibraryShared) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	c.setToolchain(ctx)

	cflags := []string{"-fPIC"}
	ldflags := c.ldflags(ctx)
	c.setCompilerFlags(ctx, cflags, ldflags)

	objFiles := c.generateObjBuildActions(ctx)

	c.generateLibraryBuildActions(ctx, objFiles)
}

func (c *ccLibraryShared) generateLibraryBuildActions(ctx common.AndroidModuleContext, objFiles []string) {
	staticLibs, sharedLibs, wholeStaticLibs, crtBegin, crtEnd := c.collectDeps(ctx)
	if ctx.Failed() {
		return
	}

	outputFile := filepath.Join(common.ModuleOutDir(ctx), ctx.ModuleName()+sharedLibraryExtension)

	TransformObjToDynamicBinary(ctx, objFiles, sharedLibs, staticLibs, wholeStaticLibs,
		crtBegin, crtEnd, c.flags, outputFile)

	c.outputFile = outputFile

	installDir := "lib"
	if c.flags.toolchain.Is64Bit() {
		installDir = "lib64"
	}

	ctx.InstallFile(installDir, outputFile)
}

func (c *ccLibraryShared) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	deps := c.ccDynamic.AndroidDynamicDependencies(ctx)
	if c.DeviceSupported() {
		deps = append(deps, "crtbegin_so", "crtend_so")
	}
	return deps
}

func (c *ccLibraryShared) SharedLibraryOutputFile() string {
	return c.outputFile
}

//
// Combined static+shared libraries
//

type ccLibrary struct {
	ccLibraryShared
	ccLibraryStatic

	variantProperties struct {
		Static struct {
			Srcs   []string `android:"arch_variant"`
			Cflags []string `android:"arch_variant"`
		} `android:"arch_variant"`
		Shared struct {
			Srcs   []string `android:"arch_variant"`
			Cflags []string `android:"arch_variant"`
		} `android:"arch_variant"`
	}
}

func NewCCLibrary() (blueprint.Module, []interface{}) {
	ccBase := &ccBase{}
	module := &ccLibrary{
		ccLibraryShared: ccLibraryShared{
			ccDynamic: ccDynamic{
				ccBase: ccBase,
			},
		},
		ccLibraryStatic: ccLibraryStatic{
			ccBase: ccBase,
		},
	}
	module.properties.System_shared_libs = []string{defaultSystemSharedLibraries}
	return common.InitAndroidModule(module, common.HostAndDeviceSupported, "both",
		&ccBase.properties, &ccBase.unused, &module.variantProperties)
}

func (c *ccLibrary) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	deps := c.ccLibraryStatic.AndroidDynamicDependencies(ctx)
	deps = append(deps, c.ccLibraryShared.AndroidDynamicDependencies(ctx)...)
	return deps
}

func (c *ccLibrary) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	c.ccLibraryShared.setToolchain(ctx)

	cflags := []string{"-fPIC"}
	ldflags := c.ldflags(ctx)
	c.ccLibraryShared.setCompilerFlags(ctx, cflags, ldflags)

	objFiles := c.generateObjBuildActions(ctx)

	objFilesStatic := c.generateCustomObjBuildActions(ctx, common.DeviceStaticLibrary,
		c.variantProperties.Static.Srcs, c.variantProperties.Static.Cflags)
	objFilesShared := c.generateCustomObjBuildActions(ctx, common.DeviceSharedLibrary,
		c.variantProperties.Shared.Srcs, c.variantProperties.Shared.Cflags)

	objFilesStatic = append(objFilesStatic, objFiles...)
	objFilesShared = append(objFilesShared, objFiles...)

	c.ccLibraryStatic.generateLibraryBuildActions(ctx, objFilesStatic)
	c.ccLibraryShared.generateLibraryBuildActions(ctx, objFilesShared)
}

//
// Objects (for crt*.o)
//

type ccObject struct {
	ccBase
	outputFile string
}

func NewCCObject() (blueprint.Module, []interface{}) {
	module := &ccObject{}
	return common.InitAndroidModule(module, common.DeviceSupported, "both", &module.properties, &module.unused)
}

func (*ccObject) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	// object files can't have any dependencies
	return nil
}

func (c *ccObject) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	c.setToolchain(ctx)
	c.setCompilerFlags(ctx, nil, nil)

	objFiles := c.generateObjBuildActions(ctx)

	ctx.VisitDirectDeps(func(m blueprint.Module) {
		if obj, ok := m.(*ccObject); ok {
			objFiles = append(objFiles, obj.outputFile)
		} else {
			ctx.ModuleErrorf("unrecognized dependency type for %q", ctx.OtherModuleName(m))
		}
	})

	var outputFile string
	if len(objFiles) == 1 {
		outputFile = objFiles[0]
	} else {
		outputFile = filepath.Join(common.ModuleOutDir(ctx), ctx.ModuleName()+".o")
		TransformObjsToObj(ctx, objFiles, c.flags, outputFile)
	}

	c.outputFile = outputFile

	ctx.CheckbuildFile(outputFile)
}

//
// Executables
//

type ccBinary struct {
	ccDynamic
	binaryProperties binaryProperties
}

type binaryProperties struct {
	Static_executable bool

	// stem: set the name of the output
	Stem string `android:"arch_variant"`
}

func (c *ccBinary) getStem(ctx common.AndroidModuleContext) string {
	if c.binaryProperties.Stem != "" {
		return c.binaryProperties.Stem
	}
	return ctx.ModuleName()
}

func (c *ccBinary) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	deps := c.ccDynamic.AndroidDynamicDependencies(ctx)
	if c.DeviceSupported() {
		if c.binaryProperties.Static_executable {
			deps = append(deps, "crtbegin_static", "crtend_android")
		} else {
			deps = append(deps, "crtbegin_dynamic", "crtend_android")
		}
	}
	return deps
}

func NewCCBinary() (blueprint.Module, []interface{}) {
	module := &ccBinary{
		ccDynamic: ccDynamic{
			ccBase: &ccBase{},
		},
	}
	module.properties.System_shared_libs = []string{defaultSystemSharedLibraries}
	return common.InitAndroidModule(module, common.DeviceSupported, "first", &module.properties,
		&module.unused)
}

func (c *ccBinary) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	if !c.binaryProperties.Static_executable && inList("libc", c.properties.Static_libs) {
		ctx.ModuleErrorf("statically linking libc to dynamic executable, please remove libc\n" +
			"from static libs or set static_executable: true")
	}

	c.setToolchain(ctx)

	linker := "/system/bin/linker"
	if c.toolchain.Is64Bit() {
		linker = "/system/bin/linker64"
	}

	cflags := []string{"-fpie"}
	ldflags := []string{}
	if c.HostOrDevice().Device() {
		ldflags = []string{
			"-nostdlib",
			"-Bdynamic",
			fmt.Sprintf("-Wl,-dynamic-linker,%s", linker),
			"-Wl,--gc-sections",
			"-Wl,-z,nocopyreloc",
		}
	}

	c.setCompilerFlags(ctx, cflags, ldflags)

	objFiles := c.generateObjBuildActions(ctx)

	staticLibs, sharedLibs, wholeStaticLibs, crtBegin, crtEnd := c.collectDeps(ctx)
	if ctx.Failed() {
		return
	}

	outputFile := filepath.Join(common.ModuleOutDir(ctx), c.getStem(ctx))

	TransformObjToDynamicBinary(ctx, objFiles, sharedLibs, staticLibs, wholeStaticLibs,
		crtBegin, crtEnd, c.flags, outputFile)

	ctx.InstallFile("bin", outputFile)
}

//
// Host static library
//

func NewCCLibraryHostStatic() (blueprint.Module, []interface{}) {
	module := &ccLibraryStatic{
		ccBase: &ccBase{},
	}
	return common.InitAndroidModule(module, common.HostSupported, "both", &module.properties, &module.unused)
}

//
// Host Shared libraries
//

func NewCCLibraryHostShared() (blueprint.Module, []interface{}) {
	module := &ccLibraryShared{
		ccDynamic: ccDynamic{
			ccBase: &ccBase{},
		},
	}
	return common.InitAndroidModule(module, common.HostSupported, "both", &module.properties, &module.unused)
}

//
// Host Binaries
//

func NewCCBinaryHost() (blueprint.Module, []interface{}) {
	module := &ccBinary{
		ccDynamic: ccDynamic{
			ccBase: &ccBase{},
		},
	}
	return common.InitAndroidModule(module, common.HostSupported, "first", &module.properties, &module.unused)
}

//
// Device libraries shipped with gcc
//

type toolchainLibrary struct {
	ccLibraryStatic
}

func (*toolchainLibrary) AndroidDynamicDependencies(ctx common.AndroidDynamicDependerModuleContext) []string {
	// toolchain libraries can't have any dependencies
	return nil
}

func NewToolchainLibrary() (blueprint.Module, []interface{}) {
	module := &toolchainLibrary{
		ccLibraryStatic: ccLibraryStatic{
			ccBase: &ccBase{},
		},
	}
	return common.InitAndroidModule(module, common.DeviceSupported, "both")
}

func (c *toolchainLibrary) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	c.setToolchain(ctx)
	c.setCompilerFlags(ctx, nil, nil)

	libName := ctx.ModuleName() + staticLibraryExtension
	outputFile := filepath.Join(common.ModuleOutDir(ctx), libName)

	CopyGccLib(ctx, libName, c.flags, outputFile)

	c.outputFile = outputFile

	ctx.CheckbuildFile(outputFile)
}
