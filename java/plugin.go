// Copyright 2019 Google Inc. All rights reserved.
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

package java

import (
	"android/soong/android"
	"android/soong/bazel"
)

func init() {
	registerJavaPluginBuildComponents(android.InitRegistrationContext)
}

func registerJavaPluginBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("java_plugin", PluginFactory)
}

// A java_plugin module describes a host java library that will be used by javac as an annotation processor.
func PluginFactory() android.Module {
	module := &Plugin{}

	module.addHostProperties()
	module.AddProperties(&module.pluginProperties)

	InitJavaModule(module, android.HostSupported)

	android.InitBazelModule(module)

	return module
}

type Plugin struct {
	Library

	pluginProperties PluginProperties
}

type PluginProperties struct {
	// The optional name of the class that javac will use to run the annotation processor.
	Processor_class *string

	// If true, assume the annotation processor will generate classes that are referenced from outside the module.
	// This necessitates disabling the turbine optimization on modules that use this plugin, which will reduce
	// parallelism and cause more recompilation for modules that depend on modules that use this plugin.
	Generates_api *bool
}

type pluginAttributes struct {
	Srcs      bazel.LabelListAttribute
	Deps      bazel.LabelListAttribute
	Javacopts bazel.StringListAttribute
}

// ConvertWithBp2build is used to convert android_app to Bazel.
func (p *Plugin) ConvertWithBp2build(ctx android.TopDownMutatorContext) {
	srcs := bazel.MakeLabelListAttribute(android.BazelLabelForModuleSrcExcludes(ctx, p.properties.Srcs, p.properties.Exclude_srcs))
	attrs := &javaLibraryAttributes{
		Srcs: srcs,
	}

	if p.properties.Javacflags != nil {
		attrs.Javacopts = bazel.MakeStringListAttribute(p.properties.Javacflags)
	}

	if p.properties.Libs != nil {
		attrs.Deps = bazel.MakeLabelListAttribute(android.BazelLabelForModuleDeps(ctx, p.properties.Libs))
	}

	props := bazel.BazelTargetModuleProperties{
		Rule_class: "java_plugin",
	}

	ctx.CreateBazelTargetModule(props, android.CommonAttributes{Name: p.Name()}, attrs)
}
