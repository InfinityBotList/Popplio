package bot

import (
	"fmt"

	"popplio/arcadia/impls"
	"popplio/arcadia/rpc"
	"popplio/arcadia/types"
	"popplio/perms"
)

// The command registry: the one list of what the staff bot answers to, and the
// two helpers every command that reaches the action layer goes through.
//
// Each command's own file holds its handler. Adding one is a line here and a
// constructor there.

func registerCommands() {
	register(
		cmdRegister(),
		cmdHelp(),
		cmdExplainMe(),
		cmdStaff(),
		cmdInviteDB(),
		cmdInvite(),
		cmdStaffGuide(),
		cmdQueue(),
		cmdClaim(),
		cmdUnclaim(),
		cmdApprove(),
		cmdDeny(),
		cmdAnalytics(),
		cmdInfo(),
		cmdLeaderboard(),
		cmdRefresh(),
		cmdGetBotRoles(),
		cmdRPC(),
		cmdRPCList(),
	)

	registerStaffRoleCommands()
}

// cmdRegister is poise's owner-only application-command registration helper.
// disgo has no button-driven equivalent, so this syncs the guild commands
// directly.
func cmdRegister() *Command {
	return &Command{
		Name:        "register",
		Category:    "Owner",
		Description: "Register application commands",
		OwnerOnly:   true,
		Run: func(c *Ctx) error {
			if err := SyncCommands(); err != nil {
				return err
			}

			return c.Ok("Registered application commands.")
		},
	}
}

// runRPCWithTarget executes an RPC against an explicit target type. The modal
// driver is the only caller that needs a target type other than Bot.
func runRPCWithTarget(c *Ctx, method types.RPCMethod, targetType types.TargetType) (rpc.Success, error) {
	return rpc.Execute(c.Context, method, rpc.Handle{
		UserID:     c.Author.ID.String(),
		TargetType: targetType,
	})
}

// requirePerm resolves the caller's permissions and checks one of them.
func requirePerm(c *Ctx, perm perms.Perm) error {
	sp, err := impls.GetUserPerms(c.Context, c.Author.ID.String())

	if err != nil {
		return err
	}

	if !sp.Has(perm) {
		return fmt.Errorf("You need the %s permission to use this command", perms.Staff.Label(perm))
	}

	return nil
}

// runRPC is the shared path from a Discord command into the RPC layer. Every
// command except the modal driver targets a bot.
func runRPC(c *Ctx, method types.RPCMethod) (rpc.Success, error) {
	return runRPCWithTarget(c, method, types.TargetTypeBot)
}
