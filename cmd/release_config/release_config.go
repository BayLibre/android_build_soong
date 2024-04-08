// Copyright 2024 Google Inc. All rights reserved.
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"android/soong/cmd/release_config/release_config_proto"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var verboseFlag bool

type StringList []string

func (l *StringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func (l *StringList) String() string {
	return fmt.Sprintf("%v", *l)
}

var releaseConfigMapPaths StringList

func DumpProtos(basePath string, message proto.Message) error {
	writer := func(suffix string, marshal func() ([]byte, error)) error {
		data, err := marshal()
		if err != nil {
			return err
		}
		return os.WriteFile(fmt.Sprintf("%s.%s", basePath, suffix), data, 0644)
	}
	err := writer("textproto", func() ([]byte, error) { return prototext.MarshalOptions{Multiline: true}.Marshal(message) })
	if err != nil {
		return err
	}

	err = writer("pb", func() ([]byte, error) { return proto.Marshal(message) })
	if err != nil {
		return err
	}

	return writer("json", func() ([]byte, error) { return json.MarshalIndent(message, "", "  ") })
}

func LoadTextproto(path string, message proto.Message) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ret := prototext.Unmarshal(data, message)
	if verboseFlag {
		debug, _ := prototext.Marshal(message)
		fmt.Printf("%s: %s\n", path, debug)
	}
	return ret
}

func WalkTextprotoFiles(root string, subdir string, Func fs.WalkDirFunc) error {
	return filepath.WalkDir(filepath.Join(root, subdir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(d.Name(), ".textproto") && d.Type().IsRegular() {
			return Func(path, d, err)
		}
		return nil
	})
}

type FlagValue struct {
	// The path providing this value.
	path string

	// Protobuf
	proto release_config_proto.FlagValue
}

func FlagValueFactory(protoPath string) (fv *FlagValue) {
	fv = &FlagValue{path: protoPath}
	if protoPath != "" {
		LoadTextproto(protoPath, &fv.proto)
	}
	return fv
}

// One directory's contribution to the a release config.
type ReleaseConfigContribution struct {
	// Paths to files providing this config.
	path string

	// Protobufs relevant to the config.
	proto release_config_proto.ReleaseConfig

	FlagValues []*FlagValue
}

// A single release_config_map.textproto and its associated data.
// Used primarily for debugging.
type ReleaseConfigMap struct {
	// The path to this release_config_map file.
	path string

	// Data received
	proto release_config_proto.ReleaseConfigMap

	ReleaseConfigContributions map[string]*ReleaseConfigContribution
	FlagDeclarations           []release_config_proto.FlagDeclaration
}

// A generated release config.
type ReleaseConfig struct {
	// the Name of the release config
	Name string

	// What contributes to this config.
	Contributions []*ReleaseConfigContribution

	// Aliases for this release
	OtherNames []string

	// The names of release configs that we inherit
	InheritNames []string

	// Generated release config
	ReleaseConfigArtifact *release_config_proto.ReleaseConfigArtifact

	// We have begun compiling this release config.
	compileInProgress bool
}

type FlagArtifact struct {
	FlagDeclaration *release_config_proto.FlagDeclaration

	Traces []*release_config_proto.Tracepoint

	// Assigned value
	Value *release_config_proto.Value
}

// Key is flag name.
type FlagArtifacts map[string]*FlagArtifact

// The generated release configs.
type ReleaseConfigs struct {
	// Ordered list of release config maps processed.
	ReleaseConfigMaps []*ReleaseConfigMap

	// Aliases
	Aliases map[string]*string

	// Dictionary of flag_name:FlagDeclaration, with no overrides applied.
	FlagArtifacts FlagArtifacts

	// Dictionary of name:ReleaseConfig
	ReleaseConfigs map[string]*ReleaseConfig

	// Generated release configs
	Artifact release_config_proto.ReleaseConfigsArtifact
}

func (src *FlagArtifact) Clone() *FlagArtifact {
	value := &release_config_proto.Value{}
	proto.Merge(value, src.Value)
	return &FlagArtifact{
		FlagDeclaration: src.FlagDeclaration,
		Traces:          src.Traces,
		Value:           value,
	}
}

func (src FlagArtifacts) Clone() (dst FlagArtifacts) {
	if dst == nil {
		dst = make(FlagArtifacts)
	}
	for k, v := range src {
		dst[k] = v.Clone()
	}
	return
}

func ReleaseConfigFactory(name string) (c *ReleaseConfig) {
	return &ReleaseConfig{Name: name}
}

func ReleaseConfigsFactory() (c *ReleaseConfigs) {
	return &ReleaseConfigs{
		Aliases:        make(map[string]*string),
		FlagArtifacts:  make(map[string]*FlagArtifact),
		ReleaseConfigs: make(map[string]*ReleaseConfig),
	}
}

func ReleaseConfigMapFactory(protoPath string) (m *ReleaseConfigMap) {
	m = &ReleaseConfigMap{
		path:                       protoPath,
		ReleaseConfigContributions: make(map[string]*ReleaseConfigContribution),
	}
	if protoPath != "" {
		LoadTextproto(protoPath, &m.proto)
	}
	return m
}

func FlagDeclarationFactory(protoPath string) (fd *release_config_proto.FlagDeclaration) {
	fd = &release_config_proto.FlagDeclaration{}
	if protoPath != "" {
		LoadTextproto(protoPath, fd)
	}
	return fd
}

func (configs *ReleaseConfigs) LoadReleaseConfigMap(path string) error {
	m := ReleaseConfigMapFactory(path)
	dir := filepath.Dir(path)
	// Record any aliases, checking for duplicates.
	for _, alias := range m.proto.Aliases {
		name := *alias.Name
		oldTarget, ok := configs.Aliases[name]
		if ok {
			if *oldTarget != *alias.Target {
				return fmt.Errorf("Conflicting alias declarations: %s vs %s",
					oldTarget, alias.Target)
			}
		}
		configs.Aliases[name] = alias.Target
	}
	WalkTextprotoFiles(dir, "flag_declarations", func(path string, d fs.DirEntry, err error) error {
		flagDeclaration := FlagDeclarationFactory(path)
		m.FlagDeclarations = append(m.FlagDeclarations, *flagDeclaration)
		name := *flagDeclaration.Name
		if def, ok := configs.FlagArtifacts[name]; !ok {
			configs.FlagArtifacts[name] = &FlagArtifact{FlagDeclaration: flagDeclaration}
		} else if !proto.Equal(def.FlagDeclaration, flagDeclaration) {
			return fmt.Errorf("Duplicate definition of %s", *flagDeclaration.Name)
		}
		// Set the initial value for the flag, if any.
		if flagDeclaration.GetValue() != nil {
			configs.FlagArtifacts[name].UpdateValue(
				FlagValue{path: path, proto: release_config_proto.FlagValue{
					Name: proto.String(name), Value: flagDeclaration.Value}})
		}
		return nil
	})
	WalkTextprotoFiles(dir, "release_configs", func(path string, d fs.DirEntry, err error) error {
		releaseConfigContribution := &ReleaseConfigContribution{path: path}
		LoadTextproto(path, &releaseConfigContribution.proto)
		name := *releaseConfigContribution.proto.Name
		if _, ok := configs.ReleaseConfigs[name]; !ok {
			configs.ReleaseConfigs[name] = ReleaseConfigFactory(name)
		}
		config := configs.ReleaseConfigs[name]
		config.InheritNames = append(config.InheritNames, releaseConfigContribution.proto.Inherits...)
		WalkTextprotoFiles(dir, filepath.Join("flag_values", name), func(path string, d fs.DirEntry, err error) error {
			flagValue := FlagValueFactory(path)
			releaseConfigContribution.FlagValues = append(releaseConfigContribution.FlagValues, flagValue)
			return nil
		})
		m.ReleaseConfigContributions[name] = releaseConfigContribution
		config.Contributions = append(config.Contributions, releaseConfigContribution)
		return nil
	})
	configs.ReleaseConfigMaps = append(configs.ReleaseConfigMaps, m)
	return nil
}

func (configs *ReleaseConfigs) GetReleaseConfig(name string) (*ReleaseConfig, error) {
	trace := []string{name}
	for target, ok := configs.Aliases[name]; ok; target, ok = configs.Aliases[name] {
		name = *target
		trace = append(trace, name)
	}
	if config, ok := configs.ReleaseConfigs[name]; ok {
		return config, nil
	}
	return nil, fmt.Errorf("Missing config %s.  Trace=%s", name, trace)
}

func (configs *ReleaseConfigs) GenerateReleaseConfigs(targetRelease string) error {
	otherNames := make(map[string][]string)
	for aliasName, aliasTarget := range configs.Aliases {
		if _, ok := configs.ReleaseConfigs[aliasName]; ok {
			return fmt.Errorf("Alias %s is a declared release config", aliasName)
		}
		if _, ok := configs.ReleaseConfigs[*aliasTarget]; !ok {
			if _, ok2 := configs.Aliases[*aliasTarget]; !ok2 {
				return fmt.Errorf("Alias %s points to non-existing config %s", aliasName, *aliasTarget)
			}
		}
		otherNames[*aliasTarget] = append(otherNames[*aliasTarget], aliasName)
	}
	for name, aliases := range otherNames {
		configs.ReleaseConfigs[name].OtherNames = aliases
	}

	for _, config := range configs.ReleaseConfigs {
		err := config.GenerateReleaseConfig(configs)
		if err != nil {
			return err
		}
	}

	releaseConfig, err := configs.GetReleaseConfig(targetRelease)
	if err != nil {
		return err
	}
	configs.Artifact = release_config_proto.ReleaseConfigsArtifact{
		ReleaseConfig: releaseConfig.ReleaseConfigArtifact,
		OtherReleaseConfigs: func() []*release_config_proto.ReleaseConfigArtifact {
			orc := []*release_config_proto.ReleaseConfigArtifact{}
			for name, config := range configs.ReleaseConfigs {
				if name != releaseConfig.Name {
					orc = append(orc, config.ReleaseConfigArtifact)
				}
			}
			return orc
		}(),
	}
	return nil
}

func (fa *FlagArtifact) UpdateValue(flagValue FlagValue) error {
	name := *flagValue.proto.Name
	fa.Traces = append(fa.Traces, &release_config_proto.Tracepoint{Source: proto.String(flagValue.path), Value: flagValue.proto.Value})
	if fa.Value.GetObsolete() {
		return fmt.Errorf("Attempting to set obsolete flag %s. Trace=%s", name, fa.Traces)
	}
	switch val := flagValue.proto.Value.Val.(type) {
	case *release_config_proto.Value_StringValue:
		fa.Value = &release_config_proto.Value{Val: &release_config_proto.Value_StringValue{val.StringValue}}
	case *release_config_proto.Value_BoolValue:
		fa.Value = &release_config_proto.Value{Val: &release_config_proto.Value_BoolValue{val.BoolValue}}
	case *release_config_proto.Value_Obsolete:
		if !val.Obsolete {
			return fmt.Errorf("%s: Cannot set obsolete=false.  Trace=%s", name, fa.Traces)
		}
		fa.Value = &release_config_proto.Value{Val: &release_config_proto.Value_Obsolete{true}}
	default:
		return fmt.Errorf("Invalid type for flag_value: %T.  Trace=%s", val, fa.Traces)
	}
	return nil
}

func (fa *FlagArtifact) Marshal() (*release_config_proto.FlagArtifact, error) {
	return &release_config_proto.FlagArtifact{
		FlagDeclaration: fa.FlagDeclaration,
		Value:           fa.Value,
		Trace:           fa.Traces,
	}, nil
}

func (config *ReleaseConfig) GenerateReleaseConfig(configs *ReleaseConfigs) error {
	if config.ReleaseConfigArtifact != nil {
		return nil
	}
	if config.compileInProgress {
		return fmt.Errorf("Loop detected for release config %s", config.Name)
	}
	config.compileInProgress = true

	// Generate any configs we need to inherit.  This will detect loops in
	// the config.
	contributionsToApply := []*ReleaseConfigContribution{}
	myInherits := []string{}
	myInheritsSet := make(map[string]bool)
	for _, inherit := range config.InheritNames {
		if _, ok := myInheritsSet[inherit]; ok {
			continue
		}
		myInherits = append(myInherits, inherit)
		myInheritsSet[inherit] = true
		iConfig, err := configs.GetReleaseConfig(inherit)
		if err != nil {
			return err
		}
		iConfig.GenerateReleaseConfig(configs)
		contributionsToApply = append(contributionsToApply, iConfig.Contributions...)
	}
	contributionsToApply = append(contributionsToApply, config.Contributions...)

	myAconfigValueSets := []string{}
	myFlags := configs.FlagArtifacts.Clone()
	for _, contrib := range contributionsToApply {
		myAconfigValueSets = append(myAconfigValueSets, contrib.proto.AconfigValueSets...)
		for _, value := range contrib.FlagValues {
			fa, ok := myFlags[*value.proto.Name]
			if !ok {
				return fmt.Errorf("Setting value for undefined flag %s in %s\n", *value.proto.Name, value.path)
			}
			if err := fa.UpdateValue(*value); err != nil {
				return err
			}
		}
	}

	config.ReleaseConfigArtifact = &release_config_proto.ReleaseConfigArtifact{
		Name:       proto.String(config.Name),
		OtherNames: config.OtherNames,
		FlagArtifacts: func() []*release_config_proto.FlagArtifact {
			ret := []*release_config_proto.FlagArtifact{}
			for _, flag := range myFlags {
				ret = append(ret, &release_config_proto.FlagArtifact{
					FlagDeclaration: flag.FlagDeclaration,
					Trace:           flag.Traces,
					Value:           flag.Value,
				})
			}
			return ret
		}(),
		AconfigValueSets: myAconfigValueSets,
		Inherits:         myInherits,
	}

	config.compileInProgress = false
	return nil
}

func main() {
	var target_release string
	var output_path string
	flag.BoolVar(&verboseFlag, "debug", false, "print debugging information")
	flag.Var(&releaseConfigMapPaths, "map", "path to a release_config_map.textproto. may be repeated")
	flag.StringVar(&target_release, "release", "trunk_staging", "TARGET_RELEASE for this build")
	flag.StringVar(&output_path, "output_path", "all_release_configs", "path for the output.  .pb and .textproto are generated.")
	flag.Parse()

	configs := ReleaseConfigsFactory()
	for _, releaseConfigMapPath := range releaseConfigMapPaths {
		err := configs.LoadReleaseConfigMap(releaseConfigMapPath)
		if err != nil {
			panic(err)
		}
	}

	// Now that we have all of the release config maps, can meld them and generate the artifacts.

	err := configs.GenerateReleaseConfigs(target_release)
	if err != nil {
		panic(err)
	}

	DumpProtos(output_path, &configs.Artifact)
}
