package core

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

// Returns an error with the given message
func ErrorHandler(status int, message string) func(t *testing.T, req *tstReq) *tstReply {
	return func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{status: status, statusMsg: message}
	}
}

func GoodPathHandler(message string) func(t *testing.T, req *tstReq) *tstReply {
	return func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{output: message}
	}
}

func WriteTempFile(t *testing.T, contents string) *os.File {
	f, err := ioutil.TempFile("", "priam-test-file")
	require.Nil(t, err)
	_, err = f.Write([]byte(contents))
	require.Nil(t, err)
	return f
}

func CleanupTempFile(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}
