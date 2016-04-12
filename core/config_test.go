package core

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

const testAppCfg = `---
currenttarget: familyCountDown
targets:
  familyCountDown:
    host: https://space.odyssey.example.com
  1:
    host: https://venus.example.com
  staging:
    host: https://earth.example.com
`

func cfgTestSetup(t *testing.T, cfg string) *config {
	cfgFile := WriteTempFile(t, stringOrDefault(cfg, testAppCfg))
	defer cfgFile.Close()
	return newAppConfig(newBufferedLogr(), cfgFile.Name())
}

func TestTargetDeleteByName(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.deleteTarget("1", "")
	assert.Contains(t, cfg.log.infoString(), "deleted target 1")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "https://venus.example.com")
}

func TestTargetDeleteByNameAndURL(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.deleteTarget("venus.example.com", "1")
	assert.Contains(t, cfg.log.infoString(), "deleted target 1")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "https://venus.example.com")
}

func TestTargetDeleteAll(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.clear()
	assert.Contains(t, cfg.log.infoString(), "all targets deleted")
	assert.Equal(t, "currenttarget: \"\"\ntargets: {}\n", GetTempFile(t, cfg.fileName))
}

func TestTargetDeleteByURL(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.deleteTarget("space.odyssey.example.com", "")
	assert.Contains(t, cfg.log.infoString(), "deleted target familyCountDown")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "space.odyssey.example.com")
}

func TestTargetDeleteCurrent(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.deleteTarget("", "")
	assert.Contains(t, cfg.log.infoString(), "deleted target familyCountDown")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "space.odyssey.example.com")
}

func TestTargetDeleteSpecific(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.deleteTarget("staging", "")
	assert.Contains(t, cfg.log.infoString(), "deleted target staging")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "earth.example.com")
}

func TestTargetDeleteNotFound(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.deleteTarget("sven", "")
	assert.Contains(t, cfg.log.infoString(), "nothing deleted, no such target found")
}

func TestTargetDeleteCurrentNotSet(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.CurrentTarget = ""
	cfg.deleteTarget("", "")
	assert.Contains(t, cfg.log.infoString(), "nothing deleted, no target set")
}

func TestTarget(t *testing.T) {
	cfg := cfgTestSetup(t, "")
	defer os.Remove(cfg.fileName)
	cfg.setTarget("", "", nil)
	assert.Contains(t, "current target is: familyCountDown, https://space.odyssey.example.com\n", cfg.log.infoString())
}

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

func TestTargetCheckURLFails(t *testing.T) {

}