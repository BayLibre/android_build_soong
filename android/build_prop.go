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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/blueprint/proptools"
)

func init() {
	ctx := InitRegistrationContext
	ctx.RegisterModuleType("build_prop", buildPropFactory)
}

type buildPropProperties struct {
	// Output file name. defaults to module name
	Stem *string

	// Whether if build.prop is for sdk build or not. Defaults to false
	Sdk_build *bool

	// List of prop names to exclude. This affects not only common build properties but also
	// properties in prop_files.
	Block_list []string

	// Path to the input prop files. The contents of the files are directly
	// emitted to the output
	Prop_files []string `android:"path"`

	// Files to be appended at the end of build.prop. These files are appended after
	// post_process_props without any further checking.
	Footer_files []string `android:"path"`

	// Name of target partition. The only possible value for now is "system".
	Partition *string

	// Whether this module is directly installable to one of the partitions. Defaults to true.
	Installable *bool
}

type buildPropModule struct {
	ModuleBase

	properties buildPropProperties

	outputFilePath OutputPath
	installPath    InstallPath
}

var _ OutputFileProducer = (*buildPropModule)(nil)

func (p *buildPropModule) installable() bool {
	return proptools.BoolDefault(p.properties.Installable, true)
}

func (p *buildPropModule) stem() string {
	return proptools.StringDefault(p.properties.Stem, p.Name())
}

// OutputFileProducer
func (p *buildPropModule) OutputFiles(tag string) (Paths, error) {
	if tag != "" {
		return nil, fmt.Errorf("unsupported tag %q", tag)
	}
	return Paths{p.outputFilePath}, nil
}

func (p *buildPropModule) getBuildVariant(config Config) string {
	if config.Eng() {
		return "eng"
	} else if config.Debuggable() {
		return "userdebug"
	} else {
		return "user"
	}
}

func (p *buildPropModule) getBuildFlavor(config Config) string {
	buildFlavor := config.DeviceProduct() + "-" + p.getBuildVariant(config)
	if InList("address", config.SanitizeDevice()) && !strings.Contains(buildFlavor, "_asan") {
		buildFlavor += "_asan"
	}
	return buildFlavor
}

func (p *buildPropModule) shouldAddBuildThumbprint(config Config) bool {
	knownOemProperties := []string{
		"ro.product.brand",
		"ro.product.name",
		"ro.product.device",
	}

	for _, knownProp := range knownOemProperties {
		if InList(knownProp, config.OemProperties()) {
			return true
		}
	}
	return false
}

type buildInfo struct {
	AaptCharacteristics string
	AbOtaPartitions     string
	AbOtaUpdater        *bool `json:",omitempty"`

	Arch               string
	ArchVariantRuntime string

	BoardApiLevel         string
	BoardApiLevelFrozen   bool
	BoardPlatform         string
	BoardShippingApiLevel string

	BootloaderBoardName string

	Brand               string
	BrandForAttestation string

	BuildFlavor   string
	BuildId       string
	BuildKeys     string
	BuildType     string
	BuildUsername string
	BuildVariant  string

	CpuAbiList   string
	CpuAbiList32 string
	CpuAbiList64 string
	CpuAbis      []string

	DefaultLocale       *string `json:",omitempty"`
	DefaultWifiChannels []string

	Device               string
	DeviceForAttestation string

	Dex2oatTargetCpuVariantRuntime      string
	Dex2oatTargetInstructionSetFeatures string

	DisplayBuildNumber bool

	DontUseVabcOta bool

	EnableUffdGc string

	FullTreble bool

	Manufacturer               string
	ManufacturerForAttestation string

	MaxPageSizeSupported string

	Model               string
	ModelForAttestation string

	NameForAttestation string

	NoBionicPageSizeMacro bool `json:",omitempty"`

	PlatformBaseOs                       string
	PlatformDisplayVersion               string
	PlatformMinSupportedTargetSdkVersion string
	PlatformPreviewSdkVersion            string
	PlatformSdkVersion                   string
	PlatformSecurityPatch                string
	PlatformVersion                      string
	PlatformVersionAllCodenames          []string
	PlatformVersionCodename              string
	PlatformVersionKnownCodenames        string
	PlatformVersionLastStable            string

	Product                       string
	Product16kDeveloperOption     bool
	ProductShippingApiLevel       string
	ProductShippingVendorApiLevel string

	PropVariables map[string][]string

	PropertySplitEnabled bool

	RecoveryDefaultRotation string
	RecoveryOverscanPercent string
	RecoveryPixelFormat     string

	RetrofitDynamicPartitions *bool `json:",omitempty"`

	SanitizeTargets []string

	ScreenDensity string

	SdkBuild bool

	SecondaryArch                          string
	SecondaryArchVariantRuntime            string
	SecondaryDex2oatCpuVariantRuntime      string
	SecondaryDex2oatInstructionSetFeatures string

	SetDebugfsRestrictions bool `json:",omitempty"`

	Blocklist []string

	SystemBrand                string
	SystemDevice               string
	SystemManufacturer         string
	SystemModel                string
	SystemName                 string
	SystemServerCompilerFilter string

	UseDynamicPartitions         *bool `json:",omitempty"`
	UseVbmetaDigestInFingerprint bool
	UsesVulkan                   bool

	VendorSecurityPatch       string
	VendorImageFileSystemType string

	ZygoteForce64 bool
}

func (p *buildPropModule) buildInfoJson(ctx ModuleContext) Path {
	c := ctx.Config()
	d := ctx.DeviceConfig()

	variables := make(map[string][]string)
	variables["PRODUCT_SYSTEM_PROPERTIES"] = c.SystemProperties()
	variables["PRODUCT_SYSTEM_DEFAULT_PROPERTIES"] = c.SystemDefaultProperties()
	variables["PRODUCT_SYSTEM_EXT_PROPERTIES"] = c.SystemExtProperties()
	variables["PRODUCT_VENDOR_PROPERTIES"] = c.VendorProperties()
	variables["PRODUCT_PRODUCT_PROPERTIES"] = c.ProductProperties()
	variables["PRODUCT_ODM_PROPERTIES"] = c.OdmProperties()
	variables["PRODUCT_OEM_PROPERTIES"] = c.OemProperties()
	variables["PRODUCT_PROPERTY_OVERRIDES"] = c.PropertyOverrides()

	info := buildInfo{
		AaptCharacteristics: c.ProductAAPTCharacteristics(),
		AbOtaPartitions:     c.AbOtaPartitions(),
		AbOtaUpdater:        c.AbOtaUpdater(),

		Arch:               d.DeviceArch(),
		ArchVariantRuntime: d.DeviceCpuVariantRuntime(),

		Blocklist: append([]string{}, p.properties.Block_list...),

		BoardApiLevel:         c.VendorApiLevel(),
		BoardApiLevelFrozen:   c.VendorApiLevelFrozen(),
		BoardPlatform:         d.BoardPlatform(),
		BoardShippingApiLevel: d.BoardShippingApiLevel(),

		BootloaderBoardName: c.BootloaderBoardName(),

		Brand:               c.ProductBrand(),
		BrandForAttestation: c.ProductBrandForAttestation(),

		BuildFlavor:   p.getBuildFlavor(c),
		BuildId:       c.BuildId(),
		BuildKeys:     c.BuildKeys(),
		BuildType:     c.BuildType(),
		BuildUsername: c.Getenv("BUILD_USERNAME"),
		BuildVariant:  p.getBuildVariant(c),

		CpuAbiList:   c.DeviceAbiList(),
		CpuAbiList32: c.DeviceAbiList32(),
		CpuAbiList64: c.DeviceAbiList64(),
		CpuAbis:      c.DeviceAbi(),

		DefaultLocale:       nil, // see below
		DefaultWifiChannels: c.ProductDefaultWifiChannels(),

		Device:               c.DeviceName(),
		DeviceForAttestation: c.ProductDeviceForAttestation(),

		Dex2oatTargetCpuVariantRuntime:      d.Dex2oatTargetCpuVariantRuntime(),
		Dex2oatTargetInstructionSetFeatures: d.Dex2oatTargetInstructionSetFeatures(),

		DisplayBuildNumber: c.DisplayBuildNumber(),

		DontUseVabcOta: c.DontUseVabcOta(),

		EnableUffdGc: c.EnableUffdGc(),

		FullTreble: c.FullTreble(),

		Manufacturer:               c.ProductManufacturer(),
		ManufacturerForAttestation: c.ProductManufacturerForAttestation(),

		MaxPageSizeSupported: c.MaxPageSizeSupported(),

		Model:               c.ProductModel(),
		ModelForAttestation: c.ProductModelForAttestation(),

		NameForAttestation: c.ProductNameForAttestation(),

		NoBionicPageSizeMacro: c.NoBionicPageSizeMacro(),

		PlatformBaseOs:                       c.PlatformBaseOS(),
		PlatformDisplayVersion:               c.PlatformDisplayVersionName(),
		PlatformMinSupportedTargetSdkVersion: c.PlatformMinSupportedTargetSdkVersion(),
		PlatformPreviewSdkVersion:            c.PlatformPreviewSdkVersion(),
		PlatformSdkVersion:                   c.PlatformSdkVersion().String(),
		PlatformSecurityPatch:                c.PlatformSecurityPatch(),
		PlatformVersion:                      c.PlatformVersionName(),
		PlatformVersionCodename:              c.PlatformSdkCodename(),
		PlatformVersionAllCodenames:          c.PlatformVersionActiveCodenames(),
		PlatformVersionKnownCodenames:        c.PlatformVersionKnownCodenames(),
		PlatformVersionLastStable:            c.PlatformVersionLastStable(),

		Product:                       c.DeviceProduct(),
		Product16kDeveloperOption:     c.Product16KDeveloperOption(),
		ProductShippingApiLevel:       d.ShippingApiLevel().String(),
		ProductShippingVendorApiLevel: d.ShippingVendorApiLevel(),

		PropVariables: variables,

		PropertySplitEnabled: c.PropertySplitEnabled(),

		RecoveryDefaultRotation: c.RecoveryDefaultRotation(),
		RecoveryOverscanPercent: c.RecoveryOverscanPercent(),
		RecoveryPixelFormat:     c.RecoveryPixelFormat(),

		RetrofitDynamicPartitions: c.RetrofitDynamicPartitions(),

		// workaround: SanitizeDevice() returns nil if empty.
		// make it []string{}, rather than nil, to simplify code
		SanitizeTargets: append([]string{}, c.SanitizeDevice()...),

		ScreenDensity: c.ScreenDensity(),

		SdkBuild: proptools.Bool(p.properties.Sdk_build),

		SecondaryArch:                          d.DeviceSecondaryArch(),
		SecondaryArchVariantRuntime:            d.DeviceSecondaryArchVariantRuntime(),
		SecondaryDex2oatCpuVariantRuntime:      d.SecondaryDex2oatCpuVariantRuntime(),
		SecondaryDex2oatInstructionSetFeatures: d.SecondaryDex2oatInstructionSetFeatures(),

		SetDebugfsRestrictions: d.BuildDebugfsRestrictionsEnabled(),

		SystemBrand:                c.SystemBrand(),
		SystemDevice:               c.SystemDevice(),
		SystemManufacturer:         c.SystemManufacturer(),
		SystemModel:                c.SystemModel(),
		SystemName:                 c.SystemName(),
		SystemServerCompilerFilter: c.SystemServerCompilerFilter(),

		UseDynamicPartitions:         c.UseDynamicPartitions(),
		UseVbmetaDigestInFingerprint: c.BoardUseVbmetaDigestInFingerprint(),
		UsesVulkan:                   c.UsesVulkan(),

		VendorSecurityPatch:       c.VendorSecurityPatch(),
		VendorImageFileSystemType: c.VendorImageFileSystemType(),

		ZygoteForce64: c.ZygoteForce64(),
	}

	if len(c.ProductLocales()) > 0 {
		info.DefaultLocale = proptools.StringPtr(c.ProductLocales()[0])
	}

	jsonPath := PathForModuleOut(ctx, "build_info.json").OutputPath
	content, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Errorf("buildinfo marshal failed: %w", err))
	}

	WriteFileRuleVerbatim(ctx, jsonPath, string(content[:]))

	return jsonPath
}

func (p *buildPropModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	p.outputFilePath = PathForModuleOut(ctx, "build.prop").OutputPath
	if !ctx.Config().KatiEnabled() {
		WriteFileRule(ctx, p.outputFilePath, "# no build.prop if kati is disabled")
		return
	}

	partition := proptools.String(p.properties.Partition)
	if partition != "system" {
		ctx.PropertyErrorf("partition", "unsupported partition %q: only \"system\" is supported", partition)
		return
	}

	rule := NewRuleBuilder(pctx, ctx)

	config := ctx.Config()

	cmd := rule.Command().BuiltTool("gen_build_prop")
	cmd.ImplicitOutput(p.outputFilePath)

	cmd.FlagWithInput("--build-hostname-file=", config.BuildHostnameFile(ctx))
	// shouldn't depend on BuildNumberFile, BuildFingerprintFile, and BuildThumbprintFile to prevent
	// from rebuilding on every incremental build.
	cmd.FlagWithArg("--build-fingerprint-file=", config.BuildFingerprintFile(ctx).String())
	cmd.FlagWithArg("--build-number-file=", config.BuildNumberFile(ctx).String())
	if p.shouldAddBuildThumbprint(config) {
		cmd.FlagWithArg("--build-thumbprint-file=", config.BuildThumbprintFile(ctx).String())
	}
	// shouldn't depend on BUILD_DATETIME_FILE to prevent from rebuilding on every incremental
	// build.
	cmd.FlagWithArg("--date-file=", ctx.Config().Getenv("BUILD_DATETIME_FILE"))
	cmd.FlagWithInput("--platform-preview-sdk-fingerprint-file=", ApiFingerprintPath(ctx))
	cmd.FlagWithInput("--build-info=", p.buildInfoJson(ctx))
	cmd.FlagWithArg("--partition=", partition)
	cmd.FlagWithArg("--out=", p.outputFilePath.String())

	cmd = rule.Command().BuiltTool("post_process_props")
	if ctx.DeviceConfig().BuildBrokenDupSysprop() {
		cmd.Flag("--allow-dup")
	}
	cmd.FlagWithArg("--sdk-version ", config.PlatformSdkVersion().String())
	cmd.FlagWithInput("--kernel-version-file-for-uffd-gc ", PathForOutput(ctx, "dexpreopt/kernel_version_for_uffd_gc.txt"))
	cmd.Text(p.outputFilePath.String())
	cmd.Flags(p.properties.Block_list)

	rule.Command().Text("echo").Text(proptools.NinjaAndShellEscape("# end of file")).FlagWithArg(">> ", p.outputFilePath.String())

	rule.Build(ctx.ModuleName(), "generating buildinfo props")

	if !p.installable() {
		p.SkipInstall()
	}

	p.installPath = PathForModuleInstall(ctx)
	ctx.InstallFile(p.installPath, p.stem(), p.outputFilePath)
}

func (p *buildPropModule) AndroidMkEntries() []AndroidMkEntries {
	return []AndroidMkEntries{AndroidMkEntries{
		Class:      "ETC",
		OutputFile: OptionalPathForPath(p.outputFilePath),
		ExtraEntries: []AndroidMkExtraEntriesFunc{
			func(ctx AndroidMkExtraEntriesContext, entries *AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", p.installPath.String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", p.stem())
				entries.SetBoolIfTrue("LOCAL_UNINSTALLABLE_MODULE", !p.installable())
			},
		},
	}}
}

// build_prop module generates {partition}/build.prop file. At first common build properties are
// printed based on Soong config variables. And then prop_files are printed as-is. Finally,
// post_process_props tool is run to check if the result build.prop is valid or not.
func buildPropFactory() Module {
	module := &buildPropModule{}
	module.AddProperties(&module.properties)
	InitAndroidArchModule(module, DeviceSupported, MultilibFirst)
	return module
}
