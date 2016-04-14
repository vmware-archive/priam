/*
Copyright (c) 2016 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
