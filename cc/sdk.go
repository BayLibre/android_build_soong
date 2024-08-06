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
	"android/soong/android"
)

// sdkTransitionMutator creates a platform and an SDK variant for modules
// that set sdk_version, and ignores sdk_version for the platform
// variant.  The SDK variant will be used for embedding in APKs
// that may be installed on older platforms.  Apexes use their own
// variants that enforce backwards compatibility.
type sdkTransitionMutator struct{}

func (sdkTransitionMutator) Split(ctx android.BaseModuleContext) []string {
	if ctx.Os() != android.Android {
		return []string{""}
	}

	if sdkInterface, ok := ctx.Module().(android.SdkInterface); ok {
		return sdkInterface.SdkVariants(ctx)
	}

	return []string{""}
}

func (sdkTransitionMutator) OutgoingTransition(ctx android.OutgoingTransitionContext, sourceVariation string) string {
	return sourceVariation
}

func (sdkTransitionMutator) IncomingTransition(ctx android.IncomingTransitionContext, incomingVariation string) string {
	if ctx.Os() != android.Android {
		return ""
	}

	if sdkInterface, ok := ctx.Module().(android.SdkInterface); ok {
		return sdkInterface.IncomingSdkTransition(ctx, incomingVariation)
	}

	if ctx.IsAddingDependency() {
		return incomingVariation
	} else {
		return ""
	}
}

func (sdkTransitionMutator) Mutate(ctx android.BottomUpMutatorContext, variation string) {
	if ctx.Os() != android.Android {
		return
	}

	if sdkInterface, ok := ctx.Module().(android.SdkInterface); ok {
		sdkInterface.MutateSdkVariant(ctx, variation)
	}
}
