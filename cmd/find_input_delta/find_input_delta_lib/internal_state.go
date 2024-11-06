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

package find_input_delta_lib

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"

	fid_proto "android/soong/cmd/find_input_delta/find_input_delta_proto_internal"
	"github.com/google/blueprint/pathtools"
	"google.golang.org/protobuf/proto"
)

type FileStates []*FileState

type InputState struct {
	Files FileStates
}

type FileState struct {
	// The name of the file
	Name string

	// mtime of the file, from time.Time.UnixNano().
	MtimeNsec int64

	// Hash, formatted as "hash_name:hash_value".
	Hash string

	// Information about the contents of this archive.
	Contents FileStates
}

func LoadState(filename string, fsys fs.ReadFileFS) (InputState, error) {
	data, err := fsys.ReadFile(filename)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return InputState{}, err
	}

	var message = &fid_proto.PartialCompileInputs{}
	proto.Unmarshal(data, message)
	ret := InputState{}

	for _, pci := range message.InputFiles {
		fs := &FileState{}
		if fs.Unmarshal(pci) != nil {
			return InputState{}, err
		}
		ret.Files = append(ret.Files, fs)
	}
	return ret, nil
}

func (fs *FileState) Unmarshal(pci *fid_proto.PartialCompileInput) error {
	fs.Name = pci.GetName()
	fs.MtimeNsec = pci.GetMtimeNsec()
	fs.Hash = pci.GetHash()
	myContents := pci.GetContents()
	if myContents != nil {
		for _, cfs := range myContents {
			content := &FileState{}
			if err := content.Unmarshal(cfs); err != nil {
				return err
			}
			fs.Contents = append(fs.Contents, content)
		}
	}
	return nil
}

type StatReadFileFS interface {
	fs.StatFS
	fs.ReadFileFS
}

func (fs *FileState) InspectContents(name string) (FileStates, error) {
	// TODO: Actually inspect the contents.
	fmt.Printf("inspecting contents for %s\n", name)
	return nil, nil
}

func CreateState(inputs []string, inspect_contents bool, fsys StatReadFileFS) (InputState, error) {
	ret := InputState{}
	slices.Sort(inputs)
	for _, input := range inputs {
		stat, err := fs.Stat(fsys, input)
		if err != nil {
			return ret, err
		}
		fileState := &FileState{
			Name:      input,
			MtimeNsec: stat.ModTime().UnixNano(),
		}
		if inspect_contents {
			contents, err := fileState.InspectContents(input)
			if err != nil {
				return InputState{}, err
			}
			if contents != nil {
				fileState.Contents = contents
			}
		}
		ret.Files = append(ret.Files, fileState)
	}
	return ret, nil
}

func (fs FileState) Marshal() (*fid_proto.PartialCompileInput, error) {
	pci := &fid_proto.PartialCompileInput{
		Name: proto.String(fs.Name),
	}
	if fs.MtimeNsec != 0 {
		pci.MtimeNsec = proto.Int64(fs.MtimeNsec)
	}
	pci.Hash = proto.String(fs.Hash)
	for _, v := range fs.Contents {
		pb, err := v.Marshal()
		if err != nil {
			return pci, err
		}
		pci.Contents = append(pci.Contents, pb)
	}
	return pci, nil
}

func (s InputState) Marshal() (*fid_proto.PartialCompileInputs, error) {
	slices.SortFunc(s.Files, func(a, b *FileState) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name == b.Name {
			return 0
		} else {
			return 1
		}
	})

	ret := &fid_proto.PartialCompileInputs{}
	for _, fs := range s.Files {
		pb, err := fs.Marshal()
		if err != nil {
			return ret, err
		}
		ret.InputFiles = append(ret.InputFiles, pb)
	}
	return ret, nil
}

func (s InputState) WriteState(path string) error {
	message, err := s.Marshal()
	if err != nil {
		return err
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return pathtools.WriteFileIfChanged(path, data, 0644)
}

func (s FileStates) Equal(other FileStates) bool {
	if len(s) != len(other) {
		return false
	}
	for idx, v := range s {
		if !v.Equal(other[idx]) {
			return false
		}
	}
	return true
}

func (fs *FileState) Equal(other *FileState) bool {
	if fs.Name != other.Name || fs.MtimeNsec != other.MtimeNsec {
		return false
	}
	if fs.Hash != other.Hash {
		return false
	}
	return fs.Contents.Equal(other.Contents)
}

func (prior InputState) Compare(other InputState, target string) FileList {
	fileList := prior.Files.Compare(other.Files, target)
	return *fileList
}

func (prior FileStates) Compare(other FileStates, name string) *FileList {
	ret := &FileList{
		Name: name,
	}
	PriorMap := make(map[string]*FileState, len(prior))
	// We know that the lists are properly sorted, so we can simply compare them.
	for _, v := range prior {
		PriorMap[v.Name] = v
	}
	otherMap := make(map[string]*FileState, len(other))
	for _, v := range other {
		otherMap[v.Name] = v
		if _, ok := PriorMap[v.Name]; !ok {
			// Added file
			ret.Additions = append(ret.Additions, v.Name)
		} else if !PriorMap[v.Name].Equal(v) {
			// Modified file
			ret.Changes = append(ret.Changes, *PriorMap[v.Name].Contents.Compare(v.Contents, v.Name))
		}
	}
	for _, v := range prior {
		if _, ok := otherMap[v.Name]; !ok {
			// Deleted file
			ret.Deletions = append(ret.Deletions, v.Name)
		}
	}
	return ret
}
