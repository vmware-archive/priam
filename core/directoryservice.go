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

package core

import (
	"priam/util"
)

// The directory service interface.
// The directory contains different entities (User, Group, Role, ...)
type DirectoryService interface {
	// Add an entity
	AddEntity(ctx *util.HttpContext, entity interface{}) error

	// Display an entity
	DisplayEntity(ctx *util.HttpContext, name string)

	// Update the given entity referenced by the name parameter.
	// Only the fields existing in the given entity will be updated.
	UpdateEntity(ctx *util.HttpContext, name string, entity interface{})

	// Delete the given entity
	DeleteEntity(ctx *util.HttpContext, name string)

	// List existing entities
	// @param count the number of entities to display
	// @param filter the filter such as 'username eq \"joe\"' for SCIM resources
	ListEntities(ctx *util.HttpContext, count int, filter string)

	// Create entities from a file
	LoadEntities(ctx *util.HttpContext, fileName string)

	// Adds or removes a user for entities that have members, like Group or Role
	UpdateMember(ctx *util.HttpContext, name, member string, remove bool)
}
