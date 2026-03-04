package ship

import cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"

func (c CLI) runAgent(args []string) int {
	return cmd.RunAgent(args, cmd.AgentDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}
