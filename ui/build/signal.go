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
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"android/soong/ui/logger"
)

func SetupSignals(log logger.Logger, cancel, cleanup func()) {
	signals := make(chan os.Signal, 5)
	// TODO: Handle other signals
	signal.Notify(signals, os.Interrupt, syscall.SIGALRM)
	go handleSignals(signals, log, cancel, cleanup)
}

func handleSignals(signals chan os.Signal, log logger.Logger, cancel, cleanup func()) {
	defer cleanup()

	var force bool

	for {
		s := <-signals
		if force {
			// So that we can better see what was stuck
			debug.SetTraceback("all")
			log.Panicln("Second signal received:", s)
		} else {
			log.Println("Got signal:", s)
			cancel()
			force = true
		}
	}
}
