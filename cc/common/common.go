package common

import (
	"blueprint"
)

type Config interface {
	SrcDir() string
	PrebuiltOS() string
}

var (
	pctx = blueprint.NewPackageContext("android/soong/cc/common")

	HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", Config.PrebuiltOS)
	SrcDir          = pctx.VariableConfigMethod("SrcDir", Config.SrcDir)

	LibcRoot = pctx.StaticVariable("LibcRoot", "${SrcDir}/bionic/libc")
	LibmRoot = pctx.StaticVariable("LibmRoot", "${SrcDir}/bionic/libm")
)
