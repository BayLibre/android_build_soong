// Copyright 2023 Google Inc. All rights reserved.
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

package device_config

import (
	"android/soong/android"
	"github.com/google/blueprint"
)

var (
	pctx = android.NewPackageContext("android/soong/device_config")

	// for device_config
	aconfigRule = pctx.AndroidStaticRule("aconfig",
		blueprint.RuleParams{
			Command: `echo '["${aconfig}", "${in}", "${overrides}"]' > $out`,
			CommandDeps: []string{
				"${aconfig}",
			},
		}, "overrides")

	// for device_config
	srcJarRule = pctx.AndroidStaticRule("aconfig_srcjar",
		blueprint.RuleParams{
			Command: `rm -rf ${out}.tmp && mkdir -p ${out}.tmp && ` +
				`echo 'class MyFlags { } // ${in} ${aconfig}' > ${out}.tmp/MyFlags.java && ` +
				`$soong_zip -jar -o ${out} -C ${out}.tmp -D ${out}.tmp && rm -rf ${out}.tmp`,
			CommandDeps: []string{
				"$aconfig",
				"$soong_zip",
			},
		})
)

func init() {
	registerDeviceConfigBuildComponents(android.InitRegistrationContext)
}

func registerDeviceConfigBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("device_config", DeviceConfigFactory)
	ctx.RegisterModuleType("device_config_override", DeviceConfigOverrideFactory)
	ctx.RegisterModuleType("device_config_override_set", DeviceConfigOverrideSetFactory)
	pctx.HostBinToolVariable("aconfig", "aconfig")
	pctx.HostBinToolVariable("soong_zip", "soong_zip")
}
