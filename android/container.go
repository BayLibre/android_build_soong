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

package android

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

var containerDependencyViolationAllowlist = map[string][]string{
	"AdServices-core":                   {"framework-res"},
	"AdServicesApk":                     {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"AdServicesLib":                     {"BackCompatLintChecker", "framework-annotations-lib", "framework-res"},
	"AppInRebootlessApex":               {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"Bluetooth":                         {"BluetoothLintChecker", "android.hidl.manager-V1.0-java", "androidx.room_room-compiler-plugin", "app-compat-annotations", "ext", "framework", "framework-annotations-lib", "framework-bluetooth-pre-jarjar", "framework-configinfrastructure", "framework-mediaprovider", "framework-res", "unsupportedappusage"},
	"CameraExtensionsProxy":             {"androidx.camera.extensions.impl"},
	"CarServiceUpdatable":               {"android.hidl.manager-V1.0-java", "framework-bluetooth", "framework-configinfrastructure", "framework-connectivity", "framework-statsd", "framework-tethering", "framework-wifi", "modules-utils-os", "modules-utils-preconditions", "modules-utils-shell-command-handler"},
	"CellBroadcastApp":                  {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"CellBroadcastCommon":               {"framework-annotations-lib", "framework-bluetooth", "framework-res", "framework-statsd"},
	"CellBroadcastServiceModule":        {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-annotations-lib", "framework-res", "framework-statsd", "unsupportedappusage"},
	"CompOSPayloadApp":                  {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"CtsShim":                           {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"CtsShimAddApkToApex":               {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"CtsShimPriv":                       {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"CtsShimPrivUpgrade":                {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"CtsShimTargetPSdk":                 {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"DevCameraGoogle":                   {"com.google.android.camera.experimental2018", "com.google.android.camera.experimental2019", "com.google.android.camera.experimental2020", "com.google.android.camera.experimental2020_midyear", "com.google.android.camera.experimental2021", "com.google.android.camera.experimental2022", "com.google.android.camera.experimental2023", "com.google.android.camera.experimental2024"},
	"DeviceConfigServiceResources":      {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"DeviceDropMonitor":                 {"vendor-pixelatoms-java"},
	"DeviceLockController":              {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"DeviceLockControllerDebug":         {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"DeviceLockControllerGoogle":        {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"DeviceLockControllerGoogleDebug":   {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"DockSetupLibrary":                  {"vendor-pixelatoms-java"},
	"ExtServices-core":                  {"framework-configinfrastructure", "framework-connectivity", "framework-res"},
	"ExtServices-sminus":                {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"ExtServices-tplus":                 {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"FederatedCompute":                  {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "auto_value_annotations", "auto_value_plugin", "ext", "framework", "framework-annotations-lib", "framework-configinfrastructure", "framework-res"},
	"GoogleSafetyCenterResources":       {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"HealthConnectBackupRestore":        {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "error_prone_android_framework", "ext", "framework", "framework-res"},
	"HealthConnectBackupRestoreLibrary": {"framework-annotations-lib", "framework-res", "nullaway_plugin"},
	"HealthConnectController":           {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"HealthConnectLibrary":              {"framework-res", "framework-configinfrastructure"},
	"MediaProvider":                     {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "app-compat-annotations", "error_prone_mediaprovider", "ext", "framework", "framework-annotations-lib", "framework-configinfrastructure", "framework-res", "framework-statsd", "glide-annotation-processor", "unsupportedappusage"},
	"NetworkStackApi31Shims":            {"framework-wifi"},
	"NetworkStackApi33Shims":            {"framework-bluetooth", "framework-wifi"},
	"NetworkStackApi34Shims":            {"framework-bluetooth", "framework-wifi"},
	"NetworkStackApi35Shims":            {"android.net.ipsec.ike", "framework-bluetooth", "framework-wifi"},
	"NetworkStackApiCurrentShims":       {"framework-configinfrastructure", "framework-statsd", "framework-wifi"},
	"NetworkStackApiStableShims":        {"framework-configinfrastructure", "framework-statsd", "framework-wifi"},
	"NfcNciApex":                        {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "framework-bluetooth", "framework-configinfrastructure", "framework-permission", "framework-permission-s", "framework-wifi"},
	"OnDevicePersonalization":           {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-adservices", "framework-annotations-lib", "framework-configinfrastructure", "framework-res", "staledataclass-annotation-processor"},
	"OsuLogin":                          {"ext", "framework", "framework-res"},
	"OverlayRemountedTest_Overlay":      {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"OverlayRemountedTest_Target":       {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"PackageManagerTestApexApp":         {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"PermissionController":              {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"PermissionController-lib":          {"androidx.compose.compiler_compiler-hosted", "framework-res", "safety-center-annotations"},
	"PersistentBackgroundServices-common-no-concurrency-modules": {"vendor-pixelatoms-java"},
	"Photopicker":                                                  {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"PhotopickerLib":                                               {"androidx.compose.compiler_compiler-hosted", "framework-configinfrastructure", "framework-res"},
	"PlatformProperties":                                           {"PlatformProperties_public", "sysprop-library-stub-platform"},
	"PrivAppInRebootlessApex":                                      {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"QualifiedNetworksService":                                     {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "framework-connectivity", "framework-wifi"},
	"SafetyCenterResources":                                        {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"SafetyCenterResourcesShared":                                  {"framework-res"},
	"ScriptExecutor":                                               {"android.hidl.manager-V1.0-java"},
	"SdkSandbox":                                                   {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-annotations-lib", "framework-res"},
	"SdkSandbox-java-lib":                                          {"framework-annotations-lib"},
	"ServiceConnectivityResources":                                 {"ext", "framework", "framework-res"},
	"ServiceUwbResources":                                          {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "ext", "framework", "framework-res"},
	"ServiceWifiResources":                                         {"ext", "framework", "framework-res"},
	"SettingsLibActionBarShadow":                                   {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibActivityEmbedding":                                 {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibAppPreference":                                     {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibBarChartPreference":                                {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibCollapsingToolbarBaseActivity":                     {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibColor":                                             {"framework-res"},
	"SettingsLibFooterPreference":                                  {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibHelpUtils":                                         {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibIllustrationPreference":                            {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibLayoutPreference":                                  {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibMainSwitchPreference":                              {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibProfileSelector":                                   {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibProgressBar":                                       {"framework-res"},
	"SettingsLibRestrictedLockUtils":                               {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibSearchWidget":                                      {"framework-res"},
	"SettingsLibSelectorWithWidgetPreference":                      {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibSettingsSpinner":                                   {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibSettingsTheme":                                     {"framework-res"},
	"SettingsLibSettingsTransition":                                {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibTopIntroPreference":                                {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibTwoTargetPreference":                               {"SettingsLibLintChecker", "framework-res"},
	"SettingsLibUtils":                                             {"SettingsLibLintChecker", "framework-res"},
	"TestAppACrashingV2":                                           {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"TestAppAv1":                                                   {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"TestAppAv2":                                                   {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"Tethering":                                                    {"connectivity-internal-api-util", "ext", "framework", "framework-bluetooth", "framework-configinfrastructure", "framework-res", "framework-wifi", "unsupportedappusage"},
	"TetheringApiCurrentLib":                                       {"connectivity-internal-api-util", "framework-bluetooth", "framework-configinfrastructure", "framework-res", "framework-wifi", "unsupportedappusage"},
	"TetheringApiStableLib":                                        {"connectivity-internal-api-util", "framework-bluetooth", "framework-configinfrastructure", "framework-res", "framework-wifi", "unsupportedappusage"},
	"TetheringNext":                                                {"connectivity-internal-api-util", "ext", "framework", "framework-bluetooth", "framework-configinfrastructure", "framework-res", "framework-wifi", "unsupportedappusage"},
	"TextClassifierNotificationLibNoManifest":                      {"framework-res"},
	"TextClassifierServiceLibNoManifest":                           {"androidx.room_room-compiler-plugin", "auto_value_plugin", "framework-res"},
	"VmLauncherApp":                                                {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"WifiDialog":                                                   {"ext", "framework", "framework-res"},
	"adservices-cobalt":                                            {"androidx.room_room-compiler-plugin", "auto_annotation_plugin", "auto_value_plugin", "framework-annotations-lib", "framework-res"},
	"adservices-service-core":                                      {"BackCompatLintChecker", "androidx.appsearch_appsearch-compiler-plugin", "androidx.room_room-compiler-plugin", "auto_annotation_plugin", "auto_oneof_plugin", "auto_value_plugin", "framework-annotations-lib", "framework-configinfrastructure", "framework-res"},
	"adservices-shared-common":                                     {"framework-annotations-lib"},
	"adservices-shared-error-logging":                              {"auto_annotation_plugin", "auto_value_plugin", "framework-annotations-lib"},
	"adservices-shared-proto-utils":                                {"framework-annotations-lib"},
	"adservices-shared-spe":                                        {"auto_annotation_plugin", "auto_value_plugin", "framework-annotations-lib"},
	"adservices-shared-storage":                                    {"framework-annotations-lib"},
	"adservices-shared-util":                                       {"framework-annotations-lib"},
	"android.car-module.impl":                                      {"framework-bluetooth", "framework-configinfrastructure", "framework-wifi", "modules-utils-preconditions"},
	"android.net.ipsec.ike.impl":                                   {"conscrypt.module.public.api", "framework-annotations-lib", "framework-configinfrastructure", "unsupportedappusage"},
	"android.net.wifi.flags-aconfig-java":                          {"aconfig-annotations-lib", "fake_device_config", "unsupportedappusage"},
	"android.os.profiling.flags-aconfig-java":                      {"aconfig-annotations-lib", "unsupportedappusage"},
	"android.permission.flags-aconfig-java":                        {"aconfig-annotations-lib", "fake_device_config", "unsupportedappusage"},
	"android.service.notification.flags-aconfig-export-java":       {"aconfig-annotations-lib", "fake_device_config", "unsupportedappusage"},
	"android.system.virtualmachine.res":                            {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"android_downloader_lib":                                       {"auto_value_plugin", "framework-res"},
	"androidx.compose.material_material-ripple":                    {"androidx.compose.compiler_compiler-hosted"},
	"androidx.legacy_legacy-preference-v14":                        {"framework-res"},
	"androidx.legacy_legacy-support-core-ui":                       {"framework-res"},
	"androidx.legacy_legacy-support-v13":                           {"framework-res"},
	"androidx.legacy_legacy-support-v4":                            {"framework-res"},
	"androidx.lifecycle_lifecycle-extensions":                      {"framework-res"},
	"bluetooth-nano-protos":                                        {"libprotobuf-java-nano"},
	"bluetooth.change-ids":                                         {"app-compat-annotations"},
	"bluetooth_flags_java_lib":                                     {"aconfig-annotations-lib", "framework-configinfrastructure", "unsupportedappusage"},
	"bouncycastle":                                                 {"unsupportedappusage"},
	"com.android.apex.maxsdk.app.available.target10k.test":         {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.apex.maxsdk.app.available.test":                   {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.apex.maxsdk.app.unavailable.test":                 {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.apex.product.app.test":                            {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.apex.system.app.test":                             {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.apex.system_ext.app.test":                         {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.apex.vendor.app.test":                             {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.modules.apkinapex.apps.futureminsdk":              {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.modules.apkinapex.apps.futuretargetsdk":           {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.modules.apkinapex.apps.installable":               {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.modules.apkinapex.apps.pastmaxsdk":                {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java"},
	"com.android.vcard":                                            {"framework-annotations-lib"},
	"com.google.android.camera.support.v23.experimental":           {"com.google.android.camera.experimental2018", "com.google.android.camera.experimental2019", "com.google.android.camera.experimental2020", "com.google.android.camera.experimental2020_midyear", "com.google.android.camera.experimental2021", "com.google.android.camera.experimental2022", "com.google.android.camera.experimental2023", "com.google.android.camera.experimental2024"},
	"conscrypt":                                                    {"unsupportedappusage"},
	"core-oj":                                                      {"core-oj-hiddenapi-annotations"},
	"cronet_aml_base_base_java":                                    {"jsr305"},
	"cronet_aml_components_cronet_android_cronet_impl_native_java": {"jsr305"},
	"cronet_aml_net_android_net_java":                              {"jsr305"},
	"device_config_reboot_flags_java_lib":                          {"aconfig-annotations-lib", "ext", "framework", "unsupportedappusage"},
	"devicelockcontroller-lib":                                     {"modules-utils-expresslog", "framework-statsd"},
	"framework-adservices.impl":                                    {"adservices_flags_lib", "framework-annotations-lib"},
	"framework-appsearch.impl":                                     {"ext", "framework", "framework-annotations-lib", "safeparcel-annotation-processor", "unsupportedappusage"},
	"framework-bluetooth.impl":                                     {"app-compat-annotations", "framework-annotations-lib", "unsupportedappusage"},
	"framework-configinfrastructure.impl":                          {"framework-annotations-lib"},
	"framework-connectivity-t.impl":                                {"app-compat-annotations", "framework-annotations-lib", "framework-bluetooth", "framework-connectivity-pre-jarjar", "framework-wifi", "unsupportedappusage"},
	"framework-connectivity.impl":                                  {"app-compat-annotations", "framework-annotations-lib", "unsupportedappusage"},
	"framework-crashrecovery.impl":                                 {"framework-annotations-lib"},
	"framework-healthfitness.impl":                                 {"framework-annotations-lib", "nullaway_plugin"},
	"framework-media.impl":                                         {"framework-annotations-lib"},
	"framework-mediaprovider.impl":                                 {"framework-annotations-lib", "unsupportedappusage"},
	"framework-ondevicepersonalization.impl":                       {"framework-annotations-lib", "ondevicepersonalization_flags_lib", "staledataclass-annotation-processor"},
	"framework-pdf-v.impl":                                         {"framework-annotations-lib"},
	"framework-pdf.impl":                                           {"framework-annotations-lib", "modules-utils-preconditions", "unsupportedappusage"},
	"framework-permission-s-shared":                                {"framework-annotations-lib", "unsupportedappusage"},
	"framework-permission-s.impl":                                  {"app-compat-annotations", "framework-annotations-lib"},
	"framework-permission.impl":                                    {"framework-annotations-lib"},
	"framework-profiling.impl":                                     {"framework-annotations-lib"},
	"framework-scheduling.impl":                                    {"framework-annotations-lib"},
	"framework-sdkextensions.impl":                                 {"framework-annotations-lib"},
	"framework-sdksandbox.impl":                                    {"framework-annotations-lib"},
	"framework-statsd.impl":                                        {"framework-annotations-lib", "framework-configinfrastructure"},
	"framework-tethering.impl":                                     {"framework-annotations-lib"},
	"framework-uwb.impl":                                           {"framework-annotations-lib", "unsupportedappusage"},
	"framework-wifi-util-lib":                                      {"framework-annotations-lib", "unsupportedappusage"},
	"framework-wifi.impl":                                          {"aconfig-annotations-lib", "app-compat-annotations", "framework-annotations-lib", "framework-configinfrastructure", "unsupportedappusage"},
	"grpc-java-core-internal":                                      {"perfmark-api-lib", "gson"},
	"guava":                                                        {"guava-jre"},
	"hilt_android":                                                 {"dagger2-compiler", "framework-res", "hilt_aggregated_deps_processor", "hilt_alias_of_processor", "hilt_android_entry_point_processor", "hilt_component_tree_deps_processor", "hilt_define_component_processor", "hilt_early_entry_point_processor", "hilt_generates_root_input_processor", "hilt_originating_element_processor", "hilt_root_processor", "hilt_viewmodel_processor"},
	"hilt_core":                                                    {"hilt_define_component_processor", "hilt_generates_root_input_processor"},
	"httpclient_impl":                                              {"httpclient_api", "framework-annotations-lib"},
	"iconloader_sc_mainline_prod":                                  {"framework-res"},
	"kotlinx_coroutines":                                           {"kotlinx_coroutines-host"},
	"libnativeloader_e2e_tests":                                    {"libnativeloader_vendor_shared_lib", "loadlibrarytest_vendor_app"},
	"loadlibrarytest_product_app":                                  {"libnativeloader_vendor_shared_lib"},
	"loadlibrarytest_testlib":                                      {"libnativeloader_vendor_shared_lib"},
	"media_mainline_flags_java_lib":                                {"aconfig-annotations-lib", "fake_device_config", "unsupportedappusage"},
	"mediaprovider_flags_java_lib":                                 {"aconfig-annotations-lib", "ext", "framework", "unsupportedappusage"},
	"mmslib":                                                       {"unsupportedappusage"},
	"mobile_data_downloader_lib":                                   {"auto_annotation_plugin", "auto_value_plugin", "dagger2-compiler", "framework-annotations-lib", "framework-res", "unsupportedappusage"},
	"modules-utils-backgroundthread":                               {"framework-annotations-lib"},
	"modules-utils-binary-xml":                                     {"framework-annotations-lib"},
	"modules-utils-build":                                          {"framework-annotations-lib"},
	"modules-utils-build_system":                                   {"framework-annotations-lib"},
	"modules-utils-bytesmatcher":                                   {"framework-annotations-lib"},
	"modules-utils-expresslog":                                     {"framework-statsd", "framework-annotations-lib"},
	"modules-utils-fastxmlserializer":                              {"framework-annotations-lib", "unsupportedappusage"},
	"modules-utils-handlerexecutor":                                {"framework-annotations-lib"},
	"modules-utils-list-slice":                                     {"framework-annotations-lib"},
	"modules-utils-locallog":                                       {"unsupportedappusage"},
	"modules-utils-os":                                             {"framework-annotations-lib"},
	"modules-utils-package-state":                                  {"framework-annotations-lib"},
	"modules-utils-preconditions":                                  {"framework-annotations-lib", "unsupportedappusage"},
	"modules-utils-shell-command-handler":                          {"framework-annotations-lib"},
	"modules-utils-statemachine":                                   {"framework-annotations-lib", "unsupportedappusage"},
	"modules-utils-uieventlogger-interface":                        {"framework-annotations-lib"},
	"net-utils-device-common":                                      {"framework-annotations-lib", "framework-configinfrastructure"},
	"net-utils-device-common-ip":                                   {"framework-annotations-lib"},
	"net-utils-device-common-struct":                               {"framework-annotations-lib"},
	"net-utils-device-common-struct-base":                          {"framework-annotations-lib"},
	"net-utils-framework-common":                                   {"framework-annotations-lib"},
	"net-utils-services-common":                                    {"framework-annotations-lib"},
	"networkstack-client":                                          {"framework-annotations-lib"},
	"okhttp":                                                       {"conscrypt.module.intra.core.api"},
	"okhttp-norepackage":                                           {"okhttp-android-util-log"},
	"ondevicepersonalization-plugin-lib":                           {"auto_value_annotations", "auto_value_plugin", "framework-annotations-lib", "framework-res"},
	"opencensus-java-api":                                          {"auto_value_annotations"},
	"pdf_viewer_flags_java_lib":                                    {"aconfig-annotations-lib", "fake_device_config", "unsupportedappusage"},
	"permissions-aconfig-flags-lib":                                {"aconfig-annotations-lib", "framework-configinfrastructure", "unsupportedappusage"},
	"rkpdapp":                                                      {"android.hidl.base-V1.0-java", "android.hidl.manager-V1.0-java", "androidx.room_room-compiler-plugin", "ext", "framework", "framework-annotations-lib", "framework-connectivity", "framework-connectivity-t", "framework-res", "framework-statsd"},
	"safety-center-config":                                         {"framework-annotations-lib", "safety-center-annotations"},
	"safety-center-internal-data":                                  {"safety-center-annotations"},
	"safety-center-pending-intents":                                {"safety-center-annotations"},
	"safety-center-persistence":                                    {"safety-center-annotations"},
	"safety-center-resources-lib":                                  {"safety-center-annotations"},
	"safety-label":                                                 {"framework-annotations-lib"},
	"sdk_sandbox_flags_lib":                                        {"aconfig-annotations-lib", "fake_device_config", "unsupportedappusage"},
	"service-adservices.impl":                                      {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure"},
	"service-appsearch":                                            {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure"},
	"service-art.impl":                                             {"auto_value_annotations", "auto_value_plugin", "ext", "framework", "framework-annotations-lib"},
	"service-bluetooth":                                            {"ext", "framework"},
	"service-bluetooth-binder-aidl":                                {"framework-annotations-lib"},
	"service-bluetooth-pre-jarjar":                                 {"ext", "framework", "framework-annotations-lib", "framework-bluetooth-pre-jarjar", "framework-configinfrastructure", "service-bluetooth.change-ids"},
	"service-configinfrastructure.impl":                            {"ext", "framework", "framework-annotations-lib"},
	"service-connectivity":                                         {"ext", "framework", "framework-annotations-lib", "framework-permission", "framework-permission-s", "framework-wifi", "libprotobuf-java-nano"},
	"service-connectivity-pre-jarjar":                              {"framework-annotations-lib", "framework-configinfrastructure", "framework-connectivity-pre-jarjar", "framework-permission", "framework-permission-s", "framework-statsd", "framework-wifi", "unsupportedappusage"},
	"service-connectivity-protos":                                  {"libprotobuf-java-nano"},
	"service-connectivity-tiramisu-pre-jarjar":                     {"framework-annotations-lib", "framework-configinfrastructure", "framework-connectivity-pre-jarjar", "framework-connectivity-t-pre-jarjar", "framework-wifi", "unsupportedappusage"},
	"service-crashrecovery.impl":                                   {"framework-annotations-lib"},
	"service-devicelock":                                           {"framework-permission", "framework-permission-s"},
	"service-entitlement":                                          {"auto_value_annotations", "auto_value_plugin"},
	"service-entitlement-api":                                      {"auto_value_annotations", "auto_value_plugin"},
	"service-entitlement-data":                                     {"auto_value_annotations", "auto_value_plugin"},
	"service-entitlement-impl":                                     {"auto_value_annotations", "auto_value_plugin"},
	"service-healthfitness.impl":                                   {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure", "framework-sdkextensions", "modules-utils-preconditions", "nullaway_plugin"},
	"service-media-s.impl":                                         {"ext", "framework", "framework-annotations-lib"},
	"service-nearby-pre-jarjar":                                    {"framework-annotations-lib", "framework-bluetooth", "framework-configinfrastructure", "framework-statsd"},
	"service-ondevicepersonalization.impl":                         {"framework-annotations-lib"},
	"service-permission-shared":                                    {"framework-annotations-lib"},
	"service-permission.impl":                                      {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure", "jsr305", "safety-center-annotations"},
	"service-profiling":                                            {"framework-annotations-lib", "framework-configinfrastructure"},
	"service-remoteauth-pre-jarjar":                                {"framework-annotations-lib", "framework-bluetooth", "framework-configinfrastructure", "framework-connectivity-pre-jarjar", "framework-connectivity-t-pre-jarjar", "framework-statsd"},
	"service-rkp.impl":                                             {"framework-annotations-lib"},
	"service-scheduling":                                           {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure", "unsupportedappusage"},
	"service-sdksandbox.impl":                                      {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure"},
	"service-statsd":                                               {"ext", "framework", "framework-annotations-lib", "framework-configinfrastructure"},
	"service-thread-pre-jarjar":                                    {"framework-annotations-lib", "framework-connectivity-pre-jarjar", "framework-connectivity-t-pre-jarjar", "framework-wifi"},
	"service-uwb":                                                  {"ext", "framework"},
	"service-uwb-pre-jarjar":                                       {"framework-annotations-lib", "framework-configinfrastructure", "framework-uwb-pre-jarjar"},
	"service-wifi":                                                 {"auto_value_annotations", "auto_value_plugin", "ext", "framework"},
	"sysuig":                                                       {"vendor-pixelatoms-java"},
	"tensorflowlite_java":                                          {"android-support-annotations"},
	"test_framework-sdkextensions.impl":                            {"framework-annotations-lib"},
	"tetheringstatsprotos":                                         {"ext", "framework"},
	"tflite_support_classifiers_java":                              {"auto_value_plugin"},
	"updatable-media":                                              {"ext", "framework"},
	"uwb_aconfig_flags_lib":                                        {"aconfig-annotations-lib", "ext", "framework", "unsupportedappusage"},
	"uwb_androidx_backend":                                         {"android-support-annotations"},
	"wifi-service-pre-jarjar":                                      {"app-compat-annotations", "auto_value_annotations", "auto_value_plugin", "framework-annotations-lib", "framework-bluetooth", "framework-configinfrastructure", "framework-wifi-pre-jarjar", "jsr305", "unsupportedappusage"},
	"wirelesscharger-adapter":                                      {"vendor-pixelatoms-java"},
}

type StubsAvailableModule interface {
	IsStubsModule() bool
}

var depIsStubsModule = func(_ BottomUpMutatorContext, _, dep Module) bool {
	if stubsModule, ok := dep.(StubsAvailableModule); ok {
		return stubsModule.IsStubsModule()
	}
	return false
}

type HidlStubsAvailableModule interface {
	IsHidlStubsModule() bool
}

var depIsHidlInterfaceStubsModule = func(_ BottomUpMutatorContext, _, dep Module) bool {
	if hidlStubsAvailableModule, ok := dep.(HidlStubsAvailableModule); ok {
		return hidlStubsAvailableModule.IsHidlStubsModule()
	}
	return false
}

type AidlStubsAvailableModule interface {
	IsAidlStubsModule() bool
}

var depIsAidlInterfaceStubsModule = func(_ BottomUpMutatorContext, _, dep Module) bool {
	if AidlStubsAvailableModule, ok := dep.(AidlStubsAvailableModule); ok {
		return AidlStubsAvailableModule.IsAidlStubsModule()
	}
	return false
}

var belongsToCommonApexes = func(_ BottomUpMutatorContext, m, dep Module) bool {
	mContainersInfo, _ := getContainerModuleInfo(m)
	depContainersInfo, _ := getContainerModuleInfo(dep)

	return HasIntersection(mContainersInfo.apexNames, depContainersInfo.apexNames)
}

var belongsToNonUpdatableApex = func(_ BottomUpMutatorContext, m, _ Module) bool {
	mContainersInfo, _ := getContainerModuleInfo(m)
	return !mContainersInfo.updatableApex
}

type SdkLibAndImportModule interface {
	RootLibraryName() string
}

var depIsSdkLibUsesLibDependency = func(ctx BottomUpMutatorContext, m, dep Module) bool {
	if _, ok := dep.(SdkLibAndImportModule); ok {
		depTag := ctx.OtherModuleDependencyTag(dep)
		return reflect.TypeOf(depTag).Name() == "usesLibraryDependencyTag"
	}
	return false
}

var prebuiltSdkLibToSourceSdkImplLibDep = func(_ BottomUpMutatorContext, m, dep Module) bool {
	_, mSdkLibOk := m.(SdkLibAndImportModule)
	depSdkLibSubModule, depSdkLibSubModuleOk := dep.(interface{ SdkLibraryName() *string })
	if mSdkLibOk && depSdkLibSubModuleOk && proptools.String(depSdkLibSubModule.SdkLibraryName()) == m.Name() {
		return dep.Name() == m.Name()+".impl"
	}
	return false
}

// Labels of exception functions, which are used to determine special dependencies that allow
// otherwise restricted inter-container dependencies
type exceptionHandleFuncLabel int

const (
	checkStubs exceptionHandleFuncLabel = iota
	checkHidlInterface
	checkAidlInterface
	checkInCommonApexes
	checkApexIsNonUpdatable
	checkSdkLibUsesLibDep
	checkSdkLibToImplLibDep
	undefined
)

// Functions cannot be used as a value passed in providers, because functions are not
// hashable. As a workaround, the exceptionHandleFunc enum values are passed using providers,
// and the corresponding functions are called from this map.
var exceptionHandleFunctionsTable = map[exceptionHandleFuncLabel]func(BottomUpMutatorContext, Module, Module) bool{
	checkStubs:              depIsStubsModule,
	checkHidlInterface:      depIsHidlInterfaceStubsModule,
	checkAidlInterface:      depIsAidlInterfaceStubsModule,
	checkInCommonApexes:     belongsToCommonApexes,
	checkApexIsNonUpdatable: belongsToNonUpdatableApex,
	checkSdkLibUsesLibDep:   depIsSdkLibUsesLibDependency,
	checkSdkLibToImplLibDep: prebuiltSdkLibToSourceSdkImplLibDep,
	undefined:               func(BottomUpMutatorContext, Module, Module) bool { return false },
}

type InstallableModule interface {
	EnforceApiContainerChecks() bool
}

type restriction struct {
	// container of the dependency
	dependency *container

	// Error message to be emitted to the user when the dependency meets this restriction
	errorMessage string

	// List of labels of allowed exception functions that allows bypassing this restriction.
	// If any of the functions mapped to each labels returns true, this dependency would be
	// considered allowed and an error will not be thrown.
	allowedExceptions []exceptionHandleFuncLabel
}
type container struct {
	// The name of the container i.e. partition, api domain
	name string

	// Map of dependency restricted containers.
	restricted []restriction
}

var (
	VendorContainer = &container{
		name:       VendorVariation,
		restricted: nil,
	}
	SystemContainer = &container{
		name: "system",
		restricted: []restriction{
			{
				dependency: VendorContainer,
				errorMessage: "Module belonging to the system partition other than HALs is " +
					"not allowed to depend on the vendor partition module, in order to support " +
					"independent development/update cycles and to support the Generic System " +
					"Image. Try depending on HALs, VNDK or AIDL instead.",
				allowedExceptions: []exceptionHandleFuncLabel{checkHidlInterface, checkAidlInterface},
			},
		},
	}
	ProductContainer = &container{
		name: ProductVariation,
		restricted: []restriction{
			{
				dependency: VendorContainer,
				errorMessage: "Module belonging to the product partition is not allowed to " +
					"depend on the vendor partition module, as this may lead to security " +
					"vulnerabilities. Try depending on the HALs or utilize AIDL instead.",
				allowedExceptions: []exceptionHandleFuncLabel{checkHidlInterface, checkAidlInterface},
			},
		},
	}
	ApexContainer = initializeApexContainer()
	CtsContainer  = &container{
		name: "cts",
		restricted: []restriction{
			{
				dependency: SystemContainer,
				errorMessage: "CTS module should not depend on the modules belonging to the " +
					"system partition, including \"framework\". Depending on the system " +
					"partition may lead to disclosure of implementation details and regression " +
					"due to API changes across platform versions. Try depending on the stubs instead.",
				allowedExceptions: []exceptionHandleFuncLabel{checkStubs},
			},
		},
	}
)

func initializeApexContainer() *container {
	apexContainer := &container{
		name: "apex",
		restricted: []restriction{
			{
				dependency: SystemContainer,
				errorMessage: "Module belonging to Apex(es) is not allowed to depend on the " +
					"modules belonging to the system partition. Either statically depend on the " +
					"module or convert the depending module to java_sdk_library and depend on " +
					"the stubs.",
				allowedExceptions: []exceptionHandleFuncLabel{checkStubs, checkInCommonApexes,
					checkApexIsNonUpdatable, checkSdkLibUsesLibDep, checkSdkLibToImplLibDep},
			},
		},
	}

	apexContainer.restricted = append(apexContainer.restricted, restriction{
		dependency: apexContainer,
		errorMessage: "Module belonging to Apex(es) is not allowed to depend on the " +
			"modules belonging to other Apex(es). Either include the depending " +
			"module in the Apex or convert the depending module to java_sdk_library " +
			"and depend on its stubs.",
		allowedExceptions: []exceptionHandleFuncLabel{checkStubs, checkInCommonApexes},
	})

	return apexContainer
}

type ContainersInfo struct {
	belongingContainers []*container

	apexNames []string

	updatableApex bool
}

func (c *ContainersInfo) BelongingContainers() []*container {
	return c.belongingContainers
}

func (c *ContainersInfo) ApexNames() []string {
	return c.apexNames
}

func satisfyAllowedExceptions(ctx BottomUpMutatorContext, allowedExceptionLabels []exceptionHandleFuncLabel, m, dep Module) bool {
	for _, label := range allowedExceptionLabels {
		if exceptionHandleFunctionsTable[label](ctx, m, dep) {
			return true
		}
	}
	return false
}

func (c *ContainersInfo) GetViolations(ctx BottomUpMutatorContext, m, dep Module, depInfo ContainersInfo) []string {
	var violations []string

	// Any containers that the module belongs to but the dependency does not belong to must be examined.
	_, containersUniqueToModule, _ := ListSetDifference(c.belongingContainers, depInfo.belongingContainers)

	// Apex container should be examined even if both the module and the dependency belong to
	// the apex container to check that the two modules belong to the same apex.
	if InList(ApexContainer, c.belongingContainers) && !InList(ApexContainer, containersUniqueToModule) {
		containersUniqueToModule = append(containersUniqueToModule, ApexContainer)
	}

	for _, containerUniqueToModule := range containersUniqueToModule {
		for _, restriction := range containerUniqueToModule.restricted {
			if InList(restriction.dependency, depInfo.belongingContainers) {
				if !satisfyAllowedExceptions(ctx, restriction.allowedExceptions, m, dep) {
					violations = append(violations, restriction.errorMessage)
				}
			}
		}
	}

	return violations
}

var ContainersInfoProvider = blueprint.NewMutatorProvider[ContainersInfo]("container_generation")

func RegisterContainerMutator(ctx RegistrationContext) {
	ctx.FinalDepsMutators(registerContainerFinalDepsMutator)
}

func registerContainerFinalDepsMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("container_generation", containerGenerationMutator).Parallel()
	ctx.BottomUp("container_combination", containerCombinationMutator).Parallel()
	ctx.BottomUp("container_enforcement", containerEnforcementMutator).Parallel()
}

// Determines if the module can be installed in the system partition or not.
// Logic is identical to that of modulePartition(...) defined in paths.go
func installInSystemPartition(ctx BottomUpMutatorContext) bool {
	module := ctx.Module()
	return !module.InstallInTestcases() &&
		!module.InstallInData() &&
		!module.InstallInRamdisk() &&
		!module.InstallInVendorRamdisk() &&
		!module.InstallInDebugRamdisk() &&
		!module.InstallInRecovery() &&
		!module.InstallInVendor() &&
		!module.InstallInOdm() &&
		!module.InstallInProduct() &&
		determineModuleKind(module.base(), ctx.blueprintBaseModuleContext()) == platformModule
}

func generateContainerInfo(ctx BottomUpMutatorContext) ContainersInfo {
	inSystem := installInSystemPartition(ctx)
	inProduct := ctx.Module().InstallInProduct()
	inVendor := ctx.Module().InstallInVendor()
	inCts := false
	inApex := false

	if m, ok := ctx.Module().(ImageInterface); ok {
		inProduct = inProduct || m.ProductVariantNeeded(ctx)
		inVendor = inVendor || m.VendorVariantNeeded(ctx)
	}

	props := ctx.Module().GetProperties()
	for _, prop := range props {
		val := reflect.ValueOf(prop).Elem()
		if val.Kind() == reflect.Struct {
			testSuites := val.FieldByName("Test_suites")
			inCts = testSuites.IsValid() && testSuites.Kind() == reflect.Slice && slices.Contains(testSuites.Interface().([]string), "cts")
		}
	}

	var apexNames []string
	var updatableApex bool
	if apexInfo, ok := ModuleProvider(ctx, ApexInfoProvider); ok {
		apexNames = apexInfo.InApexModules
		updatableApex = apexInfo.Updatable
		inApex = true
	}

	containers := []*container{}
	if inSystem {
		containers = append(containers, SystemContainer)
	}
	if inProduct {
		containers = append(containers, ProductContainer)
	}
	if inVendor {
		containers = append(containers, VendorContainer)
	}
	if inCts {
		containers = append(containers, CtsContainer)
	}
	if inApex {
		containers = append(containers, ApexContainer)
	}

	return ContainersInfo{
		belongingContainers: containers,
		apexNames:           apexNames,
		updatableApex:       updatableApex,
	}
}

func containerGenerationMutator(ctx BottomUpMutatorContext) {
	if _, ok := ctx.Module().(InstallableModule); ok {
		SetProvider(ctx, ContainersInfoProvider, generateContainerInfo(ctx))
	}
}

var visitedModuleNames sync.Map

func combineVariantsContainerInfo(ctx BottomUpMutatorContext) ContainersInfo {
	if info, ok := visitedModuleNames.Load(ctx.ModuleName()); ok {
		return info.(ContainersInfo)
	}

	var containersInfo ContainersInfo
	ctx.VisitAllModuleVariants(func(m Module) {
		variantContainersInfo, _ := OtherModuleProvider(ctx, m, ContainersInfoProvider)
		containersInfo.belongingContainers = append(containersInfo.belongingContainers, variantContainersInfo.belongingContainers...)
		containersInfo.apexNames = append(containersInfo.apexNames, variantContainersInfo.apexNames...)
		containersInfo.updatableApex = containersInfo.updatableApex || variantContainersInfo.updatableApex
	})
	containersInfo.belongingContainers = slices.Compact(containersInfo.belongingContainers)
	containersInfo.apexNames = slices.Compact(containersInfo.apexNames)

	visitedModuleNames.Store(ctx.ModuleName(), containersInfo)
	return containersInfo
}

func containerCombinationMutator(ctx BottomUpMutatorContext) {
	if _, ok := ModuleProvider(ctx, ContainersInfoProvider); ok {
		combineVariantsContainerInfo(ctx)
	}
}

func getContainerModuleInfo(module Module) (ContainersInfo, bool) {
	val, ok := visitedModuleNames.Load(module.Name())
	var info ContainersInfo
	if ok {
		info = val.(ContainersInfo)
	}
	return info, ok
}

func containerEnforcementMutator(ctx BottomUpMutatorContext) {
	if containersInfo, ok := getContainerModuleInfo(ctx.Module()); ok {
		ctx.VisitDirectDepsIgnoreBlueprint(func(dep Module) {
			if depContainersInfo, ok := getContainerModuleInfo(dep); ok {
				if allowedViolations, ok := containerDependencyViolationAllowlist[ctx.ModuleName()]; ok {
					if InList(dep.Name(), allowedViolations) {
						return
					}
				} else {
					violations := containersInfo.GetViolations(ctx, ctx.Module(), dep, depContainersInfo)
					if len(violations) > 0 {
						errorMessage := fmt.Sprintf("%s cannot depend on %s. ", ctx.ModuleName(), dep.Name())
						errorMessage += strings.Join(violations, " ")
						ctx.ModuleErrorf(errorMessage)
					}
				}
			}
		})
	}
}
