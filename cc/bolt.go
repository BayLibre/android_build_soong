// Copyright 2024 Google Inc. All rights reserved.
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
	"android/soong/android"
)

// Experimental support for optimizing binaries using BOLT.
type BoltProperties struct {
	Bolt bool
}

type Bolt struct {
	Properties BoltProperties
}

func (bolt *Bolt) props() []interface{} {
	return []interface{}{&bolt.Properties}
}

func (bolt *Bolt) Enabled(ctx ModuleContext) bool {
	// BOLT officially supports X86_64 and arm64, but Android only supports arm64 for now.
	if ctx.Arch().ArchType != android.Arm64 {
		return false
	}
	return bolt != nil && bolt.Properties.Bolt
}

func (bolt *Bolt) ApplyBolt(actx android.ModuleContext, in android.Path, out android.ModuleOutPath) {
	transformBolt(actx, in, out, "")
}
