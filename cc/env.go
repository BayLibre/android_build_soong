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
	"android/soong/android"
)

func EnvDefault(ctx android.BaseContext, key string, defaultValue string) string {
	ret := ctx.AConfig().Getenv(key)
	if ret == "" {
		return defaultValue
	}
	return ret
}

func EnvTrue(ctx android.BaseContext, key string) bool {
	return ctx.AConfig().Getenv(key) == "true"
}

func EnvFalse(ctx android.BaseContext, key string) bool {
	return ctx.AConfig().Getenv(key) == "false"
}

//
// Hooks for environment variables affecting build rules.
//

func CustomLinker(ctx android.LoadHookContext) {
	linker := EnvDefault(ctx, "CUSTOM_TARGET_LINKER", "")
	if linker != "" {
		type props struct {
			DynamicLinker string
		}

		p := &props{}
		p.DynamicLinker = linker
		ctx.AppendProperties(p)
	}
}

func Prefer32Bit(ctx android.LoadHookContext) {
	if EnvTrue(ctx, "HOST_PREFER_32_BIT") {
		type props struct {
			Target struct {
				Host struct {
					Compile_multilib string
				}
			}
		}

		p := &props{}
		p.Target.Host.Compile_multilib = "prefer32"
		ctx.AppendProperties(p)
	}
}
