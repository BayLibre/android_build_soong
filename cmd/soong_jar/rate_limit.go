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
	"runtime"
)

type RateLimit struct {
	requests chan chan<- Finish
	finished chan int
}

// NewRateLimit starts a new rate limiter with maxExecs number of executions allowed
// to happen at a time. If maxExecs is <= 0, it will default to the number of logical
// CPUs on the system.
func NewRateLimit(maxExecs int) *RateLimit {
	if maxExecs <= 0 {
		maxExecs = runtime.NumCPU()
	}

	ret := &RateLimit{
		// Allow an essentially unlimited number of outstanding requests
		requests: make(chan chan<- Finish, 10000),

		finished: make(chan int, maxExecs),
	}

	go ret.goFunc(maxExecs)

	return ret
}

// Make a request to execute. Requests will be approved on a FIFO basis.
//
// This will return an ExecutionRequest object, which you then need to call
// Wait on before you proceed.
func (r *RateLimit) RequestExecution() ExecutionRequest {
	ret := make(chan Finish, 1)
	r.requests <- ret
	return ret
}

type ExecutionRequest <-chan Finish

// Wait will block for the ExecutionRequest to be approved. You must call Finish()
// on the returned object when you're done executing.
func (e ExecutionRequest) Wait() Finish {
	return <-e
}

type Finish chan<- int

// Finish will mark your execution as finished, and allow another request to be approved.
func (f Finish) Finish() {
	f <- 1
}

// Stop the background goroutine
func (r *RateLimit) Stop() {
	close(r.finished)
}

func (r *RateLimit) goFunc(maxExecs int) {
	curExecs := 0

	for {
		if curExecs < maxExecs {
			select {
			case c := <-r.requests:
				curExecs++
				c <- r.finished
			case _, ok := <-r.finished:
				if !ok {
					return
				}

				curExecs--
				if curExecs < 0 {
					panic("curExecs < 0")
				}
			}
		} else {
			select {
			case _, ok := <-r.finished:
				if !ok {
					return
				}

				curExecs--
				if curExecs < 0 {
					panic("curExecs < 0")
				}
			}
		}
	}
}
