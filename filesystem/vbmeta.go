// Copyright (C) 2021 The Android Open Source Project
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

package filesystem

import (
	"fmt"
	"strconv"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("vbmeta", vbmetaFactory)
	pctx.HostBinToolVariable("avbtool", "avbtool")
}

var (
	extractPublicKeyRule = pctx.AndroidStaticRule("avb_extract_public_key",
		blueprint.RuleParams{
			Command: `${avbtool} extract_public_key --key $in --output $out`,
			CommandDeps: []string{
				"${avbtool}",
			},
		})
)

type vbmeta struct {
	android.ModuleBase

	properties vbmetaProperties

	output     android.OutputPath
	installDir android.InstallPath
}

type vbmetaProperties struct {
	// Name of the partition stored in vbmeta desc. Defaults to the name of this module.
	Partition_name *string

	// Set the name of the output. Defaults to <module_name>.img.
	Stem *string

	// Path to the private key that avbtool will use to sign this vbmeta image.
	Private_key *string `android:"path"`

	// Algorithm that avbtool will use to sign this vbmeta image. Default is SHA256_RSA4096.
	Algorithm *string

	// File whose content will provide the rollback index. If unspecified, the rollback index
	// is from PLATFORM_SECURITY_PATCH
	Rollback_index_file *string `android:"path"`

	// Rollback index location of this vbmeta image. Must be 0, 1, 2, etc. Default is 0.
	Rollback_index_location *int64

	// List of filesystem modules that this vbmeta has descriptors for. The filesystem modules
	// have to be signed (use_avb: true).
	Partitions proptools.Configurable[[]string]

	// List of chained partitions that this vbmeta deletages the verification. They are the
	// names of other vbmeta modules.
	Chained_partitions []string

	// List of key-value pair of avb properties
	Avb_properties []avbProperty
}

type avbProperty struct {
	// Key of given avb property
	Key *string

	// Value of given avb property
	Value *string
}

type vbmetaPartitionInfo struct {
	// Name of the partition
	Name string

	// Rollback index location, non-negative int
	RollbackIndexLocation int

	// The path to the public key of the private key used to sign this partition. Derived from
	// the private key.
	PublicKey android.Path
}

var vbmetaPartitionProvider = blueprint.NewProvider[vbmetaPartitionInfo]()

// vbmeta is the partition image that has the verification information for other partitions.
func vbmetaFactory() android.Module {
	module := &vbmeta{}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

type vbmetaDep struct {
	blueprint.BaseDependencyTag
}

type chainedPartitionDep struct {
	blueprint.BaseDependencyTag
}

var vbmetaPartitionDep = vbmetaDep{}
var vbmetaChainedPartitionDep = chainedPartitionDep{}

func (v *vbmeta) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddDependency(ctx.Module(), vbmetaPartitionDep, v.properties.Partitions.GetOrDefault(ctx, nil)...)
	ctx.AddDependency(ctx.Module(), vbmetaChainedPartitionDep, v.properties.Chained_partitions...)
}

func (v *vbmeta) installFileName() string {
	return proptools.StringDefault(v.properties.Stem, v.BaseModuleName()+".img")
}

func (v *vbmeta) partitionName() string {
	return proptools.StringDefault(v.properties.Partition_name, v.BaseModuleName())
}

// See external/avb/libavb/avb_slot_verify.c#VBMETA_MAX_SIZE
const vbmetaMaxSize = 64 * 1024

func (v *vbmeta) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	v.output = android.PathForModuleOut(ctx, v.installFileName()).OutputPath

	builder := android.NewRuleBuilder(pctx, ctx)
	cmd := builder.Command().BuiltTool("avbtool").Text("make_vbmeta_image")

	key := android.PathForModuleSrc(ctx, proptools.String(v.properties.Private_key))
	cmd.FlagWithInput("--key ", key)

	algorithm := proptools.StringDefault(v.properties.Algorithm, "SHA256_RSA4096")
	cmd.FlagWithArg("--algorithm ", algorithm)

	cmd.FlagWithArg("--rollback_index ", v.rollbackIndexCommand(ctx))
	ril := proptools.IntDefault(v.properties.Rollback_index_location, 0)
	if ril < 0 {
		ctx.PropertyErrorf("rollback_index_location", "must be 0, 1, 2, ...")
		return
	}
	cmd.FlagWithArg("--rollback_index_location ", strconv.Itoa(ril))

	for _, avb_prop := range v.properties.Avb_properties {
		key := proptools.String(avb_prop.Key)
		if key == "" {
			ctx.PropertyErrorf("avb_properties", "key must be specified")
			continue
		}
		value := proptools.String(avb_prop.Value)
		if value == "" {
			ctx.PropertyErrorf("avb_properties", "value must be specified")
			continue
		}
		cmd.FlagWithArg("--prop ", key+":"+value)
	}

	for _, p := range ctx.GetDirectDepsWithTag(vbmetaPartitionDep) {
		f, ok := p.(Filesystem)
		if !ok {
			ctx.PropertyErrorf("partitions", "%q(type: %s) is not supported",
				p.Name(), ctx.OtherModuleType(p))
			continue
		}
		signedImage := f.SignedOutputPath()
		if signedImage == nil {
			ctx.PropertyErrorf("partitions", "%q(type: %s) is not signed. Use `use_avb: true`",
				p.Name(), ctx.OtherModuleType(p))
			continue
		}
		cmd.FlagWithInput("--include_descriptors_from_image ", signedImage)
	}

	seenRils := make(map[int]bool)
	for _, cp := range ctx.GetDirectDepsWithTag(vbmetaChainedPartitionDep) {
		info, ok := android.OtherModuleProvider(ctx, cp, vbmetaPartitionProvider)
		if !ok {
			ctx.PropertyErrorf("chained_partitions", "Expected all modules in chained_partitions to provide vbmetaPartitionProvider, but %s did not", cp.Name())
			continue
		}
		if info.Name == "" {
			ctx.PropertyErrorf("chained_partitions", "name must be specified")
			continue
		}

		ril := info.RollbackIndexLocation
		if ril < 0 {
			ctx.PropertyErrorf("chained_partitions", "must be 0, 1, 2, ...")
			continue
		} else if seenRils[ril] {
			ctx.PropertyErrorf("chained_partitions", "Multiple chained partitions with the same rollback index location %d", ril)
			continue
		}
		seenRils[ril] = true

		publicKey := info.PublicKey
		cmd.FlagWithArg("--chain_partition ", fmt.Sprintf("%s:%d:%s", info.Name, ril, publicKey.String()))
		cmd.Implicit(publicKey)
	}

	cmd.FlagWithOutput("--output ", v.output)

	// libavb expects to be able to read the maximum vbmeta size, so we must provide a partition
	// which matches this or the read will fail.
	builder.Command().Text("truncate").
		FlagWithArg("-s ", strconv.Itoa(vbmetaMaxSize)).
		Output(v.output)

	builder.Build("vbmeta", fmt.Sprintf("vbmeta %s", ctx.ModuleName()))

	v.installDir = android.PathForModuleInstall(ctx, "etc")
	ctx.InstallFile(v.installDir, v.installFileName(), v.output)

	extractedPublicKey := android.PathForModuleOut(ctx, v.partitionName()+".avbpubkey")
	ctx.Build(pctx, android.BuildParams{
		Rule:   extractPublicKeyRule,
		Input:  key,
		Output: extractedPublicKey,
	})

	android.SetProvider(ctx, vbmetaPartitionProvider, vbmetaPartitionInfo{
		Name:                  v.partitionName(),
		RollbackIndexLocation: ril,
		PublicKey:             extractedPublicKey,
	})

	ctx.SetOutputFiles([]android.Path{v.output}, "")
}

// Returns the embedded shell command that prints the rollback index
func (v *vbmeta) rollbackIndexCommand(ctx android.ModuleContext) string {
	var cmd string
	if v.properties.Rollback_index_file != nil {
		f := android.PathForModuleSrc(ctx, proptools.String(v.properties.Rollback_index_file))
		cmd = "cat " + f.String()
	} else {
		cmd = "date -d 'TZ=\"GMT\" " + ctx.Config().PlatformSecurityPatch() + "' +%s"
	}
	// Take the first line and remove the newline char
	return "$(" + cmd + " | head -1 | tr -d '\n'" + ")"
}

var _ android.AndroidMkProviderInfoProducer = (*vbmeta)(nil)

func (v *vbmeta) PrepareAndroidMKProviderInfo(config android.Config) *android.AndroidMkProviderInfo {
	providerData := android.AndroidMkProviderInfo{
		PrimaryInfo: android.AndroidMkInfo{
			Class:      "ETC",
			OutputFile: android.OptionalPathForPath(v.output),
			EntryMap:   make(map[string][]string),
		},
	}
	providerData.PrimaryInfo.SetString("LOCAL_MODULE_PATH", v.installDir.String())
	providerData.PrimaryInfo.SetString("LOCAL_INSTALLED_MODULE_STEM", v.installFileName())
	return &providerData
}

var _ Filesystem = (*vbmeta)(nil)

func (v *vbmeta) OutputPath() android.Path {
	return v.output
}

func (v *vbmeta) SignedOutputPath() android.Path {
	return v.OutputPath() // vbmeta is always signed
}
