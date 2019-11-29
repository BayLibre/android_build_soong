// Copyright 2019 Google Inc. All rights reserved.
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

package android

// ConfigPlugin represents plugins that require initialization with
// configuration information.
type ConfigPlugin interface {
	// Function that will initialize a plugin with Configuration
	InitPlugin(Config) error
}

// Registered plugins
var plugins []ConfigPlugin

// RegisterPlugin is called by plugins to have Soong load
// configuration information into Config, and then pass the
// configuration to an initialization function.
//
// This is intended to be used by plugins whose modules' properties
// depend on their configuration.
//
// This is expected to be called by Soong plugins during their init(),
// if they need configuration.
func RegisterPlugin(plugin ConfigPlugin) {
	plugins = append(plugins, plugin)
}

func InitializePlugins(config Config) error {
	for _, p := range plugins {
		err := p.InitPlugin(config)
		if err != nil {
			return err
		}
	}

	return nil
}
