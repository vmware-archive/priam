package core

// The application service interface.
type ApplicationService interface {
	// Display display the given application defined by its name
	Display(ctx *HttpContext, name string)

	// Delete deletes the given application defined by its name
	Delete(ctx *HttpContext, name string)

	// List lists all applications in the catalog
	// @param count the number of applications to display
	// @param filter the filter
	List(ctx *HttpContext, count int, filter string)

	// Publish publishes the application defined by the manifestFile into VMware IDM catalog
	Publish(ctx *HttpContext, manifestFile string)
}
