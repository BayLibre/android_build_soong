package java

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

type partitionGroup int

const (
	partitionGroupNone partitionGroup = iota
	partitionGroupSystem
	partitionGroupVendor
	partitionGroupProduct
)

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

	ctx.ModuleErrorf("Cannot determine partition type")

	return partitionGroupNone
}

var (
	javaSdkLibraryAllowlistKey = android.NewOnceKey("javaSdkLibraryAllowlist")
)

func javaSdkAllowListedItem(ctx android.EarlyModuleContext, item string) bool {
	allowList := ctx.Config().Once(javaSdkLibraryAllowlistKey, func() interface{} {
		result := make(map[string]bool)

		for _, v := range ctx.Config().InterPartitionJavaLibraryAllowList() {
			result[v] = true
		}

		return result
	}).(map[string]bool)

	_, allowListed := allowList[item]

	return allowListed
}

func (j *Module) javaSdkAllowListedSource(ctx android.EarlyModuleContext) bool {
	return javaSdkAllowListedItem(ctx, j.Name()) || javaSdkAllowListedItem(ctx, j.Name()+":*")
}

func (j *Module) javaSdkAllowListedDependency(ctx android.EarlyModuleContext) bool {
	return javaSdkAllowListedItem(ctx, j.Name()) || javaSdkAllowListedItem(ctx, "*:"+j.Name())
}

type javaSdkExternalDependency interface {
	blueprint.Module
	javaSdkAllowListedDependency(ctx android.EarlyModuleContext) bool
	partitionGroup(ctx android.EarlyModuleContext) partitionGroup
}

func (j *Module) checkJavaSdkLibraryEnforce(ctx android.EarlyModuleContext, dep javaSdkExternalDependency) {
	if !ctx.Config().EnforceJavaSdkLibraryInterPartition() {
		return
	}

	vendorInterfaceEnforced := ctx.DeviceConfig().VndkVersion() != ""
	productInterfaceEnforced := ctx.Config().EnforceProductPartitionInterface()

	if j.javaSdkAllowListedSource(ctx) {
		return
	}

	if dep.javaSdkAllowListedDependency(ctx) {
		return
	}

	// If vendor interface is not enforced, skip check vendor partition at
	// inter-partition library dependency
	if !vendorInterfaceEnforced {
		if j.partitionGroup(ctx) == partitionGroupVendor || dep.partitionGroup(ctx) == partitionGroupVendor {
			return
		}
	}

	// If product interface is not enforced, skip check product partition at
	// inter-partition library dependency
	if !productInterfaceEnforced {
		if j.partitionGroup(ctx) == partitionGroupProduct || dep.partitionGroup(ctx) == partitionGroupProduct {
			return
		}
	}

	// If module and dependency library is inter-partition
	if j.partitionGroup(ctx) != dep.partitionGroup(ctx) {
		ctx.ModuleErrorf(
			"dependency %q, using of java_sdk_library is enforced at inter-partition(%s -> %s) dependencies",
			dep.Name(),
			j.partitionGroup(ctx), dep.partitionGroup(ctx))
	}
}
