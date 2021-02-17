package android

import (
	"android/soong/bazel"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type BazelModuleBase struct {
	bazelProperties bazel.Properties
}

type Bazelable interface {
	bazelProps() *bazel.Properties
	GetBazelLabel() string
	IsBp2buildAvailable() bool
	GetBazelBuildFileContents(c Config, path, name string) (string, error)
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

func (b *BazelModuleBase) GetBazelBuildFileContents(c Config, path, name string) (string, error) {
	if !strings.Contains(b.GetBazelLabel(), path) {
		return "", fmt.Errorf("%q not found in bazel_module.label %q", path, b.GetBazelLabel())
	}
	name = filepath.Join(path, name)
	f, err := c.fs.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data[:]), nil
}

func (b *BazelModuleBase) IsBp2buildAvailable() bool {
	return b.bazelProperties.Bazel_module.Bp2build_available && b.GetBazelLabel() == ""
}
