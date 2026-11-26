// Command receiptprog emits receipts with pkg/receipt from its arguments. It
// stands in for a Go end-to-end suite in the scripts.
//
//	receiptprog enabled
//	receiptprog only
//	receiptprog wanted <id>
//	receiptprog pass <id> <observed> [<effect>...]
//	receiptprog fail <id> <observed>
//	receiptprog skip <id> <observed>
//	receiptprog replay <id> <reproduced|refuted> <observed>
//	receiptprog outcome <id> <outcome>
//	receiptprog notest <id>
//	receiptprog many <id> <n>
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/RamXX/machinery/pkg/receipt"
)

const test = "receiptprog"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: receiptprog <command> [args...]")
	}
	a := args[1:]
	need := func(n int) error {
		if len(a) < n {
			return fmt.Errorf("receiptprog %s needs %d arguments", args[0], n)
		}
		return nil
	}
	switch args[0] {
	case "enabled":
		fmt.Printf("enabled=%t\n", receipt.Enabled())
		return nil
	case "only":
		fmt.Printf("only=%s\n", strings.Join(receipt.Only(), ","))
		return nil
	case "wanted":
		if err := need(1); err != nil {
			return err
		}
		fmt.Printf("wanted %s: %t\n", a[0], receipt.Wanted(a[0]))
		return nil
	case "pass":
		if err := need(2); err != nil {
			return err
		}
		return receipt.Pass(a[0], test, a[1], a[2:]...)
	case "fail":
		if err := need(2); err != nil {
			return err
		}
		return receipt.Fail(a[0], test, a[1])
	case "skip":
		if err := need(2); err != nil {
			return err
		}
		return receipt.Skip(a[0], test, a[1])
	case "replay":
		if err := need(3); err != nil {
			return err
		}
		return receipt.Replay(a[0], test, a[1], a[2])
	case "outcome":
		if err := need(2); err != nil {
			return err
		}
		return receipt.Emit(receipt.Receipt{ID: a[0], Outcome: a[1], Test: test, Observed: "custom outcome"})
	case "notest":
		if err := need(1); err != nil {
			return err
		}
		return receipt.Emit(receipt.Receipt{ID: a[0], Outcome: receipt.OutcomePass, Observed: "no test named"})
	case "many":
		if err := need(2); err != nil {
			return err
		}
		n, err := strconv.Atoi(a[1])
		if err != nil {
			return err
		}
		var wg sync.WaitGroup
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errs <- receipt.Pass(a[0], fmt.Sprintf("%s/%d", test, i), strings.Repeat("y", 250))
			}(i)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("receiptprog: unknown command %q", args[0])
}
