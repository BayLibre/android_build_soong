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
	"android/soong/ui/logger"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// This file provides cross-process synchronization methods
// i.e. making sure only one Soong process is running for a given output directory

// start of exposed methods
func BecomeSingletonOrFail(config Config, logger logger.Logger) {
	info := getLockingInfo(config)
	err := info.lock()
	if err != nil {
		logger.Fatal(err)
	}
}
func TrySurrenderSingleton(config Config, logger logger.Logger) {
	info := getLockingInfo(config)
	err := info.unlock()
	if err != nil {
		// Unfortunately there's not much we can do if we fail to surrender the singleton
		logger.Print(err)
	}
}

// end of exposed methods

var lockfileTimeout = time.Minute
var lockingInfo *lockfile

func getLockingInfo(config Config) *lockfile {
	if lockingInfo == nil {
		basedir := filepath.Join(config.SoongOutDir(), "locking")
		os.MkdirAll(basedir, 0750)
		lockPath := filepath.Join(basedir, "soong.lock")
		lockfileDescriptor, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic("failed to open " + lockPath)
		}
		lockingInfo = &lockfile{File: lockfileDescriptor, Timeout: lockfileTimeout}
	}
	return lockingInfo
}

type lockfile struct {
	File    *os.File
	Timeout time.Duration
}

func (l lockfile) Filepath() (path string) {
	return l.File.Name()
}

func (l lockfile) lock() (err error) {
	sleepInterval := time.Second
	sleepDeadline := time.Now().Add(l.Timeout)

	waited := false

	for {
		err := l.tryLock()
		if err == nil {
			if waited {
				// If we had to wait at all, then inform the user a message when the wait is done
				fmt.Printf("Acquired lock on %v; previous Soong process must have completed. Continuing...\n", l.Filepath())
			}
			return nil
		}

		waited = true
		remainingSleep := sleepDeadline.Sub(time.Now())
		numSecondsRounded := math.Floor(remainingSleep.Seconds()*10+0.5) / 10

		if remainingSleep > 0 {
			fmt.Printf("Waiting up to %vs to lock %v to ensure only one Soong process runs at once\n", numSecondsRounded, l.Filepath())
			time.Sleep(sleepInterval)
		} else {
			return fmt.Errorf("Timed out after %v while waiting to lock %v; make sure no other Soong process is using it", l.Timeout, l.Filepath())
		}
	}
}

func (l lockfile) tryLock() (err error) {
	return l.changeLockStateImpl(true)
}
func (l lockfile) unlock() (err error) {
	return l.changeLockStateImpl(false)
}
func (l lockfile) changeLockStateImpl(locked bool) (err error) {
	var lockCommand int
	if locked {
		lockCommand = syscall.F_SETLK
	} else {
		lockCommand = syscall.F_UNLCK
	}
	lockInstructions := &syscall.Flock_t{Type: syscall.F_WRLCK, Whence: int16(os.SEEK_SET), Start: 0, Len: 1}
	err = syscall.FcntlFlock(l.File.Fd(), lockCommand, lockInstructions)
	if err != nil {
		return err
	}
	return nil
}
