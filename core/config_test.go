package core

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// creates and inits a config file, removes read privilege,
// calls newAppConfig, tests error
func TestErrorReadingConfigFile(t *testing.T) {
	assert, log, cfgFile := assert.New(t), newBufferedLogr(), WriteTempFile(t, "---\n")
	defer CleanupTempFile(cfgFile)
	require.Nil(t, cfgFile.Chmod(0))
	assert.Nil(newAppConfig(log, cfgFile.Name()))
	assert.Contains(log.errString(), "could not read config file "+cfgFile.Name())
}

func TestErrorWritingConfigFile(t *testing.T) {
	assert, log, cfgFile := assert.New(t), newBufferedLogr(), WriteTempFile(t, "---\n")
	defer CleanupTempFile(cfgFile)
	cfg := newAppConfig(log, cfgFile.Name())
	assert.NotNil(cfg)
	require.Nil(t, cfgFile.Chmod(0))
	assert.False(cfg.save())
	assert.Contains(log.errString(), "could not write config file "+cfgFile.Name())
}
