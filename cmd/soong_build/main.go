// Copyright 2015 Google Inc. All rights reserved.
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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"android/soong/android"
	"android/soong/bp2build"
	"android/soong/shared"
	"android/soong/ui/metrics/bp2build_metrics_proto"

	"github.com/google/blueprint/bootstrap"
	"github.com/google/blueprint/deptools"
	"github.com/google/blueprint/metrics"
	androidProtobuf "google.golang.org/protobuf/android"
)

var (
	topDir           string
	outDir           string
	soongOutDir      string
	availableEnvFile string
	usedEnvFile      string

	runGoTests bool

	globFile    string
	globListDir string
	delveListen string
	delvePath   string

	moduleGraphFile   string
	moduleActionsFile string
	docFile           string
	bazelQueryViewDir string
	bp2buildMarker    string

	cmdlineArgs bootstrap.Args
)

func init() {
	// Flags that make sense in every mode
	flag.StringVar(&topDir, "top", "", "Top directory of the Android source tree")
	flag.StringVar(&soongOutDir, "soong_out", "", "Soong output directory (usually $TOP/out/soong)")
	flag.StringVar(&availableEnvFile, "available_env", "", "File containing available environment variables")
	flag.StringVar(&usedEnvFile, "used_env", "", "File containing used environment variables")
	flag.StringVar(&globFile, "globFile", "build-globs.ninja", "the Ninja file of globs to output")
	flag.StringVar(&globListDir, "globListDir", "", "the directory containing the glob list files")
	flag.StringVar(&outDir, "out", "", "the ninja builddir directory")
	flag.StringVar(&cmdlineArgs.ModuleListFile, "l", "", "file that lists filepaths to parse")

	// Debug flags
	flag.StringVar(&delveListen, "delve_listen", "", "Delve port to listen on for debugging")
	flag.StringVar(&delvePath, "delve_path", "", "Path to Delve. Only used if --delve_listen is set")
	flag.StringVar(&cmdlineArgs.Cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&cmdlineArgs.TraceFile, "trace", "", "write trace to file")
	flag.StringVar(&cmdlineArgs.Memprofile, "memprofile", "", "write memory profile to file")
	flag.BoolVar(&cmdlineArgs.NoGC, "nogc", false, "turn off GC for debugging")

	// Flags representing various modes soong_build can run in
	flag.StringVar(&moduleGraphFile, "module_graph_file", "", "JSON module graph file to output")
	flag.StringVar(&moduleActionsFile, "module_actions_file", "", "JSON file to output inputs/outputs of actions of modules")
	flag.StringVar(&docFile, "soong_docs", "", "build documentation file to output")
	flag.StringVar(&bazelQueryViewDir, "bazel_queryview_dir", "", "path to the bazel queryview directory relative to --top")
	flag.StringVar(&bp2buildMarker, "bp2build_marker", "", "If set, run bp2build, touch the specified marker file then exit")
	flag.StringVar(&cmdlineArgs.OutFile, "o", "build.ninja", "the Ninja file to output")
	flag.BoolVar(&cmdlineArgs.EmptyNinjaFile, "empty-ninja-file", false, "write out a 0-byte ninja file")

	// Flags that probably shouldn't be flags of soong_build but we haven't found
	// the time to remove them yet
	flag.BoolVar(&runGoTests, "t", false, "build and run go tests during bootstrap")

	// Disable deterministic randomization in the protobuf package, so incremental
	// builds with unrelated Soong changes don't trigger large rebuilds (since we
	// write out text protos in command lines, and command line changes trigger
	// rebuilds).
	androidProtobuf.DisableRand()
}

type MainlineModule struct {
	// The short name of the module.
	ShortName string

	// The root directories belonging to the module's source namespace.
	SourcePaths []string

	// The root directories belonging to the module's prebuilt SDK snapshot namespace.
	PrebuiltSdkPaths []string

	// The root directories belonging to the module's prebuilt APEX namespace.
	//
	// If the module does not require a prebuilt APEX then this should be set to an empty array.
	PrebuiltApexPaths []string

	// Additional exports that have to be exported from the source namespace to other Soong modules.
	AdditionalSoongExports []string

	// Additional exports that have to be exported from the source namespace to make.
	AdditionalMakeExports []string
}

var MainlineModules = []MainlineModule{
	{
		ShortName: "art",
		SourcePaths: []string{
			"art",
			"external/apache-harmony",
			"external/apache-xml",
			"external/okhttp",
			"external/vixl",
			"libcore",
			"libnativehelper",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/art",
		},
		AdditionalSoongExports: []string{
			"artd-aidl-java-source",
			"artd-aidl-ndk-source",
			"artd-aidl_interface",
			"dmtracedump",
			"libctstiagent",
		},
		AdditionalMakeExports: []string{
			"ahat",
			"dexdiag",
			"dexlist",
			"dexoptanalyzer",
			"libartservice",
			"libdt_fd_forward",
			"libjavacore-benchmarks",
			"libjavacore-unit-tests",
			"libnativehelper_tests",
			"libopenjdkd",
		},
	},
	{
		ShortName: "conscrypt",
		SourcePaths: []string{
			"external/conscrypt",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/conscrypt",
		},
		PrebuiltApexPaths: []string{
			"prebuilts/runtime/mainline/conscrypt/apex",
		},
		AdditionalSoongExports: []string{
			"conscrypt-support",
		},
	},
	{
		ShortName: "appsearch",
		SourcePaths: []string{
			"packages/modules/AppSearch",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/AppSearch",
		},
		AdditionalSoongExports: []string{
			"AppSearchTestUtils",
			"com.android.appsearch-bootclasspath-fragment",
			"framework-appsearch",
			"framework-appsearch-sources",
			"framework-appsearch.impl",
			"framework-appsearch.stubs",
			"framework-appsearch.stubs.module_lib",
			"framework-appsearch.stubs.system",
		},
		AdditionalMakeExports: []string{
			"service-appsearch",
		},
	},
	{
		ShortName: "bluetooth",
		SourcePaths: []string{
			"packages/modules/Bluetooth",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Bluetooth",
		},
		AdditionalSoongExports: []string{
			"android.hardware.bluetooth@1.1-service.btlinux",
			"async_fd_watcher",
			"bluetooth_packetgen",
			"controller_properties.json",
			"framework-bluetooth",
			"framework-bluetooth-sources",
			"framework-bluetooth.stubs",
			"framework-bluetooth.stubs.module_lib",
			"framework-bluetooth.stubs.system",
			"h4_packetizer_lib",
			"libbluetooth",
			"libbluetooth-binder-aidl",
			"libbluetooth-binder-common",
			"libbluetooth-types",
			"libbluetooth-types-header",
			"libbluetooth_headers",
			"libbt-rootcanal",
			"libbtcore",
			"libosi",
			"service-bluetooth-tests-sources",
			"services.bluetooth-sources",
		},
		AdditionalMakeExports: []string{
			"BluetoothInstrumentationTests",
			"BluetoothTests",
			"audio.bluetooth.default",
			"audio_set_configurations_bfbs",
			"audio_set_configurations_json",
			"audio_set_scenarios_bfbs",
			"audio_set_scenarios_json",
			"bluetooth_stack_with_facade",
			"bluetooth_test_common",
			"bluetoothtbd_test",
			"bt_did.conf",
			"bt_stack.conf",
			"net_test_audio_a2dp_hw",
			"net_test_avrcp",
			"net_test_bluetooth",
			"net_test_bta",
			"net_test_btcore",
			"net_test_btif",
			"net_test_btif_profile_queue",
			"net_test_btpackets",
			"net_test_device",
			"net_test_hci",
			"net_test_osi",
			"net_test_performance",
			"net_test_stack",
			"net_test_stack_ad_parser",
			"net_test_stack_multi_adv",
			"net_test_stack_rfcomm",
			"net_test_stack_smp",
			"net_test_types",
			"root-canal",
		},
	},
	{
		ShortName: "connectivity",
		SourcePaths: []string{
			"packages/modules/Connectivity",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Connectivity",
		},
		AdditionalSoongExports: []string{
			"ConnectivityNextEnableDefaults",
			"bpf_connectivity_headers",
			"connectivity-mainline-presubmit-cc-defaults",
			"connectivity-mainline-presubmit-java-defaults",
			"connectivity_native_aidl_interface-V1-cpp-source",
			"connectivity_native_aidl_interface-V1-java-source",
			"connectivity_native_aidl_interface-V1-ndk-source",
			"connectivity_native_aidl_interface-V2-cpp-source",
			"connectivity_native_aidl_interface-V2-java-source",
			"connectivity_native_aidl_interface-V2-ndk-source",
			"connectivity_native_aidl_interface-api",
			"connectivity_native_aidl_interface_interface",
			"cts-net-utils",
			"framework-connectivity-protos",
			"framework-connectivity-sources",
			"framework-connectivity-t.impl",
			"framework-connectivity-test-defaults",
			"framework-connectivity-tiramisu-updatable-sources",
			"framework-connectivity.impl",
			"framework-tethering-srcs",
			"framework-tethering.impl",
			"libnetd_updatable",
			"libnetworkstats",
			"libnetworkstatsfactorytestjni",
			"service-connectivity-tiramisu-pre-jarjar",
		},
		AdditionalMakeExports: []string{
			"FrameworksNetSmokeTests",
			"FrameworksNetTests",
			"libnetworkstats_test",
			"privapp_allowlist_com.android.tethering",
		},
	},
	// {
	// 	ShortName: "dnsresolver",
	// 	SourcePaths: []string{
	// 		"packages/modules/DnsResolver",
	// 	},
	// 	AdditionalMakeExports: []string{
	// 		"doh_ffi_test",
	// 		"doh_unit_test",
	// 		"resolv_gold_test",
	// 		"resolv_integration_test",
	// 		"resolv_unit_test",
	// 	},
	// },
	// {
	// 	ShortName: "extservices",
	// 	SourcePaths: []string{
	// 		"packages/modules/ExtServices",
	// 	},
	// 	AdditionalMakeExports: []string{
	// 		"ExtServicesUnitTests",
	// 		"privapp_allowlist_android.ext.services.xml",
	// 	},
	// },
	{
		ShortName: "ipsec",
		SourcePaths: []string{
			"packages/modules/IPsec",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/IPsec",
		},
		AdditionalSoongExports: []string{
			"android.net.ipsec.ike.impl",
			"android.net.ipsec.ike.xml",
			"ike-aes-xcbc",
			"ike-srcs",
			"ike-tun-utils",
		},
	},
	{
		SourcePaths: []string{
			"packages/modules/Media",
			"frameworks/av/apex",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Media",
		},
		AdditionalSoongExports: []string{
			"updatable-media-srcs",
		},
	},
	{
		ShortName: "permission",
		SourcePaths: []string{
			"packages/modules/Permission",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Permission",
		},
		AdditionalSoongExports: []string{
			"framework-permission-s-sources",
			"framework-permission-s.impl",
			"framework-permission-sources",
			"framework-permission.impl",
			"service-permission-protos",
			"service-permission.impl",
		},
	},
	{
		ShortName: "i18n",
		SourcePaths: []string{
			"packages/modules/RuntimeI18n",
			"external/icu",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/runtime/mainline/i18n/sdk",
			"prebuilts/runtime/mainline/i18n/test-exports",
		},
		PrebuiltApexPaths: []string{
			"prebuilts/runtime/mainline/i18n/apex",
		},
		AdditionalSoongExports: []string{
			"ICU4CTestRunner",
			"android-icu4j-tests",
			"cintltst32",
			"cintltst64",
			"icu4c_test_data",
			"icu4j",
			"icu4j-icudata-jarjar",
			"icu4j-icutzdata-jarjar",
			"icu4j_calendar_astronomer",
			"intltest32",
			"intltest64",
			"libandroidicu",
			"libicu.ndk",
			"libicutest_static",
			"libicuuc_stubdata",
		},
		AdditionalMakeExports: []string{
			"icu4j-platform-compat-config",
			"icu-data_host_i18n_apex",
		},
	},
	{
		ShortName: "scheduling",
		SourcePaths: []string{
			"packages/modules/Scheduling",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Scheduling",
		},
		AdditionalSoongExports: []string{
			"framework-scheduling-sources",
			"framework-scheduling.impl",
		},
	},
	{
		ShortName: "sdkext",
		SourcePaths: []string{
			"packages/modules/SdkExtensions",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/SdkExtensions",
		},
		AdditionalSoongExports: []string{
			"framework-sdkextensions-sources",
			"framework-sdkextensions.impl",
			"sdkinfo_45",
		},
	},
	{
		ShortName: "statsd",
		SourcePaths: []string{
			"packages/modules/StatsD",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/StatsD",
		},
		AdditionalSoongExports: []string{
			"CtsStatsdApp",
			"framework-statsd-sources",
			"framework-statsd.impl",
			"libkll",
			"libstatspull",
			"libstatspull_headers",
			"libstatssocket_headers",
			"statsd-aidl-ndk",
			"statsd-aidl-ndk-source",
			"statsd-aidl_interface",
			"statsd_internal_protos",
			"statsdprotolite",
			"statsdprotonano",
		},
		AdditionalMakeExports: []string{
			"statsd_test",
		},
	},
	{
		ShortName: "uwb",
		SourcePaths: []string{
			"packages/modules/Uwb",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Uwb",
		},
		AdditionalSoongExports: []string{},
	},
	{
		ShortName: "wifi",
		SourcePaths: []string{
			"packages/modules/Wifi",
		},
		PrebuiltSdkPaths: []string{
			"prebuilts/module_sdk/Wifi",
		},
		AdditionalSoongExports: []string{
			"framework-wifi-annotations",
			"framework-wifi-test-defaults",
			"framework-wifi-updatable-sources",
			"framework-wifi.impl",
		},
		AdditionalMakeExports: []string{
			"FrameworksWifiApiTests",
			"FrameworksWifiTests",
		},
	},
}

func newNameResolver(config android.Config) *android.NameResolver {
	namespacePathsToExport := make(map[string]bool)

	for _, namespaceName := range config.ExportedNamespaces() {
		namespacePathsToExport[namespaceName] = true
	}

	namespacePathsToExport["."] = true // always export the root namespace

	exportFilter := func(namespace *android.Namespace) bool {
		return namespacePathsToExport[namespace.Path]
	}

	fmt.Printf("PAUL: Configuring protected namespaces\n")
	var protectedNamespaces []*android.ProtectedNamespaceConfig
	if android.EnableApiBoundaryEnforcement(config) {
		for _, mainline := range MainlineModules {
			// Create the source namespace.
			sourceNamespace := &android.ProtectedNamespaceConfig{
				Paths:                  mainline.SourcePaths,
				AdditionalSoongExports: mainline.AdditionalSoongExports,
				AdditionalMakeExports:  mainline.AdditionalMakeExports,
			}

			// Create the prebuilt namespace.
			prebuiltPaths := append([]string(nil), mainline.PrebuiltSdkPaths...)
			prebuiltPaths = append(prebuiltPaths, mainline.PrebuiltApexPaths...)

			prebuiltNamespace := &android.ProtectedNamespaceConfig{
				Paths: prebuiltPaths,
			}

			if mainline.PrebuiltApexPaths != nil {
				// The prebuilt namespace cannot exclude the source namespace as that causes breakages in
				// branches that have SDK snapshots but no corresponding prebuilt APEX.
				prebuiltNamespace.Exclude(sourceNamespace)
			}

			sourceNamespace.Exclude(prebuiltNamespace)

			envName := fmt.Sprintf("%s_USE_PREBUILTS", strings.ToUpper(mainline.ShortName))
			prebuilts := config.IsEnvTrue(envName)

			if prebuilts {
				// The prebuilts namespace is active, source is not.
				prebuiltNamespace.Active = true
			} else {
				// The source namespace is active, prebuilt is not.
				sourceNamespace.Active = true
			}

			// Add the namespaces.
			protectedNamespaces = append(protectedNamespaces, sourceNamespace, prebuiltNamespace)
		}
	}

	return android.NewNameResolver(exportFilter, protectedNamespaces)
}

func newContext(configuration android.Config) *android.Context {
	ctx := android.NewContext(configuration)
	ctx.Register()
	resolver := newNameResolver(configuration)
	ctx.SetNameInterface(resolver)
	ctx.SetAllowMissingDependencies(configuration.AllowMissingDependencies())
	return ctx
}

func newConfig(availableEnv map[string]string) android.Config {
	configuration, err := android.NewConfig(cmdlineArgs.ModuleListFile, runGoTests, outDir, soongOutDir, availableEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	return configuration
}

// Bazel-enabled mode. Attaches a mutator to queue Bazel requests, adds a
// BeforePrepareBuildActionsHook to invoke Bazel, and then uses Bazel metadata
// for modules that should be handled by Bazel.
func runMixedModeBuild(configuration android.Config, ctx *android.Context, extraNinjaDeps []string) {
	ctx.EventHandler.Begin("mixed_build")
	defer ctx.EventHandler.End("mixed_build")

	bazelHook := func() error {
		ctx.EventHandler.Begin("bazel")
		defer ctx.EventHandler.End("bazel")
		return configuration.BazelContext.InvokeBazel(configuration)
	}
	ctx.SetBeforePrepareBuildActionsHook(bazelHook)

	ninjaDeps := bootstrap.RunBlueprint(cmdlineArgs, bootstrap.DoEverything, ctx.Context, configuration)
	ninjaDeps = append(ninjaDeps, extraNinjaDeps...)

	globListFiles := writeBuildGlobsNinjaFile(ctx, configuration.SoongOutDir(), configuration)
	ninjaDeps = append(ninjaDeps, globListFiles...)

	writeDepFile(cmdlineArgs.OutFile, *ctx.EventHandler, ninjaDeps)
}

// Run the code-generation phase to convert BazelTargetModules to BUILD files.
func runQueryView(queryviewDir, queryviewMarker string, configuration android.Config, ctx *android.Context) {
	ctx.EventHandler.Begin("queryview")
	defer ctx.EventHandler.End("queryview")
	codegenContext := bp2build.NewCodegenContext(configuration, *ctx, bp2build.QueryView)
	absoluteQueryViewDir := shared.JoinPath(topDir, queryviewDir)
	if err := createBazelQueryView(codegenContext, absoluteQueryViewDir); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	touch(shared.JoinPath(topDir, queryviewMarker))
}

func writeMetrics(configuration android.Config, eventHandler metrics.EventHandler, metricsDir string) {
	if len(metricsDir) < 1 {
		fmt.Fprintf(os.Stderr, "\nMissing required env var for generating soong metrics: LOG_DIR\n")
		os.Exit(1)
	}
	metricsFile := filepath.Join(metricsDir, "soong_build_metrics.pb")
	err := android.WriteMetrics(configuration, eventHandler, metricsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing soong_build metrics %s: %s", metricsFile, err)
		os.Exit(1)
	}
}

func writeJsonModuleGraphAndActions(ctx *android.Context, graphPath string, actionsPath string) {
	graphFile, graphErr := os.Create(shared.JoinPath(topDir, graphPath))
	actionsFile, actionsErr := os.Create(shared.JoinPath(topDir, actionsPath))
	if graphErr != nil || actionsErr != nil {
		fmt.Fprintf(os.Stderr, "Graph err: %s, actions err: %s", graphErr, actionsErr)
		os.Exit(1)
	}

	defer graphFile.Close()
	defer actionsFile.Close()
	ctx.Context.PrintJSONGraphAndActions(graphFile, actionsFile)
}

func writeBuildGlobsNinjaFile(ctx *android.Context, buildDir string, config interface{}) []string {
	ctx.EventHandler.Begin("globs_ninja_file")
	defer ctx.EventHandler.End("globs_ninja_file")

	globDir := bootstrap.GlobDirectory(buildDir, globListDir)
	bootstrap.WriteBuildGlobsNinjaFile(&bootstrap.GlobSingleton{
		GlobLister: ctx.Globs,
		GlobFile:   globFile,
		GlobDir:    globDir,
		SrcDir:     ctx.SrcDir(),
	}, config)
	return bootstrap.GlobFileListFiles(globDir)
}

func writeDepFile(outputFile string, eventHandler metrics.EventHandler, ninjaDeps []string) {
	eventHandler.Begin("ninja_deps")
	defer eventHandler.End("ninja_deps")
	depFile := shared.JoinPath(topDir, outputFile+".d")
	err := deptools.WriteDepFile(depFile, outputFile, ninjaDeps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing depfile '%s': %s\n", depFile, err)
		os.Exit(1)
	}
}

// doChosenActivity runs Soong for a specific activity, like bp2build, queryview
// or the actual Soong build for the build.ninja file. Returns the top level
// output file of the specific activity.
func doChosenActivity(configuration android.Config, extraNinjaDeps []string, logDir string) string {
	mixedModeBuild := configuration.BazelContext.BazelEnabled()
	generateBazelWorkspace := bp2buildMarker != ""
	generateQueryView := bazelQueryViewDir != ""
	generateModuleGraphFile := moduleGraphFile != ""
	generateDocFile := docFile != ""

	if generateBazelWorkspace {
		// Run the alternate pipeline of bp2build mutators and singleton to convert
		// Blueprint to BUILD files before everything else.
		runBp2Build(configuration, extraNinjaDeps)
		return bp2buildMarker
	}

	blueprintArgs := cmdlineArgs

	ctx := newContext(configuration)
	if mixedModeBuild {
		runMixedModeBuild(configuration, ctx, extraNinjaDeps)
	} else {
		var stopBefore bootstrap.StopBefore
		if generateModuleGraphFile {
			stopBefore = bootstrap.StopBeforeWriteNinja
		} else if generateQueryView {
			stopBefore = bootstrap.StopBeforePrepareBuildActions
		} else if generateDocFile {
			stopBefore = bootstrap.StopBeforePrepareBuildActions
		} else {
			stopBefore = bootstrap.DoEverything
		}

		ninjaDeps := bootstrap.RunBlueprint(blueprintArgs, stopBefore, ctx.Context, configuration)
		ninjaDeps = append(ninjaDeps, extraNinjaDeps...)

		globListFiles := writeBuildGlobsNinjaFile(ctx, configuration.SoongOutDir(), configuration)
		ninjaDeps = append(ninjaDeps, globListFiles...)

		// Convert the Soong module graph into Bazel BUILD files.
		if generateQueryView {
			queryviewMarkerFile := bazelQueryViewDir + ".marker"
			runQueryView(bazelQueryViewDir, queryviewMarkerFile, configuration, ctx)
			writeDepFile(queryviewMarkerFile, *ctx.EventHandler, ninjaDeps)
			return queryviewMarkerFile
		} else if generateModuleGraphFile {
			writeJsonModuleGraphAndActions(ctx, moduleGraphFile, moduleActionsFile)
			writeDepFile(moduleGraphFile, *ctx.EventHandler, ninjaDeps)
			return moduleGraphFile
		} else if generateDocFile {
			// TODO: we could make writeDocs() return the list of documentation files
			// written and add them to the .d file. Then soong_docs would be re-run
			// whenever one is deleted.
			if err := writeDocs(ctx, shared.JoinPath(topDir, docFile)); err != nil {
				fmt.Fprintf(os.Stderr, "error building Soong documentation: %s\n", err)
				os.Exit(1)
			}
			writeDepFile(docFile, *ctx.EventHandler, ninjaDeps)
			return docFile
		} else {
			// The actual output (build.ninja) was written in the RunBlueprint() call
			// above
			writeDepFile(cmdlineArgs.OutFile, *ctx.EventHandler, ninjaDeps)
		}
	}

	writeMetrics(configuration, *ctx.EventHandler, logDir)
	return cmdlineArgs.OutFile
}

// soong_ui dumps the available environment variables to
// soong.environment.available . Then soong_build itself is run with an empty
// environment so that the only way environment variables can be accessed is
// using Config, which tracks access to them.

// At the end of the build, a file called soong.environment.used is written
// containing the current value of all used environment variables. The next
// time soong_ui is run, it checks whether any environment variables that was
// used had changed and if so, it deletes soong.environment.used to cause a
// rebuild.
//
// The dependency of build.ninja on soong.environment.used is declared in
// build.ninja.d
func parseAvailableEnv() map[string]string {
	if availableEnvFile == "" {
		fmt.Fprintf(os.Stderr, "--available_env not set\n")
		os.Exit(1)
	}

	result, err := shared.EnvFromFile(shared.JoinPath(topDir, availableEnvFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading available environment file '%s': %s\n", availableEnvFile, err)
		os.Exit(1)
	}

	return result
}

func main() {
	flag.Parse()

	shared.ReexecWithDelveMaybe(delveListen, delvePath)
	android.InitSandbox(topDir)

	availableEnv := parseAvailableEnv()

	configuration := newConfig(availableEnv)
	extraNinjaDeps := []string{
		configuration.ProductVariablesFileName,
		usedEnvFile,
	}

	if configuration.Getenv("ALLOW_MISSING_DEPENDENCIES") == "true" {
		configuration.SetAllowMissingDependencies()
	}

	if shared.IsDebugging() {
		// Add a non-existent file to the dependencies so that soong_build will rerun when the debugger is
		// enabled even if it completed successfully.
		extraNinjaDeps = append(extraNinjaDeps, filepath.Join(configuration.SoongOutDir(), "always_rerun_for_delve"))
	}

	// Bypass configuration.Getenv, as LOG_DIR does not need to be dependency tracked. By definition, it will
	// change between every CI build, so tracking it would require re-running Soong for every build.
	logDir := availableEnv["LOG_DIR"]

	finalOutputFile := doChosenActivity(configuration, extraNinjaDeps, logDir)

	writeUsedEnvironmentFile(configuration, finalOutputFile)
}

func writeUsedEnvironmentFile(configuration android.Config, finalOutputFile string) {
	if usedEnvFile == "" {
		return
	}

	path := shared.JoinPath(topDir, usedEnvFile)
	data, err := shared.EnvFileContents(configuration.EnvDeps())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing used environment file '%s': %s\n", usedEnvFile, err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(path, data, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing used environment file '%s': %s\n", usedEnvFile, err)
		os.Exit(1)
	}

	// Touch the output file so that it's not older than the file we just
	// wrote. We can't write the environment file earlier because one an access
	// new environment variables while writing it.
	touch(shared.JoinPath(topDir, finalOutputFile))
}

func touch(path string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error touching '%s': %s\n", path, err)
		os.Exit(1)
	}

	err = f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error touching '%s': %s\n", path, err)
		os.Exit(1)
	}

	currentTime := time.Now().Local()
	err = os.Chtimes(path, currentTime, currentTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error touching '%s': %s\n", path, err)
		os.Exit(1)
	}
}

// Find BUILD files in the srcDir which...
//
// - are not on the allow list (android/bazel.go#ShouldKeepExistingBuildFileForDir())
//
// - won't be overwritten by corresponding bp2build generated files
//
// And return their paths so they can be left out of the Bazel workspace dir (i.e. ignored)
func getPathsToIgnoredBuildFiles(topDir string, generatedRoot string, srcDirBazelFiles []string, verbose bool) []string {
	paths := make([]string, 0)

	for _, srcDirBazelFileRelativePath := range srcDirBazelFiles {
		srcDirBazelFileFullPath := shared.JoinPath(topDir, srcDirBazelFileRelativePath)
		fileInfo, err := os.Stat(srcDirBazelFileFullPath)
		if err != nil {
			// Warn about error, but continue trying to check files
			fmt.Fprintf(os.Stderr, "WARNING: Error accessing path '%s', err: %s\n", srcDirBazelFileFullPath, err)
			continue
		}
		if fileInfo.IsDir() {
			// Don't ignore entire directories
			continue
		}
		if !(fileInfo.Name() == "BUILD" || fileInfo.Name() == "BUILD.bazel") {
			// Don't ignore this file - it is not a build file
			continue
		}
		srcDirBazelFileDir := filepath.Dir(srcDirBazelFileRelativePath)
		if android.ShouldKeepExistingBuildFileForDir(srcDirBazelFileDir) {
			// Don't ignore this existing build file
			continue
		}
		correspondingBp2BuildFile := shared.JoinPath(topDir, generatedRoot, srcDirBazelFileRelativePath)
		if _, err := os.Stat(correspondingBp2BuildFile); err == nil {
			// If bp2build generated an alternate BUILD file, don't exclude this workspace path
			// BUILD file clash resolution happens later in the symlink forest creation
			continue
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Ignoring existing BUILD file: %s\n", srcDirBazelFileRelativePath)
		}
		paths = append(paths, srcDirBazelFileRelativePath)
	}

	return paths
}

// Returns temporary symlink forest excludes necessary for bazel build //external/... (and bazel build //frameworks/...) to work
func getTemporaryExcludes() []string {
	excludes := make([]string, 0)

	// FIXME: 'autotest_lib' is a symlink back to external/autotest, and this causes an infinite symlink expansion error for Bazel
	excludes = append(excludes, "external/autotest/venv/autotest_lib")
	excludes = append(excludes, "external/autotest/autotest_lib")
	excludes = append(excludes, "external/autotest/client/autotest_lib/client")

	// FIXME: The external/google-fruit/extras/bazel_root/third_party/fruit dir is poison
	// It contains several symlinks back to real source dirs, and those source dirs contain BUILD files we want to ignore
	excludes = append(excludes, "external/google-fruit/extras/bazel_root/third_party/fruit")

	// FIXME: 'frameworks/compile/slang' has a filegroup error due to an escaping issue
	excludes = append(excludes, "frameworks/compile/slang")

	return excludes
}

// Read the bazel.list file that the Soong Finder already dumped earlier (hopefully)
// It contains the locations of BUILD files, BUILD.bazel files, etc. in the source dir
func getExistingBazelRelatedFiles(topDir string) ([]string, error) {
	bazelFinderFile := filepath.Join(filepath.Dir(cmdlineArgs.ModuleListFile), "bazel.list")
	if !filepath.IsAbs(bazelFinderFile) {
		// Assume this was a relative path under topDir
		bazelFinderFile = filepath.Join(topDir, bazelFinderFile)
	}
	data, err := ioutil.ReadFile(bazelFinderFile)
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(data)), "\n")
	return files, nil
}

// Run Soong in the bp2build mode. This creates a standalone context that registers
// an alternate pipeline of mutators and singletons specifically for generating
// Bazel BUILD files instead of Ninja files.
func runBp2Build(configuration android.Config, extraNinjaDeps []string) {
	eventHandler := metrics.EventHandler{}
	var metrics bp2build.CodegenMetrics
	eventHandler.Do("bp2build", func() {

		// Register an alternate set of singletons and mutators for bazel
		// conversion for Bazel conversion.
		bp2buildCtx := android.NewContext(configuration)

		// Soong internals like LoadHooks behave differently when running as
		// bp2build. This is the bit to differentiate between Soong-as-Soong and
		// Soong-as-bp2build.
		bp2buildCtx.SetRunningAsBp2build()

		// Propagate "allow misssing dependencies" bit. This is normally set in
		// newContext(), but we create bp2buildCtx without calling that method.
		bp2buildCtx.SetAllowMissingDependencies(configuration.AllowMissingDependencies())
		bp2buildCtx.SetNameInterface(newNameResolver(configuration))
		bp2buildCtx.RegisterForBazelConversion()

		// The bp2build process is a purely functional process that only depends on
		// Android.bp files. It must not depend on the values of per-build product
		// configurations or variables, since those will generate different BUILD
		// files based on how the user has configured their tree.
		bp2buildCtx.SetModuleListFile(cmdlineArgs.ModuleListFile)
		modulePaths, err := bp2buildCtx.ListModulePaths(".")
		if err != nil {
			panic(err)
		}

		extraNinjaDeps = append(extraNinjaDeps, modulePaths...)

		// Run the loading and analysis pipeline to prepare the graph of regular
		// Modules parsed from Android.bp files, and the BazelTargetModules mapped
		// from the regular Modules.
		blueprintArgs := cmdlineArgs
		ninjaDeps := bootstrap.RunBlueprint(blueprintArgs, bootstrap.StopBeforePrepareBuildActions, bp2buildCtx.Context, configuration)
		ninjaDeps = append(ninjaDeps, extraNinjaDeps...)

		globListFiles := writeBuildGlobsNinjaFile(bp2buildCtx, configuration.SoongOutDir(), configuration)
		ninjaDeps = append(ninjaDeps, globListFiles...)

		// Run the code-generation phase to convert BazelTargetModules to BUILD files
		// and print conversion metrics to the user.
		codegenContext := bp2build.NewCodegenContext(configuration, *bp2buildCtx, bp2build.Bp2Build)
		metrics = bp2build.Codegen(codegenContext)

		generatedRoot := shared.JoinPath(configuration.SoongOutDir(), "bp2build")
		workspaceRoot := shared.JoinPath(configuration.SoongOutDir(), "workspace")

		excludes := []string{
			"bazel-bin",
			"bazel-genfiles",
			"bazel-out",
			"bazel-testlogs",
			"bazel-" + filepath.Base(topDir),
		}

		if outDir[0] != '/' {
			excludes = append(excludes, outDir)
		}

		existingBazelRelatedFiles, err := getExistingBazelRelatedFiles(topDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error determining existing Bazel-related files: %s\n", err)
			os.Exit(1)
		}

		pathsToIgnoredBuildFiles := getPathsToIgnoredBuildFiles(topDir, generatedRoot, existingBazelRelatedFiles, configuration.IsEnvTrue("BP2BUILD_VERBOSE"))
		excludes = append(excludes, pathsToIgnoredBuildFiles...)

		excludes = append(excludes, getTemporaryExcludes()...)

		symlinkForestDeps := bp2build.PlantSymlinkForest(
			topDir, workspaceRoot, generatedRoot, ".", excludes)

		ninjaDeps = append(ninjaDeps, codegenContext.AdditionalNinjaDeps()...)
		ninjaDeps = append(ninjaDeps, symlinkForestDeps...)

		writeDepFile(bp2buildMarker, eventHandler, ninjaDeps)

		// Create an empty bp2build marker file.
		touch(shared.JoinPath(topDir, bp2buildMarker))
	})

	// Only report metrics when in bp2build mode. The metrics aren't relevant
	// for queryview, since that's a total repo-wide conversion and there's a
	// 1:1 mapping for each module.
	metrics.Print()
	writeBp2BuildMetrics(&metrics, configuration, eventHandler)
}

// Write Bp2Build metrics into $LOG_DIR
func writeBp2BuildMetrics(codegenMetrics *bp2build.CodegenMetrics,
	configuration android.Config, eventHandler metrics.EventHandler) {
	for _, event := range eventHandler.CompletedEvents() {
		codegenMetrics.Events = append(codegenMetrics.Events,
			&bp2build_metrics_proto.Event{
				Name:      event.Id,
				StartTime: uint64(event.Start.UnixNano()),
				RealTime:  event.RuntimeNanoseconds(),
			})
	}
	metricsDir := configuration.Getenv("LOG_DIR")
	if len(metricsDir) < 1 {
		fmt.Fprintf(os.Stderr, "\nMissing required env var for generating bp2build metrics: LOG_DIR\n")
		os.Exit(1)
	}
	codegenMetrics.Write(metricsDir)
}
