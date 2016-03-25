package core

// The directory service interface.
// The directory contains different entities (User, Group, Role, ...)
type DirectoryService interface {
	// Add an entity
	AddEntity(ctx *HttpContext, entity interface{}) error

	// Display an entity
	DisplayEntity(ctx *HttpContext, name string)

	// Delete the given entity
	DeleteEntity(ctx *HttpContext, name string)

	// List existing entities
	// @param count the number of entities to display
	// @param filter the filter such as 'username eq \"joe\"' for SCIM resources
	ListEntities(ctx *HttpContext, count int, filter string)

	// Create entities from a file
	LoadEntities(ctx *HttpContext, fileName string)
}
