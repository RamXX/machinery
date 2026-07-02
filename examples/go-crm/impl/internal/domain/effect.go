package domain

import (
	"errors"

	"crm/internal/model"
)

// Effect is the shared machine effect type (defined in the model kernel). It is
// aliased here so the Deal/Task/User Fire signatures read naturally within the
// domain package.
type Effect = model.Effect

// maxRetries is the persist-overlay retry bound (BUILD.md 9: retriesExhausted at
// 3, ~1.5s at a 500ms backoff).
const maxRetries = 3

// errIs reports whether err matches the target sentinel (errors.Is), tolerating
// a nil err.
func errIs(err, target error) bool { return err != nil && errors.Is(err, target) }
