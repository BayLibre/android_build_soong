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
		NAME                   string
		PACKAGE                string
		MODULE_TYPE            string
		OS                     string
		ARCH                   string
		IS_PRIMARY_ARCH        string
		VARIANT                string
		IS_STATIC_LIB          string
		INSTALLED_FILES        string
		BUILT_FILES            string
		STATIC_DEPS            string
		STATIC_DEP_FILES       string
		WHOLE_STATIC_DEPS      string
		WHOLE_STATIC_DEP_FILES string
		LICENSES               string

		// module_type=package
		PKG_DEFAULT_APPLICABLE_LICENSES string

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
		"os",
		"arch",
		"is_primary_arch",
		"variant",
		"is_static_lib",
		"installed_files",
		"built_files",
		"static_deps",
		"static_dep_files",
		"whole_static_deps",
		"whole_static_dep_files",
		"licenses",

		"pkg_default_applicable_licenses",

		"lic_license_kinds",
		"lic_license_text",
		"lic_package_name",

		"lk_conditions",
		"lk_url",
	}

	// A constant list of all property names in metadata
	// Order of properties here is the order of columns in the exported metadata.csv file.
	METADATA_PROPS = []string{
		MetadataProp.NAME,
		MetadataProp.PACKAGE,
		MetadataProp.MODULE_TYPE,
		MetadataProp.OS,
		MetadataProp.ARCH,
		MetadataProp.VARIANT,
		MetadataProp.IS_STATIC_LIB,
		MetadataProp.IS_PRIMARY_ARCH,
		// Space separated installed files
		MetadataProp.INSTALLED_FILES,
		// Space separated built files
		MetadataProp.BUILT_FILES,
		// Space separated module names of static dependencies
		MetadataProp.STATIC_DEPS,
		// Space separated file paths of static dependencies
		MetadataProp.STATIC_DEP_FILES,
		// Space separated module names of whole static dependencies
		MetadataProp.WHOLE_STATIC_DEPS,
		// Space separated file paths of whole static dependencies
		MetadataProp.WHOLE_STATIC_DEP_FILES,
		MetadataProp.LICENSES,
		// module_type=package
		MetadataProp.PKG_DEFAULT_APPLICABLE_LICENSES,
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
// dependencies, built/installed files, etc. It is a wrapper on a map[string]string with some utility
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

func (this *MetadataInfo) SetListValue(propertyName string, value []string) {
	this.SetStringValue(propertyName, strings.TrimSpace(strings.Join(value, " ")))
}

func (this *MetadataInfo) getStringValue(propertyName string) string {
	if !slices.Contains(METADATA_PROPS, propertyName) {
		panic(fmt.Errorf("Unknown metadata property: %s.", propertyName))
	}
	return this.properties[propertyName]
}

func (this *MetadataInfo) getAllValues() map[string]string {
	return this.properties
}

var (
	MetadataProvider = blueprint.NewProvider[*MetadataInfo]()
)

// buildMetadataProvider starts with the ModuleContext.MetadataInfo() and fills in more common metadata
// for different module types without accessing their private fields but through android.Module interface
// and public/private fields of package android. The final metadata is stored to a module's MetadataProvider.
func buildMetadataProvider(ctx ModuleContext, m *ModuleBase) {
	metadataInfo := ctx.MetadataInfo()
	metadataInfo.SetStringValue(MetadataProp.NAME, m.Name())
	metadataInfo.SetStringValue(MetadataProp.PACKAGE, ctx.ModuleDir())
	metadataInfo.SetStringValue(MetadataProp.MODULE_TYPE, ctx.ModuleType())

	switch ctx.ModuleType() {
	case "license":
		licenseModule := m.module.(*licenseModule)
		metadataInfo.SetListValue(MetadataProp.LIC_LICENSE_KINDS, licenseModule.properties.License_kinds)
		metadataInfo.SetListValue(MetadataProp.LIC_LICENSE_TEXT, PathsForModuleSrc(ctx, licenseModule.properties.License_text).Strings())
		metadataInfo.SetStringValue(MetadataProp.LIC_PACKAGE_NAME, String(licenseModule.properties.Package_name))
	case "license_kind":
		licenseKindModule := m.module.(*licenseKindModule)
		metadataInfo.SetListValue(MetadataProp.LK_CONDITIONS, licenseKindModule.properties.Conditions)
		metadataInfo.SetStringValue(MetadataProp.LK_URL, licenseKindModule.properties.Url)
	default:
		metadataInfo.SetStringValue(MetadataProp.OS, ctx.Os().String())
		metadataInfo.SetStringValue(MetadataProp.ARCH, ctx.Arch().String())
		metadataInfo.SetStringValue(MetadataProp.IS_PRIMARY_ARCH, strconv.FormatBool(ctx.PrimaryArch()))
		metadataInfo.SetStringValue(MetadataProp.VARIANT, ctx.ModuleSubDir())
		if m.primaryLicensesProperty != nil && m.primaryLicensesProperty.getName() == "licenses" {
			metadataInfo.SetListValue(MetadataProp.LICENSES, m.primaryLicensesProperty.getStrings())
		}

		var installed InstallPaths
		installed = append(installed, m.module.FilesToInstall()...)
		installed = append(installed, m.katiInstalls.InstallPaths()...)
		installed = append(installed, m.katiSymlinks.InstallPaths()...)
		installed = append(installed, m.katiInitRcInstalls.InstallPaths()...)
		installed = append(installed, m.katiVintfInstalls.InstallPaths()...)
		metadataInfo.SetListValue(MetadataProp.INSTALLED_FILES, FirstUniqueStrings(installed.Strings()))
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

	// sqlite3 command line tool
	sqlite3 = pctx.HostBinToolVariable("sqlite3", "sqlite3")

	// Command to import .csv files to sqlite3 database
	importCsv = pctx.AndroidStaticRule("importCsv",
		blueprint.RuleParams{
			Command: `rm -rf $out && ` +
				`${sqlite3} $out ".import --csv $in modules" && ` +
				`${sqlite3} $out ".import --csv ${make_metadata} make_metadata" && ` +
				`${sqlite3} $out ".import --csv ${make_modules} make_modules"`,
			CommandDeps: []string{"${sqlite3}"},
		}, "make_metadata", "make_modules")
)

func metadataSingletonFactory() Singleton {
	return &metadataSingleton{}
}

type metadataSingleton struct {
}

// Collect metadata from all Soong modules, write to a CSV file and
// import metadata from Make and Soong to a sqlite3 database.
func (this *metadataSingleton) GenerateBuildActions(ctx SingletonContext) {
	if !ctx.Config().HasDeviceProduct() {
		return
	}
	// Collect metadata of modules in Soong and write to out/soong/metadata/<product>/metadata.csv file.
	allModules := make([][]string, 0)
	columnNames := []string{"id"}
	columnNames = append(columnNames, METADATA_PROPS...)
	allModules = append(allModules, columnNames)
	rowId := -1

	ctx.VisitAllModules(func(module Module) {
		if !module.Enabled(ctx) {
			return
		}
		moduleType := ctx.ModuleType(module)
		if moduleType == "package" {
			metadataMap := map[string]string{
				MetadataProp.NAME:                            ctx.ModuleName(module),
				MetadataProp.MODULE_TYPE:                     ctx.ModuleType(module),
				MetadataProp.PKG_DEFAULT_APPLICABLE_LICENSES: strings.Join(module.base().primaryLicensesProperty.getStrings(), " "),
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
	makeModulesCsv := PathForOutput(ctx, "metadata", deviceProduct, "make-modules.csv")

	// Import metadata from Make and Soong to sqlite3 database
	metadataDb := PathForOutput(ctx, "metadata", deviceProduct, "metadata.db")
	ctx.Build(pctx, BuildParams{
		Rule:  importCsv,
		Input: modulesCsv,
		Implicits: []Path{
			makeMetadataCsv,
			makeModulesCsv,
		},
		Output: metadataDb,
		Args: map[string]string{
			"make_metadata": makeMetadataCsv.String(),
			"make_modules":  makeModulesCsv.String(),
		},
	})

	// Phony rule "metadata.db". "m metadata.db" to create the metadata database.
	ctx.Build(pctx, BuildParams{
		Rule:   blueprint.Phony,
		Inputs: []Path{metadataDb},
		Output: PathForPhony(ctx, "metadata.db"),
	})

}
