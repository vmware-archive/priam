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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/vmware/priam/testaid"
)

func cfgTestSetup(t *testing.T) *Config {
	const testAppCfg = `---
currenttarget: familyCountDown
targets:
  familyCountDown:
    host: https://space.odyssey.example.com
  1:
    host: https://venus.example.com
    mode: tenant-in-host
  staging:
    host: https://earth.example.com
    mode: tenant-in-path
  beautyOnTheBeach:
    host: https://disney.princess.com
    mode: not-right
`
	cfgFile, cfg := WriteTempFile(t, testAppCfg), &Config{}
	defer cfgFile.Close()
	require.True(t, cfg.Init(NewBufferedLogr(), cfgFile.Name()))
	return cfg
}

func TestTargetDeleteByName(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("1", "")
	assert.Contains(t, cfg.Log.InfoString(), "deleted target 1")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "https://venus.example.com")
}

func TestTargetDeleteByNameAndURL(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("venus.example.com", "1")
	assert.Contains(t, cfg.Log.InfoString(), "deleted target 1")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "https://venus.example.com")
}

func TestTargetDeleteAll(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.Clear()
	assert.Contains(t, cfg.Log.InfoString(), "all targets deleted")
	assert.Equal(t, "currenttarget: \"\"\ntargets: {}\n", GetTempFile(t, cfg.fileName))
}

func TestTargetDeleteByURL(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("space.odyssey.example.com", "")
	assert.Contains(t, cfg.Log.InfoString(), "deleted target familyCountDown")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "space.odyssey.example.com")
}

func TestTargetDeleteCurrent(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("", "")
	assert.Contains(t, cfg.Log.InfoString(), "deleted target familyCountDown")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "space.odyssey.example.com")
}

func TestTargetDeleteSpecificCurrent(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("familyCountDown", "")
	assert.Contains(t, cfg.Log.InfoString(), "deleted target familyCountDown")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "space.odyssey.example.com")

	// ensure current is not set or chosen
	assert.True(t, cfg.Init(NewBufferedLogr(), cfg.fileName))
	cfg.PrintTarget("current")
	assert.Equal(t, "no target set\n", cfg.Log.InfoString())
}

func TestTargetDeleteSpecific(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("staging", "")
	assert.Contains(t, cfg.Log.InfoString(), "deleted target staging")
	assert.NotContains(t, GetTempFile(t, cfg.fileName), "earth.example.com")
}

func TestTargetDeleteNotFound(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.DeleteTarget("sven", "")
	assert.Contains(t, cfg.Log.InfoString(), "nothing deleted, no such target found")
}

func TestTargetDeleteCurrentNotSet(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.CurrentTarget = ""
	cfg.DeleteTarget("", "")
	assert.Contains(t, cfg.Log.InfoString(), "nothing deleted, no target set")
}

func TestTarget(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.SetTarget("", "", false, nil)
	assert.Contains(t, "current target is: familyCountDown, https://space.odyssey.example.com\n", cfg.Log.InfoString())
}

func TestGetConfigFileFailure(t *testing.T) {
	fname := filepath.Join(os.TempDir(), "this file does not exist")
	err := GetYamlFile(fname, &Config{})
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
	err = PutYamlFile(fname, &failingMarshaler{})
	assert.EqualError(t, err, yamlMarshalErrorMsg)
	_, err = os.Stat(fname)
	assert.True(t, os.IsNotExist(err))
}

// creates and inits a config file, removes read privilege,
// calls newAppConfig, tests error
func TestErrorReadingConfigFile(t *testing.T) {
	assert, log, cfgFile, cfg := assert.New(t), NewBufferedLogr(), WriteTempFile(t, "---\n"), &Config{}
	defer CleanupTempFile(cfgFile)
	require.Nil(t, cfgFile.Chmod(0))
	assert.False(cfg.Init(log, cfgFile.Name()))
	assert.Contains(log.ErrString(), "could not read config file "+cfgFile.Name())
}

func TestErrorWritingConfigFile(t *testing.T) {
	assert, log, cfgFile, cfg := assert.New(t), NewBufferedLogr(), WriteTempFile(t, "---\n"), &Config{}
	defer CleanupTempFile(cfgFile)
	assert.True(cfg.Init(log, cfgFile.Name()))
	require.Nil(t, cfgFile.Chmod(0))
	assert.False(cfg.Save())
	assert.Contains(log.ErrString(), "could not write config file "+cfgFile.Name())
}

func TestDefaultHostModeIsTenantInHost(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.SetTarget("https://space.odyssey.example.com", "familyCountDown", false, nil)
	assert.True(t, cfg.IsTenantInHost(), "default host mode should be tenant in host")
}

func TestGetHostModeForTenantInPathFromConfig(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.SetTarget("https://earth.example.com", "staging", false, nil)
	assert.Contains(t, cfg.Log.InfoString(), "new target is: staging")
	assert.False(t, cfg.IsTenantInHost(), "host mode should be tenant in path")
}

func TestGetHostModeForTenantInHost(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.SetTarget("https://venus.example.com", "1", false, nil)
	assert.Equal(t, cfg.Targets[cfg.CurrentTarget][HostMode], "tenant-in-host")
	assert.True(t, cfg.IsTenantInHost(), "host mode should be tenant in host")
}

func TestUnknownModeLeadsToTenantInHost(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.SetTarget("https://disney.princess.com", "beautyOnTheBeach", false, nil)
	assert.True(t, cfg.IsTenantInHost(), "host mode should be tenant in host")
}

func TestCanDetectTenantInPathMode(t *testing.T) {
	cfg := cfgTestSetup(t)
	defer os.Remove(cfg.fileName)
	cfg.SetTarget("https://hello.me.com/SAAS/t/foo", "", false, nil)
	assert.Contains(t, cfg.Log.InfoString(), "Mode detected: tenant-in-path")
	assert.False(t, cfg.IsTenantInHost(), "host mode should be tenant in path")
}
