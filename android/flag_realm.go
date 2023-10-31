// Copyright 2018 Google Inc. All rights reserved.
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
	"fmt"

	"github.com/google/blueprint/parser"
)

type flagSelectRealm struct {
}

func newFlagSelectRealm() parser.SelectRealm {
	fmt.Printf("newFlagSelectRealm\n")
	return &flagSelectRealm{
	}
}

func (this *flagSelectRealm) GetSelectValue(condition string) (bool,parser.Expression) {
	return true, &parser.String{Value:"I HATE GOLANG"}
}

