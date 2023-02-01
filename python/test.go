// Copyright 2017 Google Inc. All rights reserved.
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

package python

import (
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/tradefed"
)

// This file contains the module types for building Python test.

func init() {
	registerPythonTestComponents(android.InitRegistrationContext)
}

func registerPythonTestComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("python_test_host", PythonTestHostFactory)
	ctx.RegisterModuleType("python_test", PythonTestFactory)
	//ctx.RegisterModuleType("python_multidevice_test_host", PythonMultideviceTestHostFactory)
}

func NewTest(hod android.HostOrDeviceSupported) *PythonTestModule {
	return &PythonTestModule{PythonBinaryModule: *NewBinary(hod)}
}

func PythonTestHostFactory() android.Module {
	return NewTest(android.HostSupportedNoCross).init()
}

func PythonTestFactory() android.Module {
	module := NewTest(android.HostAndDeviceSupported)
	module.multilib = android.MultilibBoth
	return module.init()
}

type TestProperties struct {
	// the name of the test configuration (for example "AndroidTest.xml") that should be
	// installed with the module.
	Test_config *string `android:"path,arch_variant"`

	// the name of the test configuration template (for example "AndroidTestTemplate.xml") that
	// should be installed with the module.
	Test_config_template *string `android:"path,arch_variant"`

	// list of files or filegroup modules that provide data that should be installed alongside
	// the test
	Data []string `android:"path,arch_variant"`

	// list of java modules that provide data that should be installed alongside the test.
	Java_data []string

	// Test options.
	Test_options android.CommonTestOptions
}

type PythonTestModule struct {
	PythonBinaryModule

	testProperties TestProperties
	testConfig     android.Path
	data           []android.DataPath
}

func (p *PythonTestModule) init() android.Module {
	p.AddProperties(&p.properties, &p.protoProperties)
	p.AddProperties(&p.binaryProperties)
	p.AddProperties(&p.testProperties)
	android.InitAndroidArchModule(p, p.hod, p.multilib)
	android.InitDefaultableModule(p)
	android.InitBazelModule(p)
	if p.hod == android.HostSupportedNoCross && p.testProperties.Test_options.Unit_test == nil {
		p.testProperties.Test_options.Unit_test = proptools.BoolPtr(true)
	}
	return p
}

func (p *PythonTestModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// We inherit from only the library's GenerateAndroidBuildActions, and then
	// just use buildBinary() so that the binary is not installed into the location
	// it would be for regular binaries.
	p.PythonLibraryModule.GenerateAndroidBuildActions(ctx)
	p.buildBinary(ctx)

	p.testConfig = tradefed.AutoGenTestConfig(ctx, tradefed.AutoGenTestConfigOptions{
		TestConfigProp:         p.testProperties.Test_config,
		TestConfigTemplateProp: p.testProperties.Test_config_template,
		TestSuites:             p.binaryProperties.Test_suites,
		AutoGenConfig:          p.binaryProperties.Auto_gen_config,
		DeviceTemplate:         "${PythonBinaryHostTestConfigTemplate}",
		HostTemplate:           "${PythonBinaryHostTestConfigTemplate}",
	})

	p.installedDest = ctx.InstallFile(installDir(ctx, "nativetest", "nativetest64", ctx.ModuleName()), p.installSource.Base(), p.installSource)

	for _, dataSrcPath := range android.PathsForModuleSrc(ctx, p.testProperties.Data) {
		p.data = append(p.data, android.DataPath{SrcPath: dataSrcPath})
	}

	// Emulate the data property for java_data dependencies.
	for _, javaData := range ctx.GetDirectDepsWithTag(javaDataTag) {
		for _, javaDataSrcPath := range android.OutputFilesForModule(ctx, javaData, "") {
			p.data = append(p.data, android.DataPath{SrcPath: javaDataSrcPath})
		}
	}
}

//=======
//type TestOptions struct {
//	android.CommonTestOptions
//
//	Multidevice_component *string
//}
//
//func (test *testDecorator) bootstrapperProps() []interface{} {
//	return append(test.binaryDecorator.bootstrapperProps(), &test.testProperties)
//}
//
//func (test *testDecorator) install(ctx android.ModuleContext, file android.Path) {
//	if test.testProperties.Test_options.Multidevice_component == nil {
//		test.testConfig = tradefed.AutoGenPythonBinaryHostTestConfig(ctx, test.testProperties.Test_config,
//			test.testProperties.Test_config_template, test.binaryDecorator.binaryProperties.Test_suites,
//			test.binaryDecorator.binaryProperties.Auto_gen_config)
//	} else {
//		test.testConfig = genMultideviceTestTemplate(ctx, *test.testProperties.Test_options.Multidevice_component)
//	}
//	>>>>>>> 16067b60c (multi-device soong)

func (p *PythonTestModule) AndroidMkEntries() []android.AndroidMkEntries {
	entriesList := p.PythonBinaryModule.AndroidMkEntries()
	if len(entriesList) != 1 {
		panic("Expected 1 entry")
	}
	entries := &entriesList[0]

	entries.Class = "NATIVE_TESTS"

	entries.ExtraEntries = append(entries.ExtraEntries,
		func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
			//entries.AddCompatibilityTestSuites(p.binaryProperties.Test_suites...)
			if p.testConfig != nil {
				entries.SetString("LOCAL_FULL_TEST_CONFIG", p.testConfig.String())
			}

			entries.SetBoolIfTrue("LOCAL_DISABLE_AUTO_GENERATE_TEST_CONFIG", !BoolDefault(p.binaryProperties.Auto_gen_config, true))

			entries.AddStrings("LOCAL_TEST_DATA", android.AndroidMkDataPaths(p.data)...)

			p.testProperties.Test_options.SetAndroidMkEntries(entries)
		})

	return entriesList
}

//func PythonMultideviceTestHostFactory() android.Module {
//	module := NewTest(android.HostSupportedNoCross)
//
//	return module.init()
//}
//
//var genMultideviceTestConfig = pctx.StaticRule("autogenTestConfig", blueprint.RuleParams{
//	Command:     "sed 's&{MODULE}&${name}&g;s&{COMPONENT}&'${component}'&g' $template > $out",
//	CommandDeps: []string{"$template"},
//}, "name", "template", "component")
//
//func genMultideviceTestTemplate(ctx android.ModuleContext, component string) android.Path {
//	outputFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".config")
//
//	ctx.Build(pctx, android.BuildParams{
//		Rule:        genMultideviceTestConfig,
//		Description: "multi-device test config",
//		Output:      outputFile,
//		Args: map[string]string{
//			"name":      ctx.ModuleName(),
//			"template":  "build/make/core/python_multidevice_test_config_template.xml",
//			"component": component,
//		},
//	})
//
//	return outputFile
//}
