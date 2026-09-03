//go:build windows

package cachestage

import "os"

// Windows exposes synthesized POSIX bits; confinement comes from real-entry
// validation and the user's native cache ACL.
func privateStageMode(os.FileMode) bool { return true }
