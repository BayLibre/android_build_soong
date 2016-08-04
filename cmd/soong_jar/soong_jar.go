// Copyright 2015 Google Inc. All rights reserved.
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
	"bytes"
	"compress/flate"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"android/zip"
)

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}

type fileArg struct {
	relativeRoot, file string
}

type fileArgs []fileArg

func (l *fileArgs) String() string {
	return `""`
}

func (l *fileArgs) Set(s string) error {
	if *relativeRoot == "" {
		return fmt.Errorf("must pass -C before -f")
	}

	*l = append(*l, fileArg{filepath.Clean(*relativeRoot), s})
	return nil
}

func (l *fileArgs) Get() interface{} {
	return l
}

var (
	out          = flag.String("o", "", "file to write jar file to")
	manifest     = flag.String("m", "", "input manifest file name")
	directories  = flag.Bool("d", false, "include directories in jar")
	relativeRoot = flag.String("C", "", "path to use as relative root of files in next -f or -l argument")
	listFiles    fileArgs
	files        fileArgs
	// TODO(dwillemsen): flag for compression level
)

func init() {
	flag.Var(&listFiles, "l", "file containing list of .class files")
	flag.Var(&files, "f", "file to include in jar")
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: soong_jar -o jarfile [-m manifest] -C dir [-f|-l file]...\n")
	flag.PrintDefaults()
	os.Exit(2)
}

type zipWriter struct {
	time        time.Time
	createdDirs map[string]bool
	directories bool

	errors   chan error
	writeOps chan chan *zipEntry

	rateLimit *RateLimit
}

type zipEntry struct {
	fh *zip.FileHeader
	r  io.ReadCloser
}

func main() {
	flag.Parse()

	if *out == "" {
		fmt.Fprintf(os.Stderr, "error: -o is required\n")
		usage()
	}

	w := &zipWriter{
		time:        time.Date(2009, 1, 1, 0, 0, 0, 0, time.UTC),
		createdDirs: make(map[string]bool),
		directories: *directories,
	}

	err := w.write(*out, listFiles, *manifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func (z *zipWriter) write(out string, listFiles fileArgs, manifest string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}

	defer f.Close()
	defer func() {
		if err != nil {
			os.Remove(out)
		}
	}()

	z.errors = make(chan error)
	defer close(z.errors)
	// This channel size can be essentially unlimited -- it's just used
	// as a fifo queue. The actual rate limit is handled by the RateLimit
	z.writeOps = make(chan chan *zipEntry, 10000)

	z.rateLimit = NewRateLimit(0)
	defer z.rateLimit.Stop()

	go func() {
		defer close(z.writeOps)

		for _, listFile := range listFiles {
			err = z.writeListFile(listFile)
			if err != nil {
				z.errors <- err
				return
			}
		}

		for _, file := range files {
			err = z.writeRelFile(file.relativeRoot, file.file)
			if err != nil {
				z.errors <- err
				return
			}
		}

		if manifest != "" {
			err = z.writeFile("META-INF/MANIFEST.MF", manifest)
			if err != nil {
				z.errors <- err
				return
			}
		}
	}()

	zipw := zip.NewWriter(f)
	defer zipw.Close()

loop:
	for {
		select {
		case writeOp, ok := <-z.writeOps:
			if !ok {
				break loop
			}

			select {
			case op := <-writeOp:
				var out io.WriteCloser
				if op.fh.Method == zip.Deflate {
					out, err = zipw.CreateCompressedHeader(op.fh)
				} else {
					var zw io.Writer
					zw, err = zipw.CreateHeader(op.fh)
					out = nopCloser{zw}
				}
				if err != nil {
					return err
				}

				if op.r != nil {
					_, err = io.Copy(out, op.r)
					op.r.Close()
					if err != nil {
						return err
					}
				}

				out.Close()
			case err = <-z.errors:
				return err
			}
		case err = <-z.errors:
			return err
		}
	}

	select {
	case err = <-z.errors:
		return err
	default:
		return nil
	}
}

func (z *zipWriter) writeListFile(listFile fileArg) error {
	list, err := ioutil.ReadFile(listFile.file)
	if err != nil {
		return err
	}

	files := strings.Split(string(list), "\n")

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		err = z.writeRelFile(listFile.relativeRoot, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (z *zipWriter) writeRelFile(root, file string) error {
	file = filepath.Clean(file)

	rel, err := filepath.Rel(root, file)
	if err != nil {
		return err
	}

	err = z.writeFile(rel, file)
	if err != nil {
		return err
	}

	return nil
}

func (z *zipWriter) writeFile(rel, file string) error {
	if s, err := os.Lstat(file); err != nil {
		return err
	} else if s.IsDir() {
		if z.directories {
			return z.writeDirectory(rel)
		}
		return nil
	} else if s.Mode()&os.ModeSymlink != 0 {
		return z.writeSymlink(rel, file)
	} else if !s.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file, directory, or symlink", file)
	}

	if z.directories {
		dir, _ := filepath.Split(rel)
		err := z.writeDirectory(dir)
		if err != nil {
			return err
		}
	}

	fileHeader := &zip.FileHeader{
		Name:   rel,
		Method: zip.Deflate,
	}
	fileHeader.SetModTime(z.time)

	compressChan := make(chan *zipEntry, 1)
	z.writeOps <- compressChan

	request := z.rateLimit.RequestExecution()

	go z.compressFile(fileHeader, file, request, compressChan)

	return nil
}

var compressorPool sync.Pool

func getCompressor(w io.Writer) (*flate.Writer, error) {
	fw, ok := compressorPool.Get().(*flate.Writer)
	if ok {
		fw.Reset(w)
		return fw, nil
	} else {
		return flate.NewWriter(w, 6)
	}
}

func (z *zipWriter) compressFile(fh *zip.FileHeader, file string, request ExecutionRequest, compressChan chan *zipEntry) {
	r, err := os.Open(file)
	if err != nil {
		z.errors <- err
		return
	}

	exec := request.Wait()
	defer exec.Finish()

	crc := crc32.NewIEEE()
	count, err := io.Copy(crc, r)
	if err != nil {
		r.Close()
		z.errors <- err
		return
	}

	fh.CRC32 = crc.Sum32()
	fh.UncompressedSize64 = uint64(count)

	_, err = r.Seek(0, 0)
	if err != nil {
		r.Close()
		z.errors <- err
		return
	}

	buf := new(bytes.Buffer)
	fw, err := getCompressor(buf)
	if err != nil {
		r.Close()
		z.errors <- err
		return
	}

	_, err = io.Copy(fw, r)
	if err != nil {
		r.Close()
		z.errors <- err
		return
	}
	fw.Close()
	compressorPool.Put(fw)

	ze := &zipEntry{fh: fh}

	if uint64(buf.Len()) < fh.UncompressedSize64 {
		ze.r = ioutil.NopCloser(buf)
		r.Close()
	} else {
		_, err = r.Seek(0, 0)
		if err != nil {
			z.errors <- err
		}

		ze.fh.Method = zip.Store
		ze.r = r
	}

	compressChan <- ze
}

func (z *zipWriter) writeDirectory(dir string) error {
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}

	for dir != "" && dir != "./" && !z.createdDirs[dir] {
		z.createdDirs[dir] = true

		dirHeader := &zip.FileHeader{
			Name: dir,
		}
		dirHeader.SetMode(0700 | os.ModeDir)
		dirHeader.SetModTime(z.time)

		ze := make(chan *zipEntry, 1)
		ze <- &zipEntry{
			fh: dirHeader,
		}
		z.writeOps <- ze

		dir, _ = filepath.Split(dir)
	}

	return nil
}

func (z *zipWriter) writeSymlink(rel, file string) error {
	if z.directories {
		dir, _ := filepath.Split(rel)
		if err := z.writeDirectory(dir); err != nil {
			return err
		}
	}

	fileHeader := &zip.FileHeader{
		Name: rel,
	}
	fileHeader.SetModTime(z.time)
	fileHeader.SetMode(0700 | os.ModeSymlink)

	dest, err := os.Readlink(file)
	if err != nil {
		return err
	}

	readerChan := make(chan io.ReadCloser, 1)
	readerChan <- ioutil.NopCloser(bytes.NewBufferString(dest))

	ze := make(chan *zipEntry, 1)
	ze <- &zipEntry{
		fh: fileHeader,
		r:  ioutil.NopCloser(bytes.NewBufferString(dest)),
	}
	z.writeOps <- ze

	return nil
}
