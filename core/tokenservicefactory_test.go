/*
Copyright (c) 2017 VMware, Inc. All Rights Reserved.

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
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/vmware/priam/util"
)

func TestTenantInUrlTokenService(t *testing.T) {
	factory := &TokenServiceFactoryImpl{}
	cfg := configFor(TenantInHost)
	cfg.CurrentTarget = "current"
	cfg.Targets = make(map[string]map[string]interface{})
	cfg.Targets[cfg.CurrentTarget] = map[string]interface{}{HostOption: "full-url", HostMode: "tenant-in-host"}

	svc, ok := factory.GetTokenService(cfg, "id", "secret").(TokenService)

	assert.True(t, ok, "should get back a TokenService object")
	assert.Equal(t, "/SAAS", svc.BasePath)
}

func TestTenantInPathTokenService(t *testing.T) {
	factory := &TokenServiceFactoryImpl{}
	cfg := configFor(TenantInPath)

	svc, ok := factory.GetTokenService(cfg, "id", "secret").(TokenService)

	assert.True(t, ok, "should get back a TokenService object")
	assert.Equal(t, "", svc.BasePath)
}

func configFor(mode string) *Config {
	cfg := &Config{}
	cfg.CurrentTarget = "current"
	cfg.Targets = make(map[string]map[string]interface{})
	cfg.Targets[cfg.CurrentTarget] = map[string]interface{}{HostOption: "full-url", HostMode: mode}
	return cfg
}
