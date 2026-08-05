package bot

import (
	"context"
	"fmt"
	"sync"

	"popplio/infernoplex/dclient"
	"popplio/state"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

const (
	colourBlurple = 0x7289DA
	colourGreen   = 0x00ff00
	colourRed     = 0xFF0000
)

type Ctx struct {
	Context context.Context

	Author    discord.User
	GuildID   snowflake.ID
	ChannelID snowflake.ID

	Options map[string]string

	slash     *events.ApplicationCommandInteractionCreate
	component *events.ComponentInteractionCreate
	modal     *events.ModalSubmitInteractionCreate
	replied   bool
}

func (c *Ctx) Option(name string) string {
	return c.Options[name]
}

func (c *Ctx) Say(content string) error {
	return c.SayColour(content, colourBlurple)
}

func (c *Ctx) Ok(content string) error {
	return c.SayColour(content, colourGreen)
}

func (c *Ctx) Fail(content string) error {
	return c.SayColour(content, colourRed)
}

func (c *Ctx) SayColour(content string, colour int) error {
	return c.Send(discord.MessageCreate{Embeds: []discord.Embed{{
		Description: content,
		Color:       colour,
	}}})
}

func (c *Ctx) Send(msg discord.MessageCreate) error {
	switch {
	case c.slash != nil:
		if c.replied {
			_, err := c.slash.Client().Rest().CreateFollowupMessage(c.slash.ApplicationID(), c.slash.Token(), msg)
			return err
		}

		c.replied = true
		return c.slash.CreateMessage(msg)

	case c.component != nil:
		if c.replied {
			_, err := c.component.Client().Rest().CreateFollowupMessage(c.component.ApplicationID(), c.component.Token(), msg)
			return err
		}

		c.replied = true
		return c.component.CreateMessage(msg)

	case c.modal != nil:
		if c.replied {
			_, err := c.modal.Client().Rest().CreateFollowupMessage(c.modal.ApplicationID(), c.modal.Token(), msg)
			return err
		}

		c.replied = true
		return c.modal.CreateMessage(msg)

	default:
		_, err := dclient.Get().Rest().CreateMessage(c.ChannelID, msg)
		return err
	}
}

func (c *Ctx) Defer() error {
	if c.replied {
		return nil
	}

	c.replied = true

	switch {
	case c.slash != nil:
		return c.slash.DeferCreateMessage(false)
	case c.component != nil:
		return c.component.DeferCreateMessage(false)
	case c.modal != nil:
		return c.modal.DeferCreateMessage(false)
	default:
		return nil
	}
}

func (c *Ctx) Modal(modal discord.ModalCreate) error {
	c.replied = true

	switch {
	case c.slash != nil:
		return c.slash.Modal(modal)
	case c.component != nil:
		return c.component.Modal(modal)
	default:
		return fmt.Errorf("no live interaction to answer with a modal")
	}
}

type Check func(c *Ctx) error

type Command struct {
	Name        string
	Description string
	Options     []discord.ApplicationCommandOption
	Checks      []Check

	AdminOnly bool

	OwnerOnly bool
	Run       func(c *Ctx) error
}

var (
	registry = map[string]*Command{}
	ordered  []*Command
)

func register(cmds ...*Command) {
	for _, cmd := range cmds {
		registry[cmd.Name] = cmd
		ordered = append(ordered, cmd)
	}
}

func Listener(ctx context.Context) bot.EventListener {
	registerCommands()

	return &events.ListenerAdapter{
		OnReady:                         func(e *events.Ready) { onReady(ctx, e) },
		OnApplicationCommandInteraction: func(e *events.ApplicationCommandInteractionCreate) { onSlashCommand(ctx, e) },
		OnComponentInteraction:          func(e *events.ComponentInteractionCreate) { onComponent(ctx, e) },
		OnModalSubmit:                   func(e *events.ModalSubmitInteractionCreate) { onModalSubmit(ctx, e) },
	}
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

	invoke(cmd, c)
}

func invoke(cmd *Command, c *Ctx) {
	state.Logger.Info(fmt.Sprintf("Executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))

	defer func() {
		if rec := recover(); rec != nil {
			state.Logger.Error("Command panicked", zap.String("command", cmd.Name), zap.Any("panic", rec))
			c.Fail("There was an error running this command: internal error")
		}
	}()

	if cmd.OwnerOnly && !isOwner(c.Author.ID) {
		c.Fail("You do not have permission to run this command")
		return
	}

	for _, check := range cmd.Checks {
		if err := check(c); err != nil {
			c.Fail(err.Error())
			return
		}
	}

	if cmd.Run == nil {
		c.Fail("This command is currently disabled")
		return
	}

	if err := cmd.Run(c); err != nil {
		c.Fail(err.Error())
		return
	}

	state.Logger.Info(fmt.Sprintf("Done executing command %s for user %s (%s)...", cmd.Name, c.Author.Username, c.Author.ID))
}

func buildCommands() []discord.ApplicationCommandCreate {
	cmds := make([]discord.ApplicationCommandCreate, 0, len(ordered))

	for _, cmd := range ordered {
		create := discord.SlashCommandCreate{
			Name:        cmd.Name,
			Description: truncate(cmd.Description, 100),
			Options:     cmd.Options,
		}

		if cmd.AdminOnly {
			create.DefaultMemberPermissions = json.NewNullablePtr(discord.PermissionAdministrator)
		}

		cmds = append(cmds, create)
	}

	return cmds
}

func SyncCommands() error {
	cmds := buildCommands()

	if _, err := dclient.Get().Rest().SetGlobalCommands(dclient.Get().ApplicationID(), cmds); err != nil {
		return fmt.Errorf("failed to register global commands: %w", err)
	}

	state.Logger.Info("Registered global slash commands", zap.Int("count", len(cmds)))

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

var (
	ownersMu sync.Mutex
	owners   map[snowflake.ID]struct{}
)

func isOwner(id snowflake.ID) bool {
	ownersMu.Lock()
	defer ownersMu.Unlock()

	_, ok := owners[id]
	return ok
}

func loadOwners() {
	app, err := dclient.Get().Rest().GetBotApplicationInfo()

	if err != nil {
		state.Logger.Error("Failed to fetch Infernoplex application info for owner resolution", zap.Error(err))
		return
	}

	found := map[snowflake.ID]struct{}{}

	if app.Team != nil {
		for _, member := range app.Team.Members {
			found[member.User.ID] = struct{}{}
		}
	} else if app.Owner != nil {
		found[app.Owner.ID] = struct{}{}
	}

	ownersMu.Lock()
	owners = found
	ownersMu.Unlock()
}
