// Package portablepath defines a conservative filename grammar shared by
// Linux, default macOS filesystems, and Windows/NTFS.
package portablepath

import (
	"fmt"
	"path"
	"strings"
)

var reserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true, "CLOCK$": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// ValidateBase accepts only ASCII letters, digits, dot, underscore, and
// hyphen, excluding Windows device names and dot/space aliases.
func ValidateBase(name string) error {
	if name == "" || name == "." || name == ".." || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("%q is not a portable filename", name)
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '_' && r != '-' {
			return fmt.Errorf("%q is not a portable ASCII filename", name)
		}
	}
	stem, _, _ := strings.Cut(name, ".")
	if reserved[strings.ToUpper(stem)] {
		return fmt.Errorf("%q uses a Windows-reserved device basename", name)
	}
	return nil
}

// ValidateRelative validates a slash-separated relative path independent of
// the host OS. Backslashes and drive/UNC syntax are always rejected.
func ValidateRelative(rel string) error {
	if rel == "" || strings.Contains(rel, "\\") || strings.HasPrefix(rel, "/") || (len(rel) >= 2 && rel[1] == ':') {
		return fmt.Errorf("%q is not a portable relative path", rel)
	}
	if clean := path.Clean(rel); clean != rel || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%q is not a clean portable relative path", rel)
	}
	for _, part := range strings.Split(rel, "/") {
		if err := ValidateBase(part); err != nil {
			return err
		}
	}
	return nil
}
