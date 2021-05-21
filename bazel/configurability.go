// Copyright 2021 Google Inc. All rights reserved.
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

package bazel

import (
	"fmt"
	"strings"
)

type configurationType int

const (
	NoConfig configurationType = iota
	Arch
	Os
	ArchOs
	ProductVariables
)

func (ct configurationType) String() string {
	return map[configurationType]string{
		NoConfig:         "no_config",
		Arch:             "arch",
		Os:               "os",
		ArchOs:           "arch_os",
		ProductVariables: "product_variables",
	}[ct]
}

func (ct configurationType) validateConfig(config string) {
	switch ct {
	case NoConfig:
		if config != "" {
			panic(fmt.Errorf("Cannot specify config with %s, but got %s", ct, config))
		}
	case Arch:
		if _, ok := PlatformArchMap[config]; !ok {
			panic(fmt.Errorf("Unknown arch: %s", config))
		}
	case Os:
		if _, ok := PlatformOsMap[config]; !ok {
			panic(fmt.Errorf("Unknown os: %s", config))
		}
	case ArchOs:
		if _, ok := PlatformTargetMap[config]; !ok {
			panic(fmt.Errorf("Unknown os+arch: %s", config))
		}
	case ProductVariables:
		// do nothing
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationType %d", ct))
	}
}

func (ct configurationType) SelectKey(config string) string {
	ct.validateConfig(config)
	switch ct {
	case NoConfig:
		panic(fmt.Errorf("SelectKey is unnecessary for NoConfig ConfigurationType "))
	case Arch:
		return PlatformArchMap[config]
	case Os:
		return PlatformOsMap[config]
	case ArchOs:
		return PlatformTargetMap[config]
	case ProductVariables:
		if config == CONDITIONS_DEFAULT {
			return ConditionsDefaultSelectKey
		}
		return fmt.Sprintf("%s:%s", productVariableBazelPackage, strings.ToLower(config))
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationType %d", ct))
	}
}

var (
	NoConfigAxis            = ConfigurationAxis{configurationType: NoConfig}
	ArchConfigurationAxis   = ConfigurationAxis{configurationType: Arch}
	OsConfigurationAxis     = ConfigurationAxis{configurationType: Os}
	ArchOsConfigurationAxis = ConfigurationAxis{configurationType: ArchOs}
)

func ProductVariableConfigurationAxis(variable string) ConfigurationAxis {
	return ConfigurationAxis{
		configurationType: ProductVariables,
		subType:           variable,
	}
}

type ConfigurationAxis struct {
	configurationType
	subType string
}

func (ca *ConfigurationAxis) less(other ConfigurationAxis) bool {
	if ca.configurationType < other.configurationType {
		return true
	}
	return ca.subType < other.subType
}
