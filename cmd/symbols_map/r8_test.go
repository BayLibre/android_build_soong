// Copyright 2022 Google Inc. All rights reserved.
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
	"bytes"
	"testing"
)

func Test_extractR8CompilerHash(t *testing.T) {
	testCases := []struct {
		name string
		data string
		hash string
	}{
		{
			name: "simple",
			data: `# compiler: R8
# compiler_version: 3.3.18-dev
# min_api: 10000
# compiler_hash: bab44c1a04a2201b55fe10394f477994205c34e0
# common_typos_disable
# {"id":"com.android.tools.r8.mapping","version":"2.0"}
# pg_map_id: 7fe8b95
# pg_map_hash: SHA-256 7fe8b95ae71f179f63d2a585356fb9cf2c8fb94df9c9dd50621ffa6d9e9e88da
android.car.userlib.UserHelper -> android.car.userlib.UserHelper:
`,
			hash: "bab44c1a04a2201b55fe10394f477994205c34e0",
		},
		{
			name: "empty",
			data: ``,
			hash: "",
		},
		{
			name: "non comment line",
			data: `# compiler: R8
# compiler_version: 3.3.18-dev
# min_api: 10000
android.car.userlib.UserHelper -> android.car.userlib.UserHelper:
# compiler_hash: bab44c1a04a2201b55fe10394f477994205c34e0
# common_typos_disable
# {"id":"com.android.tools.r8.mapping","version":"2.0"}
# pg_map_id: 7fe8b95
# pg_map_hash: SHA-256 7fe8b95ae71f179f63d2a585356fb9cf2c8fb94df9c9dd50621ffa6d9e9e88da
`,
			hash: "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := extractR8CompilerHash(bytes.NewBufferString(tt.data))
			if err != nil {
				t.Fatalf("unexpected error in extractR8CompilerHash: %s", err)
			}

			if g, w := hash, tt.hash; g != w {
				t.Errorf("incorrect hash, want %q got %q", w, g)
			}
		})
	}
}
