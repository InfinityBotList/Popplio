// The internal handler for the Popplio frontend bot
package botapi

import (
	"fmt"
	"popplio/state"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type ArgumentType int

const (
	ArgumentTypeString ArgumentType = iota
	ArgumentTypeInteger
	ArgumentTypeBoolean
	ArgumentTypeUser
	ArgumentTypeMember
	ArgumentTypeChannel
	ArgumentTypeRole
)

type Context struct {
	// Filled in on every command execution
	Values      ValueList
	Command     Command
	Subcommand  Subcommand
	Interaction *discordgo.Interaction
	Session     *discordgo.Session
}

func (ctx *Context) Reply(text string, ephemeral bool) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: text,
			Flags:   flags,
		},
	})
}

type Value struct {
	// Name of the value
	Name string
	// Description of the value
	Description string
	// Value filled in on every command execution
	Value any
	// Type of the argument
	Type ArgumentType
	// Whether or not the value was passed in by the user
	Valid bool
	// Whether or not value is required
	Required bool

	// This provides the extra context required for proper error handling in the helper commands
	i    *discordgo.Interaction
	sess *discordgo.Session
}

func (v Value) String() string {
	if v.Type != ArgumentTypeString {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: String() called on non-string value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(string)
}

func (v Value) Integer() int {
	if v.Type != ArgumentTypeInteger {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: Integer() called on non-integer value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(int)
}

func (v Value) Boolean() bool {
	if v.Type != ArgumentTypeBoolean {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: Boolean() called on non-boolean value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(bool)
}

func (v Value) User() *discordgo.User {
	if v.Type != ArgumentTypeUser {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: User() called on non-user value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(*discordgo.User)
}

func (v Value) Member() *discordgo.Member {
	if v.Type != ArgumentTypeMember {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: Member() called on non-member value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(*discordgo.Member)
}

func (v Value) Channel() *discordgo.Channel {
	if v.Type != ArgumentTypeChannel {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: Channel() called on non-channel value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(*discordgo.Channel)
}

func (v Value) Role() *discordgo.Role {
	if v.Type != ArgumentTypeRole {
		v.sess.InteractionRespond(v.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Internal error: Role() called on non-role value",
			},
		})
		// This panic is handled by recover() in the main handler
		panic("EXIT")
	}
	return v.Value.(*discordgo.Role)
}

// Stores the list of arguments for a command
type ValueList struct {
	values []Value
}

func (vl *ValueList) Get(name string) Value {
	for _, v := range vl.values {
		if v.Name == name {
			return v
		}
	}
	return Value{}
}

type Subcommand struct {
	Name        string
	Description string
	Arguments   []Value
	Callback    func(*Context)
}

func (sc Subcommand) GetArguments(vs []*discordgo.ApplicationCommandInteractionDataOption) ValueList {
	vl := ValueList{}
	for _, v := range sc.Arguments {
		for _, o := range vs {
			if o.Name == v.Name {
				v.Valid = true
				switch v.Type {
				case ArgumentTypeString:
					v.Value = o.Value.(string)
				case ArgumentTypeInteger:
					v.Value = o.Value.(int)
				case ArgumentTypeBoolean:
					v.Value = o.Value.(bool)
				case ArgumentTypeUser:
					v.Value = o.Value.(*discordgo.User)
				case ArgumentTypeMember:
					v.Value = o.Value.(*discordgo.Member)
				case ArgumentTypeChannel:
					v.Value = o.Value.(*discordgo.Channel)
				case ArgumentTypeRole:
					v.Value = o.Value.(*discordgo.Role)
				}
			}
		}
		vl.values = append(vl.values, v)
	}
	return vl
}

type Command struct {
	Name        string
	Description string
	Arguments   []Value
	Subcommands []Subcommand
	Callback    func(*Context)
}

func (c Command) GetArguments(vs []*discordgo.ApplicationCommandInteractionDataOption) ValueList {
	vl := ValueList{}

	for _, v := range c.Arguments {
		for _, o := range vs {
			if o.Name == v.Name {
				v.Valid = true
				switch v.Type {
				case ArgumentTypeString:
					v.Value = o.Value.(string)
				case ArgumentTypeInteger:
					v.Value = o.Value.(int)
				case ArgumentTypeBoolean:
					v.Value = o.Value.(bool)
				case ArgumentTypeUser:
					v.Value = o.Value.(*discordgo.User)
				case ArgumentTypeMember:
					v.Value = o.Value.(*discordgo.Member)
				case ArgumentTypeChannel:
					v.Value = o.Value.(*discordgo.Channel)
				case ArgumentTypeRole:
					v.Value = o.Value.(*discordgo.Role)
				}
			}
		}
		vl.values = append(vl.values, v)
	}
	return vl
}

func (c *Command) GetSubcommand(name string) (Subcommand, bool) {
	for _, sc := range c.Subcommands {
		if sc.Name == name {
			return sc, true
		}
	}
	return Subcommand{}, false
}

var cmdMap map[string]Command = make(map[string]Command)

func AddCommand(cmd Command) {
	if cmd.Name == "" {
		panic("Command name cannot be empty")
	}

	if cmd.Description == "" {
		panic("Command description cannot be empty")
	}

	for _, subcommand := range cmd.Subcommands {
		if subcommand.Name == "" {
			panic("Subcommand name cannot be empty")
		}

		if subcommand.Description == "" {
			panic("Subcommand description cannot be empty")
		}

		if subcommand.Callback == nil {
			panic("Subcommand callback cannot be nil")
		}
	}

	if cmd.Callback == nil {
		panic("Command callback cannot be nil")
	}

	cmdMap[cmd.Name] = cmd
}

// helper method used to register the commands with discord
func RegisterWithAPI(s *discordgo.Session) {
	currentUser := s.State.User
	fmt.Println(currentUser.Username)
	var cmdList []*discordgo.ApplicationCommand = make([]*discordgo.ApplicationCommand, 0, len(cmdMap))
	var i int
	for _, cmd := range cmdMap {
		cmdList = append(cmdList, &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     make([]*discordgo.ApplicationCommandOption, 0, len(cmd.Arguments)),
		})

		for _, arg := range cmd.Arguments {
			var opttype discordgo.ApplicationCommandOptionType

			switch arg.Type {
			case ArgumentTypeString:
				opttype = discordgo.ApplicationCommandOptionString
			case ArgumentTypeInteger:
				opttype = discordgo.ApplicationCommandOptionInteger
			case ArgumentTypeBoolean:
				opttype = discordgo.ApplicationCommandOptionBoolean
			case ArgumentTypeUser:
				opttype = discordgo.ApplicationCommandOptionUser
			case ArgumentTypeMember:
				opttype = discordgo.ApplicationCommandOptionUser
			case ArgumentTypeChannel:
				opttype = discordgo.ApplicationCommandOptionChannel
			case ArgumentTypeRole:
				opttype = discordgo.ApplicationCommandOptionRole
			}

			cmdList[i].Options = append(cmdList[i].Options, &discordgo.ApplicationCommandOption{
				Type:        opttype,
				Name:        arg.Name,
				Description: arg.Description,
				Required:    arg.Required,
			})
		}

		for _, subcommand := range cmd.Subcommands {
			cmdList[i].Options = append(cmdList[i].Options, &discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        subcommand.Name,
				Description: subcommand.Description,
				Options:     make([]*discordgo.ApplicationCommandOption, 0, len(subcommand.Arguments)),
			})

			for _, arg := range subcommand.Arguments {
				var opttype discordgo.ApplicationCommandOptionType

				switch arg.Type {
				case ArgumentTypeString:
					opttype = discordgo.ApplicationCommandOptionString
				case ArgumentTypeInteger:
					opttype = discordgo.ApplicationCommandOptionInteger
				case ArgumentTypeBoolean:
					opttype = discordgo.ApplicationCommandOptionBoolean
				case ArgumentTypeUser:
					opttype = discordgo.ApplicationCommandOptionUser
				case ArgumentTypeMember:
					opttype = discordgo.ApplicationCommandOptionUser
				case ArgumentTypeChannel:
					opttype = discordgo.ApplicationCommandOptionChannel
				case ArgumentTypeRole:
					opttype = discordgo.ApplicationCommandOptionRole
				}

				cmdList[i].Options[len(cmdList[i].Options)-1].Options = append(cmdList[i].Options[len(cmdList[i].Options)-1].Options, &discordgo.ApplicationCommandOption{
					Type:        opttype,
					Name:        arg.Name,
					Description: arg.Description,
					Required:    arg.Required,
				})
			}
		}

		i++
	}

	s.ApplicationCommandBulkOverwrite(currentUser.ID, "", cmdList)
}

func Start(sess *discordgo.Session) {
	// Create a discordgo event listener for the interactionCreate event
	sess.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			// Check if the command exists
			cmd, ok := cmdMap[i.ApplicationCommandData().Name]

			if !ok {
				return
			}

			// Create a new context
			ctx := &Context{
				Session:     s,
				Command:     cmd,
				Interaction: i.Interaction,
			}

			var values ValueList

			// Check if the command has subcommands
			if len(cmd.Subcommands) > 0 {
				// Check if the subcommand exists
				subcmd, ok := cmd.GetSubcommand(i.ApplicationCommandData().Options[0].Name)

				if !ok {
					return
				}

				// Set the subcommand
				ctx.Subcommand = subcmd

				// Set the arguments
				values = subcmd.GetArguments(i.ApplicationCommandData().Options[0].Options)
			} else {
				// Set the arguments
				values = cmd.GetArguments(i.ApplicationCommandData().Options)
			}

			ctx.Values = values

			// Call the callback
			go func(
				ctx *Context,
			) {
				defer func() {
					if r := recover(); r != nil {
						state.Logger.Error("Panic in command callback", zap.Any("panic", r))
					}
				}()

				ctx.Command.Callback(ctx)
			}(ctx)
		}
	})
}
