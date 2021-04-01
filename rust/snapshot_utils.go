package rust

import (
	"android/soong/android"
)

func (mod *Module) ExcludeFromVendorSnapshot() bool {
	// TODO Rust does not yet support snapshotting
	return true
}

func (mod *Module) ExcludeFromRecoverySnapshot() bool {
	// TODO Rust does not yet support snapshotting
	return true
}

func (mod *Module) SnapshotLibrary() bool {
	// TODO Rust does not yet support snapshotting
	return false
}

func (mod *Module) SnapshotRuntimeLibs() []string {
	// TODO Rust does not yet support a runtime libs notion similar to CC
	return []string{}
}

func (mod *Module) SnapshotSharedLibs() []string {
	// TODO Rust does not yet support snapshotting
	return []string{}
}

func (mod *Module) Symlinks() []string {
	// TODO update this to return the list of symlinks when Rust supports defining symlinks
	return nil
}

func (m *Module) SnapshotHeaders() android.Paths {
	// TODO Rust does not yet support snapshotting
	return android.Paths{}
}
