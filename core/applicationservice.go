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

// The application service interface.
type ApplicationService interface {
	// Display display the given application defined by its name
	Display(ctx *util.HttpContext, name string)

	// Delete deletes the given application defined by its name
	Delete(ctx *util.HttpContext, name string)

	// List lists all applications in the catalog
	// @param count the number of applications to display
	// @param filter the filter
	List(ctx *util.HttpContext, count int, filter string)

	// Publish publishes the application defined by the manifestFile into VMware IDM catalog
	Publish(ctx *util.HttpContext, manifestFile string)
}
