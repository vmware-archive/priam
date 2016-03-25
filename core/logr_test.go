package core

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
	log := newBufferedLogr()
	log.ppf("cradle", jsonObj, ppfFilter)
	assert.Equal(t, log.infoString(), filteredPpfOutput)
}

func TestFilteredJsonArrayPrettyPrint(t *testing.T) {
	expected := `---- names ----
- mona: monzano
  asa: breed
  bokonon: 
`
	var jsonObj interface{}
	assert.Nil(t, json.Unmarshal([]byte(`[{"mona": "monzano","asa":"breed","bokonon": ""}]`), &jsonObj))
	log := newBufferedLogr()
	log.ppf("names", jsonObj, []string{"mona", "asa", "bokonon"})
	assert.Equal(t, expected, log.infoString())
}

func TestVerboseFilteredJsonPrettyPrint(t *testing.T) {
	var jsonObj interface{}
	assert.Nil(t, json.Unmarshal([]byte(cradle), &jsonObj))
	log := newBufferedLogr()
	log.verboseOn = true
	log.ppf("cradle", jsonObj, ppfFilter)
	assert.Equal(t, log.infoString(), unfilteredPpfOutput)
}

func TestLogDebug(t *testing.T) {
	log := newBufferedLogr()
	log.debugOn = true
	log.debug("test1")
	assert.Contains(t, log.infoString(), "test1")
	log.debugOn = false
	log.debug("test2")
	assert.NotContains(t, log.infoString(), "test2")
}

func TestLogTrace(t *testing.T) {
	log := newBufferedLogr()
	log.traceOn = true
	log.trace("test1")
	assert.Contains(t, log.infoString(), "test1")
	log.traceOn = false
	log.trace("test2")
	assert.NotContains(t, log.infoString(), "test2")
}

func TestToStringJsonStyle(t *testing.T) {
	kazakJson := `{
  "malachi": "constant",
  "winston": "rumfoord"
}`
	kazak := map[string]string{"malachi": "constant", "winston": "rumfoord"}
	assert.Equal(t, kazakJson, toStringWithStyle(ljson, kazak))
}

type failingYamlMarshaler struct{}

const failingYamlMarshalMsg = "YAML Marshal Error"

func (ft *failingYamlMarshaler) MarshalYAML() (interface{}, error) {
	return nil, errors.New(failingYamlMarshalMsg)
}

func TestStringStyleError(t *testing.T) {
	assert.Equal(t, "&{}", toStringWithStyle(lyaml, &failingYamlMarshaler{}))
}
