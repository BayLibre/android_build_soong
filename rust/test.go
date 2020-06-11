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
	"android/soong/tradefed"
)

type TestProperties struct {
	// Disables the creation of a test-specific directory when used with
	// relative_install_path. Useful if several tests need to be in the same
	// directory, but test_per_src doesn't work.
	No_named_install_directory *bool

	// the name of the test configuration (for example "AndroidTest.xml") that should be
	// installed with the module.
	Test_config *string `android:"arch_variant"`

	// the name of the test configuration template (for example "AndroidTestTemplate.xml") that
	// should be installed with the module.
	Test_config_template *string `android:"arch_variant"`

	// list of compatibility suites (for example "cts", "vts") that the module should be
	// installed into.
	Test_suites []string `android:"arch_variant"`

	// Flag to indicate whether or not to create test config automatically. If AndroidTest.xml
	// doesn't exist next to the Android.bp, this attribute doesn't need to be set to true
	// explicitly.
	Auto_gen_config *bool

	// Indicate whether each source file in srcs represents a separate test. If this is true,
	// output test files will take the name of the source file.
	Test_per_src *bool
}

// A test module is a binary module with extra --test compiler flag
// and different default installation directory.
// In golang, inheriance is written as a component.
type testDecorator struct {
	*binaryDecorator
	Properties TestProperties
	testConfig android.Path
}

func (test *testDecorator) nativeCoverage() bool {
	return true
}

func NewRustTest(hod android.HostOrDeviceSupported) (*Module, *testDecorator) {
	module := newModule(hod, android.MultilibFirst)

	test := &testDecorator{
		binaryDecorator: &binaryDecorator{
			baseCompiler: NewBaseCompiler("nativetest", "nativetest64", InstallInData),
		},
	}

	module.compiler = test
	module.AddProperties(&test.Properties)
	return module, test
}

func (test *testDecorator) compilerProps() []interface{} {
	return append(test.binaryDecorator.compilerProps(), &test.Properties)
}

func (test *testDecorator) install(ctx ModuleContext, file android.Path) {
	test.testConfig = tradefed.AutoGenRustTestConfig(ctx,
		test.Properties.Test_config,
		test.Properties.Test_config_template,
		test.Properties.Test_suites,
		nil,
		test.Properties.Auto_gen_config)

	// default relative install path is module name
	if !Bool(test.Properties.No_named_install_directory) {
		test.baseCompiler.relative = ctx.ModuleName()
	} else if String(test.baseCompiler.Properties.Relative_install_path) == "" {
		ctx.PropertyErrorf("no_named_install_directory", "Module install directory may only be disabled if relative_install_path is set")
	}

	test.binaryDecorator.install(ctx, file)
}

func (test *testDecorator) compilerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = test.binaryDecorator.compilerFlags(ctx, flags)
	flags.RustFlags = append(flags.RustFlags, "--test")
	return flags
}

func init() {
	// Rust tests are binary files built with --test.
	android.RegisterModuleType("rust_test", RustTestFactory)
	android.RegisterModuleType("rust_test_host", RustTestHostFactory)
}

func RustTestFactory() android.Module {
	module, _ := NewRustTest(android.HostAndDeviceSupported)
	return module.Init()
}

func RustTestHostFactory() android.Module {
	module, _ := NewRustTest(android.HostSupported)
	return module.Init()
}

func (test *testDecorator) testPerSrc() bool {
	return Bool(test.Properties.Test_per_src)
}

func (test *testDecorator) srcs() []string {
	return test.binaryDecorator.Properties.Srcs
}

func (test *testDecorator) setSrc(name, src string) {
	test.binaryDecorator.Properties.Srcs = []string{src}
	test.baseCompiler.Properties.Stem = StringPtr(name)
}

func (test *testDecorator) unsetSrc() {
	test.binaryDecorator.Properties.Srcs = nil
	test.baseCompiler.Properties.Stem = StringPtr("")
}

type testPerSrc interface {
	testPerSrc() bool
	srcs() []string
	setSrc(string, string)
	unsetSrc()
}

//TODO(ivanlozano) There's potential for deduplication here with cc's implementation.
var _ testPerSrc = (*testDecorator)(nil)

func TestPerSrcMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if test, ok := m.compiler.(testPerSrc); ok && test.testPerSrc() {
			numTests := len(test.srcs())
			if numTests > 1 {
				if duplicate, found := android.CheckDuplicate(test.srcs()); found {
					mctx.PropertyErrorf("srcs", "found a duplicate entry %q", duplicate)
					return
				}

				testNames := make([]string, numTests)

				for i, src := range test.srcs() {
					testNames[i] = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
				}

				// TODO(chh): Add an "all tests" variation like cc/test.go?
				testNames = append(testNames)
				tests := mctx.CreateLocalVariations(testNames...)

				for i, src := range test.srcs() {
					tests[i].(*Module).compiler.(testPerSrc).setSrc(testNames[i], src)
				}
			} else {
				mctx.PropertyErrorf("srcs", "Multiple sources must be provided with test_per_src true")
			}
		}
	}
}
