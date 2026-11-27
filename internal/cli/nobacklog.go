package cli

import "os"

// NoBacklogEnv set to 1 keeps the commands that read only design and evidence
// from running the backlog tool, as their flag --no-backlog does (section 15.1).
const NoBacklogEnv = "MACHINERY_NO_BACKLOG"

// notRead is what such a command prints in place of its backlog lines.
const notRead = "backlog: not read"

// storyNeedsBacklog refuses --story under the switch: it names a story the
// backlog holds.
const storyNeedsBacklog = "--story reads its story from the backlog, which --no-backlog or " + NoBacklogEnv + "=1 says not to read"

// noBacklog reports the switch: the flag, or the environment variable set to 1.
func noBacklog(flag bool) bool { return flag || os.Getenv(NoBacklogEnv) == "1" }
