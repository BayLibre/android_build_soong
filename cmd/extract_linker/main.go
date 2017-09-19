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

// This tool extracts ELF LOAD segments from our linker binary, and produces an
// assembly file and linker script which will embed those segments as sections
// in another binary.
package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	var asmPath string
	var scriptPath string

	flag.StringVar(&asmPath, "s", "", "Path to save the assembly file")
	flag.StringVar(&scriptPath, "T", "", "Path to save the linker script")
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Error opening %q: %v", flag.Arg(0), err)
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		log.Fatal("Unable to read elf file: %v", err)
	}

	asm := &bytes.Buffer{}
	lscript := &bytes.Buffer{}

	baseLoadAddr := uint64(0x1000)
	fmt.Fprintln(lscript, "ENTRY(__dlwrap__start)")
	fmt.Fprintln(lscript, "SECTIONS {")
	fmt.Fprintln(lscript, "  __dlwrap_original_start = _start;")
	fmt.Fprintln(lscript, "  /DISCARD/ : { *(.interp) }")

	fmt.Fprintln(asm, ".globl __dlwrap_linker_entry")
	fmt.Fprintf(asm, ".set __dlwrap_linker_entry, 0x%x\n\n", ef.Entry)

	load := 0
	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}

		sectionName := fmt.Sprintf(".linker.sect%d", load)
		flags := ""
		if prog.Flags&elf.PF_W != 0 {
			flags += "w"
		}
		if prog.Flags&elf.PF_X != 0 {
			flags += "x"
		}
		fmt.Fprintf(asm, ".section %s, \"a%s\"\n", sectionName, flags)

		if load == 0 {
			fmt.Fprintln(asm, ".globl __dlwrap_linker_code_start")
			fmt.Fprintln(asm, "__dlwrap_linker_code_start:")
		}

		buffer := &bytes.Buffer{}
		io.Copy(buffer, prog.Open())
		bytesToAsm(asm, buffer.Bytes())

		// Fill in zeros for any BSS sections. It would be nice to keep
		// this as a true BSS, but ld/gold isn't preserving those,
		// instead combining the segments with the following segment,
		// and BSS only exists at the end of a LOAD segment.  The
		// linker doesn't use a lot of BSS, so this isn't a huge
		// problem.
		if prog.Memsz > prog.Filesz {
			fmt.Fprintf(asm, ".fill 0x%x, 1, 0\n", prog.Memsz-prog.Filesz)
		}
		fmt.Fprintln(asm)

		fmt.Fprintf(lscript, "  . = 0x%x;\n", baseLoadAddr+prog.Vaddr)
		fmt.Fprintf(lscript, "  %s : { KEEP(*(%s)) }\n", sectionName, sectionName)

		load += 1
	}

	fmt.Fprintln(lscript, "  .text : { *(.text .text.*) }")
	fmt.Fprintln(lscript, "  .rodata : { *(.rodata .rodata.* .gnu.linkonce.r.*) }")
	fmt.Fprintln(lscript, "  .data : { *(.data .data.* .gnu.linkonce.d.*) }")
	fmt.Fprintln(lscript, "  .bss : { *(.dynbss) *(.bss .bss.* .gnu.linkonce.b.*) *(COMMON) }")
	fmt.Fprintln(lscript, "}")

	if asmPath != "" {
		if err := ioutil.WriteFile(asmPath, asm.Bytes(), 0777); err != nil {
			log.Fatal("Unable to write %q: %v", asmPath, err)
		}
	}

	if scriptPath != "" {
		if err := ioutil.WriteFile(scriptPath, lscript.Bytes(), 0777); err != nil {
			log.Fatal("Unable to write %q: %v", scriptPath, err)
		}
	}
}

func bytesToAsm(asm io.Writer, buf []byte) {
	newline := true
	count := 0
	for {
		if len(buf) == 0 {
			break
		} else if len(buf) < 8 {
			if !newline {
				fmt.Fprintln(asm)
			}
			newline = false
			fmt.Fprintf(asm, ".byte ")
			for i, b := range buf {
				if i != 0 {
					fmt.Fprintf(asm, ",")
				}
				fmt.Fprintf(asm, "0x%x", b)
			}
			break
		} else {
			if newline {
				fmt.Fprintf(asm, ".quad ")
				newline = false
			} else {
				fmt.Fprintf(asm, ",")
			}

			fmt.Fprintf(asm, "0x%x", binary.LittleEndian.Uint64(buf[:8]))
			buf = buf[8:]

			count += 1
			if count > 5 {
				fmt.Fprintln(asm)
				newline = true
				count = 0
			}
		}
	}
	if !newline {
		fmt.Fprintln(asm)
	}
}
