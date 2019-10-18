package ccutil

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

type LinkableInterface interface {
	Module() android.Module
	IsCcLinkable() bool

	InRecovery() bool
	OutputFile() android.OptionalPath

	IncludeDirs() []string
	SetDepsInLinkOrder([]android.Path)
	GetDepsInLinkOrder() []android.Path

	HasStaticVariant() bool
	GetStaticVariant() LinkableInterface

	StubsVersions() []string
	SetBuildStubs()
	SetStubsVersions(string)

	BuildStaticVariant() bool
	BuildSharedVariant() bool
	SetStatic()
	SetShared()
}

type DependencyTag struct {
	blueprint.BaseDependencyTag
	Name    string
	Library bool
	Shared  bool

	ReexportFlags bool

	ExplicitlyVersioned bool
}

var (
	SharedDepTag = DependencyTag{Name: "shared", Library: true, Shared: true}
	StaticDepTag = DependencyTag{Name: "static", Library: true}

	CrtBeginDepTag = DependencyTag{Name: "crtbegin"}
	CrtEndDepTag   = DependencyTag{Name: "crtend"}
)
