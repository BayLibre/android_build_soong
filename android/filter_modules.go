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
    "fmt"
)

func registerFilterModulesMutator(ctx RegisterMutatorsContext) {
	ctx.TopDown("filter_modules", filterModulesMutator).Parallel()
}

func filterModulesMutator(ctx TopDownMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

    config := ctx.Config()

    fmt.Printf("filterModulesMutator module %s -- %t %t\n", m.String(), m.IsNecessary(), config.FilterModule(m.Name()))

    if config.FilterModule(m.Name()) {
        m.SetNecessary(true)
    }

    if (m.IsNecessary()) {
        ctx.VisitDirectDeps(func(dep Module) {
            fmt.Printf("    dep %s -- %t\n", dep.String(), m.IsNecessary())
            m.SetNecessary(true)
        })
    }
}
