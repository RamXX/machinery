//go:build !windows

package cachestage

import "os"

func privateStageMode(mode os.FileMode) bool { return mode.Perm()&0o077 == 0 }
