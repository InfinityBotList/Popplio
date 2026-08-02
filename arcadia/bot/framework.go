// Package bot is the staff Discord bot: the other half of the system that
// shares the RPC action layer with the panel.
//
// Arcadia used poise, which supplies a command framework (prefix + slash
// parsing, checks, cooldowns, help rendering, modals, pagination). disgo has no
// equivalent, so the pieces the commands actually rely on are implemented here.
package bot

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

// Ctx is what a command handler receives. It abstracts over a prefix invocation
// and a slash invocation so each command is written once.
type Ctx struct {
	Context context.Context

	// Author is who invoked the command.
	Author discord.User
	// GuildID is 0 in DMs.
	GuildID snowflake.ID
	// ChannelID is where the command was invoked.
	ChannelID snowflake.ID

	// Args are the positional arguments of a prefix invocation.
	Args []string
	// Options are the named options of a slash invocation.
	Options map[string]string

	// slash is set for slash invocations.
	slash *events.ApplicationCommandInteractionCreate
	// replied tracks whether the interaction has been answered.
	replied bool
}

// Option reads a named option, falling back to the positional prefix argument at
// the given index.
func (c *Ctx) Option(name string, index int) string {
	if v, ok := c.Options[name]; ok {
		return v
	}

	if index >= 0 && index < len(c.Args) {
		return c.Args[index]
	}

	return ""
}

// BoolOption reads a boolean option, defaulting when absent.
func (c *Ctx) BoolOption(name string, fallback bool) bool {
	v, ok := c.Options[name]

	if !ok {
		return fallback
	}

	switch strings.ToLower(v) {
	case "true", "t", "y", "yes":
		return true
	case "false", "f", "n", "no":
		return false
	default:
		return fallback
	}
}

// Say sends a plain text reply.
func (c *Ctx) Say(content string) error {
	return c.Send(discord.MessageCreate{Content: content})
}

// Send sends a full message reply.
func (c *Ctx) Send(msg discord.MessageCreate) error {
	if c.slash != nil {
		if c.replied {
			_, err := c.slash.Client().Rest().CreateFollowupMessage(
				c.slash.ApplicationID(), c.slash.Token(), messageCreateToFollowup(msg))
			return err
		}

		c.replied = true
		return c.slash.CreateMessage(msg)
	}

	_, err := state.Discord.Rest().CreateMessage(c.ChannelID, msg)
	return err
}

// Defer acknowledges a slash command that will take a while. It is a no-op for
// prefix invocations.
func (c *Ctx) Defer() error {
	if c.slash == nil || c.replied {
		return nil
	}

	c.replied = true
	return c.slash.DeferCreateMessage(false)
}

func messageCreateToFollowup(msg discord.MessageCreate) discord.MessageCreate {
	return msg
}

// Check is a command guard. Returning an error surfaces its text to the user,
// which is how "You are not staff" reaches them.
type Check func(c *Ctx) error

// Command is one bot command.
type Command struct {
	Name        string
	Category    string
	Description string
	// Options are the slash-command options, in order.
	Options []discord.ApplicationCommandOption
	Checks  []Check
	Run     func(c *Ctx) error
	// Subcommands are dispatched on the first positional argument.
	Subcommands []*Command
	// OwnerOnly restricts the command to the configured owners.
	OwnerOnly bool
}

// registry holds every registered command by name.
var registry = map[string]*Command{}

// ordered keeps registration order for the help renderer.
var ordered []*Command

func register(cmds ...*Command) {
	for _, cmd := range cmds {
		registry[cmd.Name] = cmd
		ordered = append(ordered, cmd)
	}
}

// Setup registers the commands and attaches the event listeners to Popplio's
// existing Discord client. It does not open a second gateway connection.
func Setup(ctx context.Context) {
	registerCommands()

	state.Discord.AddEventListeners(&events.ListenerAdapter{
		OnGuildsReady:                   func(e *events.GuildsReady) { onGuildsReady(ctx, e) },
		OnGuildMemberJoin:               onGuildMemberJoin,
		OnMessageCreate:                 func(e *events.MessageCreate) { onMessageCreate(ctx, e) },
		OnApplicationCommandInteraction: func(e *events.ApplicationCommandInteractionCreate) { onSlashCommand(ctx, e) },
		OnComponentInteraction:          func(e *events.ComponentInteractionCreate) { onComponent(ctx, e) },
		OnModalSubmit:                   func(e *events.ModalSubmitInteractionCreate) { onModalSubmit(ctx, e) },
	})
}

// prefix is the configured command prefix for the current environment.
func prefix() string {
	return state.Config.Arcadia.Prefix.Parse()
}

func onMessageCreate(ctx context.Context, e *events.MessageCreate) {
	if e.Message.Author.Bot {
		return
	}

	p := prefix()

	if !strings.HasPrefix(e.Message.Content, p) {
		return
	}

	fields := strings.Fields(strings.TrimPrefix(e.Message.Content, p))

	if len(fields) == 0 {
		return
	}

	cmd, ok := registry[strings.ToLower(fields[0])]

	if !ok {
		return
	}

	c := &Ctx{
		Context:   ctx,
		Author:    e.Message.Author,
		ChannelID: e.ChannelID,
		Args:      fields[1:],
		Options:   map[string]string{},
	}

	if e.GuildID != nil {
		c.GuildID = *e.GuildID
	}

	invoke(cmd, c)
}

func onSlashCommand(ctx context.Context, e *events.ApplicationCommandInteractionCreate) {
	data := e.SlashCommandInteractionData()

	cmd, ok := registry[data.CommandName()]

	if !ok {
		return
	}

	c := &Ctx{
		Context:   ctx,
		Author:    e.User(),
		ChannelID: e.Channel().ID(),
		Options:   map[string]string{},
		slash:     e,
	}

	if e.GuildID() != nil {
		c.GuildID = *e.GuildID()
	}

	for name := range data.Options {
		if v, ok := data.OptString(name); ok {
			c.Options[name] = v
			continue
		}

		if v, ok := data.OptBool(name); ok {
			c.Options[name] = fmt.Sprintf("%t", v)
			continue
		}

		if v, ok := data.OptInt(name); ok {
			c.Options[name] = fmt.Sprintf("%d", v)
		}
	}

	// A subcommand arrives as the option name; mirror it into Args so the
	// prefix and slash paths dispatch identically.
	if sub := data.SubCommandName; sub != nil {
		c.Args = append(c.Args, *sub)
	}

	invoke(cmd, c)
}

// invoke runs a command through its checks, logging like poise's pre/post hooks.
func invoke(cmd *Command, c *Ctx) {
	// Subcommand dispatch on the first positional argument.
	if len(cmd.Subcommands) > 0 && len(c.Args) > 0 {
		for _, sub := range cmd.Subcommands {
			if strings.EqualFold(sub.Name, c.Args[0]) {
				c.Args = c.Args[1:]
				invoke(sub, c)
				return
			}
		}
	}

	state.Logger.Info(fmt.Sprintf("Executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))

	defer func() {
		if rec := recover(); rec != nil {
			state.Logger.Error("Command panicked", zap.String("command", cmd.Name), zap.Any("panic", rec))
			c.Say("There was an error running this command: internal error")
		}
	}()

	if cmd.OwnerOnly && !isOwner(c.Author.ID) {
		c.Say("Whoa there, do you have permission to do this?: You are not an owner")
		return
	}

	for _, check := range cmd.Checks {
		if err := check(c); err != nil {
			c.Say(fmt.Sprintf("Whoa there, do you have permission to do this?: %s", err))
			return
		}
	}

	if cmd.Run == nil {
		c.Say("This command is currently disabled")
		return
	}

	if err := cmd.Run(c); err != nil {
		c.Say(fmt.Sprintf("There was an error running this command: %s", err))
		return
	}

	state.Logger.Info(fmt.Sprintf("Done executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))
}

func isOwner(userID snowflake.ID) bool {
	for _, owner := range state.Config.Arcadia.Owners {
		if owner == userID {
			return true
		}
	}

	return false
}

// SyncCommands registers the slash commands with Discord for the staff guild.
func SyncCommands() error {
	cmds := make([]discord.ApplicationCommandCreate, 0, len(ordered))

	for _, cmd := range ordered {
		create := discord.SlashCommandCreate{
			Name:        cmd.Name,
			Description: truncate(cmd.Description, 100),
			Options:     cmd.Options,
		}

		for _, sub := range cmd.Subcommands {
			create.Options = append(create.Options, discord.ApplicationCommandOptionSubCommand{
				Name:        sub.Name,
				Description: truncate(sub.Description, 100),
			})
		}

		cmds = append(cmds, create)
	}

	_, err := state.Discord.Rest().SetGuildCommands(state.Discord.ApplicationID(), state.Config.Servers.Staff, cmds)

	return err
}

func truncate(s string, max int) string {
	if s == "" {
		return "No description"
	}

	if len(s) <= max {
		return s
	}

	return s[:max]
}

// helpCategories groups the registered commands for the help renderer.
func helpCategories() []string {
	seen := map[string]struct{}{}

	var out []string

	for _, cmd := range ordered {
		if _, ok := seen[cmd.Category]; ok {
			continue
		}

		seen[cmd.Category] = struct{}{}
		out = append(out, cmd.Category)
	}

	sort.Strings(out)

	return out
}
