package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunC4ProcessTruncatesDeterministically(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestC4ProcessHelper", "--", "oversize")
	cmd.Env = append(os.Environ(), "MACHINERY_C4_PROCESS_HELPER=1")
	out, err := runC4Process(context.Background(), cmd, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := fmt.Sprintf("\n[output truncated at %d bytes]\n", verifyC4OutputLimit)
	if !strings.HasSuffix(out, wantSuffix) || len(out) != verifyC4OutputLimit+len(wantSuffix) {
		t.Fatalf("bounded output length/suffix = %d/%q", len(out), out[len(out)-min(len(out), 80):])
	}
}

func TestRunC4ProcessTimesOut(t *testing.T) {
	old := verifyC4Timeout
	verifyC4Timeout = 50 * time.Millisecond
	t.Cleanup(func() { verifyC4Timeout = old })
	ctx, cancel := context.WithTimeout(context.Background(), verifyC4Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestC4ProcessHelper", "--", "sleep")
	cmd.Env = append(os.Environ(), "MACHINERY_C4_PROCESS_HELPER=1")
	_, err := runC4Process(ctx, cmd, verifyC4Timeout)
	if err == nil || !strings.Contains(err.Error(), "process timed out after 50ms") {
		t.Fatalf("timeout was not explicit: %v", err)
	}
}

func TestC4ProcessHelper(t *testing.T) {
	if os.Getenv("MACHINERY_C4_PROCESS_HELPER") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "oversize":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", verifyC4OutputLimit+4096)))
	case "sleep":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}
