package core

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"github.com/stretchr/testify/assert"
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

func GetTempFile(t *testing.T, fileName string) string {
	contents, err := ioutil.ReadFile(fileName)
	require.Nil(t, err)
	return string(contents)
}

func TestCaselessEqualsWithoutAString(t *testing.T) {
	assert.False(t, caselessEqual("axel", 1985), "String should not equal integer")
}

func TestCaselessEquals(t *testing.T) {
	assert.True(t, caselessEqual("axel", "AxEL"))
}

func TestCaseEqualsWithoutAString(t *testing.T) {
	assert.False(t, caseEqual("axel", 1985), "String should not equal integer")
}

func TestCaseEqualsWithAString(t *testing.T) {
	assert.False(t, caseEqual("axel", "AxEL"))
}

func TestCaseEquals(t *testing.T) {
	assert.True(t, caseEqual("axel", "axel"))
}

func TestToStringFailsWithoutAString(t *testing.T) {
	assert.Equal(t, "", interfaceToString(1985), "String should not be converted to integer")
}

func TestToString(t *testing.T) {
	assert.Equal(t, "axel", interfaceToString("axel"))
}
