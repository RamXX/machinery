//go:build linux

package install

func setBeforeActivationExecForTest(hook func(string)) {
	beforeActivationExec = hook
}
