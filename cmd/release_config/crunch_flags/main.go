package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"android/soong/cmd/release_config/release_config_proto"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type StringList []string

func (l *StringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func (l *StringList) String() string {
	return fmt.Sprintf("%v", *l)
}

var releaseConfigDirs StringList

func RenameNext(name string) string {
	if name == "next" {
		return "ap3a"
	}
	return name
}

func WriteFile(path string, message proto.Message) error {
	data, err := prototext.MarshalOptions{Multiline: true}.Marshal(message)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0775)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func WalkValueFiles(dir string, Func fs.WalkDirFunc) error {
	valPath := filepath.Join(dir, "build_config")
	if _, err := os.Stat(valPath); err != nil {
		fmt.Printf("%s not found, ignoring.\n", valPath)
		return nil
	}

	return filepath.WalkDir(valPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(d.Name(), ".scl") && d.Type().IsRegular() {
			return Func(path, d, err)
		}
		return nil
	})
}

func ProcessBuildFlags(dir string) error {
	var rootAconfigModule string

	path := filepath.Join(dir, "build_flags.scl")
	if _, err := os.Stat(path); err != nil {
		fmt.Printf("%s not found, ignoring.\n", path)
		return nil
	} else {
		fmt.Printf("Processing %s\n", path)
	}
	commentRegexp, err := regexp.Compile("^[[:space:]]*#(?<comment>.+)")
	if err != nil {
		return err
	}
	declRegexp, err := regexp.Compile("^[[:space:]]*flag.\"(?<name>[A-Z_0-9]+)\",[[:space:]]*(?<container>[_A-Z]*),[[:space:]]*(?<value>(\"[^\"]*\"|[^\",)]*))")
	if err != nil {
		return err
	}
	declIn, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(declIn), "\n")
	var description string
	for _, line := range lines {
		if comment := commentRegexp.FindStringSubmatch(commentRegexp.FindString(line)); comment != nil {
			// Description is the text from any contiguous series of lines before a `flag()` call.
			description += fmt.Sprintf(" %s", strings.Trim(comment[commentRegexp.SubexpIndex("comment")], " "))
			continue
		}
		matches := declRegexp.FindStringSubmatch(declRegexp.FindString(line))
		if matches == nil {
			// The line is neither a comment nor a `flag()` call.
			// Discard any description we have gathered and process the next line.
			description = ""
			continue
		}
		declValue := matches[declRegexp.SubexpIndex("value")]
		declName := matches[declRegexp.SubexpIndex("name")]
		container := release_config_proto.Container(release_config_proto.Container_value[matches[declRegexp.SubexpIndex("container")]])
		description = strings.Trim(description, " ")
		flagDeclaration := &release_config_proto.FlagDeclaration{
			Name:        proto.String(declName),
			Namespace:   proto.String("android_UNKNONWN"),
			Description: proto.String(description),
			Container:   &container,
		}
		description = ""
		switch {
		case declName == "RELEASE_ACONFIG_VALUE_SETS":
			flagDeclaration = nil
			rootAconfigModule = declValue[1 : len(declValue)-1]
		case strings.HasPrefix(declValue, "\""):
			declValue = declValue[1 : len(declValue)-1]
			flagDeclaration.Value = &release_config_proto.Value{Val: &release_config_proto.Value_StringValue{declValue}}
			var workflow release_config_proto.Workflow
			switch {
			case strings.HasPrefix(declName, "RELEASE_PLATFORM_") || strings.HasPrefix(declName, "RELEASE_ACONFIG_"):
				workflow = release_config_proto.Workflow(release_config_proto.Workflow_MANUAL)
			default:
				workflow = release_config_proto.Workflow(release_config_proto.Workflow_PREBUILT)
			}
			flagDeclaration.Workflow = &workflow
		case declValue == "True":
			flagDeclaration.Value = &release_config_proto.Value{Val: &release_config_proto.Value_BoolValue{true}}
			workflow := release_config_proto.Workflow(release_config_proto.Workflow_LAUNCH)
			flagDeclaration.Workflow = &workflow
		case declValue == "False" || declValue == "None":
			flagDeclaration.Value = &release_config_proto.Value{Val: &release_config_proto.Value_BoolValue{false}}
			workflow := release_config_proto.Workflow(release_config_proto.Workflow_LAUNCH)
			flagDeclaration.Workflow = &workflow
		default:
			fmt.Printf("%s: Unexpected value %s=%s\n", path, declName, declValue)
		}
		if flagDeclaration != nil {
			declPath := filepath.Join(dir, "flag_declarations", fmt.Sprintf("%s.textproto", declName))
			err := WriteFile(declPath, flagDeclaration)
			if err != nil {
				return err
			}
		}
	}
	if rootAconfigModule != "" {
		rootProto := &release_config_proto.ReleaseConfig{
			Name:             proto.String("root"),
			AconfigValueSets: []string{rootAconfigModule},
		}
		return WriteFile(filepath.Join(dir, "release_configs", "root.textproto"), rootProto)
	}
	return nil
}

func ProcessBuildConfigs(dir, name string, paths []string, releaseProto *release_config_proto.ReleaseConfig) error {
	valRegexp, err := regexp.Compile("[[:space:]]+value.\"(?<name>[A-Z_0-9]+)\",[[:space:]]*(?<value>[^,)]*)")
	if err != nil {
		return err
	}
	for _, path := range paths {
		fmt.Printf("Processing %s\n", path)
		valIn, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("%s: error: %v\n", err)
			return err
		}
		vals := valRegexp.FindAllString(string(valIn), -1)
		for _, val := range vals {
			matches := valRegexp.FindStringSubmatch(val)
			valValue := matches[valRegexp.SubexpIndex("value")]
			valName := matches[valRegexp.SubexpIndex("name")]
			flagValue := &release_config_proto.FlagValue{
				Name: proto.String(valName),
			}
			switch {
			case valName == "RELEASE_ACONFIG_VALUE_SETS":
				flagValue = nil
				if releaseProto.AconfigValueSets == nil {
					releaseProto.AconfigValueSets = []string{}
				}
				releaseProto.AconfigValueSets = append(releaseProto.AconfigValueSets, valValue[1:len(valValue)-1])
			case strings.HasPrefix(valValue, "\""):
				valValue = valValue[1 : len(valValue)-1]
				flagValue.Value = &release_config_proto.Value{Val: &release_config_proto.Value_StringValue{valValue}}
			case valValue == "None":
				// nothing to do here.
			case valValue == "True":
				flagValue.Value = &release_config_proto.Value{Val: &release_config_proto.Value_BoolValue{true}}
			case valValue == "False":
				flagValue.Value = &release_config_proto.Value{Val: &release_config_proto.Value_BoolValue{false}}
			default:
				fmt.Println("%s: Unexpected value %s=%s\n", path, valName, valValue)
			}
			if flagValue != nil {
				valPath := filepath.Join(dir, "flag_values", RenameNext(name), fmt.Sprintf("%s.textproto", valName))
				err := WriteFile(valPath, flagValue)
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

func ProcessReleaseConfigMap(dir string) error {
	path := filepath.Join(dir, "release_config_map.mk")
	if _, err := os.Stat(path); err != nil {
		fmt.Printf("%s not found, ignoring.\n", path)
		return nil
	} else {
		fmt.Printf("Processing %s\n", path)
	}
	configRegexp, err := regexp.Compile("^..call[[:space:]]+declare-release-config,[[:space:]]+(?<name>[_a-z0-0A-Z]+),[[:space:]]+(?<files>[^,]*)(,[[:space:]]*(?<inherits>.*)|[[:space:]]*)[)]$")
	if err != nil {
		return err
	}
	aliasRegexp, err := regexp.Compile("^..call[[:space:]]+alias-release-config,[[:space:]]+(?<name>[_a-z0-9A-Z]+),[[:space:]]+(?<target>[_a-z0-9A-Z]+)")
	if err != nil {
		return err
	}

	mapIn, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cleanDir := strings.TrimLeft(dir, "../")
	var defaultContainer release_config_proto.Container
	switch {
	case strings.HasPrefix(cleanDir, "build/") || cleanDir == "vendor/google_shared/build":
		defaultContainer = release_config_proto.Container(release_config_proto.Container_ALL)
	case cleanDir == "vendor/google/release":
		defaultContainer = release_config_proto.Container(release_config_proto.Container_ALL)
	default:
		defaultContainer = release_config_proto.Container(release_config_proto.Container_VENDOR)
	}
	releaseConfigMap := &release_config_proto.ReleaseConfigMap{DefaultContainer: &defaultContainer}
	lines := strings.Split(string(mapIn), "\n")
	for _, line := range lines {
		alias := aliasRegexp.FindStringSubmatch(aliasRegexp.FindString(line))
		if alias != nil {
			fmt.Printf("processing alias %s\n", line)
			name := alias[aliasRegexp.SubexpIndex("name")]
			target := alias[aliasRegexp.SubexpIndex("target")]
			if target == "next" {
				if RenameNext(target) != name {
					return fmt.Errorf("Unexpected name for next (%s)", RenameNext(target))
				}
				target, name = name, target
			}
			releaseConfigMap.Aliases = append(releaseConfigMap.Aliases,
				&release_config_proto.ReleaseAlias{
					Name:   proto.String(name),
					Target: proto.String(target),
				})
		}
		config := configRegexp.FindStringSubmatch(configRegexp.FindString(line))
		if config == nil {
			continue
		}
		name := config[configRegexp.SubexpIndex("name")]
		releaseConfig := &release_config_proto.ReleaseConfig{
			Name: proto.String(RenameNext(name)),
		}
		configFiles := config[configRegexp.SubexpIndex("files")]
		files := strings.Split(strings.ReplaceAll(configFiles, "$(local_dir)", dir+"/"), " ")
		configInherits := config[configRegexp.SubexpIndex("inherits")]
		if len(configInherits) > 0 {
			releaseConfig.Inherits = strings.Split(configInherits, " ")
		}
		err := ProcessBuildConfigs(dir, name, files, releaseConfig)
		if err != nil {
			return err
		}

		releasePath := filepath.Join(dir, "release_configs", fmt.Sprintf("%s.textproto", RenameNext(name)))
		err = WriteFile(releasePath, releaseConfig)
		if err != nil {
			return err
		}
	}
	return WriteFile(filepath.Join(dir, "release_config_map.textproto"), releaseConfigMap)
}

func main() {
	var err error
	var dirs StringList
	flag.Var(&dirs, "dir", "directory to process")
	flag.Parse()

	if len(dirs) == 0 {
		dirs = StringList{"../release", "../../vendor/google_shared/build/release", "../../vendor/google/release"}
	}

	for _, dir := range dirs {
		err = ProcessBuildFlags(dir)
		if err != nil {
			panic(err)
		}

		err = ProcessReleaseConfigMap(dir)
		if err != nil {
			panic(err)
		}
	}
}
