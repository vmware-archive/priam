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

package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCaselessEqualsWithoutAString(t *testing.T) {
	assert.False(t, CaselessEqual("axel", 1985), "String should not equal integer")
}

func TestCaselessEquals(t *testing.T) {
	assert.True(t, CaselessEqual("axel", "AxEL"))
}

func TestCaseEqualsWithoutAString(t *testing.T) {
	assert.False(t, CaseEqual("axel", 1985), "String should not equal integer")
}

func TestCaseEqualsWithAString(t *testing.T) {
	assert.False(t, CaseEqual("axel", "AxEL"))
}

func TestCaseEquals(t *testing.T) {
	assert.True(t, CaseEqual("axel", "axel"))
}

func TestToStringFailsWithoutAString(t *testing.T) {
	assert.Equal(t, "", InterfaceToString(1985), "String should not be converted to integer")
}

func TestToString(t *testing.T) {
	assert.Equal(t, "axel", InterfaceToString("axel"))
}
