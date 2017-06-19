package main

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"android/soong/third_party/zip"
)

var testCases = []struct {
	name string

	inputFiles []string
	sorted     bool
	args       []string

	outputFiles []string
	err         error
}{
	{
		name: "arg error",

		args: []string{"a\\b:b"},

		err: fmt.Errorf("\\ characters are not currently supported"),
	},
	{ // This is modelled after the update package generation in our build
		name: "filter globs",

		inputFiles: []string{
			"RADIO/a",
			"IMAGES/system.img",
			"IMAGES/b.txt",
			"IMAGES/recovery.img",
			"IMAGES/vendor.img",
			"OTA/android-info.txt",
			"OTA/b",
		},
		args: []string{"OTA/android-info.txt:android-info.txt", "IMAGES/*.img:."},

		outputFiles: []string{
			"android-info.txt",
			"system.img",
			"recovery.img",
			"vendor.img",
		},
	},
	{
		name: "sorted filter globs",

		inputFiles: []string{
			"RADIO/a",
			"IMAGES/system.img",
			"IMAGES/b.txt",
			"IMAGES/recovery.img",
			"IMAGES/vendor.img",
			"OTA/android-info.txt",
			"OTA/b",
		},
		sorted: true,
		args:   []string{"IMAGES/*.img:.", "OTA/android-info.txt:android-info.txt"},

		outputFiles: []string{
			"recovery.img",
			"system.img",
			"vendor.img",
			"android-info.txt",
		},
	},
	{
		name: "sort all",

		inputFiles: []string{
			"RADIO/a",
			"IMAGES/system.img",
			"IMAGES/b.txt",
			"IMAGES/recovery.img",
			"IMAGES/vendor.img",
			"OTA/b",
			"OTA/android-info.txt",
		},
		sorted: true,
		args:   []string{"**"},

		outputFiles: []string{
			"IMAGES/b.txt",
			"IMAGES/recovery.img",
			"IMAGES/system.img",
			"IMAGES/vendor.img",
			"OTA/android-info.txt",
			"OTA/b",
			"RADIO/a",
		},
	},
}

func errorString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

func TestZip2Zip(t *testing.T) {
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			inputBuf := &bytes.Buffer{}
			outputBuf := &bytes.Buffer{}

			inputWriter := zip.NewWriter(inputBuf)
			for _, file := range testCase.inputFiles {
				w, err := inputWriter.Create(file)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintln(w, "test")
			}
			inputWriter.Close()
			inputBytes := inputBuf.Bytes()
			inputReader, err := zip.NewReader(bytes.NewReader(inputBytes), int64(len(inputBytes)))
			if err != nil {
				t.Fatal(err)
			}

			outputWriter := zip.NewWriter(outputBuf)
			err = zip2zip(inputReader, outputWriter, testCase.sorted, testCase.args)
			if errorString(testCase.err) != errorString(err) {
				t.Fatalf("Unexpected error:\n got: %q\nwant: %q", errorString(err), errorString(testCase.err))
			}

			outputWriter.Close()
			outputBytes := outputBuf.Bytes()
			outputReader, err := zip.NewReader(bytes.NewReader(outputBytes), int64(len(outputBytes)))
			if err != nil {
				t.Fatal(err)
			}
			var outputFiles []string
			if len(outputReader.File) > 0 {
				outputFiles = make([]string, len(outputReader.File))
				for i, file := range outputReader.File {
					outputFiles[i] = file.Name
				}
			}

			if !reflect.DeepEqual(testCase.outputFiles, outputFiles) {
				t.Fatalf("Output file list does not match:\n got: %v\nwant: %v", outputFiles, testCase.outputFiles)
			}
		})
	}
}
