// Copyright 2024 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/blueprint"
)

var (
	// Constants of property names used in metadata of modules
	MetadataProp = struct {
		NAME             string
		PACKAGE          string
		MODULE_TYPE      string
		ARCH             string
		PRIMARY_ARCH     string
		VARIANT          string
		INSTALLED_FILES  string
		BUILT_FILES      string
		STATIC_DEPS      string
		STATIC_DEP_FILES string

		// module_type=package
		PKG_DEFAULT_APPLICABLE_LICENSE string

		// module_type=license
		LIC_LICENSE_KINDS string
		LIC_LICENSE_TEXT  string
		LIC_PACKAGE_NAME  string

		// module_type=license_kind
		LK_CONDITIONS string
		LK_URL        string
	}{
		"name",
		"package",
		"module_type",
		"arch",
		"primary_arch",
		"variant",
		"installed_files",
		"built_files",
		"static_deps",
		"static_dep_files",

		"pkg_default_applicable_licenses",

		"lic_license_kinds",
		"lic_license_text",
		"lic_package_name",

		"lk_conditions",
		"lk_url",
	}

	// A constant list of all property names in metadata
	METADATA_PROPS = []string{
		MetadataProp.NAME,
		MetadataProp.PACKAGE,
		MetadataProp.MODULE_TYPE,
		MetadataProp.ARCH,
		MetadataProp.PRIMARY_ARCH,
		MetadataProp.VARIANT,
		MetadataProp.INSTALLED_FILES,
		MetadataProp.BUILT_FILES,
		MetadataProp.STATIC_DEPS,
		MetadataProp.STATIC_DEP_FILES,
		// module_type=package
		MetadataProp.PKG_DEFAULT_APPLICABLE_LICENSE,
		// module_type=license
		MetadataProp.LIC_LICENSE_KINDS,
		MetadataProp.LIC_LICENSE_TEXT, // resolve to file paths
		MetadataProp.LIC_PACKAGE_NAME,
		// module_type=license_kind
		MetadataProp.LK_CONDITIONS,
		MetadataProp.LK_URL,
	}
)

// MetadataInfo provides all metadata of a module, e.g. name, module type, package, license,
// dependencies, input/output files, etc. It is a wrapper on a map[string]string with some utility
// methods to get/set properties' values.
type MetadataInfo struct {
	properties map[string]string
}

func NewMetadataInfo() *MetadataInfo {
	return &MetadataInfo{
		properties: map[string]string{},
	}
}

func (this *MetadataInfo) SetStringValue(propertyName string, value string) {
	if !slices.Contains(METADATA_PROPS, propertyName) {
		panic(fmt.Errorf("Unknown metadata property: %s.", propertyName))
	}
	this.properties[propertyName] = value
}

func (this *MetadataInfo) SetListValue(property string, value []string) {
	this.SetStringValue(property, strings.TrimSpace(strings.Join(value, " ")))
}

func (this *MetadataInfo) getStringValue(property string) string {
	if !slices.Contains(METADATA_PROPS, property) {
		panic(fmt.Errorf("Unknown metadata property: %s.", property))
	}
	return this.properties[property]
}

func (this *MetadataInfo) getAllValues() map[string]string {
	return this.properties
}

var (
	MetadataProvider = blueprint.NewProvider[*MetadataInfo]()
)

func buildMetadataProvider(ctx ModuleContext, m *ModuleBase) {
	metadataInfo := ctx.MetadataInfo()
	metadataInfo.SetStringValue(MetadataProp.NAME, m.Name())
	metadataInfo.SetStringValue(MetadataProp.PACKAGE, ctx.ModuleDir())
	metadataInfo.SetStringValue(MetadataProp.MODULE_TYPE, ctx.ModuleType())

	switch ctx.ModuleType() {
	case "license":
		metadataInfo.SetListValue(MetadataProp.LIC_LICENSE_KINDS, m.module.(*licenseModule).properties.License_kinds)
		metadataInfo.SetListValue(MetadataProp.LIC_LICENSE_TEXT, m.module.(*licenseModule).properties.License_text)
		metadataInfo.SetStringValue(MetadataProp.LIC_PACKAGE_NAME, String(m.module.(*licenseModule).properties.Package_name))
	case "license_kind":
		metadataInfo.SetListValue(MetadataProp.LK_CONDITIONS, m.module.(*licenseKindModule).properties.Conditions)
		metadataInfo.SetStringValue(MetadataProp.LK_URL, m.module.(*licenseKindModule).properties.Url)
	default:
		metadataInfo.SetStringValue(MetadataProp.ARCH, ctx.Arch().String())
		metadataInfo.SetStringValue(MetadataProp.PRIMARY_ARCH, strconv.FormatBool(ctx.PrimaryArch()))
		metadataInfo.SetStringValue(MetadataProp.VARIANT, ctx.ModuleSubDir())

		var installed InstallPaths
		installed = append(installed, m.module.FilesToInstall()...)
		installed = append(installed, m.katiInstalls.InstallPaths()...)
		installed = append(installed, m.katiSymlinks.InstallPaths()...)
		installed = append(installed, m.katiInitRcInstalls.InstallPaths()...)
		installed = append(installed, m.katiVintfInstalls.InstallPaths()...)
		metadataInfo.SetListValue(MetadataProp.INSTALLED_FILES, installed.Strings())
	}
	ctx.setProvider(MetadataProvider, metadataInfo)
}

func init() {
	RegisterMetadataSingleton(InitRegistrationContext)
}

func RegisterMetadataSingleton(ctx RegistrationContext) {
	ctx.RegisterParallelSingletonType("metadata_singleton", metadataSingletonFactory)
}

var (
	PrepareForTestWithMetadataSingleton = FixtureRegisterWithContext(RegisterMetadataSingleton)

	sqlite3 = pctx.HostBinToolVariable("sqlite3", "sqlite3")

	importCsv = pctx.AndroidStaticRule("importCsv",
		blueprint.RuleParams{
			Command: `rm -rf $out && ` +
				`${sqlite3} $out ".import --csv $in modules" && ` +
				`${sqlite3} $out ".import --csv ${make_metadata} make_metadata"`,
			CommandDeps: []string{"${sqlite3}"},
		}, "make_metadata")
)

func metadataSingletonFactory() Singleton {
	return &metadataSingleton{}
}

type metadataSingleton struct {
}

// Collect metadata from all modules, write to a CSV file and import metadata from Make and Soong to a sqlite database.
func (this *metadataSingleton) GenerateBuildActions(ctx SingletonContext) {
	if !ctx.Config().HasDeviceProduct() {
		return
	}
	// Metadata of modules in Soong
	allModules := make([][]string, 0)
	columnNames := []string{"id"}
	columnNames = append(columnNames, METADATA_PROPS...)
	allModules = append(allModules, columnNames)
	rowId := -1

	ctx.VisitAllModules(func(module Module) {
		if !module.Enabled() {
			return
		}
		moduleType := ctx.ModuleType(module)
		if moduleType == "package" {
			metadataMap := map[string]string{
				MetadataProp.NAME:                           ctx.ModuleName(module),
				MetadataProp.MODULE_TYPE:                    ctx.ModuleType(module),
				MetadataProp.PKG_DEFAULT_APPLICABLE_LICENSE: strings.Join(module.base().primaryLicensesProperty.getStrings(), " "),
			}
			rowId = rowId + 1
			metadata := []string{strconv.Itoa(rowId)}
			for _, propertyName := range METADATA_PROPS {
				metadata = append(metadata, metadataMap[propertyName])
			}
			allModules = append(allModules, metadata)
			return
		}
		if provider, ok := ctx.moduleProvider(module, MetadataProvider); ok {
			metadataInfo := provider.(*MetadataInfo)
			rowId = rowId + 1
			metadata := []string{strconv.Itoa(rowId)}
			for _, propertyName := range METADATA_PROPS {
				metadata = append(metadata, metadataInfo.getStringValue(propertyName))
			}
			allModules = append(allModules, metadata)
			return
		}
	})
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	err := w.WriteAll(allModules)
	if err != nil {
		panic(err)
	}
	deviceProduct := ctx.Config().DeviceProduct()
	modulesCsv := PathForOutput(ctx, "metadata", deviceProduct, "metadata.csv")
	WriteFileRuleVerbatim(ctx, modulesCsv, b.String())

	// Metadata generated in Make
	makeMetadataCsv := PathForOutput(ctx, "metadata", deviceProduct, "make-metadata.csv")

	// Import both
	metadataDb := PathForOutput(ctx, "metadata", deviceProduct, "metadata.db")
	ctx.Build(pctx, BuildParams{
		Rule:     importCsv,
		Input:    modulesCsv,
		Implicit: makeMetadataCsv,
		Output:   metadataDb,
		Args: map[string]string{
			"make_metadata": makeMetadataCsv.String(),
		},
	})

	// "m metadata.db" to create the metadata database
	ctx.Build(pctx, BuildParams{
		Rule:   blueprint.Phony,
		Inputs: []Path{metadataDb},
		Output: PathForPhony(ctx, "metadata.db"),
	})

}
