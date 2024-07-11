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
	"AppInstalledOnMultipleUsers": {
		"framework", // cts -> unstable
	},

	"Bluetooth": {
		"app-compat-annotations",         // apex -> system
		"framework-bluetooth-pre-jarjar", // apex -> system
	},

	"CVE-2021-0965": {
		"framework", // cts -> unstable
	},

	"CameraExtensionsProxy": {
		"androidx.camera.extensions.impl", // system -> vendor
	},

	"CarServiceUpdatable": {
		"modules-utils-os",                    // apex -> apex
		"modules-utils-preconditions",         // apex -> apex
		"modules-utils-shell-command-handler", // apex -> apex
	},

	"CtsAdServicesNotInAllowListEndToEndTests": {
		"framework", // cts -> unstable
	},

	"CtsAdServicesPermissionsAppOptOutEndToEndTests": {
		"framework", // cts -> unstable
	},

	"CtsAdServicesPermissionsNoPermEndToEndTests": {
		"framework", // cts -> unstable
	},

	"CtsAdServicesPermissionsValidEndToEndTests": {
		"framework", // cts -> unstable
	},

	"CtsAdservicesHostTestApp": {
		"framework", // cts -> unstable
	},

	"CtsAlarmManagerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAndroidAppTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAppExitTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAppFgsStartTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAppFgsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAppOpsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAppSearchTestCases": {
		"framework", // cts -> unstable
	},

	"CtsAppTestStubsApp2": {
		"framework", // cts -> unstable
	},

	"CtsAudioHostTestApp": {
		"framework", // cts -> unstable
	},

	"CtsBRSTestCases": {
		"framework", // cts -> unstable
	},

	"CtsBackgroundActivityAppAllowCrossUidFlagDefault": {
		"framework", // cts -> unstable
	},

	"CtsBatterySavingTestCases": {
		"framework", // cts -> unstable
	},

	"CtsBluetoothTestCases": {
		"framework", // cts -> unstable
	},

	"CtsBootDisplayModeApp": {
		"framework", // cts -> unstable
	},

	"CtsBroadcastTestCases": {
		"framework", // cts -> unstable
	},

	"CtsCompanionDeviceManagerCoreTestCases": {
		"framework", // cts -> unstable
	},

	"CtsCompanionDeviceManagerMultiProcessTestCases": {
		"framework", // cts -> unstable
	},

	"CtsCompanionDeviceManagerUiAutomationTestCases": {
		"framework", // cts -> unstable
	},

	"CtsContentSuggestionsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsContentTestCases": {
		"framework", // cts -> unstable
	},

	"CtsCredentialManagerBackupRestoreApp": {
		"framework", // cts -> unstable
	},

	"CtsCrossProfileEnabledApp": {
		"framework", // cts -> unstable
	},

	"CtsCrossProfileEnabledNoPermsApp": {
		"framework", // cts -> unstable
	},

	"CtsCrossProfileNotEnabledApp": {
		"framework", // cts -> unstable
	},

	"CtsCrossProfileUserEnabledApp": {
		"framework", // cts -> unstable
	},

	"CtsDeviceAndProfileOwnerApp": {
		"framework", // cts -> unstable
	},

	"CtsDeviceAndProfileOwnerApp23": {
		"framework", // cts -> unstable
	},

	"CtsDeviceAndProfileOwnerApp25": {
		"framework", // cts -> unstable
	},

	"CtsDeviceAndProfileOwnerApp30": {
		"framework", // cts -> unstable
	},

	"CtsDeviceLockTestCases": {
		"framework", // cts -> unstable
	},

	"CtsDeviceOwnerApp": {
		"framework", // cts -> unstable
	},

	"CtsDevicePolicySimTestCases": {
		"framework", // cts -> unstable
	},

	"CtsDevicePolicyTestCases": {
		"framework", // cts -> unstable
	},

	"CtsDreamsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsDrmTestCases": {
		"framework", // cts -> unstable
	},

	"CtsEphemeralTestsEphemeralApp1": {
		"framework", // cts -> unstable
	},

	"CtsFgsBootCompletedTestCases": {
		"framework", // cts -> unstable
	},

	"CtsFgsBootCompletedTestCasesApi35": {
		"framework", // cts -> unstable
	},

	"CtsFgsStartTestHelperApi34": {
		"framework", // cts -> unstable
	},

	"CtsFgsStartTestHelperCurrent": {
		"framework", // cts -> unstable
	},

	"CtsFileDescriptorTestCases": {
		"framework", // cts -> unstable
	},

	"CtsHostsideCompatChangeTestsApp": {
		"framework", // cts -> unstable
	},

	"CtsHostsideNetworkPolicyTestsApp2": {
		"framework", // cts -> unstable
	},

	"CtsIdentityTestCases": {
		"framework", // cts -> unstable
	},

	"CtsIkeTestCases": {
		"framework", // cts -> unstable
	},

	"CtsInstalledLoadingProgressDeviceTests": {
		"framework", // cts -> unstable
	},

	"CtsInstantAppTests": {
		"framework", // cts -> unstable
	},

	"CtsIntentSenderApp": {
		"framework", // cts -> unstable
	},

	"CtsJobSchedulerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsKeystoreTestCases": {
		"framework", // cts -> unstable
	},

	"CtsLegacyNotification27TestCases": {
		"framework", // cts -> unstable
	},

	"CtsLibcoreTestCases": {
		"framework", // cts -> unstable
	},

	"CtsLibcoreWycheproofConscryptTestCases": {
		"framework", // cts -> unstable
	},

	"CtsListeningPortsTest": {
		"framework", // cts -> unstable
	},

	"CtsLocationCoarseTestCases": {
		"framework", // cts -> unstable
	},

	"CtsLocationFineTestCases": {
		"framework", // cts -> unstable
	},

	"CtsLocationNoneTestCases": {
		"framework", // cts -> unstable
	},

	"CtsLocationPrivilegedTestCases": {
		"framework", // cts -> unstable
	},

	"CtsManagedProfileApp": {
		"framework", // cts -> unstable
	},

	"CtsMediaAudioTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaBetterTogetherTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaCodecTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaDecoderTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaDrmFrameworkTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaEncoderTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaExtractorTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaMiscTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaMuxerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaPerformanceClassTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaPlayerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaProjectionSDK33TestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaProjectionSDK34TestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaProjectionTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaProviderTranscodeTests": {
		"framework", // cts -> unstable
	},

	"CtsMediaRecorderTestCases": {
		"framework", // cts -> unstable
	},

	"CtsMediaRouterHostSideTestBluetoothPermissionsApp": {
		"framework", // cts -> unstable
	},

	"CtsMediaRouterHostSideTestMediaRoutingControlApp": {
		"framework", // cts -> unstable
	},

	"CtsMediaRouterHostSideTestModifyAudioRoutingApp": {
		"framework", // cts -> unstable
	},

	"CtsMediaV2TestCases": {
		"framework", // cts -> unstable
	},

	"CtsMimeMapTestCases": {
		"framework", // cts -> unstable
	},

	"CtsModifyQuietModeEnabledApp": {
		"framework", // cts -> unstable
	},

	"CtsMusicRecognitionTestCases": {
		"framework", // cts -> unstable
	},

	"CtsNativeMediaAAudioTestCases": {
		"framework", // cts -> unstable
	},

	"CtsNetTestCases": {
		"framework", // cts -> unstable
	},

	"CtsNetTestCasesLegacyApi22": {
		"framework", // cts -> unstable
	},

	"CtsNetTestCasesMaxTargetSdk30": {
		"framework", // cts -> unstable
	},

	"CtsNetTestCasesMaxTargetSdk31": {
		"framework", // cts -> unstable
	},

	"CtsNetTestCasesMaxTargetSdk33": {
		"framework", // cts -> unstable
	},

	"CtsNetTestCasesUpdateStatsPermission": {
		"framework", // cts -> unstable
	},

	"CtsNfcTestCases": {
		"framework", // cts -> unstable
	},

	"CtsOnDevicePersonalizationTestCases": {
		"framework", // cts -> unstable
	},

	"CtsPackageInstallerApp": {
		"framework", // cts -> unstable
	},

	"CtsPackageManagerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsPackageSchemeTestsWithVisibility": {
		"framework", // cts -> unstable
	},

	"CtsPackageSchemeTestsWithoutVisibility": {
		"framework", // cts -> unstable
	},

	"CtsPermissionsSyncTestApp": {
		"framework", // cts -> unstable
	},

	"CtsPreservedSettingsApp": {
		"framework", // cts -> unstable
	},

	"CtsProtoTestCases": {
		"framework", // cts -> unstable
	},

	"CtsProviderTestCases": {
		"framework", // cts -> unstable
	},

	"CtsProxyMediaRouterTestHelperApp": {
		"framework", // cts -> unstable
	},

	"CtsRebootReadinessTestCases": {
		"framework", // cts -> unstable
	},

	"CtsResourcesLoaderTests": {
		"framework", // cts -> unstable
	},

	"CtsResourcesTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSandboxedAdIdManagerTests": {
		"framework", // cts -> unstable
	},

	"CtsSandboxedAppSetIdManagerTests": {
		"framework", // cts -> unstable
	},

	"CtsSandboxedFledgeManagerTests": {
		"framework", // cts -> unstable
	},

	"CtsSandboxedMeasurementManagerTests": {
		"framework", // cts -> unstable
	},

	"CtsSandboxedTopicsManagerTests": {
		"framework", // cts -> unstable
	},

	"CtsSdkExtensionsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSdkSandboxInprocessTests": {
		"framework", // cts -> unstable
	},

	"CtsSecureElementTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSecurityTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxEphemeralTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxTargetSdk25TestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxTargetSdk27TestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxTargetSdk28TestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxTargetSdk29TestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxTargetSdk30TestCases": {
		"framework", // cts -> unstable
	},

	"CtsSelinuxTargetSdkCurrentTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSettingsDeviceOwnerApp": {
		"framework", // cts -> unstable
	},

	"CtsSharedUserMigrationTestCases": {
		"framework", // cts -> unstable
	},

	"CtsShortFgsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSimRestrictedApisTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSliceTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSpeechTestCases": {
		"framework", // cts -> unstable
	},

	"CtsStatsSecurityApp": {
		"framework", // cts -> unstable
	},

	"CtsSuspendAppsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsSystemUiTestCases": {
		"framework", // cts -> unstable
	},

	"CtsTareTestCases": {
		"framework", // cts -> unstable
	},

	"CtsTelephonyTestCases": {
		"framework", // cts -> unstable
	},

	"CtsTetheringTest": {
		"framework", // cts -> unstable
	},

	"CtsThreadNetworkTestCases": {
		"framework", // cts -> unstable
	},

	"CtsUsageStatsTestCases": {
		"framework", // cts -> unstable
	},

	"CtsUsbManagerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsUserRestrictionTestCases": {
		"framework", // cts -> unstable
	},

	"CtsUtilTestCases": {
		"framework", // cts -> unstable
	},

	"CtsUwbTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVcnTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVideoCodecTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVideoTestCases": {
		"framework", // cts -> unstable
	},

	"CtsViewReceiveContentTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVirtualDevicesAppLaunchTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVirtualDevicesAudioTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVirtualDevicesCameraTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVirtualDevicesSensorTestCases": {
		"framework", // cts -> unstable
	},

	"CtsVirtualDevicesTestCases": {
		"framework", // cts -> unstable
	},

	"CtsWearableSensingServiceTestCases": {
		"framework", // cts -> unstable
	},

	"CtsWebViewCompatChangeApp": {
		"framework", // cts -> unstable
	},

	"CtsWidgetTestCases": {
		"framework", // cts -> unstable
	},

	"CtsWidgetTestCases29": {
		"framework", // cts -> unstable
	},

	"CtsWifiNonUpdatableTestCases": {
		"framework", // cts -> unstable
	},

	"CtsWifiTestCases": {
		"framework", // cts -> unstable
	},

	"CtsWindowManagerExternalApp": {
		"framework", // cts -> unstable
	},

	"CtsWindowManagerTestCases": {
		"framework", // cts -> unstable
	},

	"CtsZipValidateApp": {
		"framework", // cts -> unstable
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

	"DeviceDropMonitor": {
		"vendor-pixelatoms-java", // system -> vendor
	},

	"DockSetupLibrary": {
		"vendor-pixelatoms-java", // system -> vendor
	},

	"EmulatorBootSanityTester": {
		"framework", // cts -> unstable
	},

	"EmulatorCtsVerifierAutomator": {
		"framework", // cts -> unstable
	},

	"FederatedCompute": {
		"auto_value_annotations", // apex -> apex
	},

	"IncrementalTestAppValidator": {
		"framework", // cts -> unstable
	},

	"MctsMediaDrmFrameworkTestCases": {
		"framework", // cts -> unstable
	},

	"MctsMediaTranscodingTestCases": {
		"framework", // cts -> unstable
	},

	"MediaProvider": {
		"app-compat-annotations", // apex -> system
	},

	"MockSatelliteGatewayServiceApp": {
		"framework", // cts -> unstable
	},

	"MockSatelliteServiceApp": {
		"framework", // cts -> unstable
	},

	"PermissionController-lib": {
		"safety-center-annotations", // apex -> system
	},

	"PersistentBackgroundServices-common-no-concurrency-modules": {
		"vendor-pixelatoms-java", // system -> vendor
	},

	"PlatformProperties": {
		"sysprop-library-stub-platform", // apex -> system
	},

	"SdkSandboxManagerDisabledTests": {
		"framework", // cts -> unstable
	},

	"SdkSandboxManagerTests": {
		"framework", // cts -> unstable
	},

	"TelephonyDeviceTest": {
		"framework", // cts -> unstable
	},

	"TestExternalImsServiceApp": {
		"framework", // cts -> unstable
	},

	"TestSmsRetrieverApp": {
		"framework", // cts -> unstable
	},

	"Tethering": {
		"connectivity-internal-api-util", // apex -> system
	},

	"TetheringApiCurrentLib": {
		"connectivity-internal-api-util", // apex -> system
	},

	"TetheringApiStableLib": {
		"connectivity-internal-api-util", // apex -> system
	},

	"TetheringNext": {
		"connectivity-internal-api-util", // apex -> system
	},

	"android.car-module.impl": {
		"modules-utils-preconditions", // apex -> apex
	},

	"art-aconfig-flags-java-lib": {
		"framework-api-annotations-lib", // apex -> system
	},

	"bluetooth-nano-protos": {
		"libprotobuf-java-nano", // apex -> apex
	},

	"bluetooth.change-ids": {
		"app-compat-annotations", // apex -> system
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

	"connectivity-net-module-utils-bpf": {
		"net-utils-device-common-struct-base", // apex -> system
	},

	"conscrypt-aconfig-flags-lib": {
		"aconfig-annotations-lib-sdk-none", // apex -> system
	},

	"cronet_aml_base_base_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
		"jsr305", // apex -> apex
	},

	"cronet_aml_build_android_build_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_base_feature_overrides_java_proto": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_cronet_api_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_cronet_impl_common_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_cronet_impl_native_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
		"jsr305", // apex -> apex
	},

	"cronet_aml_components_cronet_android_cronet_jni_registration_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_cronet_shared_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_cronet_stats_log_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_cronet_urlconnection_impl_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_flags_java_proto": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_components_cronet_android_request_context_config_java_proto": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_net_android_net_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
		"jsr305", // apex -> apex
	},

	"cronet_aml_net_android_net_thread_stats_uid_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_third_party_jni_zero_jni_zero_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"cronet_aml_url_url_java": {
		"framework-connectivity-pre-jarjar-without-cronet", // apex -> system
	},

	"device_config_reboot_flags_java_lib": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},

	"devicelockcontroller-lib": {
		"modules-utils-expresslog", // apex -> apex
	},

	"framework-adservices.impl": {
		"adservices_flags_lib", // apex -> system
	},

	"framework-bluetooth.impl": {
		"app-compat-annotations", // apex -> system
	},

	"framework-connectivity-t.impl": {
		"app-compat-annotations",            // apex -> system
		"framework-connectivity-pre-jarjar", // apex -> system
	},

	"framework-connectivity.impl": {
		"app-compat-annotations", // apex -> system
	},

	"framework-ondevicepersonalization.impl": {
		"ondevicepersonalization_flags_lib", // apex -> system
	},

	"framework-pdf.impl": {
		"modules-utils-preconditions", // apex -> apex
	},

	"framework-permission-s.impl": {
		"app-compat-annotations", // apex -> system
	},

	"framework-wifi.impl": {
		"aconfig_storage_reader_java", // apex -> system
		"app-compat-annotations",      // apex -> system
	},

	"grpc-java-core-internal": {
		"gson",             // apex -> apex
		"perfmark-api-lib", // apex -> system
	},

	"httpclient_impl": {
		"httpclient_api", // apex -> system
	},

	"libcore-aconfig-flags-lib": {
		"framework-api-annotations-lib", // apex -> system
	},

	"libnativeloader_e2e_tests": {
		"libnativeloader_vendor_shared_lib", // system -> vendor
		"loadlibrarytest_vendor_app",        // system -> vendor
	},

	"loadlibrarytest_product_app": {
		"libnativeloader_vendor_shared_lib", // product -> vendor
	},

	"loadlibrarytest_testlib": {
		"libnativeloader_vendor_shared_lib", // system -> vendor
	},

	"mediaprovider_flags_java_lib": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},

	"net-utils-device-common-netlink": {
		"net-utils-device-common-struct-base", // apex -> system
	},

	"net-utils-device-common-struct": {
		"net-utils-device-common-struct-base", // apex -> system
	},

	"okhttp-norepackage": {
		"okhttp-android-util-log", // apex -> system
	},

	"ondevicepersonalization-plugin-lib": {
		"auto_value_annotations", // apex -> apex
	},

	"opencensus-java-api": {
		"auto_value_annotations", // apex -> apex
	},

	"safety-center-config": {
		"safety-center-annotations", // apex -> system
	},

	"safety-center-internal-data": {
		"safety-center-annotations", // apex -> system
	},

	"safety-center-pending-intents": {
		"safety-center-annotations", // apex -> system
	},

	"safety-center-persistence": {
		"safety-center-annotations", // apex -> system
	},

	"safety-center-resources-lib": {
		"safety-center-annotations", // apex -> system
	},

	"service-art.impl": {
		"auto_value_annotations", // apex -> apex
	},

	"service-bluetooth-pre-jarjar": {
		"framework-bluetooth-pre-jarjar", // apex -> system
		"service-bluetooth.change-ids",   // apex -> system
	},

	"service-connectivity": {
		"libprotobuf-java-nano", // apex -> apex
	},

	"service-connectivity-pre-jarjar": {
		"framework-connectivity-pre-jarjar", // apex -> system
	},

	"service-connectivity-protos": {
		"libprotobuf-java-nano", // apex -> apex
	},

	"service-connectivity-tiramisu-pre-jarjar": {
		"framework-connectivity-pre-jarjar",   // apex -> system
		"framework-connectivity-t-pre-jarjar", // apex -> system
	},

	"service-entitlement": {
		"auto_value_annotations", // apex -> apex
	},

	"service-entitlement-api": {
		"auto_value_annotations", // apex -> apex
	},

	"service-entitlement-data": {
		"auto_value_annotations", // apex -> apex
	},

	"service-entitlement-impl": {
		"auto_value_annotations", // apex -> apex
	},

	"service-healthfitness.impl": {
		"modules-utils-preconditions", // apex -> apex
	},

	"service-networksecurity-pre-jarjar": {
		"framework-connectivity-pre-jarjar", // apex -> system
	},

	"service-permission.impl": {
		"jsr305",                    // apex -> apex
		"safety-center-annotations", // apex -> system
	},

	"service-remoteauth-pre-jarjar": {
		"framework-connectivity-pre-jarjar",   // apex -> system
		"framework-connectivity-t-pre-jarjar", // apex -> system
	},

	"service-thread-pre-jarjar": {
		"framework-connectivity-pre-jarjar",   // apex -> system
		"framework-connectivity-t-pre-jarjar", // apex -> system
	},

	"service-uwb-pre-jarjar": {
		"framework-uwb-pre-jarjar", // apex -> system
	},

	"service-wifi": {
		"auto_value_annotations", // apex -> apex
	},

	"sysuig": {
		"vendor-pixelatoms-java", // system -> vendor
	},

	"tensorflowlite_java": {
		"android-support-annotations", // apex -> system
	},

	"tetheringstatsprotos": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},

	"uwb_aconfig_flags_lib": {
		"ext",       // apex -> system
		"framework", // apex -> system
	},

	"uwb_androidx_backend": {
		"android-support-annotations", // apex -> system
	},

	"wifi-service-pre-jarjar": {
		"app-compat-annotations",    // apex -> system
		"auto_value_annotations",    // apex -> apex
		"framework-wifi-pre-jarjar", // apex -> system
		"jsr305",                    // apex -> apex
	},

	"wirelesscharger-adapter": {
		"vendor-pixelatoms-java", // system -> vendor
	},
}
