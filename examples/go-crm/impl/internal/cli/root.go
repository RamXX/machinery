package cli

import (
	"crm/internal/domain"
	"crm/internal/repo"
	"crm/internal/session"
)

// Run is the process entrypoint exposed by crm.commands (BUILD.md 4.5:
// crm.commands exposes internal/cli/root.go). It wires the session, domain, and
// repo boundaries and drives the CommandExecution envelope for one invocation,
// returning the process exit code.
//
// It demonstrates the allowed dependency edges (commands->session,
// commands->domain, commands->repo) and deliberately does NOT import crm.authz:
// authorization is reached through domain.Service.CheckAuthorization.
//
// SCAFFOLDING STUB: the cobra command tree and the load-act-save envelope
// (BUILD.md 9) are the implementer's job.
func Run(args []string, r repo.Repo, tokenPath string, hmacKey []byte) int {
	svc := domain.NewDefaultService(r)
	sess := session.New(r, tokenPath, hmacKey)
	ce := &CommandExecution{Authorize: svc.CheckAuthorization}
	_, _, _ = sess, ce, args
	return 1 // not implemented
}
