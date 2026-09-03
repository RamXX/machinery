package main

import (
	"fmt"
	"os"

	"github.com/RamXX/machinery/internal/modelithtx"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: modelith-tx {recover|fingerprint|publish} path [digest]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "recover":
		if len(os.Args) != 3 {
			err = fmt.Errorf("recover requires a repository path")
			break
		}
		err = modelithtx.Recover(os.Args[2])
	case "fingerprint":
		if len(os.Args) != 3 {
			err = fmt.Errorf("fingerprint requires a corpus path")
			break
		}
		var digest string
		digest, err = modelithtx.Fingerprint(os.Args[2])
		if err == nil {
			fmt.Println(digest)
		}
	case "publish":
		if len(os.Args) != 4 {
			err = fmt.Errorf("publish requires a repository path and expected digest")
			break
		}
		err = modelithtx.Publish(os.Args[2], os.Args[3])
	default:
		err = fmt.Errorf("unknown action %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "modelith transaction: %v\n", err)
		os.Exit(1)
	}
}
