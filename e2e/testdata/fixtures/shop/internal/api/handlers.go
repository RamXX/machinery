// Package api exposes the order operations of the fixture project.
package api

import "example.test/shop/internal/domain"

// Handle forwards an event to the domain layer.
func Handle(event string) string {
	return domain.Apply(event)
}
