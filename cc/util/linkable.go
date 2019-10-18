package util

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

type CcLinkableInterface interface {
	Module() android.Module
	IsCcLinkable() bool

	InRecovery() bool
	OutputFile() android.OptionalPath

	IncludeDirs() []string
	SetDepsInLinkOrder([]android.Path)
	GetDepsInLinkOrder() []android.Path

	HasStaticVariant() bool
	GetStaticVariant() CcLinkableInterface

	StubsVersions() []string
	SetBuildStubs()
	SetStubsVersions(string)

	BuildStaticVariant() bool
	BuildSharedVariant() bool
	SetStatic()
	SetShared()
}

type CcDependencyTag struct {
	blueprint.BaseDependencyTag
	Name    string
	Library bool
	Shared  bool

	ReexportFlags bool

	ExplicitlyVersioned bool
}

var (
	SharedDepTag = CcDependencyTag{Name: "shared", Library: true, Shared: true}
	StaticDepTag = CcDependencyTag{Name: "static", Library: true}

	CrtBeginDepTag = CcDependencyTag{Name: "crtbegin"}
	CrtEndDepTag   = CcDependencyTag{Name: "crtend"}
)
