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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// This file provides cross-process synchronization methods
// i.e. making sure only one Soong process is running for a given 'out' directory

// start of exposed methods
func BecomeSingleton(config Config) (err error) {
	info := getLockingInfo(config)
	err = info.lock()
	if err != nil {
		fmt.Println(err)
	}
	return err
}
func SurrenderSingleton(config Config) (err error) {
	info := getLockingInfo(config)
	err = info.unlock()

	if err != nil {
		fmt.Println(err)
	}
	return err
}

// end of exposed methods

var lockfileTimeout = time.Second
var pidfileTimeout = time.Minute

func getLockingInfo(config Config) atomicPidfile {
	basePath := config.SoongOutDir() + "/locking"
	return atomicPidfile{
		lockfile{basePath + "/soong.lock", lockfileTimeout},
		pidFile{basePath + "/soong.pid", pidfileTimeout},
	}
}

type lockfile struct {
	Filepath string
	Timeout  time.Duration
}

func (l lockfile) lock() (err error) {
	numSleeps := 10
	sleepInterval := time.Duration(l.Timeout.Nanoseconds() / int64(numSleeps))
	for i := 0; i < numSleeps; i++ {
		err := l.tryLock()
		if err == nil {
			return nil
		}
		time.Sleep(sleepInterval)
	}
	return errors.New(fmt.Sprintf("Timed out after %v while waiting to lock %s", l.Timeout, l.Filepath))
}
func (l lockfile) tryLock() (err error) {
	err = os.MkdirAll(path.Dir(l.Filepath), 0777)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(l.Filepath, os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return file.Close()
	}
	return nil
}
func (l lockfile) unlock() error {
	return os.Remove(l.Filepath)
}

type pidFile struct {
	Filepath string
	Timeout  time.Duration
}

type atomicPidfile struct {
	lockFile lockfile
	pidFile  pidFile
}

func (a atomicPidfile) lock() (err error) {
	maxSleep := a.pidFile.Timeout
	sleepInterval := time.Second
	numSleeps := int(maxSleep.Nanoseconds() / sleepInterval.Nanoseconds())
	var pid uint64
	for i := 0; i < numSleeps; i++ {
		// re-lock lock before reading pidFile
		// this prevents a possible race condition where one process is halfway done writing a pid while the other reads the pid
		err := a.lockFile.lock()
		if err != nil {
			if i == 0 {
				// if we couldn't get the lockFile on the first attempt
				// then we assume that that means that a previous Soong process failed to clean up the lockFile
				// and that therefore it's fine for us to clean it up now'
				fmt.Printf("Will clean up presumably stale lockfile %s; it appears to have existed for longer than %v\n", a.lockFile.Filepath, a.lockFile.Timeout)
			} else {
				// if we couldn't get the lockFile on an attempt other than the first, then give up
				a.lockFile.unlock()
				return err
			}
		}

		// re-read the pidFile on every iteration in case it gets deleted
		text, err := ioutil.ReadFile(a.pidFile.Filepath)

		// check for the existence of the process mentioned in the pidFile
		if err == nil {
			// parse the file text into a pid
			pid, err = strconv.ParseUint(strings.TrimSpace(string(text)), 10, 0)
			if err == nil {
				// find the process mentioned in the file
				proc, err := os.FindProcess(int(pid))
				if err == nil {
					// FindProcess can return a process and an empty error even when the process is not running
					// So now we have to check if it's really running
					err = proc.Signal(syscall.Signal(0))
					if err == nil {
						// Previous exists; try again later
						a.lockFile.unlock()
						fmt.Printf(
							"Waiting up to %s for existing Soong process (pid %v) specified in (%v) to terminate.\n",
							maxSleep, pid, a.pidFile.Filepath)
						time.Sleep(sleepInterval)
						continue
					}
				}
				fmt.Printf("Overwriting presumably stale pidfile %v; pid %v does not exist\n", a.pidFile.Filepath, pid)
			}
		}
		// couldn't find an existing process; save our pid to the pidFile
		fileContents := []byte(strconv.FormatUint(uint64(os.Getpid()), 10))
		err = ioutil.WriteFile(a.pidFile.Filepath, fileContents, 0666)
		a.lockFile.unlock()
		if err != nil {
			return err
		}
		return nil
	}
	// Timeout
	return errors.New(fmt.Sprintf("Timed out after %s while waiting for the existing Soong process (pid %v) to terminate.", maxSleep, pid))
}
func (a atomicPidfile) unlock() (err error) {
	subErr := a.lockFile.lock()
	if subErr != nil {
		err = subErr
		// if we got the lock, that's great and it's what's supposed to happen
		// if we failed to get the lock, then we still might as well clean up the pidFile anyway
	}

	subErr = os.Remove(a.pidFile.Filepath)
	if subErr != nil {
		err = subErr
	}

	// if we removed the pidFile, that's great and it's what's supposed to happen
	// if we failed to remove the pidFile, we might as well try to remove the lock anyway
	subErr = a.lockFile.unlock()
	if subErr != nil {
		err = subErr
	}
	return err
}
