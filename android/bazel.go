package android

import "android/soong/bazel"

type BazelModuleBase struct {
	bazelProperties bazel.Properties
}

type Bazelable interface {
	bazelProps() *bazel.Properties
	GetBazelLabel() string
	IsBp2buildAvailable() bool
}

type BazelModule interface {
	Module
	Bazelable
}

func InitBazelModule(module BazelModule) {
	module.AddProperties(module.bazelProps())
}

func (b *BazelModuleBase) bazelProps() *bazel.Properties {
	return &b.bazelProperties
}

func (b *BazelModuleBase) GetBazelLabel() string {
	return b.bazelProperties.Bazel_module.Label
}

func (b *BazelModuleBase) IsBp2buildAvailable() bool {
	return b.bazelProperties.Bazel_module.Bp2build_available
}
