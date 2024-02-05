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

const (
	NAME            = "name"
	PACKAGE         = "package"
	MODULE_TYPE     = "module_type"
	ARCH            = "arch"
	PRIMARY_ARCH    = "primary_arch"
	VARIANT         = "variant"
	INSTALLED_FILES = "installed_files"

	// module_type=package
	PKG_DEFAULT_APPLICABLE_LICENSE = "pkg_default_applicable_licenses"

	// module_type=license
	LIC_LICENSE_KINDS = "lic_license_kinds"
	LIC_LICENSE_TEXT  = "lic_license_text"
	LIC_PACKAGE_NAME  = "lic_package_name"

	// module_type=license_kind
	LK_CONDITIONS = "lk_conditions"
	LK_URL        = "lk_url"
)

var METADATA_PROPERTIES = []string{
	NAME,
	PACKAGE,
	MODULE_TYPE,
	ARCH,
	PRIMARY_ARCH,
	VARIANT,
	INSTALLED_FILES,
	// module_type=package
	PKG_DEFAULT_APPLICABLE_LICENSE,
	// module_type=license
	LIC_LICENSE_KINDS,
	LIC_LICENSE_TEXT, // resolve to file paths
	LIC_PACKAGE_NAME,
	// module_type=license_kind
	LK_CONDITIONS,
	LK_URL,
}

// MetadataInfo provides all metadata of a module, e.g. name, module type, package, license,
// dependencies, input/output files, etc. It is a wrapper on a map[string]string with some utility
// methods to get/set propertie's values.
type MetadataInfo struct {
	properties map[string]string
}

func (this *MetadataInfo) SetStringValue(propertyName string, value string) {
	if !slices.Contains(METADATA_PROPERTIES, propertyName) {
		panic(fmt.Errorf("Unknown metadata property: %s.", propertyName))
	}
	this.properties[propertyName] = value
}

func (this *MetadataInfo) SetListValue(property string, value []string) {
	this.SetStringValue(property, strings.Join(value, " "))
}

func (this *MetadataInfo) getStringValue(property string) string {
	if !slices.Contains(METADATA_PROPERTIES, property) {
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
	metadataInfo := MetadataInfo{
		properties: map[string]string{},
	}
	metadataInfo.SetStringValue(NAME, m.Name())
	metadataInfo.SetStringValue(PACKAGE, ctx.ModuleDir())
	metadataInfo.SetStringValue(MODULE_TYPE, ctx.ModuleType())
	switch ctx.ModuleType() {
	case "license":
		metadataInfo.SetListValue(LIC_LICENSE_KINDS, m.module.(*licenseModule).properties.License_kinds)
		metadataInfo.SetListValue(LIC_LICENSE_TEXT, m.module.(*licenseModule).properties.License_text)
		metadataInfo.SetStringValue(LIC_PACKAGE_NAME, String(m.module.(*licenseModule).properties.Package_name))
	case "license_kind":
		metadataInfo.SetListValue(LK_CONDITIONS, m.module.(*licenseKindModule).properties.Conditions)
		metadataInfo.SetStringValue(LK_URL, m.module.(*licenseKindModule).properties.Url)
	default:
		metadataInfo.SetStringValue(ARCH, ctx.Arch().String())
		metadataInfo.SetStringValue(PRIMARY_ARCH, strconv.FormatBool(ctx.PrimaryArch()))
		metadataInfo.SetStringValue(VARIANT, ctx.ModuleSubDir())

		var installed InstallPaths
		installed = append(installed, m.module.FilesToInstall()...)
		installed = append(installed, m.katiInstalls.InstallPaths()...)
		installed = append(installed, m.katiSymlinks.InstallPaths()...)
		installed = append(installed, m.katiInitRcInstalls.InstallPaths()...)
		installed = append(installed, m.katiVintfInstalls.InstallPaths()...)
		metadataInfo.SetListValue(INSTALLED_FILES, installed.Strings())
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

func (this *metadataSingleton) GenerateBuildActions(ctx SingletonContext) {
	// Metadata of modules in Soong
	allModules := make([][]string, 0)
	allModules = append(allModules, METADATA_PROPERTIES)

	ctx.VisitAllModules(func(module Module) {
		if !module.Enabled() {
			return
		}
		moduleType := ctx.ModuleType(module)
		if moduleType == "package" {
			metadataMap := map[string]string{
				NAME:                           ctx.ModuleName(module),
				MODULE_TYPE:                    ctx.ModuleType(module),
				PKG_DEFAULT_APPLICABLE_LICENSE: strings.Join(module.base().primaryLicensesProperty.getStrings(), " "),
			}
			metadata := []string{}
			for _, propertyName := range METADATA_PROPERTIES {
				metadata = append(metadata, metadataMap[propertyName])
			}
			allModules = append(allModules, metadata)
			return
		}
		if provider, ok := ctx.moduleProvider(module, MetadataProvider); ok {
			metadataInfo := provider.(MetadataInfo)
			var metadata []string
			for _, propertyName := range METADATA_PROPERTIES {
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

	ctx.Build(pctx, BuildParams{
		Rule:   blueprint.Phony,
		Inputs: []Path{metadataDb},
		Output: PathForPhony(ctx, "metadata.db"),
	})

}
