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

package shared

import (
	"io/ioutil"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// Save takes a protobuf message, marshals to an array of bytes
// and is then saved to a file.
func Save(pb proto.Message, filepath string) (err error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	tempFilepath := filepath + ".tmp"
	if err := ioutil.WriteFile(tempFilepath, []byte(data), 0644 /* rw-r--r-- */); err != nil {
		return err
	}

	if err := os.Rename(tempFilepath, filepath); err != nil {
		return err
	}

	return nil
}

// SaveTextProto takes a protobuf message, marshals to textproto
// and is then saved to a file.
func SaveTextProto(pb proto.Message, filepath string, header []string) (err error) {
	lines := header
	lines = append(lines, prototext.Format(pb))
	tempFilepath := filepath + ".tmp"
	if err := ioutil.WriteFile(tempFilepath, []byte(strings.Join(lines, "\n")), 0666 /* rw-rw-rw- */); err != nil {
		return err
	}

	if err := os.Rename(tempFilepath, filepath); err != nil {
		return err
	}

	return nil
}
