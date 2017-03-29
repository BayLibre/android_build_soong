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
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"android/soong/ui/logger"
)

// This file provides cross-process synchronization methods
// i.e. making sure only one Soong process is running for a given output directory

// start of public api
func BecomeSingletonOrFail(ctx Context, config Config) (lock *fileLock) {
	lockingInfo, err := newLock(config.OutDir())
	if err != nil {
		ctx.Logger.Fatal(err)
	}
	err = lockSynchronous(*lockingInfo, newSleepWaiter(lockfilePollDuration, lockfileTimeout), ctx.Logger)
	if err != nil {
		ctx.Logger.Fatal(err)
	}
	return lockingInfo
}

var lockfileTimeout = time.Minute
var lockfilePollDuration = time.Second

// end of public api

type lockable interface {
	tryLock() error
	unlock() error
	description() string
}

type fileLock struct {
	File *os.File
}

func (l fileLock) description() (path string) {
	return l.File.Name()
}
func (l fileLock) tryLock() (err error) {
	var lockCommand int
	lockCommand = syscall.F_SETLK

	return syscall.Flock(int(l.File.Fd()), lockCommand)
}
func (l fileLock) unlock() (err error) {
	return l.File.Close()
}

func lockSynchronous(lock lockable, waiter waiter, logger logger.Logger) (err error) {

	waited := false

	for {
		err = lock.tryLock()
		if err == nil {
			if waited {
				// If we had to wait at all, then when the wait is done, we inform the user
				logger.Printf("Acquired lock on %v; previous Soong process must have completed. Continuing...\n", lock.description())
			}
			return nil
		}

		waited = true

		done, description := waiter.checkDeadline()

		if done {
			return fmt.Errorf("Tried to lock %s, but timed out %s . Make sure no other Soong process is using it",
				lock.description(), waiter.summarize())
		} else {
			logger.Printf("Waiting up to %s to lock %v to ensure no other Soong process is running in the same output directory\n", description, lock.description())
			waiter.wait()
		}
	}
}

func newLock(basedir string) (lock *fileLock, err error) {
	lockPath := filepath.Join(basedir, ".lock")

	os.MkdirAll(basedir, 0750)
	lockfileDescriptor, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, errors.New("failed to open " + lockPath)
	}
	lockingInfo := &fileLock{File: lockfileDescriptor}

	return lockingInfo, nil
}

type waiter interface {
	wait()
	checkDeadline() (done bool, remainder string)
	summarize() (summary string)
}

type sleepWaiter struct {
	sleepInterval time.Duration
	deadline      time.Time

	totalWait time.Duration
}

func newSleepWaiter(interval time.Duration, duration time.Duration) (waiter *sleepWaiter) {
	return &sleepWaiter{interval, time.Now().Add(duration), duration}
}

func (s sleepWaiter) wait() {
	time.Sleep(s.sleepInterval)
}
func (s *sleepWaiter) checkDeadline() (done bool, remainder string) {
	remainingSleep := s.deadline.Sub(time.Now())
	numSecondsRounded := math.Floor(remainingSleep.Seconds()*10+0.5) / 10
	if remainingSleep > 0 {
		return false, fmt.Sprintf("%vs", numSecondsRounded)
	} else {
		return true, ""
	}
}
func (s sleepWaiter) summarize() (summary string) {
	return fmt.Sprintf("polling every %v until %v", s.sleepInterval, s.totalWait)
}

// countWaiter only exists for the purposes of testing lockSynchronous
type countWaiter struct {
	numWaitsElapsed int
	maxNumWaits     int
}

func newCountWaiter(count int) (waiter *countWaiter) {
	return &countWaiter{0, count}
}

func (c *countWaiter) wait() {
	c.numWaitsElapsed++
}
func (c *countWaiter) checkDeadline() (done bool, remainder string) {
	numWaitsRemaining := c.maxNumWaits - c.numWaitsElapsed
	if numWaitsRemaining < 1 {
		return true, ""
	}
	fmt.Printf("proc_sync countWaiter deadline diff = %v\n", numWaitsRemaining)
	return false, fmt.Sprintf("%v waits remain", numWaitsRemaining)
}
func (c countWaiter) summarize() (summary string) {
	return fmt.Sprintf("waiting %v times", c.maxNumWaits)
}

// countLock only exists for the purposes of testing lockSynchronous
type countLock struct {
	nextIndex    int
	successIndex int
}

// returns a countLock that succeeds on iteration <index>
func testLockCountingTo(index int) (lock *countLock) {
	return &countLock{nextIndex: 0, successIndex: index}
}
func (c *countLock) description() (message string) {
	return fmt.Sprintf("counter that counts from %v to %v", c.nextIndex, c.successIndex)
}
func (c *countLock) tryLock() (err error) {
	currentIndex := c.nextIndex
	c.nextIndex++
	if currentIndex == c.successIndex {
		fmt.Printf("proc_sync tryLock succeeding at index %v\n", currentIndex)
		return nil
	}
	fmt.Printf("proc_sync tryLock failing at index %v\n", currentIndex)
	return fmt.Errorf("Lock busy: %s", c.description())
}
func (c *countLock) unlock() (err error) {
	if c.nextIndex == c.successIndex {
		return nil
	}
	return fmt.Errorf("Not locked: %s", c.description())
}
