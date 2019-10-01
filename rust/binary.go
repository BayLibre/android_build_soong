// Copyright 2019 The Android Open Source Project
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

package rust

import (
	"path/filepath"
	"strings"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/rust/config"
)

func init() {
	android.RegisterModuleType("rust_binary", RustBinaryFactory)
	android.RegisterModuleType("rust_binary_host", RustBinaryHostFactory)
	// Rust tests are binary files built with --test.
	android.RegisterModuleType("rust_test", RustTestFactory)
	android.RegisterModuleType("rust_test_host", RustTestHostFactory)
}

type BinaryCompilerProperties struct {
	// path to the main source file that contains the program entry point (e.g. src/main.rs)
	Srcs []string `android:"path,arch_variant"`

	// passes -C prefer-dynamic to rustc, which tells it to dynamically link the stdlib (assuming it has no dylib dependencies already)
	Prefer_dynamic *bool

	// used by multi testcase modules to store the per testcase module SubName
	Stem *string
}

type binaryDecorator struct {
	*baseCompiler

	Properties           BinaryCompilerProperties
	distFile             android.OptionalPath
	unstrippedOutputFile android.Path
	isTest               bool
}

var _ compiler = (*binaryDecorator)(nil)

// rust_binary produces a binary that is runnable on a device.
func RustBinaryFactory() android.Module {
	module := NewRustBinary(android.HostAndDeviceSupported, false)
	return module.Init()
}

func RustBinaryHostFactory() android.Module {
	module := NewRustBinary(android.HostSupported, false)
	return module.Init()
}

func RustTestFactory() android.Module {
	module := NewRustBinary(android.HostAndDeviceSupported, true)
	return module.Init()
}

func RustTestHostFactory() android.Module {
	module := NewRustBinary(android.HostSupported, true)
	return module.Init()
}

func NewRustBinary(hod android.HostOrDeviceSupported, isTest bool) *Module {
	module := newModule(hod, android.MultilibFirst)

	dir := "bin"
	if isTest {
		dir = "testcases"
	}
	module.compiler = &binaryDecorator{
		baseCompiler: NewBaseCompiler(dir, ""), // TODO(chh): set up dir64?
		isTest:       isTest,
	}

	return module
}

func (binary *binaryDecorator) preferDynamic() bool {
	return Bool(binary.Properties.Prefer_dynamic)
}

func (binary *binaryDecorator) compilerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = binary.baseCompiler.compilerFlags(ctx, flags)

	if binary.isTest {
		// Compile a Rust test file like a binary with --test.
		flags.RustFlags = append(flags.RustFlags, "--test")
	}

	if ctx.toolchain().Bionic() {
		// no-undefined-version breaks dylib compilation since __rust_*alloc* functions aren't defined, but we can apply this to binaries.
		flags.LinkFlags = append(flags.LinkFlags,
			"-Wl,--gc-sections",
			"-Wl,-z,nocopyreloc",
			"-Wl,--no-undefined-version")
	}

	if binary.preferDynamic() {
		flags.RustFlags = append(flags.RustFlags, "-C prefer-dynamic")
	}
	return flags
}

func (binary *binaryDecorator) compilerDeps(ctx DepsContext, deps Deps) Deps {
	deps = binary.baseCompiler.compilerDeps(ctx, deps)

	if binary.preferDynamic() || len(deps.Dylibs) > 0 {
		for _, stdlib := range config.Stdlibs {
			deps.Dylibs = append(deps.Dylibs, stdlib+"_"+ctx.toolchain().RustTriple())
		}
	}

	if ctx.toolchain().Bionic() {
		deps = binary.baseCompiler.bionicDeps(ctx, deps)
		deps.CrtBegin = "crtbegin_dynamic"
		deps.CrtEnd = "crtend_android"
	}

	return deps
}

func (binary *binaryDecorator) compilerProps() []interface{} {
	return append(binary.baseCompiler.compilerProps(),
		&binary.Properties)
}

func (binary *binaryDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) android.Path {
	fileName := binary.getStem(ctx) + ctx.toolchain().ExecutableSuffix()

	srcPath := srcPathFromModuleSrcs(ctx, binary.Properties.Srcs)

	outputFile := android.PathForModuleOut(ctx, fileName)
	binary.unstrippedOutputFile = outputFile

	flags.RustFlags = append(flags.RustFlags, deps.depFlags...)

	TransformSrcToBinary(ctx, srcPath, deps, flags, outputFile, deps.linkDirs)

	return outputFile
}

func (test *binaryDecorator) testPerSrc() bool {
	return test.isTest
}

func (test *binaryDecorator) srcs() []string {
	return test.Properties.Srcs
}

func (test *binaryDecorator) setSrc(name, src string) {
	test.Properties.Srcs = []string{src}
	test.Properties.Stem = StringPtr(name)
	test.baseCompiler.Properties.Stem = StringPtr(name)
	// TODO(chh): keep only one Stem?
}

func (test *binaryDecorator) unsetSrc() {
	test.Properties.Srcs = nil
	test.Properties.Stem = StringPtr("")
	test.baseCompiler.Properties.Stem = StringPtr("")
}

type testPerSrc interface {
	testPerSrc() bool
	srcs() []string
	setSrc(string, string)
	unsetSrc()
}

var _ testPerSrc = (*binaryDecorator)(nil)

func TestPerSrcMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if test, ok := m.compiler.(testPerSrc); ok {
			numTests := len(test.srcs())
			if test.testPerSrc() && numTests > 0 {
				if duplicate, found := cc.CheckDuplicate(test.srcs()); found {
					mctx.PropertyErrorf("srcs", "found a duplicate entry %q", duplicate)
					return
				}
				testNames := make([]string, numTests)
				for i, src := range test.srcs() {
					testNames[i] = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
				}
				// TODO(chh): Add an "all tests" variation like cc/test.go?
				tests := mctx.CreateLocalVariations(testNames...)
				for i, src := range test.srcs() {
					tests[i].(*Module).compiler.(testPerSrc).setSrc(testNames[i], src)
				}
			}
		}
	}
}
