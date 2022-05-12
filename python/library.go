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

// This file contains the module types for building Python library.

import (
	"fmt"
	"os"

	"android/soong/android"
	"android/soong/bazel"

	"github.com/google/blueprint/proptools"
)

func init() {
	registerPythonLibraryComponents(android.InitRegistrationContext)
}

func registerPythonLibraryComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("python_library_host", PythonLibraryHostFactory)
	ctx.RegisterModuleType("python_library", PythonLibraryFactory)
}

func PythonLibraryHostFactory() android.Module {
	module := newModule(android.HostSupported, android.MultilibFirst)

	android.InitBazelModule(module)

	return module.init()
}

type bazelPythonLibraryAttributes struct {
	Srcs         bazel.LabelListAttribute
	Deps         bazel.LabelListAttribute
	Srcs_version *string
}

type bazelPythonProtoLibraryAttributes struct {
	Deps               bazel.LabelListAttribute
	Local_include_dirs []string
}

func pythonLibBp2Build(ctx android.TopDownMutatorContext, m *Module) {
	// TODO(b/182306917): this doesn't fully handle all nested props versioned
	// by the python version, which would have been handled by the version split
	// mutator. This is sufficient for very simple python_library modules under
	// Bionic.
	py3Enabled := proptools.BoolDefault(m.properties.Version.Py3.Enabled, true)
	py2Enabled := proptools.BoolDefault(m.properties.Version.Py2.Enabled, false)
	var python_version *string
	if py2Enabled && !py3Enabled {
		python_version = &pyVersion2
	} else if !py2Enabled && py3Enabled {
		python_version = &pyVersion3
	} else if !py2Enabled && !py3Enabled {
		ctx.ModuleErrorf("bp2build converter doesn't understand having neither py2 nor py3 enabled")
	} else {
		// do nothing, since python_version defaults to PY2ANDPY3
	}

	baseAttrs := m.makeArchVariantBaseAttributes(ctx)

	if m.Name() == "libprotobuf-python" {
		fmt.Fprintf(os.Stderr, "Srcs: %#v\n", baseAttrs.Srcs)
	}

	partitionedSrcs := bazel.PartitionLabelListAttribute(ctx, &baseAttrs.Srcs, bazel.LabelPartitions{
		"proto": android.ProtoSrcLabelPartition,
		"py":    bazel.LabelPartition{Keep_remainder: true},
	})
	baseAttrs.Srcs = partitionedSrcs["py"]

	if m.Name() == "libprotobuf-python" {
		fmt.Fprintf(os.Stderr, "partitionedSrcs[proto]: %#v\n", partitionedSrcs["proto"])
		fmt.Fprintf(os.Stderr, "partitionedSrcs[py]: %#v\n", partitionedSrcs["py"])
	}

	if !partitionedSrcs["proto"].IsEmpty() {
		protoInfo, _ := android.Bp2buildProtoProperties(ctx, &m.ModuleBase, partitionedSrcs["proto"])
		protoLabel := bazel.Label{Label: ":" + protoInfo.Name}

		pyProtoLibraryName := m.Name() + "_py_proto"
		ctx.CreateBazelTargetModule(bazel.BazelTargetModuleProperties{
			Rule_class:        "py_proto_library",
			Bzl_load_location: "//build/bazel/rules/python:py_proto.bzl",
		}, android.CommonAttributes{
			Name: pyProtoLibraryName,
		}, &bazelPythonProtoLibraryAttributes{
			Deps:               bazel.MakeSingleLabelListAttribute(protoLabel),
			Local_include_dirs: protoInfo.Local_include_dirs,
		})

		baseAttrs.Deps.Add(bazel.MakeLabelAttribute(":" + pyProtoLibraryName))
	}

	attrs := &bazelPythonLibraryAttributes{
		Srcs:         baseAttrs.Srcs,
		Deps:         baseAttrs.Deps,
		Srcs_version: python_version,
	}

	props := bazel.BazelTargetModuleProperties{
		Rule_class:        "py_library",
		Bzl_load_location: "//build/bazel/rules/python:library.bzl",
	}

	if m.Name() == "fg_foo" {
		fmt.Fprintf(os.Stderr, "Creating bazel target for fg_foo: %#v\n", baseAttrs.Data)
	}
	ctx.CreateBazelTargetModule(props, android.CommonAttributes{
		Name: m.Name(),
		Data: baseAttrs.Data,
	}, attrs)
}

func PythonLibraryFactory() android.Module {
	module := newModule(android.HostAndDeviceSupported, android.MultilibBoth)

	android.InitBazelModule(module)

	return module.init()
}
