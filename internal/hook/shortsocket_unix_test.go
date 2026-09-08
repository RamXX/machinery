//go:build !windows

package hook

// shortSocketBase is a base directory short enough for a Unix-domain socket
// address. The suite's own temporary root is deliberately deep and private,
// which is right for state and wrong for a 104-byte socket path.
const shortSocketBase = "/tmp"
