// Copyright 2016 Google Inc. All rights reserved.
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
	"debug/elf"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	arch   archArg
	need   needArgs
	soname = flag.String("soname", "", "verify the soname matches")
)

type archArg struct {
	class   elf.Class
	machine elf.Machine
}

func (a *archArg) String() string {
	return a.machine.String()
}

func (a *archArg) Set(s string) error {
	mapping := map[string]archArg{
		"arm":    {elf.ELFCLASS32, elf.EM_ARM},
		"arm64":  {elf.ELFCLASS64, elf.EM_AARCH64},
		"mips":   {elf.ELFCLASS32, elf.EM_MIPS},
		"mips64": {elf.ELFCLASS64, elf.EM_MIPS},
		"x86":    {elf.ELFCLASS32, elf.EM_386},
		"x86_64": {elf.ELFCLASS64, elf.EM_X86_64},
	}
	if arch, ok := mapping[s]; ok {
		a.class = arch.class
		a.machine = arch.machine
		return nil
	}
	return fmt.Errorf("Unknown machine type for %s", s)
}

type needArgs []string

func (a *needArgs) String() string {
	return strings.Join(*a, " ")
}

func (a *needArgs) Set(s string) error {
	*a = append(*a, s)
	return nil
}

func init() {
	flag.Var(&arch, "arch", "verify library is the correct architecture")
	flag.Var(&need, "need", "verify library uses only the listed libraries")
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: verify_prebuilt_lib <lib.so>")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
	}

	file, err := elf.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: cannot open", flag.Arg(0), err)
		return
	}
	defer file.Close()

	if arch.machine != elf.EM_NONE {
		if file.FileHeader.Class != arch.class {
			fmt.Fprintln(os.Stderr, "Warning: ELF Class does not match:", arch.class, file.FileHeader.Class)
		}

		if file.FileHeader.Machine != arch.machine {
			fmt.Fprintln(os.Stderr, "Warning: Architecture does not match:", arch.machine, file.FileHeader.Machine)
		}
	}

	if file.FileHeader.Type != elf.ET_DYN {
		fmt.Fprintln(os.Stderr, "Warning: File is not dynamic:", file.FileHeader.Type)
		return
	}

	if file_soname, err := file.DynString(elf.DT_SONAME); err == nil {
		if len(file_soname) == 0 {
			fmt.Fprintln(os.Stderr, "Warning: Missing required DT_SONAME")
		} else if len(file_soname) > 1 {
			fmt.Fprintln(os.Stderr, "Warning: More than one DT_SONAME?")
		} else if *soname != "" && *soname != file_soname[0] {
			fmt.Fprintln(os.Stderr, "Warning: DT_SONAME does not match:", *soname, file_soname[0])
		}
	} else {
		fmt.Fprintln(os.Stderr, "Warning: Failed to read DT_SONAME:", err)
	}

	if file_needed, err := file.DynString(elf.DT_NEEDED); err == nil {
		for i := range file_needed {
			var found bool
			for j := range need {
				if file_needed[i] == need[j] {
					found = true
					break
				}
			}

			if !found {
				fmt.Fprintln(os.Stderr, "Warning: Library requires extra library:", file_needed[i])
			}
		}

		for i := range need {
			var found bool
			for j := range file_needed {
				if need[i] == file_needed[j] {
					found = true
					break
				}
			}

			if !found {
				fmt.Fprintln(os.Stderr, "Warning: Library does not require:", need[i])
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "Warning: Failed to read DT_NEEDED:", err)
	}
}
