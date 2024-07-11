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

var containerDependencyViolationAllowlist = map[string][]string{

	"frameworks-base-ddm-unittests": {
		"junit", // cts -> system
	},
	"core-oj": {
		"core-oj-hiddenapi-annotations", // apex -> system
	},
	"art-aconfig-flags-java-lib": {
		"framework-api-annotations-lib", // apex -> system
	},
	"libcore-aconfig-flags-lib": {
		"framework-api-annotations-lib", // apex -> system
	},
	"android.os.profiling.flags-aconfig-java": {
		"aconfig-annotations-lib", // apex -> system
	},
	"okhttp-norepackage": {
		"okhttp-android-util-log", // apex -> system
	},
	"hilt_core": {
		"hilt_define_component_processor",     // apex -> system
		"hilt_generates_root_input_processor", // apex -> system
	},
	"CtsInputMethodServiceCommon": {
		"junit", // cts -> system
	},
	"PlatformProperties": {
		"PlatformProperties_public",     // apex -> system
		"sysprop-library-stub-platform", // apex -> system
	},
	"bluetooth-nano-protos": {
		"libprotobuf-java-nano", // apex -> apex
	},
	"service-connectivity-protos": {
		"libprotobuf-java-nano", // apex -> apex
	},
	"service-entitlement-data": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
	},
	"opencensus-java-api": {
		"auto_value_annotations", // apex -> apex
	},
	"safety-center-resources-lib": {
		"safety-center-annotations", // apex -> system
	},
	"safety-center-persistence": {
		"safety-center-annotations", // apex -> system
	},
	"tensorflowlite_java": {
		"android-support-annotations", // apex -> system
	},
	"cronet_aml_base_base_java": {
		"jsr305", // apex -> apex
	},
	"service-entitlement-impl": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
	},
	"service-entitlement-api": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
	},
	"uwb_androidx_backend": {
		"android-support-annotations", // apex -> system
	},
	"service-entitlement": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
	},
	"safety-center-internal-data": {
		"safety-center-annotations", // apex -> system
	},
	"safety-center-pending-intents": {
		"safety-center-annotations", // apex -> system
	},
	"cronet_aml_net_android_net_java": {
		"jsr305", // apex -> apex
	},
	"grpc-java-core-internal": {
		"gson",             // apex -> system
		"perfmark-api-lib", // apex -> system
	},
	"tflite_support_classifiers_java": {
		"auto_value_plugin", // apex -> system
	},
	"cronet_aml_components_cronet_android_cronet_impl_native_java": {
		"jsr305", // apex -> apex
	},
	"androidx.compose.material_material-ripple": {
		"androidx.compose.compiler_compiler-hosted", // apex -> system
	},
	"sdk_sandbox_flags_lib": {
		"aconfig-annotations-lib", // apex -> system
		"fake_device_config",      // apex -> system
	},
	"pdf_viewer_flags_java_lib": {
		"aconfig-annotations-lib", // apex -> system
		"fake_device_config",      // apex -> system
	},
	"android.net.wifi.flags-aconfig-java": {
		"aconfig-annotations-lib", // apex -> system
		"fake_device_config",      // apex -> system
	},
	"media_mainline_flags_java_lib": {
		"aconfig-annotations-lib", // apex -> system
		"fake_device_config",      // apex -> system
	},
	"android.permission.flags-aconfig-java": {
		"aconfig-annotations-lib", // apex -> system
		"fake_device_config",      // apex -> system
	},
	"android.service.notification.flags-aconfig-export-java": {
		"aconfig-annotations-lib", // apex -> system
		"fake_device_config",      // apex -> system
	},
	"bluetooth.change-ids": {
		"app-compat-annotations", // apex -> system
	},
	"framework-pdf.impl": {
		"modules-utils-preconditions", // apex -> apex
	},
	"framework-adservices.impl": {
		"adservices_flags_lib", // apex -> system
	},
	"bluetooth_flags_java_lib": {
		"aconfig-annotations-lib", // apex -> system
	},
	"permissions-aconfig-flags-lib": {
		"aconfig-annotations-lib", // apex -> system
	},
	"framework-permission-s.impl": {
		"app-compat-annotations", // apex -> system
	},
	"adservices-shared-error-logging": {
		"auto_annotation_plugin", // apex -> system
		"auto_value_plugin",      // apex -> system
	},
	"framework-healthfitness.impl": {
		"nullaway_plugin", // apex -> system
	},
	"adservices-shared-spe": {
		"auto_annotation_plugin", // apex -> system
		"auto_value_plugin",      // apex -> system
	},
	"framework-ondevicepersonalization.impl": {
		"ondevicepersonalization_flags_lib",   // apex -> system
		"staledataclass-annotation-processor", // apex -> system
	},
	"httpclient_impl": {
		"httpclient_api", // apex -> system
	},
	"safety-center-config": {
		"safety-center-annotations", // apex -> system
	},
	"net-utils-device-common-struct": {
		"net-utils-device-common-struct-base", // apex -> system
	},
	"connectivity-net-module-utils-bpf": {
		"net-utils-device-common-struct-base", // apex -> system
	},
	"net-utils-device-common-netlink": {
		"net-utils-device-common-struct-base", // apex -> system
	},
	"framework-wifi.impl": {
		"aconfig-annotations-lib", // apex -> system
		"app-compat-annotations",  // apex -> system
	},
	"framework-bluetooth.impl": {
		"app-compat-annotations", // apex -> system
	},
	"hilt_android": {
		"dagger2-compiler",                    // apex -> system
		"hilt_aggregated_deps_processor",      // apex -> system
		"hilt_alias_of_processor",             // apex -> system
		"hilt_android_entry_point_processor",  // apex -> system
		"hilt_component_tree_deps_processor",  // apex -> system
		"hilt_define_component_processor",     // apex -> system
		"hilt_early_entry_point_processor",    // apex -> system
		"hilt_generates_root_input_processor", // apex -> system
		"hilt_originating_element_processor",  // apex -> system
		"hilt_root_processor",                 // apex -> system
		"hilt_viewmodel_processor",            // apex -> system
	},
	"SettingsLibSettingsTransition": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibHelpUtils": {
		"SettingsLibLintChecker", // apex -> system
	},
	"adservices-cobalt": {
		"androidx.room_room-compiler-plugin", // apex -> system
		"auto_annotation_plugin",             // apex -> system
		"auto_value_plugin",                  // apex -> system
	},
	"android_downloader_lib": {
		"auto_value_plugin", // apex -> system
	},
	"SettingsLibRestrictedLockUtils": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibUtils": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibActionBarShadow": {
		"SettingsLibLintChecker", // apex -> system
	},
	"HealthConnectBackupRestoreLibrary": {
		"nullaway_plugin", // apex -> system
	},
	"ondevicepersonalization-plugin-lib": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
	},
	"PhotopickerLib": {
		"androidx.compose.compiler_compiler-hosted", // apex -> system
	},
	"TextClassifierServiceLibNoManifest": {
		"androidx.room_room-compiler-plugin", // apex -> system
		"auto_value_plugin",                  // apex -> system
	},
	"mobile_data_downloader_lib": {
		"auto_annotation_plugin", // apex -> system
		"auto_value_plugin",      // apex -> system
		"dagger2-compiler",       // apex -> system
	},
	"adservices-service-core": {
		"BackCompatLintChecker",                        // apex -> system
		"androidx.appsearch_appsearch-compiler-plugin", // apex -> system
		"androidx.room_room-compiler-plugin",           // apex -> system
		"auto_annotation_plugin",                       // apex -> system
		"auto_oneof_plugin",                            // apex -> system
		"auto_value_plugin",                            // apex -> system
	},
	"updatable-media": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-art.impl": {
		"aconfig-annotations-lib", // apex -> system
		"auto_value_annotations",  // apex -> apex
		"auto_value_plugin",       // apex -> system
		"ext",                     // apex -> system
		"framework",               // apex -> system
	},
	"framework-appsearch.impl": {
		"ext",                             // apex -> system
		"framework",                       // apex -> system
		"safeparcel-annotation-processor", // apex -> system
	},
	"mediaprovider_flags_java_lib": {
		"aconfig-annotations-lib", // apex -> system
		"ext",                     // apex -> system
		"framework",               // apex -> system
	},
	"service-sdksandbox.impl": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-bluetooth-pre-jarjar": {
		"ext",                            // apex -> system
		"framework",                      // apex -> system
		"framework-bluetooth-pre-jarjar", // apex -> system
		"service-bluetooth.change-ids",   // apex -> system
	},
	"service-healthfitness.impl": {
		"ext",                         // apex -> system
		"framework",                   // apex -> system
		"modules-utils-preconditions", // apex -> apex
		"nullaway_plugin",             // apex -> system
	},
	"device_config_reboot_flags_java_lib": {
		"aconfig-annotations-lib", // apex -> system
		"ext",                     // apex -> system
		"framework",               // apex -> system
	},
	"tetheringstatsprotos": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-permission.impl": {
		"ext",                       // apex -> system
		"framework",                 // apex -> system
		"jsr305",                    // apex -> apex
		"safety-center-annotations", // apex -> system
	},
	"uwb_aconfig_flags_lib": {
		"aconfig-annotations-lib", // apex -> system
		"ext",                     // apex -> system
		"framework",               // apex -> system
	},
	"service-media-s.impl": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-scheduling": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-statsd": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-adservices.impl": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-bluetooth": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"service-appsearch": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"wirelesscharger-adapter": {
		"vendor-pixelatoms-java", // system -> vendor
	},
	"PersistentBackgroundServices-common-no-concurrency-modules": {
		"vendor-pixelatoms-java", // system -> vendor
	},
	"framework-connectivity.impl": {
		"app-compat-annotations", // apex -> system
	},
	"framework-connectivity-t.impl": {
		"app-compat-annotations",            // apex -> system
		"framework-connectivity-pre-jarjar", // apex -> system
	},
	"SettingsLibBarChartPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibIllustrationPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"service-remoteauth-pre-jarjar": {
		"framework-connectivity-pre-jarjar",   // apex -> system
		"framework-connectivity-t-pre-jarjar", // apex -> system
	},
	"loadlibrarytest_testlib": {
		"libnativeloader_vendor_shared_lib", // system -> vendor
	},
	"SettingsLibSettingsSpinner": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibProfileSelector": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibTwoTargetPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibMainSwitchPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibAppPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibFooterPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibCollapsingToolbarBaseActivity": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibTopIntroPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibSelectorWithWidgetPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"SettingsLibLayoutPreference": {
		"SettingsLibLintChecker", // apex -> system
	},
	"android.car-module.impl": {
		"modules-utils-preconditions", // apex -> apex
	},
	"com.google.android.camera.support.v23.experimental": {
		"com.google.android.camera.experimental2018",         // system -> vendor
		"com.google.android.camera.experimental2019",         // system -> vendor
		"com.google.android.camera.experimental2020",         // system -> vendor
		"com.google.android.camera.experimental2020_midyear", // system -> vendor
		"com.google.android.camera.experimental2021",         // system -> vendor
		"com.google.android.camera.experimental2022",         // system -> vendor
		"com.google.android.camera.experimental2023",         // system -> vendor
		"com.google.android.camera.experimental2024",         // system -> vendor
	},
	"TetheringApiStableLib": {
		"connectivity-internal-api-util", // apex -> system
	},
	"AdServicesLib": {
		"BackCompatLintChecker", // apex -> system
	},
	"TetheringApiCurrentLib": {
		"connectivity-internal-api-util", // apex -> system
	},
	"v3-ec-p256-1-companion-usesperm": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-rsa-2048-declperm": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSignatureQueryService_v3-tgt-33": {
		"cts_signature_query_service", // cts -> system
	},
	"WifiDialog": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"OsuLogin": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsDynamicMimeUpdateAppBothGroups": {
		"CtsDynamicMimeCommon", // cts -> system
	},
	"CtsSkQPTestCases": {
		"android-support-design", // cts -> system
		"ctstestrunner-axt",      // cts -> system
	},
	"CtsDeviceServicesTestSecondApp": {
		"cts-wm-overlayapp-base", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingHasPermission": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesUnexportedActivityViaAction": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsRequiredSplitTypeSplitApp": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSignatureQueryServiceTest_v2": {
		"cts_signature_query_service", // cts -> system
	},
	"hotspot": {
		"android-support-v4", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-default-caps-sharedUid-companion": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppInBackground": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSignatureQueryService_v3": {
		"cts_signature_query_service", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingReceivesPermissionProtectedUri": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsMultiUserStorageApp": {
		"CtsExternalStorageTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesUnexportedProviderViaAction": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-companion-uses-knownSigner": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"rkpdapp": {
		"androidx.room_room-compiler-plugin", // apex -> system
		"ext",                                // apex -> system
		"framework",                          // apex -> system
	},
	"v3-ec-p256-with-por_1_2_4-companion-usesperm": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-no-shUid-cap-declperm2": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSignatureQueryService_v2-tgt-33": {
		"cts_signature_query_service", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-no-shUid-cap-sharedUid": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-no-shUid-cap-sharedUid-companion2": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-rsa-2048-decl-knownSigner-str-const-ec-p256-1": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-no-perm-cap-sharedUid": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSignatureQueryService_v2": {
		"cts_signature_query_service", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-no-shUid-cap-sharedUid-companion": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-por_Y_1_2-default-caps-sharedUid": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-por_Z_1_2-default-caps-sharedUid-companion": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsIsolatedSplitApp": {
		"ctstestrunner-axt", // cts -> system
		"truth",             // cts -> system
	},
	"CtsPkgInstallerPermRequestApp": {
		"CtsPkgInstallerConstants", // cts -> system
		"truth",                    // cts -> system
	},
	"v3-rsa-2048-decl-knownSigner-ec-p256-1-3": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-rsa-2048-decl-knownSigner-str-res-ec-p256-1": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsLocationPolicyApp": {
		"telephony-common", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-default-caps-sharedUid-companion3": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"ExtServices-tplus": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsDevicePrereleaseSdkApp": {
		"fake-framework", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsTrueNumberProviderA": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsProtoTestCases": {
		"ctstestrunner-axt",     // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
		"libprotobuf-java-nano", // cts -> system
	},
	"CtsAudioHostTestApp": {
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"junit",             // cts -> system
		"truth",             // cts -> system
	},
	"CtsMediaSessionTestHelper": {
		"CtsMediaHostTestCommon", // cts -> system
	},
	"SdkSandbox": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"ServiceWifiResources": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsSyncAccountAccessStubs": {
		"android-support-annotations", // cts -> system
	},
	"CtsIntentReceiverApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsIntentSenderApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"truth",                             // cts -> system
	},
	"CtsLauncherAppsTestsSupport": {
		"junit", // cts -> system
	},
	"Bluetooth": {
		"BluetoothLintChecker",               // apex -> system
		"androidx.room_room-compiler-plugin", // apex -> system
		"app-compat-annotations",             // apex -> system
		"error_prone_android_framework",      // apex -> system
		"ext",                                // apex -> system
		"framework",                          // apex -> system
		"framework-bluetooth-pre-jarjar",     // apex -> system
	},
	"SettingsLibActivityEmbedding": {
		"SettingsLibLintChecker", // apex -> system
	},
	"CtsBackupTransportApp": {
		"testng", // cts -> system
		"truth",  // cts -> system
	},
	"CtsAngleNativeDriverCheck": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsPackageInstallerApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
	},
	"CtsDynamicMimeHelperApp": {
		"CtsDynamicMimeCommon", // cts -> system
	},
	"CtsDynamicMimeUpdateAppFirstGroup": {
		"CtsDynamicMimeCommon", // cts -> system
	},
	"CtsDynamicMimeUpdateAppSecondGroup": {
		"CtsDynamicMimeCommon", // cts -> system
	},
	"CtsFullBackupContentApp": {
		"CtsFullBackupRulesLib", // cts -> system
	},
	"MicrodroidTestAppUpdated": {
		"MicrodroidDeviceTestHelper", // cts -> system
		"VmAttestationTestUtil",      // cts -> system
		"androidx.test.runner",       // cts -> system
		"cbor-java",                  // cts -> system
		"com.android.microdroid.test.vmshare_service-java", // cts -> system
		"compatibility-common-util-devicesidelib",          // cts -> system
		"ext",       // cts -> system
		"framework", // cts -> system
		"truth",     // cts -> system
	},
	"CtsAccountManagementDevicePolicyApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"MicrodroidVmShareApp": {
		"com.android.microdroid.test.vmshare_service-java", // cts -> system
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CellBroadcastServiceModule": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"StatusCheckerApp": {
		"StatusCheckerApp_Secondary", // cts -> system
		"ctstestrunner-axt",          // cts -> system
	},
	"AppUsingOtherApp": {
		"ctstestrunner-axt", // cts -> system
	},
	"ServiceUwbResources": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"AudioRecorderTestApp_MediaRecorder": {
		"AudioRecorderTestApp_Base", // cts -> system
	},
	"CtsUsePermissionDiffCert": {
		"CtsPermissionDeclareUtilLib", // cts -> system
		"testng",                      // cts -> system
		"truth",                       // cts -> system
	},
	"CtsStatsSecurityApp": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppComponentFactoryTestCases": {
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
	},
	"CtsAppEnumerationQueriesNothing": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAccountManagerCrossUserApp": {
		"truth", // cts -> system
	},
	"CtsAppEnumerationQueriesServiceViaAction": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"com.replica.replicaisland": {
		"ext",       // cts -> system
		"framework", // cts -> system
		"junit",     // cts -> system
	},
	"CtsAppEnumerationWildcardDocumentEditorActivitySource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationForceQueryableNormalInstall": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsListeningPortsTest": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSignatureQueryService": {
		"cts_signature_query_service", // cts -> system
	},
	"CtsSignatureQueryServiceTest": {
		"cts_signature_query_service", // cts -> system
	},
	"CtsNdefTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsProfileKeyValueApp": {
		"truth", // cts -> system
	},
	"CtsMimeMapTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"mimemap-testing",   // cts -> system
	},
	"CtsDynamicMimePreferredApp": {
		"CtsDynamicMimeCommon", // cts -> system
	},
	"CtsSignatureQueryServiceTest_v2-tgt-33": {
		"cts_signature_query_service", // cts -> system
	},
	"HealthConnectBackupRestore": {
		"error_prone_android_framework", // apex -> system
		"ext",                           // apex -> system
		"framework",                     // apex -> system
	},
	"CtsNativeMediaAAudioTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"nativetesthelper",  // cts -> system
	},
	"IncrementalTestAppUncompressed": {
		"incremental-install-common-lib", // cts -> system
	},
	"FullBackupOnlyTrueWithAgentApp": {
		"CtsFullBackupOnlyLib", // cts -> system
	},
	"v3-ec-p256-with-por-1_2_3_4_5-default-caps": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-1-sharedUid": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-1-sharedUid-companion2": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256_2-companion-uses-knownSigner": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-2-sharedUid-companion": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256_3-companion-uses-knownSigner": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-with-por_1_2_3-1-no-caps-2-default-declperm": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsDeviceServicesTestThirdApp": {
		"cts-wm-overlayapp-base", // cts -> system
	},
	"DeviceConfigServiceResources": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsAppEnumerationWildcardBrowsableActivitySource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"v3-ec-p256-with-por_1_2_3-no-caps-declperm": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-default-caps": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"v3-ec-p256-with-por_1_2-default-caps-sharedUid": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsRestoreSessionApp": {
		"truth", // cts -> system
	},
	"CtsAppEnumerationWildcardBrowserActivitySource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsWindowManagerExternalApp": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"IncrementalTestApp": {
		"incremental-install-common-lib", // cts -> system
	},
	"CtsAngleDriverTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"SimpleSmsApp": {
		"cts-devicepolicy-suspensionchecker", // cts -> system
	},
	"TelephonyProviderDeviceTest": {
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
		"truth",             // cts -> system
	},
	"TestIme": {
		"cts-devicepolicy-suspensionchecker", // cts -> system
	},
	"CtsAppToBlame1": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsHostsideNetworkTestsApp2": {
		"NetworkStackApiStableShims", // cts -> system
	},
	"ServiceConnectivityResources": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"devicelockcontroller-lib": {
		"modules-utils-expresslog", // apex -> apex
	},
	"CtsZipValidateApp": {
		"ctstestrunner-axt",           // cts -> system
		"ext",                         // cts -> system
		"framework",                   // cts -> system
		"junit",                       // cts -> system
		"mockito-target-minus-junit4", // cts -> system
		"truth",                       // cts -> system
		"zip_validation_test_jar",     // cts -> system
	},
	"CtsInvalidRequiredSplitTypeSplitApp": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppThatRequestsSystemAlertWindow22": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsDynamicLinkerTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"SharedLibraryInfoTestApp": {
		"guava",               // cts -> system
		"modules-utils-build", // cts -> system
	},
	"loadlibrarytest_product_app": {
		"libnativeloader_vendor_shared_lib", // product -> vendor
	},
	"CtsAppEnumerationStubSharedUser": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAngleDriverTestCasesSecondary": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsInlineMockingTestCases": {
		"android-support-v4",              // cts -> system
		"ctstestrunner-axt",               // cts -> system
		"dexmaker-inline-mockmaker-tests", // cts -> system
		"dexmaker-mockmaker-tests",        // cts -> system
		"mockito-target-inline",           // cts -> system
	},
	"CtsAppEnumerationWildcardShareActivitySource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsToastLegacyTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"MediaProvider": {
		"app-compat-annotations",     // apex -> system
		"error_prone_mediaprovider",  // apex -> system
		"ext",                        // apex -> system
		"framework",                  // apex -> system
		"glide-annotation-processor", // apex -> system
	},
	"OnDevicePersonalization": {
		"ext",                                 // apex -> system
		"framework",                           // apex -> system
		"staledataclass-annotation-processor", // apex -> system
	},
	"FederatedCompute": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
		"ext",                    // apex -> system
		"framework",              // apex -> system
	},
	"SafetyCenterResources": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsAppEnumerationFilters": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsViewTestCasesSdk28": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAppEnumerationWildcardWebActivitySource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsThemeDeviceTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"ApkWithoutLocaleConfig": {
		"androidx.legacy_legacy-support-v4", // cts -> system
	},
	"ApkWithLocaleConfig": {
		"androidx.legacy_legacy-support-v4", // cts -> system
	},
	"ApkRemoveAppLocalesInLocaleConfig": {
		"androidx.legacy_legacy-support-v4", // cts -> system
	},
	"CtsAppEnumerationQueriesUnexportedServiceViaAction": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"AudioRecorderTestApp_AudioRecord": {
		"AudioRecorderTestApp_Base", // cts -> system
	},
	"GoogleSafetyCenterResources": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"appcompat_preinstall_override2_versioncode1_release": {
		"pre_install_override_lib", // cts -> system
	},
	"AppWithLongAttributionTag": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppWithAttributionInheritingFromSameAsOther": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppThatCanBeForcedIntoForegroundStates": {
		"AppOpsForegroundControlServiceAidl", // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
	},
	"AppWithDuplicateAttribution": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppWithTooManyAttributions": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppWithReceiverAttribution": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppForDiscreteTest": {
		"AppOpsForegroundControlServiceAidl", // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
	},
	"CtsAppToCollect": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppToBlame2": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppWithAttributionInheritingFromSelf": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppWithAttributionInheritingFromExisting": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"AppInstalledOnMultipleUsers": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"DebuggableApp": {
		"CtsAllowBackupLib", // cts -> system
	},
	"CtsUseMicOrCameraForSensorPrivacy": {
		"SensorPrivacyTestAppUtils", // cts -> system
	},
	"CtsDataExtractionRulesApp": {
		"CtsFullBackupRulesLib", // cts -> system
	},
	"CtsAppEnumerationSyncadapterTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationSharedUidTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsExternalPrintService": {
		"junit", // cts -> system
	},
	"CtsAppEnumerationWildcardContactsActivitySource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationWebActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsFgsStartTestHelperApi34": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsFgsStartTestHelperCurrent": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsEffectTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsInstantAppTests": {
		"ext",                         // cts -> system
		"framework",                   // cts -> system
		"mockito-target-minus-junit4", // cts -> system
		"truth",                       // cts -> system
	},
	"CtsEncryptionAttributeApp": {
		"CtsFullBackupRulesLib", // cts -> system
	},
	"CtsLibcoreApiEvolutionTestCases": {
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
	},
	"CtsSelinuxQCompatApp": {
		"selinux_app_empty", // cts -> system
	},
	"CtsNoAppDataStorageApp": {
		"androidx.test.runner", // cts -> system
		"truth",                // cts -> system
	},
	"CtsSaxTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAppEnumerationForceQueryable": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesPackageHasProvider": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsStorageEscalationApp28": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppEnumerationBrowserActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsHostsideNumberBlockingAppTest": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAdminApp": {
		"guava", // cts -> system
	},
	"appcompat_preinstall_override_versioncode1_debuggable": {
		"pre_install_override_lib", // cts -> system
	},
	"CtsColorModeTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAppEnumerationAppWidgetProviderSharedUidTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesProviderViaAuthority": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsUidIsolationTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
	},
	"CtsHostsideCompatChangeTestsApp": {
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"junit",             // cts -> system
		"testng",            // cts -> system
		"truth",             // cts -> system
	},
	"CameraExtensionsProxy": {
		"androidx.camera.extensions.impl", // system -> vendor
	},
	"CtsRenderscriptLegacyTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingUsesOptionalLibrary": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsLibcoreFileIOTestCases": {
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
	},
	"DeviceDropMonitor": {
		"vendor-pixelatoms-java", // system -> vendor
	},
	"CtsAppEnumerationQueriesNothingReceivesNonPersistableUri": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsToastTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsArtTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsFalseNumberProviderB": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CarServiceUpdatable": {
		"modules-utils-os",                    // apex -> apex
		"modules-utils-preconditions",         // apex -> apex
		"modules-utils-shell-command-handler", // apex -> apex
	},
	"IncrementalTestApp2_v2": {
		"incremental-install-common-lib", // cts -> system
	},
	"IncrementalTestApp2_v1": {
		"incremental-install-common-lib", // cts -> system
	},
	"CtsExternalStorageApp": {
		"CtsExternalStorageTestLib", // cts -> system
	},
	"appcompat_preinstall_override_versioncode2_debuggable": {
		"pre_install_override_lib", // cts -> system
	},
	"EmulatorBootSanityTester": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
	},
	"BackupNotAllowedApp": {
		"CtsAllowBackupLib", // cts -> system
	},
	"FullBackupOnlyFalseWithAgentApp": {
		"CtsFullBackupOnlyLib", // cts -> system
	},
	"CtsRsBlasTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsBackupRestoreAnyVersionApp": {
		"CtsRestoreAnyVersionLib", // cts -> system
	},
	"CtsSplitApp_number_provider_a": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsStorageEscalationApp29Full": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSharedUserMigrationDataTestApp1": {
		"CtsSharedUserMigrationTestLibs", // cts -> system
	},
	"appcompat_preinstall_override_versioncode1_release": {
		"pre_install_override_lib", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingHasProvider": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsUseMicOrCameraAndOverlayForSensorPrivacy": {
		"SensorPrivacyTestAppUtils", // cts -> system
	},
	"CtsDataExtractionRulesApplicabilityApp": {
		"CtsFullBackupRulesLib", // cts -> system
	},
	"CtsSplitApp_number_proxy": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingUsesLibrary": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationSyncadapterSharedUidTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationBrowserWildcardActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationPrefixWildcardWebActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationPreferredActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingTargetsQ": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationWildcardActionSource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingSeesInstaller": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationShareActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationStub": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationContactsActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingReceivesUri": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsNdkBinderTestCases": {
		"ctstestrunner-axt", // cts -> system
		"nativetesthelper",  // cts -> system
	},
	"NonDebuggableApp": {
		"CtsAllowBackupLib", // cts -> system
	},
	"CtsAppEnumerationQueriesUnexportedProviderViaAuthority": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationDocumentsActivityTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsMicrodroidDisabledTestCases": {
		"androidx.test.runner",                    // cts -> system
		"compatibility-common-util-devicesidelib", // cts -> system
		"truth", // cts -> system
	},
	"CtsAppEnumerationQueriesProviderViaAction": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationAppWidgetProviderTarget": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationSharedUidSource": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesActivityViaAction": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesPackage": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationQueriesNothingReceivesPersistableUri": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsAppEnumerationNoApi": {
		"CtsAppEnumerationTestLib", // cts -> system
	},
	"CtsContextCrossProfileApp": {
		"truth", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsFalseJni": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsDpiTestCases2": {
		"android.cts.dpi",   // cts -> system
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
	},
	"CtsReadExternalStorageApp": {
		"CtsExternalStorageTestLib", // cts -> system
	},
	"CtsRequiredSplitTypeSplitAppUpdated": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsPkgInstallTinyApp": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsFalseNumberProviderA": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsFalseNumberProxy": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsTrueJni": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsTrue": {
		"ctstestrunner-axt", // cts -> system
		"truth",             // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsTrueNumberProviderB": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsIsolatedSplitAppExtractNativeLibsTrueNumberProxy": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSplitApp_number_provider_b": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsBackupRestoreAnyVersionAppUpdate": {
		"CtsRestoreAnyVersionLib", // cts -> system
	},
	"CtsStorageEscalationApp29Scoped": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsSelinuxRCompatApp": {
		"selinux_app_empty", // cts -> system
	},
	"CtsUnaffiliatedAccountAuthenticators": {
		"CtsAccountTestsCommon", // cts -> system
		"ctstestrunner-axt",     // cts -> system
	},
	"CtsMockingDebuggableTestCases": {
		"ctstestrunner-axt",        // cts -> system
		"dexmaker-mockmaker-tests", // cts -> system
		"mockito-target",           // cts -> system
	},
	"CtsMockingTestCases": {
		"ctstestrunner-axt",        // cts -> system
		"dexmaker-mockmaker-tests", // cts -> system
		"mockito-target",           // cts -> system
	},
	"FullBackupOnlyFalseNoAgentApp": {
		"CtsFullBackupOnlyLib", // cts -> system
	},
	"CtsRsCppTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAssistApp": {
		"CtsAssistCommon", // cts -> system
	},
	"CtsHostsideNetworkPolicyTestsApp2": {
		"ext",                 // cts -> system
		"framework",           // cts -> system
		"modules-utils-build", // cts -> system
	},
	"DevCameraGoogle": {
		"com.google.android.camera.experimental2018",         // system -> vendor
		"com.google.android.camera.experimental2019",         // system -> vendor
		"com.google.android.camera.experimental2020",         // system -> vendor
		"com.google.android.camera.experimental2020_midyear", // system -> vendor
		"com.google.android.camera.experimental2021",         // system -> vendor
		"com.google.android.camera.experimental2022",         // system -> vendor
		"com.google.android.camera.experimental2023",         // system -> vendor
		"com.google.android.camera.experimental2024",         // system -> vendor
	},
	"CtsSharedUserMigrationDataTestApp2": {
		"CtsSharedUserMigrationTestLibs", // cts -> system
	},
	"EmulatorCtsVerifierAutomator": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
	},
	"CVE-2021-0965": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsLibcoreLegacy22TestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsBackupRestoreAnyVersionNoRestoreApp": {
		"CtsRestoreAnyVersionLib", // cts -> system
	},
	"CtsAppThatRequestsSystemAlertWindow23": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsJvmtiAttachingTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsPkgInstallTinyAppV2": {
		"ext",       // cts -> system
		"framework", // cts -> system
	},
	"CtsAppRestrictionsTargetApp": {
		"DpmWrapper", // cts -> system
	},
	"CtsSdkExtensionsTestCases": {
		"ctstestrunner-axt",         // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
		"modules-utils-build",       // cts -> system
		"test_util_current_version", // cts -> system
		"test_util_derive_sdk",      // cts -> system
		"truth",                     // cts -> system
	},
	"CtsViewReceiveContentTestCases": {
		"android-common",              // cts -> system
		"ctstestrunner-axt",           // cts -> system
		"ext",                         // cts -> system
		"framework",                   // cts -> system
		"metrics-helper-lib",          // cts -> system
		"mockito-target-minus-junit4", // cts -> system
		"truth",                       // cts -> system
	},
	"CtsProfileFullBackupApp": {
		"truth", // cts -> system
	},
	"CtsSyncInvalidAccountAuthorityTestCases": {
		"android-support-annotations", // cts -> system
		"ctstestrunner-axt",           // cts -> system
	},
	"CtsRebootReadinessTestCases": {
		"ext",       // cts -> system
		"framework", // cts -> system
		"truth",     // cts -> system
	},
	"BackupAllowedApp": {
		"CtsAllowBackupLib", // cts -> system
	},
	"CtsPreconditions": {
		"compatibility-device-preconditions", // cts -> system
	},
	"CtsNetTestCasesLegacyPermission22": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsCalendarcommon2TestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsBionicAppTestCases": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsResolverServiceTestCases": {
		"ctstestrunner-axt", // cts -> system
		"truth",             // cts -> system
	},
	"CtsExtendedMockingTestCases": {
		"android-support-v4",                // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"dexmaker-extended-mockmaker-tests", // cts -> system
		"dexmaker-inline-mockmaker-tests",   // cts -> system
		"dexmaker-mockmaker-tests",          // cts -> system
		"mockito-target-extended",           // cts -> system
	},
	"MicrodroidTestAppNoPerm": {
		"MicrodroidDeviceTestHelper",                       // cts -> system
		"MicrodroidTestHelper",                             // cts -> system
		"androidx.test.runner",                             // cts -> system
		"com.android.microdroid.test.vmshare_service-java", // cts -> system
		"compatibility-common-util-devicesidelib",          // cts -> system
		"truth", // cts -> system
	},
	"appcompat_preinstall_override_versioncode2_release": {
		"pre_install_override_lib", // cts -> system
	},
	"CtsPermissionTestCasesSdk28": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsGestureTestCases": {
		"androidx.test.runner", // cts -> system
	},
	"AdServicesApk": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"NotificationTrampolineApi30": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationTrampoline": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationTrampolineApi32": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure00": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure01": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure02": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure03": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure04": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure05": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure06": {
		"NotificationTrampolineBase", // cts -> system
	},
	"NotificationPressure07": {
		"NotificationTrampolineBase", // cts -> system
	},
	"MainlineModuleDetector": {
		"compatibility-device-util-axt", // cts -> system
	},
	"MockSatelliteServiceApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"telephony-common",              // cts -> system
	},
	"CtsErrorsApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsAppDataIsolationAppB": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsOverlayTarget": {
		"compatibility-device-util-axt", // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCrossProfileNotEnabledApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsAccessSerialModern": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsAccessDeviceIdentifiers": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsPkgInstallerPermWhitelistApp": {
		"CtsPkgInstallerConstants",      // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
	},
	"CtsAppDataIsolationAppSharedA": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppDataIsolationAppDirectBootA": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppDataIsolationAppApi29A": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppDataIsolationAppA": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppDataIsolationAppSharedB": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsTvProviderTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDeviceAdminServiceB": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsMediaExtractorHostTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsMediaMetricsHostTestApp": {
		"CtsMediaHostTestCommon",        // cts -> system
		"collector-device-lib",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"cts-midi-lib",                  // cts -> system
		"truth",                         // cts -> system
	},
	"CtsMediaRouterHostSideTestBluetoothPermissionsApp": {
		"CtsMediaHostTestCommon",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsMediaRouterHostSideTestModifyAudioRoutingApp": {
		"CtsMediaHostTestCommon",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsMediaRouterHostSideTestMediaRoutingControlApp": {
		"CtsMediaHostTestCommon",                              // cts -> system
		"com.android.media.flags.bettertogether-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                       // cts -> system
		"ext",                                                 // cts -> system
		"flag-junit",                                          // cts -> system
		"framework",                                           // cts -> system
	},
	"CtsMediaSessionHostTestApp": {
		"CtsMediaHostTestCommon",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsProxyMediaRouterSecondaryUserTestHelperApp": {
		"CtsMediaHostTestCommon",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsProxyMediaRouterTestHelperApp": {
		"CtsMediaHostTestCommon",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"TestApp1": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsCorpOwnedManagedProfile": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsCrossProfileEnabledApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsCrossProfileUserEnabledApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsDeviceAdminApp23": {
		"DpmWrapper",                    // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsCorpOwnedManagedProfile2": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsDeviceAdminApp24": {
		"DpmWrapper",                    // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDeviceAdminApp29": {
		"DpmWrapper",                    // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDeviceAdminService2": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDeviceAdminService4": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDeviceAdminService1": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsTransferOwnerIncomingApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
		"testng",                            // cts -> system
	},
	"CtsTransferOwnerOutgoingApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
		"testng",                            // cts -> system
	},
	"CtsDevicePolicyAssistApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
	},
	"CtsDeviceAdminService3": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsWifiConfigCreator": {
		"compatibility-device-util-axt", // cts -> system
	},
	"SharingApp1": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"SharingApp2": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"TestApp2": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"TestApp3": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"TestApp4": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsDynamicMimeTestApp": {
		"CtsDynamicMimeCommon",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
	},
	"CtsJobSchedulerSharedUid": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsPreservedSettingsApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAbiOverrideTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsKeyValueBackupRestoreApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsInstalledLoadingProgressTestsApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsInstalledLoadingProgressDeviceTests": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsInstalledLoadingProgressTestsRegisterApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"NonUiInCallServiceWoExport": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsWrapWrapDebugMallocDebugTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_tests_tests_wrap_src",      // cts -> system
	},
	"CtsAutoRestoreApp": {
		"compatibility-device-util-axt", // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsMediaBitstreamsDeviceSideTestApp": {
		"compatibility-device-util-axt",         // cts -> system
		"media-bitstreams-common-devicesidelib", // cts -> system
	},
	"CtsUseEmbeddedDexApp_Canonical": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsUseEmbeddedDexApp_Canonical_PerProcess": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsUseEmbeddedDexApp_DexCompressed_PerProcess": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsUseEmbeddedDexApp_NotPreferred": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsUseEmbeddedDexApp_DexCompressed": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsUseEmbeddedDexAppSplit_Canonical": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsUseEmbeddedDexAppSplit_CompressedDex": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAdServicesCobaltTest": {
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-adservices-lib",      // cts -> system
		"truth",                         // cts -> system
	},
	"CtsDeviceConfigTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"guava",                         // cts -> system
	},
	"CtsEmptyDeviceOwner": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsJobSchedulerJobPerm": {
		"android-support-annotations",   // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsBackupOtherSoundsSettingsApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"IncrementalTestAppValidator": {
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"ext",                            // cts -> system
		"framework",                      // cts -> system
		"incremental-install-common-lib", // cts -> system
	},
	"CtsMediaTranscodingTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
	},
	"CtsOpenGlPerf2TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSettingsSuggestionsTest": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsSplitAppDiffRevision": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"ThirdPtyInCallServiceTestApp": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CarModeTestAppTwo": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"SelfManagedCSTestAppOne": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"Api29InCallServiceTestApp": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"ThirdPtyDialerTestAppTwo": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"ThirdPtyDialerTestApp": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CarModeTestAppSelfManaged": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CarModeTestApp": {
		"CtsTelecomMockLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CallStreamingServiceTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"TelecomDeviceTest": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsNNAPIJavaTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsInputMethodServiceDeviceTests": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest",                      // cts -> system
		"hamcrest-library",              // cts -> system
	},
	"CtsAppPredictionServiceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"TestSmsRetrieverApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"hamcrest-library",              // cts -> system
	},
	"CtsHostsideVoiceInteractionTestsApp": {
		"compatibility-device-util-axt", // cts -> system
		"junit",                         // cts -> system
		"testng",                        // cts -> system
	},
	"CtsAppBindingService2": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsAdExtServicesAdServicesCobaltTest": {
		"android.ext.adservices",        // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-adservices-lib",      // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSelinuxEphemeralTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsSpeechTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsProcStatsApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsDropBoxManagerTestCasesAPI34": {
		"compatibility-device-util-axt", // cts -> system
		"cts_flags_tests_java",          // cts -> system
		"dropbox_flags_lib",             // cts -> system
		"flag-junit",                    // cts -> system
	},
	"CtsProcStatsProtoApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsTelephony3TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsNativeHardwareTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"nativetesthelper",              // cts -> system
	},
	"CtsPackageWatchdogTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsHarmfulAppWarningTestApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsNeedSplitApp": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsBackgroundRestrictionsTestCases": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
	},
	"CtsIcu4cTestApp": {
		"compatibility-device-util-axt",          // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"modules-utils-native-coverage-listener", // cts -> system
		"nativetesthelper",                       // cts -> system
	},
	"CtsUwbTestCases": {
		"com.uwb.support.dltdoa",        // cts -> system
		"com.uwb.support.fira",          // cts -> system
		"com.uwb.support.multichip",     // cts -> system
		"com.uwb.support.oemextension",  // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"uwb_flags_lib",                 // cts -> system
	},
	"CtsGraphicsStatsApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsWrapNoWrapTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_tests_tests_wrap_src",      // cts -> system
	},
	"TestExternalImsServiceApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsVoiceSettingsService": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"TelephonyDeviceTest": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsUffdGcTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDeviceLockTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsNotificationIncidentTestApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsIkeTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"ike-test-utils",                // cts -> system
		"modules-utils-build",           // cts -> system
		"net-tests-utils",               // cts -> system
	},
	"CtsAppBindingServiceB": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsDocumentProvider": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSelinuxTargetSdk28TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsDocumentClient": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSplitAppDiffVersion": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAngleDumpsysGpuTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAppBindingService1": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsPermissionsSyncTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsProfileSerialNumberApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsAppBindingService4": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsControlsDeviceTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsSystemIntentTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
	},
	"CtsIntentSignatureTestCases": {
		"android.test.base-minus-junit", // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"cts-signature-common",          // cts -> system
	},
	"CtsTelephonyPreparerApp": {
		"compatibility-device-preconditions", // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"ctstestrunner-axt",                  // cts -> system
	},
	"CtsSelinuxTargetSdk30TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsRoleSecurityTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSplitInstantApp": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSecureElementAccessControlTestCases3": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSettingsAPITestCases": {
		"TvSettingsAPI",                 // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"truth",                         // cts -> system
	},
	"CtsProviderUiTestCases": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsExternalServiceService": {
		"CtsExternalServiceCommon",      // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"MockSatelliteGatewayServiceApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"telephony-common",              // cts -> system
	},
	"CtsSampleDeviceTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsSelinuxTargetSdk29TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsWriteExternalStorageApp": {
		"CtsWriteExternalStorageWriteGiftLib", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
	},
	"CtsCarrierApiTargetPrepApp": {
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsTelephonyCleanerApp": {
		"compatibility-device-preconditions", // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"ctstestrunner-axt",                  // cts -> system
	},
	"TestSmsApp22": {
		"compatibility-device-util-axt", // cts -> system
	},
	"MockPointingUiApp": {
		"compatibility-device-util-axt", // cts -> system
		"telephony-common",              // cts -> system
	},
	"TestFinancialSmsApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"TestSmsApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsAttentionServiceDeviceTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsDropBoxManagerTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_flags_tests_java",          // cts -> system
		"dropbox_flags_lib",             // cts -> system
		"flag-junit",                    // cts -> system
	},
	"CtsAppBindingService3": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsSearchUiServiceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppBindingService5": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsAppBindingService6": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CompanionDeviceTestApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsAppBindingService7": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
	},
	"CtsMediaPreparerApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsSplitAppDiffCert": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWidgetTestCases29": {
		"android-common",                // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSplitApp": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsStorageAccessTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsWallpaperEffectsGenerationServiceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsJniTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsVrTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsWrapWrapDebugTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_tests_tests_wrap_src",      // cts -> system
	},
	"CtsEncryptionApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsTareTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsNonProductionReadyNativeApiTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsSeccompDeviceApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsBroadcastRadioTestCases": {
		"android.hardware.radio.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"ctstestrunner-axt",                         // cts -> system
		"flag-junit",                                // cts -> system
	},
	"CtsModifyQuietModeEnabledApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsNativeResourcesTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"nativetesthelper",              // cts -> system
	},
	"CtsDeviceTaskSwitchingAppB": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"AssociationRevokedTestApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsPermissionTestCasesTelephony": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSoundTriggerTestCases": {
		"CtsSoundTriggerInstrumentation", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
	},
	"CtsAppCloningStorageStatsApp": {
		"CtsStorageAppLib",              // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsSimRestrictedApisTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"telephony-common",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsOnDevicePersonalizationTestCases": {
		"compatibility-device-util-axt",          // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"framework",                              // cts -> system
		"framework-ondevicepersonalization.impl", // cts -> system
		"mockito-target-minus-junit4",            // cts -> system
	},
	"CtsThermalTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsCredentialManagerBackupRestoreApp": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsLibnativehelperTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"nativetesthelper",              // cts -> system
	},
	"CtsFragmentTestCases": {
		"android-common",                // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsActivityRecognitionTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsApacheHttpTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestserver",                 // cts -> system
		"mockwebserver",                 // cts -> system
	},
	"CtsAppIntegrityDeviceTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsOpenGLTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSelinuxTargetSdk25TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsContentSuggestionsTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsEphemeralTestsNormalApp": {
		"compatibility-device-util-axt", // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWriteExternalStorageApp2": {
		"CtsWriteExternalStorageWriteGiftLib", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
	},
	"CtsStorageStatsApp": {
		"CtsStorageAppLib",                     // cts -> system
		"android.app.usage.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"flag-junit",                           // cts -> system
	},
	"CtsUsbSerialTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsPrivilegedUpdateTests": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSettingsProviderInvalidKeyTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSplitAppRevisionA": {
		"compatibility-device-util-axt", // cts -> system
		"hamcrest-library",              // cts -> system
		"truth",                         // cts -> system
	},
	"CtsBackupSyncAdapterSettingsApp": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsNNAPIBenchmarkTestCases": {
		"NeuralNetworksApiBenchmark_Lib", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"junit",                          // cts -> system
		"test_mlts_models_assets",        // cts -> system
	},
	"MctsMediaTranscodingTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"testng",                        // cts -> system
	},
	"CtsSecureElementTestCases": {
		"android.nfc.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"ext",                            // cts -> system
		"flag-junit",                     // cts -> system
		"framework",                      // cts -> system
		"testng",                         // cts -> system
	},
	"CtsDeviceTaskSwitchingAppA": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAccelerationTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsMidiTestCases": {
		"android.media.midi-aconfig-java", // cts -> system
		"compatibility-device-util-axt",   // cts -> system
		"cts-midi-lib",                    // cts -> system
		"ctstestrunner-axt",               // cts -> system
		"flag-junit",                      // cts -> system
		"junit",                           // cts -> system
		"junit-params",                    // cts -> system
	},
	"CtsSecureElementAccessControlTestCases1": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAssistService": {
		"CtsAssistCommon",               // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsBatteryStatsApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsTransitionTestCases": {
		"android-common",                // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"hamcrest-library",              // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsTranslationTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWifiNonUpdatableTestCases": {
		"android.net.wifi.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"ext",                                 // cts -> system
		"flag-junit",                          // cts -> system
		"framework",                           // cts -> system
		"junit",                               // cts -> system
		"junit-params",                        // cts -> system
		"net-utils-framework-common",          // cts -> system
		"truth",                               // cts -> system
		"wifi_aconfig_flags_lib",              // cts -> system
	},
	"CtsNativeMidiTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-midi-lib",                  // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSimpleCpuTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsDevicePolicySingleAdminTestApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsTelecomTestCases2": {
		"CtsTelecomUtilLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"telephony_flags_core_java_lib", // cts -> system
	},
	"CtsViewInspectorAnnotationProcessorTestCases": {
		"compatibility-device-util-axt",       // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"view-inspector-annotation-processor", // cts -> system
	},
	"CtsSelinuxTargetSdk27TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsTestHarnessModeDeviceApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
	},
	"CtsSmartspaceServiceTestCases": {
		"android.app.smartspace.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"ctstestrunner-axt",                         // cts -> system
		"truth",                                     // cts -> system
	},
	"CtsHostsideTvInputApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAnimationTestCases": {
		"android-common",                // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsCrossProfileEnabledNoPermsApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsDownloadManagerInstaller": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsNetTestCasesLegacyApi22": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"framework",                     // cts -> system
		"framework-connectivity-t.impl", // cts -> system
		"framework-connectivity.impl",   // cts -> system
		"framework-tethering.impl",      // cts -> system
	},
	"CtsCalendarProviderTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWrapWrapNoDebugTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_tests_tests_wrap_src",      // cts -> system
	},
	"CtsBatteryHealthTestCases": {
		"android.os.flags-aconfig-java",     // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
		"platformprotosnano",                // cts -> system
		"truth",                             // cts -> system
	},
	"CtsPdfTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsSelinuxTargetSdkCurrentTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsHostsideTvInputMonitor": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsContactsProviderWipe": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
	},
	"CtsBugreportTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsStoragedTestApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsProcStatsHelperApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsDrmTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsDownloadManagerApi28": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsOmapiTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAppWidgetApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
	},
	"CtsTelecomTestCases3": {
		"CtsTelecomUtilLib",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsSecureElementAccessControlTestCases2": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsCallLogDirectBootApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsRenderscriptTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"xmp_toolkit",                   // cts -> system
	},
	"CtsDeviceTaskSwitchingControl": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsTileServiceTestCases": {
		"SystemUI-proto",                 // cts -> system
		"com_android_systemui_flags_lib", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
	},
	"CtsKeystorePerformanceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"core-tests-support",            // cts -> system
		"cts-keystore-test-util",        // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"platformprotosnano",            // cts -> system
	},
	"PermissionController-lib": {
		"androidx.compose.compiler_compiler-hosted", // apex -> system
		"safety-center-annotations",                 // apex -> system
	},
	"CtsContentCaptureServiceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsMusicRecognitionTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsNetHttpTestCases": {
		"CtsNetHttpTestsLib", // cts -> system
	},
	"TetheringNext": {
		"connectivity-internal-api-util", // apex -> system
		"ext",                            // apex -> system
		"framework",                      // apex -> system
	},
	"service-configinfrastructure.impl": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsAudioRecordPermissionTests": {
		"audiorecordpermissiontests_shared", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"mediatestutils",                    // cts -> system
	},
	"CtsNetSecPolicyUsesCleartextTrafficFalseTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
	},
	"MicrodroidTestApp": {
		"MicrodroidDeviceTestHelper", // cts -> system
		"VmAttestationTestUtil",      // cts -> system
		"androidx.test.runner",       // cts -> system
		"cbor-java",                  // cts -> system
		"com.android.microdroid.test.vmshare_service-java", // cts -> system
		"compatibility-common-util-devicesidelib",          // cts -> system
		"truth", // cts -> system
	},
	"DockSetupLibrary": {
		"vendor-pixelatoms-java", // system -> vendor
	},
	"CtsHibernationTestCases": {
		"android.content.pm.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"guava",                                 // cts -> system
		"hamcrest-library",                      // cts -> system
		"junit",                                 // cts -> system
		"modules-utils-build_system",            // cts -> system
		"safety-center-internal-data",           // cts -> system
		"testng",                                // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsPackageInstallAppOpDeniedTestCases": {
		"android.content.pm.flags-aconfig-java", // cts -> system
		"androidx.legacy_legacy-support-v4",     // cts -> system
		"compatibility-device-util-axt",         // cts -> system
	},
	"CtsPackageSchemeTestsWithoutVisibility": {
		"androidx.test.runner", // cts -> system
		"ext",                  // cts -> system
		"framework",            // cts -> system
		"junit",                // cts -> system
		"truth",                // cts -> system
	},
	"CtsPackageSchemeTestsWithVisibility": {
		"androidx.test.runner", // cts -> system
		"ext",                  // cts -> system
		"framework",            // cts -> system
		"junit",                // cts -> system
		"truth",                // cts -> system
	},
	"CtsPackageInstallerTapjackingTestCases": {
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsPackageInstallAppOpDefaultTestCases": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
	},
	"CtsRestoreSessionApp1": {
		"truth", // cts -> system
	},
	"CtsRestoreSessionApp2": {
		"truth", // cts -> system
	},
	"wifi-service-pre-jarjar": {
		"app-compat-annotations",    // apex -> system
		"auto_value_annotations",    // apex -> apex
		"auto_value_plugin",         // apex -> system
		"framework-wifi-pre-jarjar", // apex -> system
		"jsr305",                    // apex -> apex
	},
	"CtsOpenGlPerfTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"service-uwb-pre-jarjar": {
		"framework-uwb-pre-jarjar", // apex -> system
	},
	"service-connectivity-pre-jarjar": {
		"framework-connectivity-pre-jarjar", // apex -> system
	},
	"CtsAlarmManagerTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsContactsProviderTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSharesheetTestCases": {
		"android.nfc.flags-aconfig-java",             // cts -> system
		"android.service.chooser.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",              // cts -> system
		"ctstestrunner-axt",                          // cts -> system
		"flag-junit",                                 // cts -> system
		"mockito-target-minus-junit4",                // cts -> system
	},
	"CtsMatchFlagTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsClassLoaderFactoryPathClassLoaderTestCases": {
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
	},
	"CtsTextClassifierTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsClassLoaderFactoryInMemoryDexClassLoaderTestCases": {
		"ctstestrunner-axt", // cts -> system
		"junit",             // cts -> system
	},
	"CtsDatabaseTestCases": {
		"android-common",                       // cts -> system
		"android.database.sqlite-aconfig-java", // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"ctstestrunner-axt",                    // cts -> system
		"flag-junit",                           // cts -> system
		"junit",                                // cts -> system
		"ravenwood-junit",                      // cts -> system
	},
	"Photopicker": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsBroadcastTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"guava",                         // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsQuickAccessWalletTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsAttributionSourceTestCases": {
		"android.permission.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"flag-junit",                            // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsServiceKillTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsSensorPrivacyTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsAppFgsStartTestCases": {
		"am_flags_lib",                  // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"flag-junit",                    // cts -> system
		"framework",                     // cts -> system
	},
	"CtsLibcoreTestRunner": {
		"cts-core-test-runner-axt", // cts -> system
	},
	"CtsContentProviderTestsWithoutVisibility": {
		"androidx.test.runner", // cts -> system
		"junit",                // cts -> system
		"truth",                // cts -> system
	},
	"CtsContentProviderTestsWithVisibility": {
		"androidx.test.runner", // cts -> system
		"junit",                // cts -> system
		"truth",                // cts -> system
	},
	"CtsTimeTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"device-time-shell-utils",       // cts -> system
	},
	"CtsAdServicesEndToEndTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdExtServicesTopicsAppUpdateTests": {
		"CtsSampleTopicsApp1",           // cts -> system
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdExtServicesEndToEndTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdExtServicesAdIdEndToEndTest": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdIdEndToEndTest": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-adservices-lib",      // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesTopicsConnectionTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesMddTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdExtServicesTopicsConnectionTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesEndToEndTestMeasurement": {
		"adservices-test-utility",   // cts -> system
		"framework-adservices-lib",  // cts -> system
		"framework-sdksandbox.impl", // cts -> system
	},
	"CtsAdServicesTopicsAppUpdateTests": {
		"CtsSampleTopicsApp1",           // cts -> system
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdservicesHostTestApp": {
		"adservices-clients",                           // cts -> system
		"adservices-test-utility",                      // cts -> system
		"androidx.appsearch_appsearch-compiler-plugin", // cts -> system
		"com.google.android.material_material",         // cts -> system
		"ext",                                          // cts -> system
		"framework",                                    // cts -> system
	},
	"CtsAppSetIdEndToEndTest": {
		"adservices-clients",            // cts -> system
		"adservices-shared-exceptions",  // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-adservices-lib",      // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesExtDataStorageServiceTest": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-adservices-lib",      // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsHiddenApiBlocklistDebugClassTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsHiddenApiKillswitchDebugClassTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsCurrentApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAdExtServicesEndToEndTestMeasurement": {
		"adservices-test-utility",   // cts -> system
		"android.ext.adservices",    // cts -> system
		"framework-adservices-lib",  // cts -> system
		"framework-sdksandbox.impl", // cts -> system
	},
	"CtsHiddenApiKillswitchSdkListTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsApacheHttpLegacyUsesLibraryApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsHiddenApiBlocklistCurrentApiTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAdExtServicesMddTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsHiddenApiKillswitchWildcardTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAdExtServicesExtDataStorageServiceTest": {
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdExtServicesAppSetIdEndToEndTest": {
		"adservices-clients",            // cts -> system
		"adservices-shared-exceptions",  // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsHiddenApiBlocklistApi28TestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAndroidTestRunnerCurrentApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsApacheHttpLegacy27ApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAndroidTestBaseCurrentApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsSystemApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsHiddenApiBlocklistApi27TestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsApacheHttpLegacyCurrentApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAndroidTestBaseUsesLibraryApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsHiddenApiBlocklistTestApiTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAndroidTestMockCurrentApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsAndroidTestBase29ApiSignatureTestCases": {
		"cts-api-signature-test", // cts -> system
	},
	"CtsPermissionPolicyTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"flag-junit",                    // cts -> system
		"guava",                         // cts -> system
		"permission-test-util-lib",      // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSharedUserMigrationTestCases": {
		"CtsSharedUserMigrationTestLibs", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"ext",                            // cts -> system
		"framework",                      // cts -> system
		"permission-test-util-lib",       // cts -> system
		"services.core",                  // cts -> system
	},
	"CtsCarBuiltinSimpleApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWebViewStartupApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
	},
	"CtsGraphicsTestCases": {
		"SurfaceFlingerProperties",                     // cts -> system
		"com.android.text.flags-aconfig-java",          // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"ctsdeviceutillegacy-axt",                      // cts -> system
		"ctstestrunner-axt",                            // cts -> system
		"flag-junit",                                   // cts -> system
		"framework_graphics_flags_java_lib",            // cts -> system
		"hamcrest-library",                             // cts -> system
		"hwui_flags_java_lib",                          // cts -> system
		"junit",                                        // cts -> system
		"junit-params",                                 // cts -> system
		"mockito-target-minus-junit4",                  // cts -> system
		"testng",                                       // cts -> system
	},
	"CtsTaskFpsCallbackTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsUiRenderingTestCases27": {
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsBootDisplayModeApp": {
		"SurfaceFlingerProperties",      // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"junit-params",                  // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsEphemeralTestsEphemeralApp2": {
		"ctsdeviceutillegacy-axt", // cts -> system
		"ctstestrunner-axt",       // cts -> system
	},
	"CtsInstantUpgradeApp": {
		"ctsdeviceutillegacy-axt", // cts -> system
		"ctstestrunner-axt",       // cts -> system
	},
	"CtsWebViewCompatChangeApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsEphemeralTestsEphemeralApp1": {
		"ctsdeviceutillegacy-axt", // cts -> system
		"ctstestrunner-axt",       // cts -> system
		"ext",                     // cts -> system
		"framework",               // cts -> system
		"testng",                  // cts -> system
	},
	"CtsGameFrameRateTestCases": {
		"SurfaceFlingerProperties",              // cts -> system
		"android.server.app.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctsdeviceutillegacy-axt",               // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"ctstestserver",                         // cts -> system
		"flag-junit",                            // cts -> system
		"framework_graphics_flags_java_lib",     // cts -> system
		"junit",                                 // cts -> system
		"surfaceflinger_flags_java_lib",         // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsSyncAccountAccessOtherCertTestCases": {
		"accountaccesslib",  // cts -> system
		"ctstestrunner-axt", // cts -> system
	},
	"CtsAccountManagerTestCases": {
		"CtsAccountTestsCommon", // cts -> system
		"ctstestrunner-axt",     // cts -> system
	},
	"CtsDelegateApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"devicepolicy-deviceside-common",    // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsCertInstallerApp": {
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"devicepolicy-deviceside-common",    // cts -> system
		"testng",                            // cts -> system
		"truth",                             // cts -> system
	},
	"CtsProfileOwnerApp": {
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"devicepolicy-deviceside-common", // cts -> system
		"junit",                          // cts -> system
	},
	"CtsDeviceOwnerApp": {
		"DpmWrapper",                        // cts -> system
		"NeneInternal",                      // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"bouncycastle-unbundled",            // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-security-test-support-library", // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"devicepolicy-deviceside-common",    // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"truth",                             // cts -> system
	},
	"CtsManagedProfileApp": {
		"NeneInternal",                      // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"devicepolicy-deviceside-common",    // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"guava",                             // cts -> system
		"junit",                             // cts -> system
		"permission-test-util-lib",          // cts -> system
		"testng",                            // cts -> system
		"truth",                             // cts -> system
	},
	"CtsVoiceInteractionService": {
		"CtsVoiceInteractionCommon",     // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsVoiceInteractionApp": {
		"CtsVoiceInteractionCommon",     // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsCompanionDeviceManagerMultiProcessTestCases": {
		"cts-companion-common", // cts -> system
		"ext",                  // cts -> system
		"framework",            // cts -> system
		"junit",                // cts -> system
	},
	"CtsCompanionDeviceManagerNoCompanionServicesTestCases": {
		"cts-companion-common", // cts -> system
		"junit",                // cts -> system
	},
	"CtsNetSecPolicyUsesCleartextTrafficTrueTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
	},
	"CtsContactKeysProviderPrivilegedApp": {
		"android.provider.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"junit",                               // cts -> system
		"truth",                               // cts -> system
	},
	"CtsContactKeysManagerTestCases": {
		"android.provider.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"junit",                               // cts -> system
		"truth",                               // cts -> system
	},
	"CtsRoleTestCases": {
		"android.permission.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"truth",                                 // cts -> system
	},
	"ExtServices-sminus": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsVoiceSettingsTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsGameManagerTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSliceTestCases": {
		"compatibility-device-util-axt",      // cts -> system
		"ctsdeviceutillegacy-axt",            // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
		"metrics-helper-lib",                 // cts -> system
		"mockito-target-inline-minus-junit4", // cts -> system
		"sts-device-util",                    // cts -> system
	},
	"CtsNetApi23TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"core-tests-support",            // cts -> system
		"cts-net-utils",                 // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"junit",                         // cts -> system
		"junit-params",                  // cts -> system
		"mockwebserver",                 // cts -> system
		"truth",                         // cts -> system
	},
	"CtsHostsideNetworkCapTestsAppWithoutProperty": {
		"cts-net-utils",       // cts -> system
		"modules-utils-build", // cts -> system
	},
	"CtsHostsideNetworkCapTestsAppWithProperty": {
		"cts-net-utils",       // cts -> system
		"modules-utils-build", // cts -> system
	},
	"CtsHostsideNetworkCapTestsAppSdk33": {
		"cts-net-utils",       // cts -> system
		"modules-utils-build", // cts -> system
	},
	"CtsNetSecConfigInvalidPinTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigResourcesSrcTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigCertificateTransparencyTestCases": {
		"android.security.flags-aconfig-java",   // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigNestedDomainConfigTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigAttributeTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigCertificateTransparencyDefaultTestCases": {
		"android.security.flags-aconfig-java",   // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigBasicDomainConfigTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigBasicDebugDisabledTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigBasicDebugEnabledTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigDownloadManagerTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigPrePCleartextTrafficTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsNetSecConfigCleartextTrafficTestCases": {
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"tests-tests-networksecurityconfig-lib", // cts -> system
	},
	"CtsLauncherAppsTests": {
		"Nene",                              // cts -> system
		"ShortcutManagerTestUtils",          // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"junit",                             // cts -> system
		"testng",                            // cts -> system
	},
	"CtsGameServiceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsExternalServiceTestCases": {
		"CtsExternalServiceCommon",           // cts -> system
		"android.content.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"flag-junit",                         // cts -> system
	},
	"CtsThreadNetworkTestCases": {
		"ThreadNetworkTestUtils",                                    // cts -> system
		"compatibility-device-util-axt",                             // cts -> system
		"ctstestrunner-axt",                                         // cts -> system
		"framework-connectivity-module-api-stubs-including-flagged", // cts -> system
		"guava",           // cts -> system
		"net-tests-utils", // cts -> system
		"truth",           // cts -> system
	},
	"CtsBluetoothTestCases": {
		"PlatformProperties",            // cts -> system
		"bluetooth-test-util-lib",       // cts -> system
		"bluetooth_flags_java_lib",      // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"flag-junit",                    // cts -> system
		"framework",                     // cts -> system
	},
	"CtsNearbyFastPairTestCases": {
		"bluetooth-test-util-lib",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"framework-connectivity-t.impl", // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSafetyCenterTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build",           // cts -> system
		"safety-center-config",          // cts -> system
		"safety-center-pending-intents", // cts -> system
		"safety-center-test-util-lib",   // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCameraApi25TestCases": {
		"CtsCameraUtils",                // cts -> system
		"android-ex-camera2",            // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsTextTestCases": {
		"com.android.text.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
		"ctsdeviceutillegacy-axt",             // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"flag-junit",                          // cts -> system
		"junit-params",                        // cts -> system
		"mockito-target-minus-junit4",         // cts -> system
		"ravenwood-junit",                     // cts -> system
	},
	"CtsProcessTestHelper1": {
		"CtsProcessTestCore", // cts -> system
		"ext",                // cts -> system
		"framework",          // cts -> system
	},
	"CtsProcessTestHelper2": {
		"CtsProcessTestCore", // cts -> system
		"ext",                // cts -> system
		"framework",          // cts -> system
	},
	"CtsProcessTestHelper4": {
		"CtsProcessTestCore", // cts -> system
		"ext",                // cts -> system
		"framework",          // cts -> system
	},
	"CtsProcessTestHelper3": {
		"CtsProcessTestCore", // cts -> system
		"ext",                // cts -> system
		"framework",          // cts -> system
	},
	"service-wifi": {
		"auto_value_annotations", // apex -> apex
		"auto_value_plugin",      // apex -> system
		"ext",                    // apex -> system
		"framework",              // apex -> system
	},
	"CtsAppThatUsesAppOps": {
		"AppOpsUserServiceAidl", // cts -> system
		"appops-test-util-lib",  // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
		"junit",                 // cts -> system
		"truth",                 // cts -> system
	},
	"CtsAppOps2TestCases": {
		"appops-test-util-lib",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCppToolsTestCases": {
		"tradefed", // cts -> system
	},
	"net-tests-utils-host-common": {
		"tradefed", // cts -> system
	},
	"MicrodroidTestPreparer": {
		"tradefed", // cts -> system
	},
	"CtsVibratorTestCases": {
		"android.os.vibrator.flags-aconfig-java", // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"flag-junit",                             // cts -> system
		"guava",                                  // cts -> system
		"hamcrest-library",                       // cts -> system
		"junit",                                  // cts -> system
		"junit-params",                           // cts -> system
		"testng",                                 // cts -> system
		"truth",                                  // cts -> system
	},
	"CtsUsbManagerTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
	},
	"CtsNoPermissionTestCases": {
		"CtsNoPermissionTestCasesBase", // cts -> system
	},
	"CtsWifiTestCases": {
		"android.wifi.mockwifi",         // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"flag-junit",                    // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"junit-params",                  // cts -> system
		"net-utils-framework-common",    // cts -> system
		"truth",                         // cts -> system
		"wifi_aconfig_flags_lib",        // cts -> system
	},
	"CtsStaticSharedLibConsumerApp1BadCertDigest": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CellBroadcastApp": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsNoPermissionTestCases25": {
		"CtsNoPermissionTestCasesBase", // cts -> system
	},
	"CtsVcnTestCases": {
		"android.net.vcn.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"cts-net-utils",                      // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"ext",                                // cts -> system
		"flag-junit",                         // cts -> system
		"framework",                          // cts -> system
		"ike-tun-utils",                      // cts -> system
		"net-tests-utils",                    // cts -> system
		"telephony-cts-utils",                // cts -> system
	},
	"CtsLegacyNotification29TestCases": {
		"CtsAppTestStubsShared",         // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"permission-test-util-lib",      // cts -> system
	},
	"CtsBatterySavingAppTargetApi25": {
		"CtsBatterSavingAppTargetLib", // cts -> system
	},
	"CtsKeystoreWycheproofTestCases": {
		"bouncycastle-bcpkix-unbundled", // cts -> system
		"bouncycastle-unbundled",        // cts -> system
		"cts-core-test-runner-axt",      // cts -> system
		"cts-keystore-test-util",        // cts -> system
		"gson",                          // cts -> system
		"wycheproof-keystore",           // cts -> system
	},
	"CtsTelephonyProviderTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"telephony-cts-utils",           // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesPermissionsValidEndToEndTests": {
		"adservices-clients",                   // cts -> system
		"adservices-test-fixtures",             // cts -> system
		"adservices-test-utility",              // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"ext",                                  // cts -> system
		"framework",                            // cts -> system
		"framework-sdksandbox.impl",            // cts -> system
		"modules-utils-extended-mockito-rule",  // cts -> system
		"modules-utils-testable-device-config", // cts -> system
		"truth",                                // cts -> system
	},
	"CtsAdServicesNotInAllowListEndToEndTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesPermissionsAppOptOutEndToEndTests": {
		"adservices-clients",                   // cts -> system
		"adservices-test-fixtures",             // cts -> system
		"adservices-test-utility",              // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"ext",                                  // cts -> system
		"framework",                            // cts -> system
		"framework-sdksandbox.impl",            // cts -> system
		"modules-utils-extended-mockito-rule",  // cts -> system
		"modules-utils-testable-device-config", // cts -> system
		"truth",                                // cts -> system
	},
	"CtsAdExtServicesPermissionsAppOptOutEndToEndTests": {
		"adservices-clients",                   // cts -> system
		"adservices-test-fixtures",             // cts -> system
		"adservices-test-utility",              // cts -> system
		"android.ext.adservices",               // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"framework-sdksandbox.impl",            // cts -> system
		"modules-utils-extended-mockito-rule",  // cts -> system
		"modules-utils-testable-device-config", // cts -> system
		"truth",                                // cts -> system
	},
	"CtsAdExtServicesPermissionsValidEndToEndTests": {
		"adservices-clients",                   // cts -> system
		"adservices-test-fixtures",             // cts -> system
		"adservices-test-utility",              // cts -> system
		"android.ext.adservices",               // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"framework-sdksandbox.impl",            // cts -> system
		"modules-utils-extended-mockito-rule",  // cts -> system
		"modules-utils-testable-device-config", // cts -> system
		"truth",                                // cts -> system
	},
	"CtsAdExtServicesNotInAllowListEndToEndTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdExtServicesPermissionsNoPermEndToEndTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAdServicesPermissionsNoPermEndToEndTests": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"framework-sdksandbox.impl",     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAssistTestCases": {
		"CtsAssistCommon",               // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsTelephony2TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"telephony-cts-utils",           // cts -> system
		"telephony_flags_core_java_lib", // cts -> system
	},
	"service-uwb": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"HealthConnectController": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsBlobStoreTestCases": {
		"BlobStoreTestUtils",            // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"guava",                         // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWrapHwasanTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_tests_tests_hwasan_src",    // cts -> system
	},
	"CtsBatterySavingAppTargetApiCurrent": {
		"CtsBatterSavingAppTargetLib", // cts -> system
	},
	"CtsTelephony5TestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"telephony-cts-utils",           // cts -> system
	},
	"CtsJobSchedulerSharedUidTestCases": {
		"compatibility-device-util-axt", // cts -> system
	},
	"SdkSandboxManagerDisabledTests": {
		"SdkSandboxTestUtils",  // cts -> system
		"androidx.test.runner", // cts -> system
		"ext",                  // cts -> system
		"framework",            // cts -> system
		"truth",                // cts -> system
	},
	"CtsLegacyNotification34TestCases": {
		"CtsAppTestStubsShared",         // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"permission-test-util-lib",      // cts -> system
	},
	"CtsStaticSharedLibConsumerApp1": {
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsLegacyNotification30TestCases": {
		"CtsAppTestStubsShared", // cts -> system
		"ctstestrunner-axt",     // cts -> system
		"junit",                 // cts -> system
		"truth",                 // cts -> system
	},
	"CtsLegacyNotification27TestCases": {
		"CtsAppTestStubsShared",         // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"permission-test-util-lib",      // cts -> system
	},
	"CtsSandboxedAdIdManagerTests": {
		"SdkSandboxTestUtils",           // cts -> system
		"adservices-test-utility",       // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCarTestCases": {
		"aconfig-annotations-lib",          // cts -> system
		"android-support-v4",               // cts -> system
		"android.car.feature-aconfig-java", // cts -> system
		"android.car.test.utils",           // cts -> system
		"android.car.testapi",              // cts -> system
		"car-integration-test-utils-lib",   // cts -> system
		"compatibility-device-util-axt",    // cts -> system
		"ctstestrunner-axt",                // cts -> system
		"flag-junit",                       // cts -> system
		"hamcrest-library",                 // cts -> system
		"libprotobuf-java-lite",            // cts -> system
		"truth",                            // cts -> system
	},
	"CtsAccessibilityServiceSdk29TestCases": {
		"CtsAccessibilityCommon", // cts -> system
		"ctstestrunner-axt",      // cts -> system
	},
	"odsign_e2e_tests": {
		"cts-install-lib-host",      // cts -> system
		"frameworks-base-hostutils", // cts -> system
		"tradefed",                  // cts -> system
	},
	"CtsAppSearchTestCases": {
		"CtsAppSearchTestUtils",         // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"testng",                        // cts -> system
	},
	"CtsCompanionDeviceManagerCoreTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-companion-common",          // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
	},
	"CtsSdkSandboxInprocessTests": {
		"SdkSandboxTestUtils",       // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
		"framework-adservices.impl", // cts -> system
		"modules-utils-build",       // cts -> system
		"truth",                     // cts -> system
	},
	"CtsSandboxedAppSetIdManagerTests": {
		"SdkSandboxTestUtils",           // cts -> system
		"adservices-test-utility",       // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsShortcutMultiuserTest": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsCrashDetailHostTestCases": {
		"compatibility-host-util",   // cts -> system
		"cts-install-lib-host",      // cts -> system
		"frameworks-base-hostutils", // cts -> system
		"tradefed",                  // cts -> system
	},
	"CtsSharedLibsApiSignatureTestCases": {
		"cts-api-signature-multilib-test", // cts -> system
		"cts-api-signature-test",          // cts -> system
	},
	"CtsSandboxedTopicsManagerTests": {
		"SdkSandboxTestUtils",           // cts -> system
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsShortcutUpgradeVersion2": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupLauncher1": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsLegacyNotification28TestCases": {
		"CtsAppTestStubsShared",         // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"permission-test-util-lib",      // cts -> system
	},
	"CtsIcu4cTestCases": {
		"ICU4CTestRunner", // cts -> system
	},
	"CtsAdminTestCases": {
		"Nene",                          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsShortFgsTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"platformprotosnano",            // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCompanionDeviceManagerUiAutomationTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-companion-common",          // cts -> system
		"cts-companion-uicommon",        // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
	},
	"CtsLocationNoneTestCases": {
		"LocationCtsCommon",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsLocationGnssTestCases": {
		"LocationCtsCommon",             // cts -> system
		"apache-commons-math",           // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"libprotobuf-java-nano",         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsLocationFineTestCases": {
		"LocationCtsCommon",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsLocationPrivilegedTestCases": {
		"LocationCtsCommon",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsShortcutBackupLauncher2": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsLocationCoarseTestCases": {
		"LocationCtsCommon",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsShortcutBackupLauncher3": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsSimPhonebookProviderTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-simphonebook-rules",        // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"hamcrest-library",              // cts -> system
		"junit",                         // cts -> system
		"telephony-common-testing",      // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppTestStubsApp2": {
		"CtsAppTestStubsShared",             // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ctstestserver",                     // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
		"telephony-common",                  // cts -> system
		"voip-common",                       // cts -> system
	},
	"CtsNetSecPolicyUsesCleartextTrafficUnspecifiedTestCases": {
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
	},
	"CtsShortcutBackupPublisher2": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"service-thread-pre-jarjar": {
		"framework-connectivity-pre-jarjar",   // apex -> system
		"framework-connectivity-t-pre-jarjar", // apex -> system
	},
	"CtsShortcutBackupPublisher3": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutUpgradeVersion1": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsLeanbackJankApp": {
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"glide",                             // cts -> system
	},
	"CtsTelecomTestCases": {
		"CtsTelecomUtilLib",             // cts -> system
		"TelecomTestAppUtilsLib",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"telecom_flags_core_java_lib",   // cts -> system
		"telephony_flags_core_java_lib", // cts -> system
	},
	"CtsNetTestCasesInternetPermission": {
		"ctstestrunner-axt", // cts -> system
	},
	"CtsNetTestCasesUpdateStatsPermission": {
		"ctstestrunner-axt",             // cts -> system
		"framework",                     // cts -> system
		"framework-connectivity-t.impl", // cts -> system
		"framework-connectivity.impl",   // cts -> system
		"framework-tethering.impl",      // cts -> system
		"net-tests-utils",               // cts -> system
	},
	"CtsTvTestCases": {
		"CtsTvTestCases_lib", // cts -> system
	},
	"CtsDeviceAndProfileOwnerApp": {
		"MetricsRecorder",                    // cts -> system
		"ShortcutManagerTestUtils",           // cts -> system
		"androidx.legacy_legacy-support-v4",  // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"cts-devicepolicy-suspensionchecker", // cts -> system
		"cts-keystore-test-util",             // cts -> system
		"cts-security-test-support-library",  // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"devicepolicy-deviceside-common",     // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
		"statsdprotolite",                    // cts -> system
	},
	"CtsDeviceAndProfileOwnerApp23": {
		"MetricsRecorder",                    // cts -> system
		"ShortcutManagerTestUtils",           // cts -> system
		"androidx.legacy_legacy-support-v4",  // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"cts-devicepolicy-suspensionchecker", // cts -> system
		"cts-keystore-test-util",             // cts -> system
		"cts-security-test-support-library",  // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"devicepolicy-deviceside-common",     // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
		"statsdprotolite",                    // cts -> system
	},
	"CtsDreamsTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
	},
	"MtsProfilingModuleTests": {
		"android.os.profiling.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",           // cts -> system
		"ctstestrunner-axt",                       // cts -> system
		"framework-profiling-proto",               // cts -> system
		"framework-profiling.impl",                // cts -> system
		"junit",                                   // cts -> system
		"modules-utils-build",                     // cts -> system
		"service-profiling",                       // cts -> system
		"testng",                                  // cts -> system
	},
	"CtsAppWidgetTestCasesBalApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsDpiTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsIdentityTestCases": {
		"cbor-java",                             // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"cts-keystore-user-auth-helper-library", // cts -> system
		"cts-security-test-support-library",     // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"ext",                                   // cts -> system
		"framework",                             // cts -> system
		"identity-credential-util",              // cts -> system
		"junit",                                 // cts -> system
		"platformprotosnano",                    // cts -> system
	},
	"CtsDeviceAndProfileOwnerApp25": {
		"MetricsRecorder",                    // cts -> system
		"ShortcutManagerTestUtils",           // cts -> system
		"androidx.legacy_legacy-support-v4",  // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"cts-devicepolicy-suspensionchecker", // cts -> system
		"cts-keystore-test-util",             // cts -> system
		"cts-security-test-support-library",  // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"devicepolicy-deviceside-common",     // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
		"statsdprotolite",                    // cts -> system
	},
	"CtsNetTestCasesMaxTargetSdk30": {
		"ApfGeneratorLib",                  // cts -> system
		"CtsNetTestsNonUpdatableLib",       // cts -> system
		"DhcpPacketLib",                    // cts -> system
		"FrameworksNetCommonTests",         // cts -> system
		"NetworkStackApiStableShims",       // cts -> system
		"TetheringIntegrationTestsBaseLib", // cts -> system
		"bouncycastle-unbundled",           // cts -> system
		"core-tests-support",               // cts -> system
		"cts-net-utils",                    // cts -> system
		"ctstestrunner-axt",                // cts -> system
		"framework",                        // cts -> system
		"framework-connectivity-t.impl",    // cts -> system
		"framework-connectivity.impl",      // cts -> system
		"framework-tethering.impl",         // cts -> system
		"junit",                            // cts -> system
		"junit-params",                     // cts -> system
		"modules-utils-build",              // cts -> system
		"net-tests-utils",                  // cts -> system
		"net-utils-framework-common",       // cts -> system
		"truth",                            // cts -> system
		"voip-common",                      // cts -> system
	},
	"CtsRotationResolverServiceDeviceTestCases": {
		"android-common",                // cts -> system
		"android-support-v4",            // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsNetTestCasesMaxTargetSdk33": {
		"ApfGeneratorLib",                  // cts -> system
		"CtsNetTestsNonUpdatableLib",       // cts -> system
		"DhcpPacketLib",                    // cts -> system
		"FrameworksNetCommonTests",         // cts -> system
		"NetworkStackApiStableShims",       // cts -> system
		"TetheringIntegrationTestsBaseLib", // cts -> system
		"bouncycastle-unbundled",           // cts -> system
		"core-tests-support",               // cts -> system
		"cts-net-utils",                    // cts -> system
		"ctstestrunner-axt",                // cts -> system
		"framework",                        // cts -> system
		"framework-connectivity-t.impl",    // cts -> system
		"framework-connectivity.impl",      // cts -> system
		"framework-tethering.impl",         // cts -> system
		"junit",                            // cts -> system
		"junit-params",                     // cts -> system
		"modules-utils-build",              // cts -> system
		"net-tests-utils",                  // cts -> system
		"net-utils-framework-common",       // cts -> system
		"truth",                            // cts -> system
		"voip-common",                      // cts -> system
	},
	"CtsDeviceAndProfileOwnerApp30": {
		"MetricsRecorder",                    // cts -> system
		"ShortcutManagerTestUtils",           // cts -> system
		"androidx.legacy_legacy-support-v4",  // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"cts-devicepolicy-suspensionchecker", // cts -> system
		"cts-keystore-test-util",             // cts -> system
		"cts-security-test-support-library",  // cts -> system
		"ctstestrunner-axt",                  // cts -> system
		"devicepolicy-deviceside-common",     // cts -> system
		"ext",                                // cts -> system
		"framework",                          // cts -> system
		"statsdprotolite",                    // cts -> system
	},
	"CtsWearableSensingServiceTestCases": {
		"android-common",                          // cts -> system
		"android-support-v4",                      // cts -> system
		"android.app.wearable.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",           // cts -> system
		"ext",                                     // cts -> system
		"framework",                               // cts -> system
	},
	"CtsLocaleConfigTestCases": {
		"android.content.res.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",          // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"flag-junit",                             // cts -> system
		"junit",                                  // cts -> system
	},
	"CtsDeviceInfo": {
		"compatibility-device-info",     // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"Tethering": {
		"connectivity-internal-api-util", // apex -> system
		"ext",                            // apex -> system
		"framework",                      // apex -> system
	},
	"CtsShortcutBackupPublisher1": {
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsSandboxedFledgeManagerTests": {
		"SdkSandboxTestUtils",           // cts -> system
		"adservices-clients",            // cts -> system
		"adservices-service-core",       // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSettingsDeviceOwnerApp": {
		"DpmWrapper",                    // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsWidgetTestCases": {
		"android-common", // cts -> system
		"android.view.inputmethod.flags-aconfig-java", // cts -> system
		"androidx.test.espresso.core",                 // cts -> system
		"com.android.text.flags-aconfig-java",         // cts -> system
		"compatibility-device-util-axt",               // cts -> system
		"ctsdeviceutillegacy-axt",                     // cts -> system
		"ctstestrunner-axt",                           // cts -> system
		"ext",                                         // cts -> system
		"flag-junit",                                  // cts -> system
		"framework",                                   // cts -> system
		"mockito-target-minus-junit4",                 // cts -> system
		"testables",                                   // cts -> system
		"truth",                                       // cts -> system
	},
	"CtsLocaleManagerTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsWallpaperTestCases": {
		"TestParameterInjector",                        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
	},
	"CtsAppFgsTestCases": {
		"CtsExternalServiceCommon",              // cts -> system
		"android.content.pm.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ext",                                   // cts -> system
		"flag-junit",                            // cts -> system
		"framework",                             // cts -> system
		"libprotobuf-java-lite",                 // cts -> system
	},
	"CtsAmbientContextServiceTestCases": {
		"android-common",                // cts -> system
		"android-support-v4",            // cts -> system
		"compatibility-device-util-axt", // cts -> system
	},
	"CtsPreferenceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsAdServicesDebuggableDeviceTestCases": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"auto_annotation_plugin",        // cts -> system
		"auto_value_annotations",        // cts -> system
		"auto_value_plugin",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"framework-adservices-lib",      // cts -> system
		"mockwebserver",                 // cts -> system
		"testng",                        // cts -> system
	},
	"CtsUiRenderingTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"flag-junit",                    // cts -> system
		"hwui_flags_java_lib",           // cts -> system
		"junit-params",                  // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"testng",                        // cts -> system
	},
	"CtsAdExtServicesDebuggableDeviceTestCases": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"auto_annotation_plugin",        // cts -> system
		"auto_value_annotations",        // cts -> system
		"auto_value_plugin",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"framework-adservices-lib",      // cts -> system
		"testng",                        // cts -> system
	},
	"CtsAndroidAppTestCases": {
		"TestParameterInjector",               // cts -> system
		"android.security.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",       // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"ext",                                 // cts -> system
		"flag-junit",                          // cts -> system
		"framework",                           // cts -> system
		"junit",                               // cts -> system
		"mockito-target-minus-junit4",         // cts -> system
	},
	"CtsAdExtServicesDeviceTestCases": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"android.ext.adservices",        // cts -> system
		"auto_annotation_plugin",        // cts -> system
		"auto_value_annotations",        // cts -> system
		"auto_value_plugin",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"framework-adservices-lib",      // cts -> system
		"testng",                        // cts -> system
	},
	"CtsVoiceRecognitionTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit-params",                  // cts -> system
		"truth",                         // cts -> system
	},
	"CtsTetheringTest": {
		"TetheringCommonTests",          // cts -> system
		"TetheringIntegrationTestsLib",  // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"cts-net-utils",                 // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"junit-params",                  // cts -> system
		"net-tests-utils",               // cts -> system
	},
	"CtsNetTestCases": {
		"ApfGeneratorLib",                  // cts -> system
		"CtsNetTestsNonUpdatableLib",       // cts -> system
		"DhcpPacketLib",                    // cts -> system
		"FrameworksNetCommonTests",         // cts -> system
		"NetworkStackApiCurrentShims",      // cts -> system
		"TetheringIntegrationTestsBaseLib", // cts -> system
		"bouncycastle-unbundled",           // cts -> system
		"core-tests-support",               // cts -> system
		"cts-net-utils",                    // cts -> system
		"ctstestrunner-axt",                // cts -> system
		"framework",                        // cts -> system
		"framework-connectivity-t.impl",    // cts -> system
		"framework-connectivity.impl",      // cts -> system
		"framework-tethering.impl",         // cts -> system
		"junit",                            // cts -> system
		"junit-params",                     // cts -> system
		"modules-utils-build",              // cts -> system
		"net-tests-utils",                  // cts -> system
		"net-utils-framework-common",       // cts -> system
		"truth",                            // cts -> system
		"voip-common",                      // cts -> system
	},
	"CtsNetTestCasesMaxTargetSdk31": {
		"ApfGeneratorLib",                  // cts -> system
		"CtsNetTestsNonUpdatableLib",       // cts -> system
		"DhcpPacketLib",                    // cts -> system
		"FrameworksNetCommonTests",         // cts -> system
		"NetworkStackApiStableShims",       // cts -> system
		"TetheringIntegrationTestsBaseLib", // cts -> system
		"bouncycastle-unbundled",           // cts -> system
		"core-tests-support",               // cts -> system
		"cts-net-utils",                    // cts -> system
		"ctstestrunner-axt",                // cts -> system
		"framework",                        // cts -> system
		"framework-connectivity-t.impl",    // cts -> system
		"framework-connectivity.impl",      // cts -> system
		"framework-tethering.impl",         // cts -> system
		"junit",                            // cts -> system
		"junit-params",                     // cts -> system
		"modules-utils-build",              // cts -> system
		"net-tests-utils",                  // cts -> system
		"net-utils-framework-common",       // cts -> system
		"truth",                            // cts -> system
		"voip-common",                      // cts -> system
	},
	"CtsGrammaticalInflectionTestCases": {
		"android.app.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"junit",                          // cts -> system
	},
	"CtsUsageStatsTestApp2": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsShortcutBackupLauncher4old": {
		"CtsShortcutBackupLauncher4oldLib",          // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupLauncher4new": {
		"CtsShortcutBackupLauncher4oldLib",          // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"VideoEncodingApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctsmediav2common",              // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"VideoEncodingMinApp": {
		"compatibility-device-util-axt", // cts -> system
		"ctsmediav2common",              // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsMediaV2TestCases": {
		"aconfig_mediacodec_flags_java_lib", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctsmediav2common",                  // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ctstestserver",                     // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
	},
	"CtsMediaEditingTestCases": {
		"androidx.media3.media3-common",      // cts -> system
		"androidx.media3.media3-effect",      // cts -> system
		"androidx.media3.media3-exoplayer",   // cts -> system
		"androidx.media3.media3-test-utils",  // cts -> system
		"androidx.media3.media3-transformer", // cts -> system
		"compatibility-device-util-axt",      // cts -> system
		"ctsmediav2common",                   // cts -> system
		"ctstestrunner-axt",                  // cts -> system
	},
	"CtsVideoCodecTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctsmediav2common",              // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsSandboxedMeasurementManagerTests": {
		"SdkSandboxTestUtils",           // cts -> system
		"adservices-clients",            // cts -> system
		"adservices-test-utility",       // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSyncManagerTestsCases": {
		"CtsSyncManagerCommon",              // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
	},
	"CtsAdServicesDeviceTestCases": {
		"adservices-clients",            // cts -> system
		"adservices-test-fixtures",      // cts -> system
		"adservices-test-utility",       // cts -> system
		"auto_annotation_plugin",        // cts -> system
		"auto_value_annotations",        // cts -> system
		"auto_value_plugin",             // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"framework-adservices-lib",      // cts -> system
		"testng",                        // cts -> system
	},
	"libnativeloader_e2e_tests": {
		"libnativeloader_vendor_shared_lib", // system -> vendor
		"loadlibrarytest_vendor_app",        // system -> vendor
	},
	"CtsSdkSandboxHostSideTests": {
		"SdkSandboxHostTestUtils",     // cts -> system
		"modules-utils-build-testing", // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsUtilTestCases": {
		"core-test-rules",   // cts -> system
		"cts-install-lib",   // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"ravenwood-junit",   // cts -> system
	},
	"CtsResourcesLoaderTests": {
		"androidx.test.espresso.core",   // cts -> system
		"androidx.test.runner",          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"truth",                         // cts -> system
	},
	"PermissionController": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},
	"CtsPackageManagerMultiUserTestApp": {
		"cts-install-lib", // cts -> system
	},
	"CtsManagedProfileOwnerApp": {
		"cts-install-lib", // cts -> system
	},
	"CtsSecureFrpInstallTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-install-lib",               // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsVideoTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-media-common",              // cts -> system
		"ctsmediautil",                  // cts -> system
		"ctsmediav2common",              // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsMediaEncoderTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
	},
	"CtsMediaRecorderTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
	},
	"CtsMediaDrmFrameworkTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"hamcrest-library",  // cts -> system
		"testng",            // cts -> system
	},
	"CtsVirtualDevicesSensorTestCases": {
		"CtsVirtualDeviceCommonLib", // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
	},
	"CtsVirtualDeviceStreamedTestApp": {
		"CtsVirtualDeviceCommonLib", // cts -> system
	},
	"CtsMediaProjectionTestCases": {
		"cts-media-common",           // cts -> system
		"ctstestrunner-axt",          // cts -> system
		"ext",                        // cts -> system
		"framework",                  // cts -> system
		"junit",                      // cts -> system
		"junit-params",               // cts -> system
		"platform-compat-test-rules", // cts -> system
		"testng",                     // cts -> system
		"truth",                      // cts -> system
	},
	"CtsNotificationExtenders34TestApp": {
		"cts-install-lib", // cts -> system
		"junit",           // cts -> system
		"truth",           // cts -> system
	},
	"CtsVirtualDevicesAudioTestCases": {
		"CtsVirtualDeviceCommonLib", // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
	},
	"CtsProcessTest": {
		"CtsProcessTestCore", // cts -> system
	},
	"CtsIntentRedirectionTestApp": {
		"cts-install-lib", // cts -> system
	},
	"CtsAtomicInstallTestCases": {
		"androidx.test.runner", // cts -> system
		"cts-install-lib",      // cts -> system
		"truth",                // cts -> system
	},
	"CtsMediaProviderTranscodeTests": {
		"collector-device-lib-platform", // cts -> system
		"cts-install-lib",               // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"mockito-target",                // cts -> system
		"truth",                         // cts -> system
	},
	"CtsMediaMiscTestCases": {
		"CtsCameraUtils",                    // cts -> system
		"aconfig_mediacodec_flags_java_lib", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-media-common",                  // cts -> system
		"ctsdeviceutillegacy-axt",           // cts -> system
		"ctsmediautil",                      // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ctstestserver",                     // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"hamcrest-library",                  // cts -> system
		"junit",                             // cts -> system
		"junit-params",                      // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
		"testng",                            // cts -> system
		"truth",                             // cts -> system
	},
	"CtsMediaDecoderTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
	},
	"CtsMediaPlayerTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
	},
	"MctsMediaDrmFrameworkTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"hamcrest-library",  // cts -> system
		"testng",            // cts -> system
	},
	"CtsMediaMuxerTestCases": {
		"cts-media-common",           // cts -> system
		"ctstestrunner-axt",          // cts -> system
		"exoplayer-mediamuxer_tests", // cts -> system
		"ext",                        // cts -> system
		"framework",                  // cts -> system
	},
	"CtsMediaStressTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-media-common",              // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsMediaProjectionSDK34TestCases": {
		"cts-media-common",           // cts -> system
		"ctstestrunner-axt",          // cts -> system
		"ext",                        // cts -> system
		"framework",                  // cts -> system
		"junit",                      // cts -> system
		"junit-params",               // cts -> system
		"platform-compat-test-rules", // cts -> system
		"testng",                     // cts -> system
		"truth",                      // cts -> system
	},
	"CtsMediaExtractorTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ctstestserver",     // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
	},
	"CtsMediaParserTestCases": {
		"cts-media-common",                // cts -> system
		"ctstestrunner-axt",               // cts -> system
		"exoplayer-cts_media-test_assets", // cts -> system
		"exoplayer-cts_media-test_utils",  // cts -> system
	},
	"CtsMediaCodecTestCases": {
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"framework",         // cts -> system
		"testng",            // cts -> system
	},
	"CtsVirtualDevicesTestCases": {
		"CtsVirtualDeviceCommonLib", // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
	},
	"CtsAppExitTestCases": {
		"CtsExternalServiceCommon",      // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"libprotobuf-java-lite",         // cts -> system
	},
	"com.android.cts.helpers.aosp": {
		"cts-helpers-core",       // cts -> system
		"cts-helpers-interfaces", // cts -> system
	},
	"CtsGetBindingUidImportanceTest": {
		"CtsAppTestStubsShared",          // cts -> system
		"android.app.flags-aconfig-java", // cts -> system
		"flag-junit",                     // cts -> system
	},
	"CtsNotificationTestCases": {
		"CtsAppTestStubsShared",                           // cts -> system
		"android.app.flags-aconfig-java",                  // cts -> system
		"android.service.notification.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                   // cts -> system
		"ctstestrunner-axt",                               // cts -> system
		"flag-junit",                                      // cts -> system
		"junit",                                           // cts -> system
		"notification_flags_lib",                          // cts -> system
		"permission-test-util-lib",                        // cts -> system
	},
	"CtsFgsBootCompletedTestCases": {
		"CtsAppTestStubsShared",       // cts -> system
		"am_flags_lib",                // cts -> system
		"ctstestrunner-axt",           // cts -> system
		"ext",                         // cts -> system
		"flag-junit",                  // cts -> system
		"framework",                   // cts -> system
		"mockito-target-minus-junit4", // cts -> system
	},
	"CtsNfcTestCases": {
		"CtsAppTestStubsShared",                 // cts -> system
		"android.nfc.flags-aconfig-java",        // cts -> system
		"android.permission.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"flag-junit",                            // cts -> system
		"framework",                             // cts -> system
		"framework-nfc.impl",                    // cts -> system
		"testng",                                // cts -> system
	},
	"CtsDomainVerificationDeviceStandaloneTestCases": {
		"CtsDomainVerificationAndroidConstantsLibrary", // cts -> system
		"CtsDomainVerificationJavaConstantsLibrary",    // cts -> system
		"android.content.pm.flags-aconfig-java",        // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"junit",                                        // cts -> system
		"truth",                                        // cts -> system
	},
	"CtsAppTestCases": {
		"CtsAppTestStubsShared",                               // cts -> system
		"WallpaperTest",                                       // cts -> system
		"android.app.flags-aconfig-java",                      // cts -> system
		"android.content.pm.flags-aconfig-java",               // cts -> system
		"android.multiuser.flags-aconfig-java",                // cts -> system
		"com.android.media.flags.bettertogether-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                       // cts -> system
		"ctstestrunner-axt",                                   // cts -> system
		"ctstestserver",                                       // cts -> system
		"flag-junit",                                          // cts -> system
		"mockito-target-minus-junit4",                         // cts -> system
		"permission-test-util-lib",                            // cts -> system
		"platformprotosnano",                                  // cts -> system
		"ravenwood-junit",                                     // cts -> system
	},
	"CtsAppOpsTestCases": {
		"AppOpsForegroundControlServiceAidl",    // cts -> system
		"AppOpsUserServiceAidl",                 // cts -> system
		"CtsVirtualDeviceCommonLib",             // cts -> system
		"android.permission.flags-aconfig-java", // cts -> system
		"androidx.legacy_legacy-support-v4",     // cts -> system
		"appops-test-util-lib",                  // cts -> system
		"bluetooth-test-util-lib",               // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ext",                                   // cts -> system
		"flag-junit",                            // cts -> system
		"framework",                             // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsMediaProjectionSDK33TestCases": {
		"cts-media-common",           // cts -> system
		"ctstestrunner-axt",          // cts -> system
		"ext",                        // cts -> system
		"framework",                  // cts -> system
		"junit",                      // cts -> system
		"junit-params",               // cts -> system
		"platform-compat-test-rules", // cts -> system
		"testng",                     // cts -> system
		"truth",                      // cts -> system
	},
	"CtsVirtualDevicesCameraTestCases": {
		"CtsVirtualDeviceCommonLib", // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
		"junit-params",              // cts -> system
	},
	"CtsShortcutManagerTestCases": {
		"CtsShortcutManagerLib",             // cts -> system
		"ShortcutManagerTestUtils",          // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
		"permission-test-util-lib",          // cts -> system
	},
	"CtsNotificationExtendersCurrentTestApp": {
		"cts-install-lib", // cts -> system
		"junit",           // cts -> system
		"truth",           // cts -> system
	},
	"CtsAppCloningNotLaunchableCloneProfileApp": {
		"cts-install-lib", // cts -> system
	},
	"CtsAppCloningLaunchableCloneProfileApp": {
		"cts-install-lib", // cts -> system
	},
	"CtsPackageUninstallTestCases": {
		"ShortcutManagerTestUtils",              // cts -> system
		"android.content.pm.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"cts-install-lib",                       // cts -> system
	},
	"CtsCameraTestCases": {
		"CtsCameraUtils",                         // cts -> system
		"MediaPerformanceClassCommon",            // cts -> system
		"android-ex-camera2",                     // cts -> system
		"camera_platform_flags_java_lib",         // cts -> system
		"collector-device-lib-platform",          // cts -> system
		"compatibility-device-util-axt",          // cts -> system
		"cts-hardware-lib",                       // cts -> system
		"cts-install-lib",                        // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"flag-junit",                             // cts -> system
		"mockito-target-minus-junit4",            // cts -> system
		"modules-utils-native-coverage-listener", // cts -> system
		"truth",                                  // cts -> system
	},
	"CtsShortcutBackupPublisher4new": {
		"CtsShortcutBackupPublisher4oldLib",         // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupPublisher4new_nobackup": {
		"CtsShortcutBackupPublisher4oldLib",         // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupPublisher4new_nomanifest": {
		"CtsShortcutBackupPublisher4oldLib",         // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupPublisher4new_wrongkey": {
		"CtsShortcutBackupPublisher4oldLib",         // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupPublisher4old": {
		"CtsShortcutBackupPublisher4oldLib",         // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"CtsShortcutBackupPublisher4old_nomanifest": {
		"CtsShortcutBackupPublisher4oldLib",         // cts -> system
		"ShortcutManagerTestUtils",                  // cts -> system
		"compatibility-device-util-axt",             // cts -> system
		"hostsidetests-shortcuts-deviceside-common", // cts -> system
	},
	"service-connectivity-tiramisu-pre-jarjar": {
		"framework-connectivity-pre-jarjar",   // apex -> system
		"framework-connectivity-t-pre-jarjar", // apex -> system
	},
	"CtsLeanbackJankTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"CtsLocalVoiceInteraction": {
		"CtsVoiceInteractionCommon",     // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
	},
	"CtsAutoFillServiceTestCases": {
		"TestParameterInjector",         // cts -> system
		"androidx.test.espresso.core",   // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctsdeviceutillegacy-axt",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppWidgetTestCases": {
		"android.appwidget.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"ctstestrunner-axt",                    // cts -> system
		"junit",                                // cts -> system
		"mockito-target-minus-junit4",          // cts -> system
	},
	"CtsSystemUiTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"permission-test-util-lib",      // cts -> system
	},
	"CtsBatterySavingTestCases": {
		"BatterySavingCtsCommon",            // cts -> system
		"android.os.flags-aconfig-java",     // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"flag-junit",                        // cts -> system
		"framework",                         // cts -> system
		"mockito-target-minus-junit4",       // cts -> system
		"platformprotosnano",                // cts -> system
		"truth",                             // cts -> system
	},
	"CtsBiometricsDeviceApp": {
		"compatibility-device-util-axt", // cts -> system
		"cts-input-lib",                 // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsTelecomCujTestCases": {
		"TelecomTestAppUtilsLib",        // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"flag-junit",                    // cts -> system
		"telecom_flags_core_java_lib",   // cts -> system
		"telephony_flags_core_java_lib", // cts -> system
	},
	"CtsWebkitTestCases": {
		"CtsWebkitTestCasesSharedWithSdk", // cts -> system
	},
	"CtsScopedStorageTestAppE30": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsScopedStorageTestAppSystemGalleryBypassDB": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsScopedStorageTestAppE30FileManager": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsScopedStorageGeneralTestOnlyApp": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsScopedStorageTestAppE": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsScopedStorageTestAppFileManagerBypassDB": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsScopedStorageTestAppSystemGallery30BypassDB": {
		"cts-scopedstorage-lib", // cts -> system
		"ext",                   // cts -> system
		"framework",             // cts -> system
	},
	"CtsInputHostTestHelperApp": {
		"cts-input-lib",     // cts -> system
		"ctstestrunner-axt", // cts -> system
		"truth",             // cts -> system
	},
	"CtsDeviceStateManagerTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_window_jetpack_utils",      // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
	},
	"CtsAccessibilityTestCases": {
		"CtsAccessibilityCommon",                        // cts -> system
		"CtsAccessibilityServiceUtils",                  // cts -> system
		"android.view.accessibility.flags-aconfig-java", // cts -> system
		"ctstestrunner-axt",                             // cts -> system
		"hamcrest-library",                              // cts -> system
	},
	"CtsAdbHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsWindowManagerDeviceAm": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsHealthFitnessDeviceTestCasesHistoricAccessLimitWithPermission": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build",           // cts -> system
		"testng",                        // cts -> system
	},
	"CtsDeviceIdleHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsWindowManagerDeviceKeyguard": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsRollbackManagerTestCases": {
		"android.content.pm.flags-aconfig-java",    // cts -> system
		"android.crashrecovery.flags-aconfig-java", // cts -> system
		"cts-install-lib",                          // cts -> system
		"cts-rollback-lib",                         // cts -> system
		"flag-junit",                               // cts -> system
	},
	"CtsWindowManagerJetpackSignedApp": {
		"compatibility-device-util-axt", // cts -> system
		"cts_window-extensions",         // cts -> system
		"cts_window-sidecar",            // cts -> system
		"cts_window_jetpack_utils",      // cts -> system
	},
	"CtsBackgroundActivityAppB": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
	},
	"CtsBackgroundActivityAppA33": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
	},
	"CtsPermissionMultiDeviceTestCases": {
		"android.companion.virtual.flags-aconfig-java", // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"ctstestrunner-axt",                            // cts -> system
		"flag-junit",                                   // cts -> system
		"modules-utils-build_system",                   // cts -> system
		"permission-multidevice-test-util-lib",         // cts -> system
		"permission-test-util-lib",                     // cts -> system
	},
	"cts-dalvik-host-test-runner": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"vogarexpect-no-deps",     // cts -> system
	},
	"CtsThemeHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsWifiBroadcastsHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsDynamicMimeSingleAppRebootHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsCompilationTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"guava",                   // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"service-connectivity": {
		"ext",                   // apex -> system
		"framework",             // apex -> system
		"libprotobuf-java-nano", // apex -> apex
	},
	"CtsStagedInstallHostTestCases": {
		"cts-install-lib-host", // cts -> system
		"cts-shim-host-lib",    // cts -> system
		"cts-tradefed",         // cts -> system
		"hamcrest",             // cts -> system
		"hamcrest-library",     // cts -> system
		"tradefed",             // cts -> system
		"truth",                // cts -> system
	},
	"CtsUiAutomationTestCases": {
		"CtsAccessibilityCommon",       // cts -> system
		"CtsAccessibilityServiceUtils", // cts -> system
		"ctstestrunner-axt",            // cts -> system
		"truth",                        // cts -> system
	},
	"CtsADPFHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"cts_adpf_common",         // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsDeqpTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsUsageStatsTestCases": {
		"android.app.usage.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"ctstestrunner-axt",                    // cts -> system
		"ext",                                  // cts -> system
		"flag-junit",                           // cts -> system
		"framework",                            // cts -> system
		"junit",                                // cts -> system
		"permission-test-util-lib",             // cts -> system
		"sts-device-util",                      // cts -> system
	},
	"CtsDynamicMimeIndependentGroupRebootHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsDynamicMimeHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsJvmtiAttachingHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsBackupHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"platformprotos",          // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsApexTestCases": {
		"cts-shim-host-lib", // cts -> system
		"cts-tradefed",      // cts -> system
		"tradefed",          // cts -> system
	},
	"CtsWindowManagerDeviceDisplay": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsWindowManagerDeviceInsets": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsPackageSettingHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsGwpAsanTestCases": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsSurfaceControlTests": {
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-hardware-lib",                             // cts -> system
		"cts-input-lib",                                // cts -> system
		"ctsdeviceutillegacy-axt",                      // cts -> system
		"ctstestrunner-axt",                            // cts -> system
		"flag-junit",                                   // cts -> system
		"hwui_flags_java_lib",                          // cts -> system
		"junit-params",                                 // cts -> system
		"mockito-target-minus-junit4",                  // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsGpuProfilingDataTestCases": {
		"compatibility-host-util",        // cts -> system
		"cts-tradefed",                   // cts -> system
		"host-libprotobuf-java-full",     // cts -> system
		"perfetto_config-full",           // cts -> system
		"perfetto_trace-full",            // cts -> system
		"platform-test-annotations-host", // cts -> system
		"tradefed",                       // cts -> system
	},
	"CtsWindowManagerDeviceTaskFragment": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsDumpsysHostTestCases": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
		"truth",        // cts -> system
	},
	"CtsJdwpSecurityHostTestCases": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsHarmfulAppWarningHostTestCases": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsSettingsHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"guava",                   // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsViewTestCases": {
		"android.view.flags-aconfig-java",             // cts -> system
		"android.view.inputmethod.flags-aconfig-java", // cts -> system
		"com.android.input.flags-aconfig-java",        // cts -> system
		"compatibility-device-util-axt",               // cts -> system
		"cts-hardware-lib",                            // cts -> system
		"cts-input-lib",                               // cts -> system
		"ctsdeviceutillegacy-axt",                     // cts -> system
		"ctstestrunner-axt",                           // cts -> system
		"flag-junit",                                  // cts -> system
		"junit-params",                                // cts -> system
		"mockito-target-minus-junit4",                 // cts -> system
		"truth",                                       // cts -> system
		"ui-trace-collector",                          // cts -> system
	},
	"CtsAdbManagerHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsUsbTests": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsSensitiveContentProtectionTestCases": {
		"android.view.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",   // cts -> system
		"ctstestrunner-axt",               // cts -> system
		"flag-junit",                      // cts -> system
		"testng",                          // cts -> system
		"truth",                           // cts -> system
	},
	"CtsMemunreachableTestCases": {
		"compatibility-host-util",     // cts -> system
		"compatibility-host-util-axt", // cts -> system
		"cts-tradefed",                // cts -> system
		"tradefed",                    // cts -> system
		"truth",                       // cts -> system
	},
	"compatibility-host-telephony-preconditions": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsJdwpTunnelHostTestCases": {
		"cts-tradefed", // cts -> system
		"jdi-support",  // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsDynamicMimeComplexFilterClearGroupRebootHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsClassloaderSplitsHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsWindowManagerDeviceWindow": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsAtraceHostTestCases": {
		"cts-tradefed",   // cts -> system
		"tradefed",       // cts -> system
		"trebuchet-core", // cts -> system
	},
	"CtsWindowManagerDeviceActivity": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsAppBindingHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"guava",                   // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsDeleteKeepDataHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsDynamicMimePreferredActivitiesHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsWindowManagerDeviceBackNavigation": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsInputTestCases": {
		"CtsVirtualDeviceCommonLib",               // cts -> system
		"android.view.flags-aconfig-java",         // cts -> system
		"com.android.hardware.input-aconfig-java", // cts -> system
		"com.android.input.flags-aconfig-java",    // cts -> system
		"compatibility-device-util-axt",           // cts -> system
		"cts-input-lib",                           // cts -> system
		"flag-junit",                              // cts -> system
	},
	"CtsBootStatsTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"framework-protos",        // cts -> system
		"libprotobuf-java-full",   // cts -> system
		"platformprotos",          // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsAngleIntegrationHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsUsesNativeLibraryTest": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsHardwareTestCases": {
		"CtsVirtualDeviceCommonLib",                      // cts -> system
		"android.companion.virtual.flags-aconfig-java",   // cts -> system
		"android.hardware.biometrics.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                  // cts -> system
		"cts-input-lib",                                  // cts -> system
		"cts-kernelinfo-lib",                             // cts -> system
		"ctstestrunner-axt",                              // cts -> system
		"flag-junit",                                     // cts -> system
		"junit",                                          // cts -> system
		"junit-params",                                   // cts -> system
		"mockito-target-minus-junit4",                    // cts -> system
	},
	"CtsSignedConfigHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"guava",                   // cts -> system
		"hamcrest-library",        // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsHdmiCecHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsBRSTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsDexMetadataHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsMultiUserHostTestCases": {
		"compatibility-host-util",        // cts -> system
		"cts-tradefed",                   // cts -> system
		"platform-test-annotations-host", // cts -> system
		"tradefed",                       // cts -> system
	},
	"CtsSeccompHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"compatibility-host-provider-preconditions": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsWindowManagerDeviceOther": {
		"CtsAccessibilityCommon",                       // cts -> system
		"android.app.flags-aconfig-java",               // cts -> system
		"android.permission.flags-aconfig-java",        // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"cts-input-lib",                                // cts -> system
		"cts-wm-overlayapp-base",                       // cts -> system
		"cts-wm-shared",                                // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
		"hamcrest-library",                             // cts -> system
		"metrics-helper-lib",                           // cts -> system
		"platform-compat-test-rules",                   // cts -> system
		"truth",                                        // cts -> system
		"ui-trace-collector",                           // cts -> system
	},
	"CtsTestHarnessModeTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsJvmtiRunTest903HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest930HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsBiometricsTestCases": {
		"android.hardware.biometrics.flags-aconfig-java", // cts -> system
		"com_android_systemui_flags_lib",                 // cts -> system
		"compatibility-device-util-axt",                  // cts -> system
		"cts-input-lib",                                  // cts -> system
		"ctstestrunner-axt",                              // cts -> system
		"flag-junit",                                     // cts -> system
		"mockito-target-minus-junit4",                    // cts -> system
		"platformprotosnano",                             // cts -> system
	},
	"CtsJvmtiRunTest1925HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1958HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest944HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsDynamicMimeRemoveRebootHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsDisplayTestCases": {
		"android.companion.virtual.flags-aconfig-java", // cts -> system
		"android.hardware.flags-aconfig-java",          // cts -> system
		"compatibility-device-util-axt",                // cts -> system
		"flag-junit",                                   // cts -> system
	},
	"CtsUsesLibraryHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsIncrementalInstallHostTestCases": {
		"compatibility-host-util",             // cts -> system
		"cts-tradefed",                        // cts -> system
		"guava",                               // cts -> system
		"incremental-install-common-host-lib", // cts -> system
		"tradefed",                            // cts -> system
	},
	"CtsSampleHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsMediaPerformanceClassTestCases": {
		"MediaPerformanceClassCommon",   // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctsmediav2common",              // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"mediapc-requirements",          // cts -> system
	},
	"CtsDynamicMimeChangedGroupAppUpdateHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsHostsideNumberBlockingTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsHostsideNetworkTests": {
		"cts-tradefed",                       // cts -> system
		"modules-utils-build-testing",        // cts -> system
		"net-tests-utils-host-device-common", // cts -> system
		"tradefed",                           // cts -> system
	},
	"CtsFileSystemTestCases": {
		"MediaPerformanceClassCommon",   // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"mediapc-requirements",          // cts -> system
	},
	"CtsCredentialManagerHostSideTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"platformprotos",          // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsSustainedPerformanceHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsHostsideNetworkPolicyTests": {
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsThreadLocalRandomHostTest": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsGpuMetricsHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsEdiHostTestCases": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"hamcrest-library",            // cts -> system
		"modules-utils-build-testing", // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsDynamicMimeComplexFilterRebootHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsInputMethodTestLauncher": {
		"cts-inputmethod-util", // cts -> system
	},
	"CtsDynamicMimeSingleAppGroupRebootHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsInputMethodStandaloneTestApp": {
		"compatibility-device-util-axt", // cts -> system
		"cts-inputmethod-util",          // cts -> system
	},
	"CtsHealthFitnessDeviceTestCasesNoPermission": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
	},
	"CtsHealthFitnessShowMigrationInfoIntentAbsentTests": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build",           // cts -> system
		"testng",                        // cts -> system
	},
	"CtsMediaCUJLargeTest": {
		"androidx.media3.media3-common",         // cts -> system
		"androidx.media3.media3-exoplayer",      // cts -> system
		"androidx.media3.media3-exoplayer-dash", // cts -> system
		"androidx.media3.media3-ui",             // cts -> system
		"ctsmediacujcommon",                     // cts -> system
		"ctsmediav2common",                      // cts -> system
		"ctstestrunner-axt",                     // cts -> system
	},
	"CtsHealthFitnessDeviceTestCasesRateLimiter": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build",           // cts -> system
		"testng",                        // cts -> system
	},
	"CtsMediaCUJSmallTest": {
		"androidx.media3.media3-common",    // cts -> system
		"androidx.media3.media3-exoplayer", // cts -> system
		"androidx.media3.media3-ui",        // cts -> system
		"ctsmediacujcommon",                // cts -> system
		"ctstestrunner-axt",                // cts -> system
	},
	"CtsHealthFitnessDeviceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build",           // cts -> system
		"testng",                        // cts -> system
	},
	"CtsSensorTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-sensors-tests",             // cts -> system
		"ctstestrunner-axt",             // cts -> system
	},
	"SdkSandboxManagerTests": {
		"CtsMediationTestSdkApi",                       // cts -> system
		"CtsSdkProviderApi",                            // cts -> system
		"SdkSandboxTestUtils",                          // cts -> system
		"android.content.pm.flags-aconfig-java",        // cts -> system
		"androidx.test.runner",                         // cts -> system
		"compatibility-device-util-axt-minus-dexmaker", // cts -> system
		"ext",                   // cts -> system
		"flag-junit",            // cts -> system
		"framework",             // cts -> system
		"modules-utils-build",   // cts -> system
		"sdk_sandbox_flags_lib", // cts -> system
		"truth",                 // cts -> system
	},
	"CtsBackgroundActivityAppB33": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
	},
	"CtsBackgroundActivityAppC33": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
	},
	"CtsBackgroundActivityAppAllowCrossUidFlagDefault": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
		"ext",                      // cts -> system
		"framework",                // cts -> system
	},
	"CtsBackgroundActivityAppC": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
	},
	"CtsBackgroundActivityAppA": {
		"bal-testapp",              // cts -> system
		"cts_window-extensions",    // cts -> system
		"cts_window-sidecar",       // cts -> system
		"cts_window_jetpack_utils", // cts -> system
	},
	"CtsCarrierApiTargetPrep": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsPackageManagerMultiUserHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsAppUsageHostTestCases": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsGpuToolsHostTestCases": {
		"compatibility-host-util",        // cts -> system
		"cts-tradefed",                   // cts -> system
		"platform-test-annotations-host", // cts -> system
		"tradefed",                       // cts -> system
	},
	"CtsAbiOverrideHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsJvmtiRunTest992HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsScopedStorageBypassDatabaseOperationsTest": {
		"cts-scopedstorage-lib",      // cts -> system
		"modules-utils-build_system", // cts -> system
		"truth",                      // cts -> system
	},
	"CtsJvmtiRunTest1975HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsFgsBootCompletedTestCasesApi35": {
		"CtsAppTestStubsShared",       // cts -> system
		"am_flags_lib",                // cts -> system
		"ctstestrunner-axt",           // cts -> system
		"ext",                         // cts -> system
		"flag-junit",                  // cts -> system
		"framework",                   // cts -> system
		"mockito-target-minus-junit4", // cts -> system
	},
	"CtsLibcoreOjTestCases": {
		"libcore-expectations-knownfailures-jar", // cts -> system
		"testng",                                 // cts -> system
	},
	"CtsLibcoreWycheproofConscryptTestCases": {
		"cts-core-test-runner-axt",               // cts -> system
		"ext",                                    // cts -> system
		"framework",                              // cts -> system
		"libcore-expectations-knownfailures-jar", // cts -> system
		"wycheproof",                             // cts -> system
	},
	"CtsLibcoreWycheproofBCTestCases": {
		"bouncycastle-repackaged-unbundled",      // cts -> system
		"cts-core-test-runner-axt",               // cts -> system
		"libcore-expectations-knownfailures-jar", // cts -> system
		"wycheproof",                             // cts -> system
	},
	"CtsIcuTestCases": {
		"cts-core-test-runner-axt", // cts -> system
	},
	"CtsLibcoreJsr166TestCases": {
		"cts-core-test-runner-axt",               // cts -> system
		"libcore-expectations-knownfailures-jar", // cts -> system
	},
	"CtsLibcoreTestCases": {
		"conscrypt-support",                      // cts -> system
		"cts-core-test-runner-axt",               // cts -> system
		"ext",                                    // cts -> system
		"framework",                              // cts -> system
		"libcore-expectations-knownfailures-jar", // cts -> system
		"libcore-expectations-virtualdeviceknownfailures-jar", // cts -> system
		"mockito-target-minus-junit4",                         // cts -> system
	},
	"CtsLibcoreOkHttpTestCases": {
		"bouncycastle-unbundled",                 // cts -> system
		"cts-core-test-runner-axt",               // cts -> system
		"libcore-expectations-knownfailures-jar", // cts -> system
		"okhttp-nojarjar",                        // cts -> system
		"okhttp-tests-nojarjar",                  // cts -> system
	},
	"CtsJvmtiRunTest1937HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest947HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1923HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1984HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest993HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest920HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1942HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1990HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest923HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest918HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest2002HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest997HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1910HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest919HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest922HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1953HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsSdkSandboxWebkitTestCases": {
		"CtsSdkSandboxTestScenario",     // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctswebkitsharedenv",            // cts -> system
	},
	"CtsJvmtiRunTest986HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1927HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest914HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest912HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1924HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1933HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest904HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1917HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest905HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1913HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest984HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest927HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1989HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1994HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1983HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1901HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1936HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRedefineClassesHostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1988HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1974HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1926HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1928HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest907HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest983HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1922HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1900HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1908HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1971HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1902HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsMediaBitstreamsTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-host-utils",          // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsVirtualDevicesAppLaunchTestCases": {
		"CtsVirtualDeviceCommonLib", // cts -> system
		"ext",                       // cts -> system
		"framework",                 // cts -> system
	},
	"CtsJvmtiRunTest1939HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest985HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest989HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1981HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest931HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest910HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest2007HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest2004HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest924HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest995HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1921HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiTrackingHostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1930HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest994HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest982HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest940HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest945HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest2005HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest996HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest2001HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1911HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest902HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1943HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1976HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest951HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1940HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1969HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1931HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1979HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1941HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1904HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest990HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsWindowManagerJetpackTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts_window-extensions",         // cts -> system
		"cts_window-extensions-core",    // cts -> system
		"cts_window-sidecar",            // cts -> system
	},
	"CtsJvmtiRunTest1998HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest2003HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1934HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest911HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1912HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest917HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1992HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1962HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1996HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest928HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1995HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest913HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsPrintTestCases": {
		"print-test-util-lib", // cts -> system
	},
	"CtsScopedStorageRedactUriTest": {
		"cts-scopedstorage-lib",      // cts -> system
		"modules-utils-build_system", // cts -> system
		"truth",                      // cts -> system
	},
	"CtsJvmtiRunTest1977HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest908HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiTaggingHostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsInstalledLoadingProgressHostTests": {
		"compatibility-host-util", // cts -> system
		"cts-host-utils",          // cts -> system
		"cts-install-lib-host",    // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsJvmtiRunTest915HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest942HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1906HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsPackageManagerPreferredActivityHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-host-utils",          // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsJvmtiRunTest1903HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsRollbackManagerHostTestCases": {
		"cts-install-lib-host", // cts -> system
		"cts-shim-host-lib",    // cts -> system
		"cts-tradefed",         // cts -> system
		"tradefed",             // cts -> system
		"truth",                // cts -> system
	},
	"CtsJvmtiRunTest1982HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1997HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1999HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1907HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsDomainVerificationHostTestCases": {
		"CtsDomainVerificationJavaConstantsLibraryHost", // cts -> system
		"compatibility-host-util",                       // cts -> system
		"cts-host-utils",                                // cts -> system
		"cts-tradefed",                                  // cts -> system
		"tradefed",                                      // cts -> system
	},
	"CtsJvmtiRunTest1914HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1915HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1932HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1978HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsExtractNativeLibsHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-host-utils",          // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsJvmtiRunTest1909HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1970HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsCallLogTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-host-utils",          // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsVideoQualityFloorHostTestCases": {
		"cts-host-utils",    // cts -> system
		"cts-shim-host-lib", // cts -> system
		"cts-tradefed",      // cts -> system
		"tradefed",          // cts -> system
	},
	"CtsJvmtiRunTest906HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest932HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsInstallHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-shim-host-lib",       // cts -> system
		"cts-tradefed",            // cts -> system
		"hamcrest",                // cts -> system
		"hamcrest-library",        // cts -> system
		"tradefed",                // cts -> system
		"truth",                   // cts -> system
	},
	"CtsJvmtiRunTest988HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1916HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsVideoEncodingQualityHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-host-utils",          // cts -> system
		"cts-shim-host-lib",       // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsAppCloningHostTest": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"testng",                      // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsAppCloningIntentRedirectionTest": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"testng",                      // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsScopedStorageGeneralTest": {
		"cts-scopedstorage-lib",      // cts -> system
		"modules-utils-build_system", // cts -> system
		"truth",                      // cts -> system
	},
	"CtsAppCloningContactsSharingTest": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"testng",                      // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsJvmtiRunTest2006HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1920HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest991HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1991HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest926HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1967HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsJvmtiRunTest1968HostTestCases": {
		"CtsJvmtiHostTestBase", // cts -> system
	},
	"CtsAccessibilityServiceTestCases": {
		"CtsAccessibilityCommon",                        // cts -> system
		"CtsVirtualDeviceCommonLib",                     // cts -> system
		"android.companion.virtual.flags-aconfig-java",  // cts -> system
		"android.view.accessibility.flags-aconfig-java", // cts -> system
		"com.android.window.flags.window-aconfig-java",  // cts -> system
		"com_android_server_accessibility_flags_lib",    // cts -> system
		"compatibility-device-util-axt",                 // cts -> system
		"ctstestrunner-axt",                             // cts -> system
		"flag-junit",                                    // cts -> system
		"hamcrest-library",                              // cts -> system
		"mockito-target-minus-junit4",                   // cts -> system
		"sts-device-util",                               // cts -> system
	},
	"CtsTelephonyTestCases": {
		"CtsTelecomUtilLib",             // cts -> system
		"android.telephony.mockmodem",   // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"cts-tradefed",                  // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"flag-junit",                    // cts -> system
		"framework",                     // cts -> system
		"hamcrest-library",              // cts -> system
		"telecom_flags_core_java_lib",   // cts -> system
		"telephony-common",              // cts -> system
		"telephony_flags_core_java_lib", // cts -> system
		"truth",                         // cts -> system
		"voip-common",                   // cts -> system
	},
	"CtsScopedStorageDeviceOnlyTest": {
		"cts-scopedstorage-lib",        // cts -> system
		"flag-junit",                   // cts -> system
		"mediaprovider_flags_java_lib", // cts -> system
		"modules-utils-build_system",   // cts -> system
		"truth",                        // cts -> system
	},
	"CtsBackupTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ctstestserver",                 // cts -> system
		"device-time-shell-utils",       // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"permission-test-util-lib",      // cts -> system
		"testng",                        // cts -> system
	},
	"CtsShortcutHostTestCases": {
		"cts-tradefed", // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsCarBuiltinApiHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsSecurityTestCases": {
		"NeneInternal",                            // cts -> system
		"android-common",                          // cts -> system
		"android.security.flags-aconfig-java",     // cts -> system
		"androidx.test.runner",                    // cts -> system
		"compatibility-common-util-devicesidelib", // cts -> system
		"compatibility-device-util-axt",           // cts -> system
		"cts-install-lib",                         // cts -> system
		"ctstestrunner-axt",                       // cts -> system
		"ctstestserver",                           // cts -> system
		"ext",                                     // cts -> system
		"flag-junit",                              // cts -> system
		"framework",                               // cts -> system
		"guava",                                   // cts -> system
		"hamcrest-library",                        // cts -> system
		"permission-test-util-lib",                // cts -> system
		"sts-device-util",                         // cts -> system
	},
	"CtsAdExtServicesHostTests": {
		"adservices-host-side-test-utility", // cts -> system
		"cts-statsd-atom-host-test-utils",   // cts -> system
		"cts-tradefed",                      // cts -> system
		"platformprotos",                    // cts -> system
		"tradefed",                          // cts -> system
	},
	"CtsClasspathsTestCases": {
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsPackageManagerIncrementalStatsHostTestCases": {
		"android.content.pm.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",                    // cts -> system
		"cts-host-utils",                             // cts -> system
		"cts-statsd-atom-host-test-utils",            // cts -> system
		"cts-tradefed",                               // cts -> system
		"flag-junit-host",                            // cts -> system
		"tradefed",                                   // cts -> system
	},
	"CtsStrictJavaPackagesTestCases": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"smali-dexlib2-no-guava",      // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsBlobStoreHostTestCases": {
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsDevicePolicyManagerTestCases": {
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"guava",                           // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsMediaParserHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsLocationTimeZoneManagerHostTest": {
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"host-time-shell-utils",           // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsHostsideHiddenapiTests": {
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"host-libprotobuf-java-full",      // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsStatsdHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"core_cts_test_resources",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"host-libprotobuf-java-full",      // cts -> system
		"perfetto_config-full",            // cts -> system
		"platformprotos",                  // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsProviderTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"junit",                         // cts -> system
		"mockito-target-minus-junit4",   // cts -> system
		"sts-device-util",               // cts -> system
		"telephony-common",              // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsNNAPIStatsdAtomHostTestCases": {
		"cts-statsd-atom-host-test-utils", // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsPackageManagerStatsHostTestCases": {
		"android.content.pm.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",                    // cts -> system
		"cts-host-utils",                             // cts -> system
		"cts-statsd-atom-host-test-utils",            // cts -> system
		"cts-tradefed",                               // cts -> system
		"flag-junit-host",                            // cts -> system
		"tradefed",                                   // cts -> system
	},
	"CtsAppSearchHostTestCases": {
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsAdServicesHostTests": {
		"adservices-host-side-test-utility", // cts -> system
		"cts-statsd-atom-host-test-utils",   // cts -> system
		"cts-tradefed",                      // cts -> system
		"platformprotos",                    // cts -> system
		"tradefed",                          // cts -> system
	},
	"CtsInputHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"hamcrest-library",                // cts -> system
		"host-libprotobuf-java-full",      // cts -> system
		"platformprotos",                  // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsLocaleManagerHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"host-libprotobuf-java-full",      // cts -> system
		"tradefed",                        // cts -> system
	},
	"MicrodroidHostTestCases": {
		"MicrodroidHostTestHelper",        // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"microdroid_payload_metadata",     // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsGrammaticalInflectionHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"host-libprotobuf-java-full",      // cts -> system
		"platformprotos",                  // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsHostsideTvTests": {
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsClasspathDeviceInfoTestCases": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"junit",                       // cts -> system
		"modules-utils-build-testing", // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsBiometricsHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"host-libprotobuf-java-full",      // cts -> system
		"platformprotos",                  // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsCarHostTestCases": {
		"android.car.feature-aconfig-java-host", // cts -> system
		"car-cts-host-util",                     // cts -> system
		"compatibility-host-util",               // cts -> system
		"cts-statsd-atom-host-test-utils",       // cts -> system
		"cts-tradefed",                          // cts -> system
		"flag-junit-host",                       // cts -> system
		"libprotobuf-java-full",                 // cts -> system
		"tradefed",                              // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsSensorRatePermissionTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-sensors-tests",             // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCarHostNonRecoverableTestCases": {
		"car-cts-host-util",               // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsAccountsHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-tradefed",                    // cts -> system
		"device-policy-log-verifier-util", // cts -> system
		"platform-test-annotations-host",  // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsPackageManagerParsingHostTestCases": {
		"CtsPackageManagerParsingAnnotationProcessorApi", // cts -> system
		"compatibility-host-util",                        // cts -> system
		"cts-host-utils",                                 // cts -> system
		"cts-tradefed",                                   // cts -> system
		"tradefed",                                       // cts -> system
	},
	"CtsInputMethodTestCases32": {
		"compatibility-device-util-axt", // cts -> system
		"cts-inputmethod-util",          // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
	},
	"CtsPermissionTestCases": {
		"CtsAccessibilityCommon",                // cts -> system
		"CtsVirtualDeviceCommonLib",             // cts -> system
		"android-ex-camera2",                    // cts -> system
		"android.permission.flags-aconfig-java", // cts -> system
		"bluetooth-test-util-lib",               // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"guava",                                 // cts -> system
		"nativetesthelper",                      // cts -> system
		"permission-test-util-lib",              // cts -> system
		"platform-test-rules",                   // cts -> system
		"platformprotosnano",                    // cts -> system
		"safety-center-internal-data",           // cts -> system
		"sts-device-util",                       // cts -> system
		"testng",                                // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsInputMethodTestCases": {
		"android.view.inputmethod.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",               // cts -> system
		"cts-inputmethod-util",                        // cts -> system
		"cts_window-extensions",                       // cts -> system
		"ctstestrunner-axt",                           // cts -> system
		"flag-junit",                                  // cts -> system
		"ravenwood-junit",                             // cts -> system
		"statsdprotonano",                             // cts -> system
		"testng",                                      // cts -> system
	},
	"CtsPermissionUiTestCases": {
		"CtsAccessibilityCommon",                // cts -> system
		"android.permission.flags-aconfig-java", // cts -> system
		"bluetooth-test-util-lib",               // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"flag-junit",                            // cts -> system
		"modules-utils-build_system",            // cts -> system
		"permission-test-util-lib",              // cts -> system
		"platform-test-rules",                   // cts -> system
		"sts-device-util",                       // cts -> system
	},
	"CtsHealthConnectHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsNotificationExtendersTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"cts-tradefed",               // cts -> system
		"guava",                      // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsSyncContentHostTestCases": {
		"compatibility-host-util",         // cts -> system
		"cts-tradefed",                    // cts -> system
		"device-policy-log-verifier-util", // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsCarrierApiTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-tradefed",                  // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsSilentUpdateHostTestCases": {
		"compatibility-host-util", // cts -> system
		"cts-tradefed",            // cts -> system
		"tradefed",                // cts -> system
	},
	"CtsTelephonyProviderHostCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"compatibility-host-util",    // cts -> system
		"cts-tradefed",               // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsHostsideWebViewTests": {
		"CompatChangeGatingTestBase", // cts -> system
		"compatibility-host-util",    // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsSecurityHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"compatibility-host-util",    // cts -> system
		"cts-host-utils",             // cts -> system
		"cts-tradefed",               // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsAppCompatHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"cts-tradefed",               // cts -> system
		"guava",                      // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsMediaHostTestCases": {
		"CompatChangeGatingTestBase",                               // cts -> system
		"CtsMediaHostTestCommon",                                   // cts -> system
		"com.android.media.flags.bettertogether-aconfig-java-host", // cts -> system
		"compatibility-host-util",                                  // cts -> system
		"cts-host-utils",                                           // cts -> system
		"cts-statsd-atom-host-test-utils",                          // cts -> system
		"cts-tradefed",                                             // cts -> system
		"flag-junit-host",                                          // cts -> system
		"modules-utils-build-testing",                              // cts -> system
		"tradefed",                                                 // cts -> system
	},
	"CtsTaggingHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"HarrierCommon",              // cts -> system
		"compatibility-host-util",    // cts -> system
		"cts-tradefed",               // cts -> system
		"flag-junit-host",            // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsVoiceInteractionHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"cts-tradefed",               // cts -> system
		"guava",                      // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsTelephonyHostCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"compatibility-host-util",    // cts -> system
		"cts-tradefed",               // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsSystemUiHostTestCases": {
		"CompatChangeGatingTestBase",      // cts -> system
		"compatibility-host-util",         // cts -> system
		"core_cts_test_resources",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"platformprotos",                  // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsBootDisplayModeTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"cts-tradefed",               // cts -> system
		"guava",                      // cts -> system
		"tradefed",                   // cts -> system
	},
	"CtsTelecomHostCases": {
		"CompatChangeGatingTestBase",      // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"tradefed",                        // cts -> system
	},
	"CtsInputMethodServiceHostTestCases": {
		"CompatChangeGatingTestBase",         // cts -> system
		"compatibility-host-util",            // cts -> system
		"cts-inputmethodservice-common-host", // cts -> system
		"cts-tradefed",                       // cts -> system
		"tradefed",                           // cts -> system
	},
	"CtsWindowManagerBackgroundActivityTestCases": {
		"android.security.flags-aconfig-java",          // cts -> system
		"com.android.window.flags.window-aconfig-java", // cts -> system
		"cts-core-test-runner-axt",                     // cts -> system
		"cts_window-extensions",                        // cts -> system
		"cts_window-sidecar",                           // cts -> system
		"cts_window_jetpack_utils",                     // cts -> system
	},
	"CtsScopedStorageCoreHostTest": {
		"cts-tradefed", // cts -> system
		"testng",       // cts -> system
		"tradefed",     // cts -> system
	},
	"CtsSettingsTestCases": {
		"Harrier", // cts -> system
		"Nene",    // cts -> system
		"TestApp", // cts -> system
		"compatibility-common-util-devicesidelib", // cts -> system
		"compatibility-device-util-axt",           // cts -> system
		"ctstestrunner-axt",                       // cts -> system
		"junit",                                   // cts -> system
		"truth",                                   // cts -> system
	},
	"CtsVoiceInteractionTestCases": {
		"ActivityContext",                // cts -> system
		"CtsAttentionServiceDevice",      // cts -> system
		"CtsSoundTriggerInstrumentation", // cts -> system
		"CtsVoiceInteractionCommon",      // cts -> system
		"Harrier",                        // cts -> system
		"Nene",                           // cts -> system
		"TestApp",                        // cts -> system
		"android.app.wearable.flags-aconfig-java", // cts -> system
		"android.permission.flags-aconfig-java",   // cts -> system
		"compatibility-device-util-axt",           // cts -> system
		"ctstestrunner-axt",                       // cts -> system
		"platform-compat-test-rules",              // cts -> system
		"testng",                                  // cts -> system
	},
	"CtsScopedStorageHostTest": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"testng",                      // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsContentTestCases": {
		"CtsContentUtils",                     // cts -> system
		"Harrier",                             // cts -> system
		"HelloWorldResHardeningLib",           // cts -> system
		"Nene",                                // cts -> system
		"ShortcutManagerTestUtils",            // cts -> system
		"TestParameterInjector",               // cts -> system
		"accountaccesslib",                    // cts -> system
		"android.security.flags-aconfig-java", // cts -> system
		"androidx.legacy_legacy-support-v4",   // cts -> system
		"apache-commons-compress",             // cts -> system
		"compatibility-device-util-axt",       // cts -> system
		"cts-install-lib",                     // cts -> system
		"ctstestrunner-axt",                   // cts -> system
		"ext",                                 // cts -> system
		"flag-junit",                          // cts -> system
		"framework",                           // cts -> system
		"junit",                               // cts -> system
		"platformprotosnano",                  // cts -> system
		"ravenwood-junit",                     // cts -> system
		"services.core",                       // cts -> system
		"testng",                              // cts -> system
		"truth",                               // cts -> system
	},
	"CtsCameraHeadlessSystemUserTestCases": {
		"CtsCameraUtils",                 // cts -> system
		"Harrier",                        // cts -> system
		"android-ex-camera2",             // cts -> system
		"camera_platform_flags_java_lib", // cts -> system
		"compatibility-device-util-axt",  // cts -> system
		"ctstestrunner-axt",              // cts -> system
		"flag-junit",                     // cts -> system
	},
	"CtsResourcesTestCases": {
		"CtsContentUtils",                        // cts -> system
		"Harrier",                                // cts -> system
		"HelloWorldResHardeningLib",              // cts -> system
		"Nene",                                   // cts -> system
		"ShortcutManagerTestUtils",               // cts -> system
		"accountaccesslib",                       // cts -> system
		"android.content.res.flags-aconfig-java", // cts -> system
		"androidx.legacy_legacy-support-v4",      // cts -> system
		"apache-commons-compress",                // cts -> system
		"compatibility-device-util-axt",          // cts -> system
		"cts-install-lib",                        // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"ext",                                    // cts -> system
		"framework",                              // cts -> system
		"junit",                                  // cts -> system
		"platformprotosnano",                     // cts -> system
		"ravenwood-junit",                        // cts -> system
		"services.core",                          // cts -> system
		"testng",                                 // cts -> system
		"truth",                                  // cts -> system
	},
	"CtsPackageManagerTestCases": {
		"CtsContentUtils",                       // cts -> system
		"Harrier",                               // cts -> system
		"HelloWorldResHardeningLib",             // cts -> system
		"Nene",                                  // cts -> system
		"ShortcutManagerTestUtils",              // cts -> system
		"accountaccesslib",                      // cts -> system
		"android.content.pm.flags-aconfig-java", // cts -> system
		"android.multiuser.flags-aconfig-java",  // cts -> system
		"android.security.flags-aconfig-java",   // cts -> system
		"androidx.legacy_legacy-support-v4",     // cts -> system
		"apache-commons-compress",               // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"cts-install-lib",                       // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"ext",                                   // cts -> system
		"flag-junit",                            // cts -> system
		"framework",                             // cts -> system
		"junit",                                 // cts -> system
		"platformprotosnano",                    // cts -> system
		"ravenwood-junit",                       // cts -> system
		"services.core",                         // cts -> system
		"testng",                                // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsMultiUserTestCases": {
		"Harrier",                              // cts -> system
		"Nene",                                 // cts -> system
		"TestApp",                              // cts -> system
		"android.multiuser.flags-aconfig-java", // cts -> system
		"android.os.flags-aconfig-java",        // cts -> system
		"compatibility-device-util-axt",        // cts -> system
		"ctstestrunner-axt",                    // cts -> system
		"flag-junit",                           // cts -> system
	},
	"CtsWindowManagerTestCases": {
		"CtsContentUtils",                   // cts -> system
		"Harrier",                           // cts -> system
		"HelloWorldResHardeningLib",         // cts -> system
		"Nene",                              // cts -> system
		"ShortcutManagerTestUtils",          // cts -> system
		"accountaccesslib",                  // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"apache-commons-compress",           // cts -> system
		"compatibility-device-util-axt",     // cts -> system
		"cts-install-lib",                   // cts -> system
		"ctstestrunner-axt",                 // cts -> system
		"ext",                               // cts -> system
		"framework",                         // cts -> system
		"junit",                             // cts -> system
		"platformprotosnano",                // cts -> system
		"ravenwood-junit",                   // cts -> system
		"services.core",                     // cts -> system
		"testng",                            // cts -> system
		"truth",                             // cts -> system
	},
	"CtsPersistentDataBlockManagerTestCases": {
		"Harrier",           // cts -> system
		"ctstestrunner-axt", // cts -> system
		"truth",             // cts -> system
	},
	"CtsSuspendAppsTestCases": {
		"Harrier",                       // cts -> system
		"Nene",                          // cts -> system
		"TestApp",                       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"platform-test-rules",           // cts -> system
	},
	"CtsDomainVerificationDeviceMultiUserTestCases": {
		"CtsDomainVerificationAndroidConstantsLibrary", // cts -> system
		"CtsDomainVerificationJavaConstantsLibrary",    // cts -> system
		"Harrier",                       // cts -> system
		"Nene",                          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"junit",                         // cts -> system
		"truth",                         // cts -> system
	},
	"CtsAppCloningDeviceTestCases": {
		"Harrier",                       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"junit",                         // cts -> system
	},
	"CtsCarBuiltinApiTestCases": {
		"Harrier",                          // cts -> system
		"android.car",                      // cts -> system
		"android.car.feature-aconfig-java", // cts -> system
		"android.car.test.utils",           // cts -> system
		"compatibility-device-util-axt",    // cts -> system
		"ctstestrunner-axt",                // cts -> system
	},
	"CtsAppEnumerationTestCases": {
		"CtsAppEnumerationTestLib", // cts -> system
		"Harrier",                  // cts -> system
		"android.security.flags-aconfig-java-host", // cts -> system
		"compatibility-device-util-axt",            // cts -> system
		"cts-install-lib",                          // cts -> system
		"ctstestrunner-axt",                        // cts -> system
		"hamcrest-library",                         // cts -> system
	},
	"CtsJobSchedulerTestCases": {
		"Harrier",                       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
	},
	"CtsPhotoPickerTest": {
		"Harrier",                      // cts -> system
		"cts-install-lib",              // cts -> system
		"framework-mediaprovider.impl", // cts -> system
		"modules-utils-build",          // cts -> system
	},
	"CtsPackageInstallTestCases": {
		"Harrier",                               // cts -> system
		"Nene",                                  // cts -> system
		"android.content.pm.flags-aconfig-java", // cts -> system
		"androidx.legacy_legacy-support-v4",     // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"cts-install-lib",                       // cts -> system
		"truth",                                 // cts -> system
	},
	"CtsAppCloningMediaProviderHostTest": {
		"compatibility-host-util",     // cts -> system
		"cts-tradefed",                // cts -> system
		"modules-utils-build-testing", // cts -> system
		"testng",                      // cts -> system
		"tradefed",                    // cts -> system
	},
	"CtsKeystoreTestCases": {
		"DeviceAdminApp",                        // cts -> system
		"Harrier",                               // cts -> system
		"Nene",                                  // cts -> system
		"android-key-attestation",               // cts -> system
		"android.security.flags-aconfig-java",   // cts -> system
		"bouncycastle-unbundled",                // cts -> system
		"compatibility-device-util-axt",         // cts -> system
		"core-tests-support",                    // cts -> system
		"cts-keystore-test-util",                // cts -> system
		"cts-keystore-user-auth-helper-library", // cts -> system
		"cts-security-test-support-library",     // cts -> system
		"ctstestrunner-axt",                     // cts -> system
		"ext",                                   // cts -> system
		"flag-junit",                            // cts -> system
		"framework",                             // cts -> system
		"guava",                                 // cts -> system
		"hamcrest-library",                      // cts -> system
		"junit",                                 // cts -> system
		"platformprotosnano",                    // cts -> system
		"testng",                                // cts -> system
	},
	"CtsAdminPackageInstallerTestCases": {
		"Harrier",                           // cts -> system
		"Nene",                              // cts -> system
		"androidx.legacy_legacy-support-v4", // cts -> system
		"cts-install-lib",                   // cts -> system
	},
	"CtsMediaAudioTestCases": {
		"CtsVirtualDeviceCommonLib",        // cts -> system
		"Harrier",                          // cts -> system
		"Nene",                             // cts -> system
		"android.media.audio-aconfig-java", // cts -> system
		"com.android.media.audioserver-aconfig-java", // cts -> system
		"compatibility-device-util-axt",              // cts -> system
		"cts-media-common",                           // cts -> system
		"ctstestrunner-axt",                          // cts -> system
		"ext",                                        // cts -> system
		"flag-junit",                                 // cts -> system
		"framework",                                  // cts -> system
		"hamcrest-library",                           // cts -> system
		"junit-params",                               // cts -> system
		"mediatestutils",                             // cts -> system
		"testng",                                     // cts -> system
	},
	"CtsInputMethodInstallTestCases": {
		"Harrier", // cts -> system
		"Nene",    // cts -> system
		"TestApp", // cts -> system
		"android.view.inputmethod.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",               // cts -> system
		"cts-inputmethod-util",                        // cts -> system
		"ctstestrunner-axt",                           // cts -> system
		"flag-junit",                                  // cts -> system
		"testng",                                      // cts -> system
	},
	"CtsPermissionMultiUserTestCases": {
		"Harrier",                       // cts -> system
		"Nene",                          // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build_system",    // cts -> system
	},
	"CtsOsTestCases": {
		"Harrier",                       // cts -> system
		"android.os.flags-aconfig-java", // cts -> system
		"backstage_power_flags_lib",     // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"device-time-shell-utils",       // cts -> system
		"flag-junit",                    // cts -> system
		"guava",                         // cts -> system
		"hamcrest-library",              // cts -> system
		"junit",                         // cts -> system
		"junit-params",                  // cts -> system
		"platformprotosnano",            // cts -> system
		"ravenwood-junit",               // cts -> system
		"sdk_sandbox_flags_lib",         // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsMediaBetterTogetherTestCases": {
		"Harrier",           // cts -> system
		"cts-media-common",  // cts -> system
		"ctstestrunner-axt", // cts -> system
		"ext",               // cts -> system
		"flag-junit",        // cts -> system
		"framework",         // cts -> system
		"truth",             // cts -> system
	},
	"CtsUserRestrictionTestCases": {
		"Harrier",              // cts -> system
		"Nene",                 // cts -> system
		"androidx.test.runner", // cts -> system
		"ext",                  // cts -> system
		"framework",            // cts -> system
		"junit",                // cts -> system
		"truth",                // cts -> system
	},
	"CtsAppSecurityHostTestCases": {
		"CompatChangeGatingTestBase",                                    // cts -> system
		"CtsAppSecurityUtils",                                           // cts -> system
		"CtsPkgInstallerConstants",                                      // cts -> system
		"android.security.flags-aconfig-java-host",                      // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",                                       // cts -> system
		"cts-host-utils",                                                // cts -> system
		"cts-statsd-atom-host-test-utils",                               // cts -> system
		"cts-tradefed",                                                  // cts -> system
		"flag-junit-host",                                               // cts -> system
		"hamcrest-library",                                              // cts -> system
		"sts-host-util",                                                 // cts -> system
		"tradefed",                                                      // cts -> system
		"truth",                                                         // cts -> system
	},
	"CtsOverlayHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsHealthConnectControllerTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-lib",         // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"cts-install-lib",               // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"testng",                        // cts -> system
	},
	"CtsOsHostTestCases": {
		"compatibility-host-util",     // cts -> system
		"compatibility-host-util-axt", // cts -> system
		"cts-tradefed",                // cts -> system
		"tradefed",                    // cts -> system
		"truth",                       // cts -> system
	},
	"CtsExerciseRouteTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-lib",         // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build_system",    // cts -> system
		"testng",                        // cts -> system
	},
	"CtsInstantAppsHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsAdoptableHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsPermissionsHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsHealthConnectHostSideDeviceTestCases": {
		"compatibility-device-util-axt", // cts -> system
		"cts-healthconnect-lib",         // cts -> system
		"cts-healthconnect-utils",       // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"modules-utils-build_system",    // cts -> system
		"testng",                        // cts -> system
	},
	"CtsPackageManagerHostTestCases": {
		"CompatChangeGatingTestBase",                                    // cts -> system
		"CtsAppSecurityUtils",                                           // cts -> system
		"CtsPkgInstallerConstants",                                      // cts -> system
		"android.content.pm.flags-aconfig-java-host",                    // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",                                       // cts -> system
		"cts-host-utils",                                                // cts -> system
		"cts-statsd-atom-host-test-utils",                               // cts -> system
		"cts-tradefed",                                                  // cts -> system
		"flag-junit-host",                                               // cts -> system
		"hamcrest-library",                                              // cts -> system
		"sts-host-util",                                                 // cts -> system
		"tradefed",                                                      // cts -> system
		"truth",                                                         // cts -> system
	},
	"CtsCorruptApkHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsAppDataIsolationHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsDirectBootHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsApkVerityInstallHostTestCases": {
		"CompatChangeGatingTestBase",                                    // cts -> system
		"CtsAppSecurityUtils",                                           // cts -> system
		"CtsPkgInstallerConstants",                                      // cts -> system
		"android.security.flags-aconfig-java-host",                      // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",                                       // cts -> system
		"cts-host-utils",                                                // cts -> system
		"cts-statsd-atom-host-test-utils",                               // cts -> system
		"cts-tradefed",                                                  // cts -> system
		"flag-junit-host",                                               // cts -> system
		"hamcrest-library",                                              // cts -> system
		"sts-host-util",                                                 // cts -> system
		"tradefed",                                                      // cts -> system
		"truth",                                                         // cts -> system
	},
	"CtsStorageHostTestCases": {
		"CompatChangeGatingTestBase",                                    // cts -> system
		"CtsAppSecurityUtils",                                           // cts -> system
		"CtsPkgInstallerConstants",                                      // cts -> system
		"android.app.usage.flags-aconfig-java-host",                     // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",                                       // cts -> system
		"cts-host-utils",                                                // cts -> system
		"cts-statsd-atom-host-test-utils",                               // cts -> system
		"cts-tradefed",                                                  // cts -> system
		"flag-junit-host",                                               // cts -> system
		"hamcrest-library",                                              // cts -> system
		"sts-host-util",                                                 // cts -> system
		"tradefed",                                                      // cts -> system
		"truth",                                                         // cts -> system
	},
	"CtsUseEmbeddedDexHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"CtsResumeOnRebootHostTestCases": {
		"CompatChangeGatingTestBase", // cts -> system
		"CtsAppSecurityUtils",        // cts -> system
		"CtsPkgInstallerConstants",   // cts -> system
		"com.android.internal.pm.pkg.component.flags-aconfig-java-host", // cts -> system
		"compatibility-host-util",         // cts -> system
		"cts-host-utils",                  // cts -> system
		"cts-statsd-atom-host-test-utils", // cts -> system
		"cts-tradefed",                    // cts -> system
		"flag-junit-host",                 // cts -> system
		"hamcrest-library",                // cts -> system
		"sts-host-util",                   // cts -> system
		"tradefed",                        // cts -> system
		"truth",                           // cts -> system
	},
	"vm-tests-tf": {
		"compatibility-host-vm-targetprep", // cts -> system
	},
	"CtsDevicePolicySimTestCases": {
		"ActivityContext",               // cts -> system
		"DeviceAdminApp",                // cts -> system
		"EventLib",                      // cts -> system
		"Harrier",                       // cts -> system
		"Interactive",                   // cts -> system
		"MetricsRecorder",               // cts -> system
		"TestApp",                       // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"cts-net-utils",                 // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"framework",                     // cts -> system
		"statsdprotolite",               // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsDevicePolicyTestCases": {
		"ActivityContext",               // cts -> system
		"DeviceAdminApp",                // cts -> system
		"EventLib",                      // cts -> system
		"Harrier",                       // cts -> system
		"Interactive",                   // cts -> system
		"MetricsRecorder",               // cts -> system
		"TestApp",                       // cts -> system
		"bedstead-flags",                // cts -> system
		"compatibility-device-util-axt", // cts -> system
		"cts-net-utils",                 // cts -> system
		"ctstestrunner-axt",             // cts -> system
		"ext",                           // cts -> system
		"flag-junit",                    // cts -> system
		"framework",                     // cts -> system
		"statsdprotolite",               // cts -> system
		"testng",                        // cts -> system
		"truth",                         // cts -> system
	},
	"CtsCredentialManagerTestCases": {
		"ActivityContext",                        // cts -> system
		"DeviceAdminApp",                         // cts -> system
		"EventLib",                               // cts -> system
		"Harrier",                                // cts -> system
		"Interactive",                            // cts -> system
		"MetricsRecorder",                        // cts -> system
		"TestApp",                                // cts -> system
		"android-common",                         // cts -> system
		"android-support-v4",                     // cts -> system
		"android.credentials.flags-aconfig-java", // cts -> system
		"compatibility-device-util-axt",          // cts -> system
		"cts-net-utils",                          // cts -> system
		"ctstestrunner-axt",                      // cts -> system
		"flag-junit",                             // cts -> system
		"mockito-target-minus-junit4",            // cts -> system
		"statsdprotolite",                        // cts -> system
		"testng",                                 // cts -> system
		"truth",                                  // cts -> system
	},
	"CtsStatsdAtomHostTestCases": {
		"android.hardware.usb.flags-aconfig-java-host", // cts -> system
		"android.os.flags-aconfig-java-host",           // cts -> system
		"compatibility-host-util",                      // cts -> system
		"core_cts_test_resources",                      // cts -> system
		"cts-statsd-atom-host-test-utils",              // cts -> system
		"cts-tradefed",                                 // cts -> system
		"flag-junit-host",                              // cts -> system
		"host-libprotobuf-java-full",                   // cts -> system
		"perfetto_config-full",                         // cts -> system
		"tradefed",                                     // cts -> system
		"truth",                                        // cts -> system
	},
	"sysuig": {
		"vendor-pixelatoms-java", // system -> vendor
	},
}
