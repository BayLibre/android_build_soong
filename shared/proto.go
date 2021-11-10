package shared

import (
	"io/ioutil"
	"os"

	"google.golang.org/protobuf/proto"
)

// Save takes a protobuf message, marshals to an array of bytes
// and is then saved to a file.
func Save(pb proto.Message, filepath string) (err error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	tempFilepath := filepath + ".tmp"
	if err := ioutil.WriteFile(tempFilepath, []byte(data), 0644 /* rw-r--r-- */); err != nil {
		return err
	}

	if err := os.Rename(tempFilepath, filepath); err != nil {
		return err
	}

	return nil
}
