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
)

func init() {
	// Rust tests are binary files built with --test.
	android.RegisterModuleType("rust_test", RustTestFactory)
	android.RegisterModuleType("rust_test_host", RustTestHostFactory)
}

func RustTestFactory() android.Module {
	module := NewRustBinary(android.HostAndDeviceSupported, true)
	return module.Init()
}

func RustTestHostFactory() android.Module {
	module := NewRustBinary(android.HostSupported, true)
	return module.Init()
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
				if duplicate, found := android.CheckDuplicate(test.srcs()); found {
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
