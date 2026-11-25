package contract

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	serviceRe = regexp.MustCompile(`^\s*service\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rpcRe     = regexp.MustCompile(`^\s*rpc\s+([A-Za-z_][A-Za-z0-9_]*)?\s*\(`)
	bindRe    = regexp.MustCompile(`^\s*//\s*machinery:action\s+(\S+)\s*$`)
)

// parseProto lists the rpcs of a .proto file. Binding comments
// ("// machinery:action <Entity>.<action>") directly above an rpc line bind it;
// several comment lines are allowed. A binding that is not directly above an
// rpc line is a problem: it would otherwise be silently ignored.
func parseProto(rel string, data []byte) ([]Operation, []string) {
	var ops []Operation
	var problems []string
	service := ""
	var pending []string
	pendingLine := 0
	flush := func() {
		if len(pending) > 0 {
			problems = append(problems, fmt.Sprintf("%s:%d: machinery:action comment is not directly above an rpc line", rel, pendingLine))
		}
		pending = nil
	}
	for i, line := range strings.Split(string(data), "\n") {
		if m := bindRe.FindStringSubmatch(line); m != nil {
			if len(pending) == 0 {
				pendingLine = i + 1
			}
			pending = append(pending, m[1])
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue // another comment line may sit between the binding and the rpc
		}
		if m := rpcRe.FindStringSubmatch(line); m != nil {
			op := Operation{Contract: rel, ID: m[1], Method: "RPC", Actions: pending}
			op.Path = m[1]
			if service != "" {
				op.Path = service + "/" + m[1]
			}
			ops = append(ops, op)
			pending = nil
			continue
		}
		flush()
		if m := serviceRe.FindStringSubmatch(line); m != nil {
			service = m[1]
		}
	}
	flush()
	return ops, problems
}
