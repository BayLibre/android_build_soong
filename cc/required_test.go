// Copyright 2021 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cc

import (
	"testing"

	"android/soong/android"
)

var prepareForCcRequiredTest = android.GroupFixturePreparers(
	prepareForCcTest,
	android.PrepareForTestWithAndroidMk,
)

func TestModuleAliasForRequired(t *testing.T) {
	bp := `
		cc_module_alias_for_required {
			name: "cc_module_alias_for_required",
		}

		cc_library {
			name: "foo_vendor",
			vendor: true,
		}

		cc_library {
			name: "bar",
			vendor_available: true,
		}

		cc_library {
			name: "foo_recovery",
			recovery: true,
			vendor_available: true,
		}
	`

	prepareForCcRequiredTest.RunTestWithBp(t, bp)

	assertAlias := func(aliasFrom, aliasTo string) {
		a := moduleAliasForRequired.aliasMap[aliasFrom]
		android.AssertStringEquals(t, `module alias "`+aliasFrom+`"`, aliasTo, a.name)
	}

	assertAlias("foo_vendor.vendor", "foo_vendor")
	assertAlias("foo_recovery.recovery", "foo_recovery")
}

func TestModuleAliasForRequired_NameConflictErrors(t *testing.T) {
	bp := `
		cc_module_alias_for_required {
			name: "cc_module_alias_for_required",
		}

		cc_library {
			name: "foo_vendor",
			vendor: true,
		}

		cc_library {
			name: "foo_vendor.vendor",
			vendor: true,
		}
	`

	prepareForCcRequiredTest.
		ExtendWithErrorHandler(android.FixtureExpectsAtLeastOneErrorMatchingPattern("name conflict with module alias")).
		RunTestWithBp(t, bp)
}
