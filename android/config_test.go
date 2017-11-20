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

package android

import (
	"testing"
)

type configType struct {
	populateMe *bool `json:"omitempty"`
}

func (c *configType) SetDefaultConfig() {
}

func TestOmitEmpty(t *testing.T) {

	config := configType{}

	err := validateConfigAnnotations(&config)
	expectedError := `Field configType.populateMe has tag json:"omitempty" which specifies to change its json field name to "omitempty".
Did you mean to use an annotation of ",omitempty"?
(Alternatively, to change the json name of the field, rename the field in source instead.)`
	if err.Error() != expectedError {
		t.Errorf("Incorrect error; expected:\n"+
			"%s\n"+
			"got:\n"+
			"%s",
			expectedError, err.Error())
	}

}
