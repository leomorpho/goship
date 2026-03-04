package ship

import (
	cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func (c CLI) runNew(args []string) int {
	return cmd.RunNew(args, cmd.NewDeps{
		Out:                        c.Out,
		Err:                        c.Err,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	})
}

func (c CLI) runUpgrade(args []string) int {
	return cmd.RunUpgrade(args, cmd.UpgradeDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runDoctor(args []string) int {
	return policies.RunDoctor(args, policies.DoctorDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}
