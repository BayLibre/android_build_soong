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

package main

import (
	"debug/elf"
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	input  = flag.String("i", "", "input file")
	output = flag.String("o", "", "output file")
	symbol = flag.String("s", "", "symbol to inject into")
	value  = flag.String("v", "", "value to inject into symbol")
)

func main() {
	flag.Parse()

	usageError := func(s string) {
		fmt.Fprintln(os.Stderr, s)
		flag.Usage()
		os.Exit(1)
	}

	if *input == "" {
		usageError("-i is required")
	}

	if *output == "" {
		usageError("-o is required")
	}

	if *symbol == "" {
		usageError("-s is required")
	}

	if *value == "" {
		usageError("-v is required")
	}

	r, err := os.Open(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	defer r.Close()

	w, err := os.OpenFile(*output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
	defer w.Close()

	err = injectSymbol(r, w, *symbol, *value)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Remove(*output)
		os.Exit(2)
	}
}

type ReadSeekerAt interface {
	io.ReaderAt
	io.ReadSeeker
}

func injectSymbol(r ReadSeekerAt, w io.Writer, symbol, value string) error {
	elfFile, err := elf.NewFile(r)
	if err != nil {
		return err
	}

	symbols, err := elfFile.Symbols()
	if err != nil {
		return err
	}

	for _, s := range symbols {
		if elf.ST_TYPE(s.Info) != elf.STT_OBJECT {
			continue
		}
		if s.Name == symbol {
			if uint64(len(value))+1 > s.Size {
				return fmt.Errorf("value length %d overflows symbol size %d", len(value), s.Size)
			}

			offset, err := calculateSymbolOffset(elfFile, s)
			if err != nil {
				return err
			}
			return copyAndInject(r, w, offset, s.Size, value)
		}
	}

	return fmt.Errorf("symbol not found")
}

func calculateSymbolOffset(file *elf.File, symbol elf.Symbol) (uint64, error) {
	errOffset := ^uint64(0)
	if symbol.Section == elf.SHN_UNDEF || int(symbol.Section) >= len(file.Sections) {
		return errOffset, fmt.Errorf("invalid section index %d", symbol.Section)
	}
	section := file.Sections[symbol.Section]
	switch file.Type {
	case elf.ET_REL:
		// "In relocatable files, st_value holds a section offset for a defined symbol.
		// That is, st_value is an offset from the beginning of the section that st_shndx identifies."
		return file.Sections[symbol.Section].Addr + symbol.Value, nil
	case elf.ET_EXEC, elf.ET_DYN:
		// "In executable and shared object files, st_value holds a virtual address. To make these
		// files’ symbols more useful for the dynamic linker, the section offset (file interpretation)
		// gives way to a virtual address (memory interpretation) for which the section number is
		// irrelevant."
		if symbol.Value < section.Addr {
			return errOffset, fmt.Errorf("symbol starts before the start of its section")
		}
		section_offset := symbol.Value - section.Addr
		if section_offset+symbol.Size > section.Size {
			return errOffset, fmt.Errorf("symbol extends past the end of its section")
		}
		return section.Offset + section_offset, nil
	default:
		return errOffset, fmt.Errorf("unsupported elf file type %d", file.Type)
	}
}

func copyAndInject(r io.ReadSeeker, w io.Writer, offset, size uint64, value string) error {
	var err error

	// helper that asserts a two-value function returning an int64 and an error has err != nil
	must := func(n int64, err error) {
		if err != nil {
			panic(err)
		}
	}

	// helper that asserts a two-value function returning an int and an error has err != nil
	must2 := func(n int, err error) {
		must(int64(n), err)
	}

	// convert a panic into returning an error
	defer func() {
		if err == nil {
			if r := recover(); r != nil {
				err, _ = r.(error)
				if err == nil {
					panic(r)
				}
			}
		}
	}()

	buf := make([]byte, size)
	copy(buf, value)

	// Reset the input file
	must(r.Seek(0, io.SeekStart))
	// Copy the first bytes up to the symbol offset
	must(io.CopyN(w, r, int64(offset)))
	// Skip the symbol contents in the input file
	must(r.Seek(int64(size), io.SeekCurrent))
	// Write the injected value in the output file
	must2(w.Write(buf))
	// Write the remainder of the file
	must(io.Copy(w, r))

	return err
}
