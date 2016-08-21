// Copyright 2017 Google Inc. All rights reserved.
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

package build

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Environment adds a number of useful manipulation functions to the list of
// strings returned by os.Environ() and used in exec.Cmd.Env.
type Environment []string

// OsEnvironment wraps the current environment returned by os.Environ()
func OsEnvironment() *Environment {
	env := Environment(os.Environ())
	return &env
}

// Get returns the value associated with the key, and whether it exists.
// It's equivalent to the os.LookupEnv function, but with this copy of the
// Environment.
func (e *Environment) Get(key string) (string, bool) {
	for _, env := range *e {
		idx := strings.IndexRune(env, '=')
		if env[0:idx] == key {
			return env[idx+1:], true
		}
	}
	return "", false
}

// Set sets the value associated with the key, overwriting the current value
// if it exists.
func (e *Environment) Set(key, value string) {
	e.Unset(key)
	*e = append(*e, key+"="+value)
}

// Unset removes the specified keys from the Environment.
func (e *Environment) Unset(keys ...string) {
	out := make([]string, 0, len(*e))
EnvLoop:
	for _, env := range *e {
		key := env[0:strings.IndexRune(env, '=')]
		for _, k := range keys {
			if key == k {
				continue EnvLoop
			}
		}
		out = append(out, env)
	}
	*e = out
}

// Environ returns the []string required for exec.Cmd.Env
func (e *Environment) Environ() []string {
	return []string(*e)
}

// Copy returns a copy of the Environment so that independent changes may be made.
func (e *Environment) Copy() *Environment {
	ret := Environment(make([]string, len(*e)))
	for i, v := range *e {
		ret[i] = v
	}
	return &ret
}

// AppendFromKati reads a shell script written by Kati that exports or unsets
// environment variables, and applies those to the local Environment.
func (e *Environment) AppendFromKati(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())

		if len(text) == 0 || text[0] == '#' {
			continue
		}

		cmd := strings.SplitN(text, " ", 2)
		if cmd[0] == "unset" {
			e.Unset(cmd[1][1 : len(cmd[1])-1])
		} else if cmd[0] == "export" {
			keyvalue := strings.SplitN(cmd[1], "'='", 2)
			key := strings.TrimPrefix(keyvalue[0], "'")
			value := strings.TrimSuffix(keyvalue[1], "'")
			e.Set(key, value)
		} else {
			return fmt.Errorf("Unknown kati environment command: %q", cmd)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
