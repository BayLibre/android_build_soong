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

package paths

import (
	"context"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type LogProcess struct {
	Pid     int
	Command string
}

type LogEntry struct {
	Basename string
	Args     []string
	Parents  []LogProcess
}

const timeoutDuration = time.Duration(250) * time.Millisecond

type fallbackFunc func(name string) (addr string, cleanup func(), err error)
type fallbackDef struct {
	name      string
	linuxOnly bool

	f fallbackFunc
}

var fallbacks = []fallbackDef{
	{
		name:      "/proc/self/fd trick",
		linuxOnly: true,
		f: func(name string) (string, func(), error) {
			d, err := os.Open(filepath.Dir(name))
			if err != nil {
				return "", nil, err
			}

			return fmt.Sprintf("/proc/self/fd/%d/%s", d.Fd(), filepath.Base(name)), func() {
				d.Close()
			}, nil
		},
	},
	{
		name: "/tmp symlink",
		f: func(name string) (addr string, cleanup func(), err error) {
			d, err := ioutil.TempDir("/tmp", "log_sock")
			if err != nil {
				return
			}
			defer func() {
				if err != nil {
					os.RemoveAll(d)
				}
			}()

			dir := filepath.Dir(name)

			absDir, err := filepath.Abs(dir)
			if err != nil {
				return
			}

			err = os.Symlink(absDir, filepath.Join(d, "d"))
			if err != nil {
				return
			}

			addr = filepath.Join(d, "d", filepath.Base(name))

			cleanup = func() {
				os.RemoveAll(d)
			}
			return
		},
	},
}

const maxSocketNameSize = len(syscall.RawSockaddrUnix{}.Path)

func getSocketAddr(name string, fbs []fallbackDef) (string, func()) {
	maxNameLen := len(syscall.RawSockaddrUnix{}.Path)

	if len(name) < maxNameLen {
		return name, func() {}
	}

	for _, def := range fbs {
		if runtime.GOOS != "linux" && def.linuxOnly {
			continue
		}

		addr, cleanup, err := def.f(name)
		if err != nil {
			continue
		}
		if len(addr) < maxNameLen {
			return addr, cleanup
		}
		cleanup()
	}

	return name, func() {}
}

func dial(name string, fbs []fallbackDef) (net.Conn, error) {
	socket, cleanup := getSocketAddr(name, fbs)
	defer cleanup()

	dialer := &net.Dialer{}
	return dialer.Dial("unix", socket)
}

func listen(name string, fbs []fallbackDef) (net.Listener, error) {
	socket, cleanup := getSocketAddr(name, fbs)
	defer cleanup()

	return net.Listen("unix", socket)
}

func SendLog(logSocket string, entry *LogEntry, done chan interface{}) {
	sendLog(logSocket, entry, done, fallbacks)
}

func sendLog(logSocket string, entry *LogEntry, done chan interface{}, fbs []fallbackDef) {
	defer close(done)

	conn, err := dial(logSocket, fbs)
	if err != nil {
		return
	}
	defer conn.Close()

	enc := gob.NewEncoder(conn)
	enc.Encode(entry)
}

func LogListener(ctx context.Context, logSocket string) (chan *LogEntry, error) {
	return logListener(ctx, logSocket, fallbacks)
}

func logListener(ctx context.Context, logSocket string, fbs []fallbackDef) (chan *LogEntry, error) {
	ret := make(chan *LogEntry, 5)

	if err := os.Remove(logSocket); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ln, err := listen(logSocket, fbs)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				ln.Close()
			}
		}
	}()

	go func() {
		defer close(ret)

		for {
			conn, err := ln.Accept()
			if err != nil {
				ln.Close()
				break
			}
			conn.SetDeadline(time.Now().Add(timeoutDuration))

			go func() {
				defer conn.Close()

				dec := gob.NewDecoder(conn)
				entry := &LogEntry{}
				if err := dec.Decode(entry); err != nil {
					return
				}
				ret <- entry
			}()
		}
	}()
	return ret, nil
}
