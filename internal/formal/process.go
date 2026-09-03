package formal

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/runtimeclosure"
)

const formalOutputLimit = 1 << 20

const formalJavaProbeTimeout = 10 * time.Second

var formalAfterJavaProbe = func(string) {}

type boundedOutput struct {
	data      []byte
	limit     int
	truncated bool
}

func redactPrivatePath(value, privateRoot, logical string) string {
	value = strings.ReplaceAll(value, filepath.Clean(privateRoot), logical)
	return strings.ReplaceAll(value, filepath.ToSlash(filepath.Clean(privateRoot)), logical)
}

func redactPrivateError(err error, privateRoot, logical string) error {
	if err == nil {
		return nil
	}
	return errors.New(redactPrivatePath(err.Error(), privateRoot, logical))
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	overflow := len(b.data)+len(p) > b.limit
	if room := b.limit - len(b.data); room > 0 {
		take := len(p)
		if take > room {
			take = room
		}
		b.data = append(b.data, p[:take]...)
	}
	if overflow {
		b.truncated = true
	}
	return len(p), nil
}

func runBoundedProcess(ctx context.Context, cmd *exec.Cmd, timeout time.Duration) (string, error) {
	buf := &boundedOutput{limit: formalOutputLimit}
	cmd.Stdout, cmd.Stderr = buf, buf
	err := processcontrol.Run(ctx, cmd)
	out := string(buf.data)
	if buf.truncated {
		out += fmt.Sprintf("\n[output truncated at %d bytes]\n", formalOutputLimit)
		err = errors.Join(err, fmt.Errorf("process combined output exceeded %d-byte limit", formalOutputLimit))
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = errors.Join(err, fmt.Errorf("process timed out after %s", timeout))
	}
	return out, err
}

func openFormalJava(workdir string) (*runtimeclosure.Java, error) {
	java, err := runtimeclosure.OpenJava()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), formalJavaProbeTimeout)
	cmd := exec.CommandContext(ctx, java.Path(), "-XshowSettings:properties", "-version")
	cmd.Env = runtimeclosure.Environment(workdir, workdir, java.Path())
	out, probeErr := runBoundedProcess(ctx, cmd, formalJavaProbeTimeout)
	cancel()
	identityErr := java.BindIdentity(out)
	validateErr := java.Validate()
	if err := errors.Join(probeErr, identityErr, validateErr); err != nil {
		return nil, errors.Join(fmt.Errorf("verify supported Java runtime: %w", err), java.Close())
	}
	formalAfterJavaProbe(java.Path())
	if err := java.Validate(); err != nil {
		return nil, errors.Join(fmt.Errorf("revalidate %s before engine execution: %w", java.Identity(), err), java.Close())
	}
	return java, nil
}
