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
	"bytes"
	"strings"
	"testing"
)

func TestFilter(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{
			in:  "/abspath/external/libcxx/src/include/atomic_support.h:42:1:\n",
			out: "external/libcxx/src/include/atomic_support.h:42:1:\n",
		},
		{
			in: `404 warnings generated.
Suppressed 389 warnings (380 in non-user code, 8 NOLINT, 1 with check filters).
Use -header-filter=.* to display errors from all non-system headers. Use -system-headers to display errors from system headers as well.
`,
			out: "Suppressed 389 warnings (380 in non-user code, 8 NOLINT, 1 with check filters).\n",
		},
		{
			in: `1 warning generated.
Suppressed 1 warnings (1 with check filters).
`,
			out: "Suppressed 1 warnings (1 with check filters).\n",
		},
		{
			in: `1 warning generated.
Other error
Suppressed 1 warnings (1 with check filters).
`,
			out: `Other error
Suppressed 1 warnings (1 with check filters).
`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.in, func(t *testing.T) {
			r := strings.NewReader(testCase.in)
			buf := &bytes.Buffer{}

			err := filter(r, buf, "/abspath/")
			if err != nil {
				t.Fatal(err)
			}

			if buf.String() != testCase.out {
				t.Error("Output does not match expected:")
				t.Error(buf.String())
				t.Error(testCase.out)
			}
		})
	}
}
