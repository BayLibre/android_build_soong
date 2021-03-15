// Copyright 2021 Google Inc. All rights reserved.
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

import "android/soong/android"

// This file contains the module implementation for java_defaults.

func init() {
	RegisterDefaultsBuildComponents(android.InitRegistrationContext)
}

func RegisterDefaultsBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("java_defaults", DefaultsFactory)
}

//
// Defaults
//
type Defaults struct {
	android.ModuleBase
	android.DefaultsModuleBase
	android.ApexModuleBase
}

// java_defaults provides a set of properties that can be inherited by other java or android modules.
//
// A module can use the properties from a java_defaults module using `defaults: ["defaults_module_name"]`.  Each
// property in the defaults module that exists in the depending module will be prepended to the depending module's
// value for that property.
//
// Example:
//
//     java_defaults {
//         name: "example_defaults",
//         srcs: ["common/**/*.java"],
//         javacflags: ["-Xlint:all"],
//         aaptflags: ["--auto-add-overlay"],
//     }
//
//     java_library {
//         name: "example",
//         defaults: ["example_defaults"],
//         srcs: ["example/**/*.java"],
//     }
//
// is functionally identical to:
//
//     java_library {
//         name: "example",
//         srcs: [
//             "common/**/*.java",
//             "example/**/*.java",
//         ],
//         javacflags: ["-Xlint:all"],
//     }
func DefaultsFactory() android.Module {
	module := &Defaults{}

	module.AddProperties(
		&CommonProperties{},
		&DeviceProperties{},
		&DexProperties{},
		&DexpreoptProperties{},
		&android.ProtoProperties{},
		&aaptProperties{},
		&androidLibraryProperties{},
		&appProperties{},
		&appTestProperties{},
		&overridableAppProperties{},
		&testProperties{},
		&ImportProperties{},
		&AARImportProperties{},
		&sdkLibraryProperties{},
		&commonToSdkLibraryAndImportProperties{},
		&DexImportProperties{},
		&android.ApexProperties{},
		&RuntimeResourceOverlayProperties{},
		&LintProperties{},
		&appTestHelperAppProperties{},
	)

	android.InitDefaultsModule(module)
	return module
}
