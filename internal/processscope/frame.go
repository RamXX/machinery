package processscope

import (
	"encoding/json"
	"net"
	"os"
)

type frameReader struct {
	c       *net.UnixConn
	buf     []byte
	pending []*os.File
}

func newFrameReader(c *net.UnixConn) *frameReader {
	return &frameReader{c: c}
}

func selfDigest() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return fileDigest(exe)
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"t":"gstartfail","code":"INTERNAL_ERROR","message":"marshal"}`)
	}
	return b
}
