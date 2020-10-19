// Copyright 2020 Google Inc. All rights reserved.
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
)

type partitionGroup int

const (
	partitionGroupNone partitionGroup = iota
	partitionGroupSystem
	partitionGroupVendor
	partitionGroupProduct
)

func (g partitionGroup) String() string {
	switch g {
	case partitionGroupSystem:
		return "system"
	case partitionGroupVendor:
		return "vendor"
	case partitionGroupProduct:
		return "product"
	}

	return ""
}

func (j *Module) partitionGroup(ctx android.EarlyModuleContext) partitionGroup {
	if j.Platform() || j.SystemExtSpecific() {
		return partitionGroupSystem
	}

	if j.SocSpecific() || j.DeviceSpecific() {
		return partitionGroupVendor
	}

	if j.ProductSpecific() {
		return partitionGroupProduct
	}

	panic("Cannot determine partition type")
}

func (j *Module) allowListedInterPartitionJavaLibrary(ctx android.EarlyModuleContext) bool {
	return inList(j.Name(), ctx.Config().InterPartitionJavaLibraryAllowList())
}

type javaSdkLibraryEnforceContext interface {
	allowListedInterPartitionJavaLibrary(ctx android.EarlyModuleContext) bool
	partitionGroup(ctx android.EarlyModuleContext) partitionGroup
}

var _ javaSdkLibraryEnforceContext = (*Module)(nil)

func (j *Module) checkPartitionsForJavaDependency(ctx android.EarlyModuleContext, propName string, dep *Module) {
	if dep.allowListedInterPartitionJavaLibrary(ctx) {
		return
	}

	// If product interface is not enforced, skip check between system and product partition
	if !ctx.Config().EnforceProductPartitionInterface() {
		pToS := j.partitionGroup(ctx) == partitionGroupProduct && dep.partitionGroup(ctx) == partitionGroupSystem
		sToP := j.partitionGroup(ctx) == partitionGroupSystem && dep.partitionGroup(ctx) == partitionGroupProduct

		if pToS || sToP {
			return
		}
	}

	// If module and dependency library is inter-partition
	if j.partitionGroup(ctx) != dep.partitionGroup(ctx) {
		errorFormat := "dependency on java_library (%q) is not allowed across the partitions (%s -> %s), use java_sdk_library instead"
		ctx.PropertyErrorf(propName, errorFormat, dep.Name(), j.partitionGroup(ctx), dep.partitionGroup(ctx))
	}
}
