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
	. "github.com/vmware/priam/util"
)

// default implementation of the factory
type TokenServiceFactoryImpl struct{}

/* Factory that we can mock to return the right token service */
type TokenServiceFactory interface {
	GetTokenService(cfg *Config, cliClientID string, cliClientSecret string) TokenGrants
}

func (factory TokenServiceFactoryImpl) GetTokenService(cfg *Config, cliClientID string, cliClientSecret string) TokenGrants {
	if cfg.IsTenantInHost() {
		return TokenService{
			BasePath:        "/SAAS",
			AuthorizePath:   "/auth/oauth2/authorize",
			TokenPath:       "/auth/oauthtoken",
			LoginPath:       "/API/1.0/REST/auth/system/login",
			CliClientID:     cliClientID,
			CliClientSecret: cliClientSecret}
	}
	// Note: defining a base yoken service structure to avoid copy/pasting the same values
	// for AuthorizePath, tokenPath, ... did not pass "go vet": "composite literal uses unkeyed fields"
	return TokenService{
		BasePath:        "",
		AuthorizePath:   "/auth/oauth2/authorize",
		TokenPath:       "/auth/oauthtoken",
		LoginPath:       "/API/1.0/REST/auth/system/login",
		CliClientID:     cliClientID,
		CliClientSecret: cliClientSecret}
}
