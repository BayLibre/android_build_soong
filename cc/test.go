// Copyright 2016 Google Inc. All rights reserved.
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

import (
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"android/soong"
	"android/soong/android"
)

type TestLinkerProperties struct {
	// if set, build against the gtest library. Defaults to true.
	Gtest bool

	// Create a separate binary for each source file.  Useful when there is
	// global state that can not be torn down and reset between each test suite.
	Test_per_src *bool
}

func init() {
	soong.RegisterModuleType("cc_test", testFactory)
	soong.RegisterModuleType("cc_test_library", testLibraryFactory)
	soong.RegisterModuleType("cc_benchmark", benchmarkFactory)
	soong.RegisterModuleType("cc_test_host", testHostFactory)
	soong.RegisterModuleType("cc_benchmark_host", benchmarkHostFactory)
}

// Module factory for tests
func testFactory() (blueprint.Module, []interface{}) {
	module := NewTest(android.HostAndDeviceSupported)
	return module.Init()
}

// Module factory for test libraries
func testLibraryFactory() (blueprint.Module, []interface{}) {
	module := NewTestLibrary(android.HostAndDeviceSupported)
	return module.Init()
}

// Module factory for benchmarks
func benchmarkFactory() (blueprint.Module, []interface{}) {
	module := NewBenchmark(android.HostAndDeviceSupported)
	return module.Init()
}

// Module factory for host tests
func testHostFactory() (blueprint.Module, []interface{}) {
	module := NewTest(android.HostSupported)
	return module.Init()
}

// Module factory for host benchmarks
func benchmarkHostFactory() (blueprint.Module, []interface{}) {
	module := NewBenchmark(android.HostSupported)
	return module.Init()
}

func testPerSrcMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if test, ok := m.linker.(*testDecorator); ok && m.linker.binary() != nil {
			if Bool(test.Properties.Test_per_src) {
				testNames := make([]string, len(m.compiler.(*baseCompiler).Properties.Srcs))
				for i, src := range m.compiler.(*baseCompiler).Properties.Srcs {
					testNames[i] = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
				}
				tests := mctx.CreateLocalVariations(testNames...)
				for i, src := range m.compiler.(*baseCompiler).Properties.Srcs {
					tests[i].(*Module).compiler.(*baseCompiler).Properties.Srcs = []string{src}
					tests[i].(*Module).linker.binary().Properties.Stem = testNames[i]
				}
			}
		}
	}
}

type testDecorator struct {
	Properties TestLinkerProperties
	linker
	installer
}

func (test *testDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = test.linker.linkerFlags(ctx, flags)

	if !test.Properties.Gtest {
		return flags
	}

	flags.CFlags = append(flags.CFlags, "-DGTEST_HAS_STD_STRING")
	if ctx.Host() {
		flags.CFlags = append(flags.CFlags, "-O0", "-g")

		switch ctx.Os() {
		case android.Windows:
			flags.CFlags = append(flags.CFlags, "-DGTEST_OS_WINDOWS")
		case android.Linux:
			flags.CFlags = append(flags.CFlags, "-DGTEST_OS_LINUX")
			flags.LdFlags = append(flags.LdFlags, "-lpthread")
		case android.Darwin:
			flags.CFlags = append(flags.CFlags, "-DGTEST_OS_MAC")
			flags.LdFlags = append(flags.LdFlags, "-lpthread")
		}
	} else {
		flags.CFlags = append(flags.CFlags, "-DGTEST_OS_LINUX_ANDROID")
	}

	return flags
}

func (test *testDecorator) linkerDeps(ctx BaseModuleContext, deps Deps) Deps {
	if test.Properties.Gtest {
		if ctx.sdk() && ctx.Device() {
			switch ctx.selectedStl() {
			case "ndk_libc++_shared", "ndk_libc++_static":
				deps.StaticLibs = append(deps.StaticLibs, "libgtest_main_ndk_libcxx", "libgtest_ndk_libcxx")
			case "ndk_libgnustl_static":
				deps.StaticLibs = append(deps.StaticLibs, "libgtest_main_ndk_gnustl", "libgtest_ndk_gnustl")
			default:
				deps.StaticLibs = append(deps.StaticLibs, "libgtest_main_ndk", "libgtest_ndk")
			}
		} else {
			deps.StaticLibs = append(deps.StaticLibs, "libgtest_main", "libgtest")
		}
	}

	deps = test.linker.linkerDeps(ctx, deps)

	return deps
}

func (test *testDecorator) linkerInit(ctx BaseModuleContext) {
	test.linker.linkerInit(ctx)
	runpath := "../../lib"
	if ctx.toolchain().Is64Bit() {
		runpath += "64"
	}
	test.linker.addRunpath(runpath)
}

func (test *testDecorator) linkerProps() []interface{} {
	return append(test.linker.linkerProps(), &test.Properties)
}

func (test *testDecorator) install(ctx ModuleContext, file android.Path) {
	test.installer.setDir(filepath.Join("nativetest", ctx.ModuleName()),
		filepath.Join("nativetest64", ctx.ModuleName()),
		InstallInData)
	test.installer.install(ctx, file)
}

func NewTest(hod android.HostOrDeviceSupported) *Module {
	module := NewBinary(hod)
	module.multilib = android.MultilibBoth

	test := &testDecorator{
		linker:    module.linker,
		installer: module.installer,
	}
	test.Properties.Gtest = true
	module.linker = test
	module.installer = test
	return module
}

func NewTestLibrary(hod android.HostOrDeviceSupported) *Module {
	module, _ := NewLibrary(android.HostAndDeviceSupported, false, true)
	test := &testDecorator{
		linker:    module.linker,
		installer: module.installer,
	}
	test.Properties.Gtest = true
	module.linker = test
	module.installer = test
	return module
}

type benchmarkDecorator struct {
	linker
	installer
}

func (benchmark *benchmarkDecorator) linkerInit(ctx BaseModuleContext) {
	benchmark.linker.linkerInit(ctx)
	runpath := "../../lib"
	if ctx.toolchain().Is64Bit() {
		runpath += "64"
	}
	benchmark.linker.addRunpath(runpath)
}

func (benchmark *benchmarkDecorator) install(ctx ModuleContext, file android.Path) {
	benchmark.installer.setDir(filepath.Join("nativetest", ctx.ModuleName()),
		filepath.Join("nativetest64", ctx.ModuleName()),
		InstallInData)
	benchmark.installer.install(ctx, file)
}

func (benchmark *benchmarkDecorator) linkerDeps(ctx BaseModuleContext, deps Deps) Deps {
	deps = benchmark.linker.linkerDeps(ctx, deps)
	deps.StaticLibs = append(deps.StaticLibs, "libgoogle-benchmark")
	return deps
}

func NewBenchmark(hod android.HostOrDeviceSupported) *Module {
	module := NewBinary(hod)
	module.multilib = android.MultilibBoth
	benchmark := &benchmarkDecorator{
		linker:    module.linker,
		installer: module.installer,
	}
	module.linker = benchmark
	module.installer = benchmark
	return module
}
