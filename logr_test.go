package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

const cradle string = `{
  "wampeter": "ice-nine", "granfalloon": "hoosiers", "karass": "cynics",
  "foma": ["life is good", ["busy","busy","busy"]],
  "sinookas": ["sarcasm", {"rock": "grateful dead"}, {"classical": "vivaldi"}],
  "duffle": {"papa": "manzano", "narrator": "jonah"}
}`

func TestFilteredJsonPrettyPrint(t *testing.T) {
	ppfOutput := `---- cradle ----
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
	filter := []string{"cradle", "duffle", "papa", "karass", "sinookas", "rock", "foma"}
	var jsonObj interface{}
	assert.Nil(t, json.Unmarshal([]byte(cradle), &jsonObj))
	log := newBufferedLogr()
	log.ppf("cradle", jsonObj, filter)
	assert.Equal(t, log.infoString(), ppfOutput)
}
