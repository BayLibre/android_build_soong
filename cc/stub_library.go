// Copyright 2020 Google Inc. All rights reserved.
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
	"strings"
	"sync"

	"android/soong/android"
)

var (
	stubLibrariesKey  = android.NewOnceKey("stubLibraries")
	stubLibrariesLock sync.Mutex
)

func stubLibraries(config android.Config) map[string]string {
	return config.Once(stubLibrariesKey, func() interface{} {
		return make(map[string]string)
	}).(map[string]string)
}

// StubLibraryMutator is a Mutator to scan stubbed libraries
func StubLibraryMutator(mctx android.BottomUpMutatorContext) {
	m, ok := mctx.Module().(*Module)
	if !ok {
		return
	}
	if !m.Enabled() {
		return
	}

	if (m.IsStubs() || m.HasStubsVariants()) && m.IsForPlatform() {
		stubLibraries := stubLibraries(mctx.Config())
		stubLibrariesLock.Lock()
		defer stubLibrariesLock.Unlock()

		name := m.BaseModuleName()
		stubLibraries[name] = name + ".so"
	}
}

func init() {
	android.RegisterModuleType("stub_libraries_txt", StubLibrariesTxtFactory)
}

type stubLibrariesTxt struct {
	android.ModuleBase
	outputFile android.OutputPath
}

// StubLibrariesTxtFactory is factory function to generate a stubLibrariesTxt object
func StubLibrariesTxtFactory() android.Module {
	m := &stubLibrariesTxt{}
	android.InitAndroidModule(m)
	return m
}

func (txt *stubLibrariesTxt) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	var list = android.SortedStringMapValues(stubLibraries(ctx.Config()))

	txt.outputFile = android.PathForModuleOut(ctx, txt.Name()).OutputPath
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Output:      txt.outputFile,
		Description: "Writing " + txt.outputFile.String(),
		Args: map[string]string{
			"content": strings.Join(list, "\\n"),
		},
	})

	installPath := android.PathForModuleInstall(ctx, "etc")
	ctx.InstallFile(installPath, txt.Name(), txt.outputFile)
}

func (txt *stubLibrariesTxt) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(txt.outputFile),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_STEM", txt.outputFile.Base())
			},
		},
	}}
}
