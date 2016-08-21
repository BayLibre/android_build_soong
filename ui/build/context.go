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

package build

import (
	"context"
	"io"
	"os"

	"android/soong/ui/logger"
)

type StdIoInterface interface {
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
}

type StdIoImpl struct{}

func (StdIoImpl) Stdin() io.Reader  { return os.Stdin }
func (StdIoImpl) Stdout() io.Writer { return os.Stdout }
func (StdIoImpl) Stderr() io.Writer { return os.Stderr }

type CustomStdIo struct {
	StdIn  io.Reader
	StdOut io.Writer
	StdErr io.Writer
}

func (c CustomStdIo) Stdin() io.Reader  { return c.StdIn }
func (c CustomStdIo) Stdout() io.Writer { return c.StdOut }
func (c CustomStdIo) Stderr() io.Writer { return c.StdErr }

// Context combines a context.Context, logger.Logger, and StdIO redirection.
// These all are agnostic of the current build, and may be used for multiple
// builds, while the Config objects contain per-build information.
type Context *ContextImpl
type ContextImpl struct {
	context.Context
	logger.Logger
	StdIoInterface
}
