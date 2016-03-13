package main

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigFileFailure(t *testing.T) {
	fname := filepath.Join(os.TempDir(), "this file does not exist")
	err := getYamlFile(fname, &config{})
	assert.NotNil(t, err)
}

type failingMarshaler struct{}

const yamlMarshalErrorMsg = "YAML Marshal Error"

func (ft *failingMarshaler) MarshalYAML() (interface{}, error) {
	return nil, errors.New(yamlMarshalErrorMsg)
}

func TestYamlMarshalError(t *testing.T) {
	fname := filepath.Join(os.TempDir(), "bad_yaml_test_file")
	_, err := os.Stat(fname)
	assert.True(t, os.IsNotExist(err))
	err = putYamlFile(fname, &failingMarshaler{})
	assert.EqualError(t, err, yamlMarshalErrorMsg)
	_, err = os.Stat(fname)
	assert.True(t, os.IsNotExist(err))
}

func initConfigFileTest(t *testing.T) (assrt *assert.Assertions, cfgFile *os.File, cleanup func(), log *logr) {
	assrt, log = assert.New(t), newBufferedLogr()
	var err error
	cfgFile, err = ioutil.TempFile("", "priam-test-config")
	assrt.Nil(err)
	cleanup = func() {
		cfgFile.Close()
		os.Remove(cfgFile.Name())
	}
	_, err = cfgFile.Write([]byte("---\n"))
	assrt.Nil(err)
	return
}

// creates and inits a config file, removes read privilege,
// calls newAppConfig, tests error
func TestErrorReadingConfigFile(t *testing.T) {
	assert, cfgFile, cleanup, log := initConfigFileTest(t)
	defer cleanup()
	assert.Nil(cfgFile.Chmod(0))
	assert.Nil(newAppConfig(log, cfgFile.Name()))
	assert.Contains(log.errString(), "could not read config file "+cfgFile.Name())
}

func TestErrorWritingConfigFile(t *testing.T) {
	assert, cfgFile, cleanup, log := initConfigFileTest(t)
	defer cleanup()
	cfg := newAppConfig(log, cfgFile.Name())
	assert.NotNil(cfg)
	assert.Nil(cfgFile.Chmod(0))
	assert.False(cfg.save())
	assert.Contains(log.errString(), "could not write config file "+cfgFile.Name())
}
