package shared

import (
	"io/ioutil"
	"os"

	"google.golang.org/protobuf/proto"
)

// Save takes a protobuf message, marshals to an array of bytes
// and is then saved to a file.
func Save(pb proto.Message, filename string) (err error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	tempFilename := "." + filename + ".tmp"
	if err := ioutil.WriteFile(tempFilename, []byte(data), 0644 /* rw-r--r-- */); err != nil {
		return err
	}

	if err := os.Rename(tempFilename, filename); err != nil {
		return err
	}

	return nil
}

