// Copyright 2022 Google Inc. All rights reserved.
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

	"android/soong/android"
)

// TODO: This could go into a utils package if it's generally useful
// As it stands this is used by Droidstubs to separate all of the independent things
// into more organized code.

type CompoundModuleMultiplexer struct {
	// An Array of CompoundModulePart objects, one for each file this module can produce.
	Parts []interface{}
}

type outputFilesPart interface {
	OutputFiles(tag string) (android.Paths, bool)
}

func (m *CompoundModuleMultiplexer) OutputFiles(tag string) (android.Paths, error) {
	for _, part := range m.Parts {
		if output, ok := part.(outputFilesPart); ok {
			result, ok := output.OutputFiles(tag)
			if ok {
				return result, nil
			}
		}
	}
	return nil, fmt.Errorf("unsupported module reference tag %q", tag)
}

type generateAndroidBuildActionsPart interface {
	GenerateAndroidBuildActions(ctx android.ModuleContext)
}

func (m *CompoundModuleMultiplexer) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	for _, part := range m.Parts {
		if output, ok := part.(generateAndroidBuildActionsPart); ok {
			output.GenerateAndroidBuildActions(ctx)
		}
	}
}

type generateMkExtraFootersPart interface {
	GenerateMkExtraFooters(w io.Writer, name, prefix, moduleDir string)
}

func (m *CompoundModuleMultiplexer) GenerateMkExtraFooters(w io.Writer, name, prefix, moduleDir string) {
	for _, part := range m.Parts {
		if output, ok := part.(generateMkExtraFootersPart); ok {
			output.GenerateMkExtraFooters(w, name, prefix, moduleDir)
		}
	}

}
