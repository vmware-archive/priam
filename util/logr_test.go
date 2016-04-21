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
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

const cradle string = `{
  "wampeter": "ice-nine", "granfalloon": "hoosiers", "karass": "cynics",
  "foma": ["life is good", ["busy","busy","busy"]],
  "sinookas": ["sarcasm", {"rock": "grateful dead"}, {"classical": "vivaldi"}],
  "duffle": {"papa": "manzano", "narrator": "jonah"}
}`

var ppfFilter = []string{"duffle", "papa", "karass", "sinookas", "rock", "foma"}

const filteredPpfOutput string = `---- cradle ----
duffle: 
  papa: manzano
karass: cynics
sinookas: 
  - sarcasm
  - rock: grateful dead
foma: 
  - life is good
  - - busy
    - busy
    - busy
`
const unfilteredPpfOutput string = `---- cradle ----
duffle:
  narrator: jonah
  papa: manzano
foma:
- life is good
- - busy
  - busy
  - busy
granfalloon: hoosiers
karass: cynics
sinookas:
- sarcasm
- rock: grateful dead
- classical: vivaldi
wampeter: ice-nine
`

func TestFilteredJsonPrettyPrint(t *testing.T) {
	var jsonObj interface{}
	assert.Nil(t, json.Unmarshal([]byte(cradle), &jsonObj))
	log := NewBufferedLogr()
	log.PPF("cradle", jsonObj, ppfFilter...)
	assert.Equal(t, log.InfoString(), filteredPpfOutput)
}

func TestFilteredJsonArrayPrettyPrint(t *testing.T) {
	expected := `---- names ----
- mona: monzano
  asa: breed
  bokonon: 
`
	var jsonObj interface{}
	assert.Nil(t, json.Unmarshal([]byte(`[{"mona": "monzano","asa":"breed","bokonon": ""}]`), &jsonObj))
	log := NewBufferedLogr()
	log.PPF("names", jsonObj, "mona", "asa", "bokonon")
	assert.Equal(t, expected, log.InfoString())
}

func TestVerboseFilteredJsonPrettyPrint(t *testing.T) {
	var jsonObj interface{}
	assert.Nil(t, json.Unmarshal([]byte(cradle), &jsonObj))
	log := NewBufferedLogr()
	log.VerboseOn = true
	log.PPF("cradle", jsonObj, ppfFilter...)
	assert.Equal(t, log.InfoString(), unfilteredPpfOutput)
}

func TestLogDebug(t *testing.T) {
	log := NewBufferedLogr()
	log.DebugOn = true
	log.Debug("test1")
	assert.Contains(t, log.InfoString(), "test1")
	log.DebugOn = false
	log.Debug("test2")
	assert.NotContains(t, log.InfoString(), "test2")
}

func TestLogTrace(t *testing.T) {
	log := NewBufferedLogr()
	log.TraceOn = true
	log.Trace("test1")
	assert.Contains(t, log.InfoString(), "test1")
	log.TraceOn = false
	log.Trace("test2")
	assert.NotContains(t, log.InfoString(), "test2")
}

func TestToStringJsonStyle(t *testing.T) {
	kazakJson := `{
  "malachi": "constant",
  "winston": "rumfoord"
}`
	kazak := map[string]string{"malachi": "constant", "winston": "rumfoord"}
	assert.Equal(t, kazakJson, ToStringWithStyle(LJson, kazak))
}

type failingYamlMarshaler struct{}

const failingYamlMarshalMsg = "YAML Marshal Error"

func (ft *failingYamlMarshaler) MarshalYAML() (interface{}, error) {
	return nil, errors.New(failingYamlMarshalMsg)
}

func TestStringStyleError(t *testing.T) {
	assert.Equal(t, "&{}", ToStringWithStyle(LYaml, &failingYamlMarshaler{}))
}
