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

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
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

// Say replies with a message in the bot's neutral colour.
//
// Every reply the bot makes is an embed, including one-liners: a bare content
// message is indistinguishable from a staff member talking, which matters in
// the staff server where the bot's answers and the conversation share a channel.
// The text itself is unchanged — the embed is the container, not a rewrite — so
// the strings frozen in arcadia/conformance still read exactly as they did.
func (c *Ctx) Say(content string) error {
	return c.SayColour(content, impls.ColourBlurple)
}

// Ok replies to something that worked.
func (c *Ctx) Ok(content string) error {
	return c.SayColour(content, impls.ColourGreen)
}

// Fail replies to something that did not.
//
// This is what the command guards and the panic handler use, so a refusal is
// visibly different from an answer without either of them having to say so.
func (c *Ctx) Fail(content string) error {
	return c.SayColour(content, impls.ColourRed)
}

// SayColour replies with a message in an explicit colour.
func (c *Ctx) SayColour(content string, colour int) error {
	return c.Send(discord.MessageCreate{Embeds: []discord.Embed{{
		Description: content,
		Color:       colour,
	}}})
}

// Send sends a full message reply.
func (c *Ctx) Send(msg discord.MessageCreate) error {
	if c.slash != nil {
		if c.replied {
			_, err := c.slash.Client().Rest().CreateFollowupMessage(
				c.slash.ApplicationID(), c.slash.Token(), msg)
			return err
		}

		c.replied = true
		return c.slash.CreateMessage(msg)
	}

	_, err := dclient.Get().Rest().CreateMessage(c.ChannelID, msg)
	return err
}

// SendTracked sends a reply and returns the id of the resulting message.
//
// The component-driven commands (queue, claim) key their session on that id. A
// slash invocation must be answered through the interaction or Discord shows the
// caller "the application did not respond", so this replies the right way for
// the invocation and then resolves the message id either way.
func (c *Ctx) SendTracked(msg discord.MessageCreate) (snowflake.ID, error) {
	if c.slash == nil {
		sent, err := dclient.Get().Rest().CreateMessage(c.ChannelID, msg)

		if err != nil {
			return 0, err
		}

		return sent.ID, nil
	}

	if err := c.Send(msg); err != nil {
		return 0, err
	}

	sent, err := dclient.Get().Rest().GetInteractionResponse(c.slash.ApplicationID(), c.slash.Token())

	if err != nil {
		return 0, err
	}

	return sent.ID, nil
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

// Listener registers the commands and returns the staff bot's event listener.
//
// It is built before the gateway connects so no early event is missed; the
// caller hands it to dclient.Setup.
func Listener(ctx context.Context) bot.EventListener {
	registerCommands()
	indexCategories()

	adapter := &events.ListenerAdapter{
		OnGuildsReady:                   func(e *events.GuildsReady) { onGuildsReady(ctx, e) },
		OnGuildMemberJoin:               onGuildMemberJoin,
		OnApplicationCommandInteraction: func(e *events.ApplicationCommandInteractionCreate) { onSlashCommand(ctx, e) },
		OnComponentInteraction:          func(e *events.ComponentInteractionCreate) { onComponent(ctx, e) },
		OnModalSubmit:                   func(e *events.ModalSubmitInteractionCreate) { onModalSubmit(ctx, e) },
	}

	// Slash commands are the primary interface. The legacy prefix listener is
	// only attached when explicitly enabled, because reading message content
	// needs the privileged Message Content intent.
	if state.Config.Arcadia.PrefixCommands {
		adapter.OnMessageCreate = func(e *events.MessageCreate) { onMessageCreate(ctx, e) }
	}

	return adapter
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
			c.Fail("There was an error running this command: internal error")
		}
	}()

	if cmd.OwnerOnly && !isOwner(c.Author.ID) {
		c.Fail("Whoa there, do you have permission to do this?: You are not an owner")
		return
	}

	for _, check := range cmd.Checks {
		if err := check(c); err != nil {
			c.Fail(fmt.Sprintf("Whoa there, do you have permission to do this?: %s", err))
			return
		}
	}

	if cmd.Run == nil {
		c.Fail("This command is currently disabled")
		return
	}

	if err := cmd.Run(c); err != nil {
		c.Fail(fmt.Sprintf("There was an error running this command: %s", err))
		return
	}

	state.Logger.Info(fmt.Sprintf("Done executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))
}

func isOwner(userID snowflake.ID) bool {
	return perms.IsConfigOwner(userID.String())
}

// commandGuilds are the guilds the slash commands are published to.
//
// Registration is per-guild rather than global because the commands are staff
// tooling that no other server should see, and because guild commands take
// effect immediately instead of Discord's ~1 hour global propagation. The full
// set goes to all three: the per-command guards already decide where each one is
// actually usable, so a command registered in the wrong guild is rejected with
// its normal message rather than silently missing.
func commandGuilds() []snowflake.ID {
	return []snowflake.ID{
		state.Config.Servers.Main,
		state.Config.Servers.Staff,
		state.Config.Servers.Testing,
	}
}

// buildCommands renders the registry into Discord's application command shape.
func buildCommands() []discord.ApplicationCommandCreate {
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
				Options:     sub.Options,
			})
		}

		cmds = append(cmds, create)
	}

	return cmds
}

// SyncCommands publishes the slash commands to every staff guild and clears any
// application-wide copy of them.
func SyncCommands() error {
	cmds := buildCommands()

	for _, guildID := range commandGuilds() {
		if guildID == 0 {
			continue
		}

		_, err := dclient.Get().Rest().SetGuildCommands(dclient.Get().ApplicationID(), guildID, cmds)

		if err != nil {
			return fmt.Errorf("failed to register commands in guild %s: %w", guildID, err)
		}

		state.Logger.Info("Registered slash commands", zap.String("guildID", guildID.String()), zap.Int("count", len(cmds)))
	}

	return pruneGlobalCommands(cmds)
}

// pruneGlobalCommands deletes the global registration of any command that is
// also registered per guild.
//
// Discord delivers a global command to every guild the application is in and
// does not let a guild command of the same name stand in for it: both are listed
// side by side, so a command registered both ways shows up twice in the picker
// in every server. Everything here goes out per guild (see commandGuilds), which
// makes a global copy a leftover — from a deployment that registered globally —
// and the reason the commands were doubled.
//
// Global commands whose names this bot does not register are left alone and only
// logged: they are not ours to delete, and they are not what doubles anything.
func pruneGlobalCommands(registered []discord.ApplicationCommandCreate) error {
	appID := dclient.Get().ApplicationID()

	existing, err := dclient.Get().Rest().GetGlobalCommands(appID, false)

	if err != nil {
		return fmt.Errorf("failed to read the global commands: %w", err)
	}

	if len(existing) == 0 {
		return nil
	}

	ours := make(map[string]struct{}, len(registered))

	for _, cmd := range registered {
		ours[cmd.CommandName()] = struct{}{}
	}

	var foreign []string

	for _, cmd := range existing {
		if _, mine := ours[cmd.Name()]; !mine {
			foreign = append(foreign, cmd.Name())
			continue
		}

		if err := dclient.Get().Rest().DeleteGlobalCommand(appID, cmd.ID()); err != nil {
			return fmt.Errorf("failed to remove the global copy of /%s: %w", cmd.Name(), err)
		}

		state.Logger.Info("Removed a duplicate global slash command", zap.String("command", cmd.Name()))
	}

	if len(foreign) > 0 {
		state.Logger.Warn("The application has global slash commands this bot does not register; they appear in every server it is in",
			zap.Strings("commands", foreign))
	}

	return nil
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

// categories is the sorted set of command categories, computed once at
// registration rather than on every help invocation.
var categories []string

// helpCategories groups the registered commands for the help renderer.
func helpCategories() []string {
	return categories
}

// indexCategories fills categories from the registered commands.
func indexCategories() {
	seen := make(map[string]struct{}, len(ordered))
	categories = categories[:0]

	for _, cmd := range ordered {
		if _, ok := seen[cmd.Category]; ok {
			continue
		}

		seen[cmd.Category] = struct{}{}
		categories = append(categories, cmd.Category)
	}

	sort.Strings(categories)
}
