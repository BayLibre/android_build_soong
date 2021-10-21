package mk2rbc

import (
	"fmt"
	"os"
	"path/filepath"
)

type KnownVariable struct {
	name      string
	class     varClass
	valueType starlarkType
}

type KnownVariables map[string]KnownVariable

func (pcv KnownVariables) NewVariable(name string, varClass varClass, valueType starlarkType) {
	v, exists := pcv[name]
	if !exists {
		pcv[name] = KnownVariable{name, varClass, valueType}
		return
	}
	// Conflict resolution:
	//    * config class trumps everything
	//    * any type trumps unknown type
	match := varClass == v.class
	if !match {
		if varClass == VarClassConfig {
			v.class = VarClassConfig
			match = true
		} else if v.class == VarClassConfig {
			match = true
		}
	}
	if valueType != v.valueType {
		if valueType != starlarkTypeUnknown {
			if v.valueType == starlarkTypeUnknown {
				v.valueType = valueType
			} else {
				match = false
			}
		}
	}
	if !match {
		fmt.Fprintf(os.Stderr, "cannot redefine %s as %v/%v (already defined as %v/%v)\n",
			name, varClass, valueType, v.class, v.valueType)
	}
}

// Implements mkparser.Scope, to be used by mkparser.Value.Value()
type fileNameScope struct {
	ScopeBase
	rootDir string
}

func (s fileNameScope) Get(name string) string {
	if name != "BUILD_SYSTEM" {
		return fmt.Sprintf("$(%s)", name)
	}
	return filepath.Join(s.rootDir, "build", "make", "core")
}

func CreateKnownVariables(rootDir string) (KnownVariables, error) {
	result := make(KnownVariables)
	for _, kv := range []string{
		// Kernel-related variables that we know are lists.
		"BOARD_VENDOR_KERNEL_MODULES",
		"BOARD_VENDOR_RAMDISK_KERNEL_MODULES",
		"BOARD_VENDOR_RAMDISK_KERNEL_MODULES_LOAD",
		"BOARD_RECOVERY_KERNEL_MODULES",
		// Other variables we know are lists
		"ART_APEX_JARS",
	} {
		result.NewVariable(kv, VarClassSoong, starlarkTypeList)
	}

	path := filepath.Join(rootDir, "build", "make", "core", "product.mk")
	if err := FindConfigVariables(path, result); err != nil {
		return nil, fmt.Errorf("%s\n(check --root[=%s], it should point to the source root)",
			err, rootDir)
	}

	path = filepath.Join(rootDir, "build", "make", "core", "soong_config.mk")
	err := FindSoongVariables(path, fileNameScope{rootDir: rootDir}, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
