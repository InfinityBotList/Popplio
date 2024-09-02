package discord_dovewing

import (
	"context"
	"errors"
	"strconv"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

func disgoFlagsToArray(u *discord.User) []string {
	var arr = []string{}

	if u.Bot {
		if u.PublicFlags.Has(discord.UserFlagBotHTTPInteractions) {
			arr = append(arr, "BOT_HTTP_INTERACTIONS")
		}

		if u.PublicFlags.Has(discord.UserFlagVerifiedBot) {
			arr = append(arr, "VERIFIED_BOT")
		}
	}

	return arr
}

func disgoPlatformStatus(status discord.OnlineStatus) dovetypes.PlatformStatus {
	switch status {
	case discord.OnlineStatusOnline:
		return dovetypes.PlatformStatusOnline
	case discord.OnlineStatusIdle:
		return dovetypes.PlatformStatusIdle
	case discord.OnlineStatusDND:
		return dovetypes.PlatformStatusDoNotDisturb
	default:
		return dovetypes.PlatformStatusOffline
	}
}

type DisgoState struct {
	config        *DisgoStateConfig                    // Config for the discord state
	memberCache   cache.GroupedCache[discord.Member]   // Member cache
	guildsCache   cache.Cache[discord.Guild]           // Guild cache
	presenceCache cache.GroupedCache[discord.Presence] // Presence cache
	initialized   bool                                 // Whether the platform has been initted or not
}

type DisgoStateConfig struct {
	Client         bot.Client          // Discord session
	PreferredGuild *snowflake.ID       // Which guilds should be checked first for users, good if theres one guild with the majority of users
	BaseState      *dovewing.BaseState // Base state
}

func (c DisgoStateConfig) New() (*DisgoState, error) {
	if c.Client == nil {
		return nil, errors.New("discord not enabled")
	}

	if c.BaseState == nil {
		return nil, errors.New("base state not provided")
	}

	return &DisgoState{
		config: &c,
	}, nil
}

func (d *DisgoState) PlatformName() string {
	return "discord"
}

func (d *DisgoState) Init() error {
	caches := d.config.Client.Caches()

	if caches == nil {
		return errors.New("cache not enabled")
	}

	memberCache := caches.MemberCache()

	if memberCache == nil {
		return errors.New("member cache not enabled")
	}

	d.memberCache = memberCache

	presenceCache := caches.PresenceCache()

	if presenceCache == nil {
		return errors.New("presence cache not enabled")
	}

	d.presenceCache = presenceCache

	guildsCache := caches.GuildCache()

	if guildsCache == nil {
		return errors.New("guild cache not enabled")
	}

	d.guildsCache = guildsCache

	d.initialized = true
	return nil
}

func (d *DisgoState) Initted() bool {
	return d.initialized
}

func (d *DisgoState) GetState() *dovewing.BaseState {
	return d.config.BaseState
}

func (d *DisgoState) ValidateId(id string) (string, error) {
	// Before wasting time searching state, ensure the ID is actually a valid snowflake
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return "", err
	}

	// For all practical purposes, a simple length check can handle a lot of illegal IDs
	if len(id) <= 16 || len(id) > 20 {
		return "", errors.New("invalid snowflake")
	}

	return id, nil
}

func (d *DisgoState) PlatformSpecificCache(ctx context.Context, idStr string) (*dovetypes.PlatformUser, error) {
	id, err := snowflake.Parse(idStr)

	if err != nil {
		return nil, err
	}

	// First try for main server
	if d.config.PreferredGuild != nil {
		member, ok := d.memberCache.Get(*d.config.PreferredGuild, id)

		if ok {
			p, pOk := d.presenceCache.Get(*d.config.PreferredGuild, id)

			var status = discord.OnlineStatusOffline
			if pOk {
				status = p.Status
			}

			return &dovetypes.PlatformUser{
				ID:          idStr,
				Username:    member.User.Username,
				Avatar:      member.User.EffectiveAvatarURL(),
				DisplayName: member.EffectiveName(),
				Bot:         member.User.Bot,
				Flags:       disgoFlagsToArray(&member.User),
				ExtraData: map[string]any{
					"nickname":        member.Nick,
					"mutual_guild":    d.config.PreferredGuild,
					"preferred_guild": true,
					"public_flags":    member.User.PublicFlags,
				},
				Status: disgoPlatformStatus(status),
			}, nil
		}
	}

	var puser *dovetypes.PlatformUser
	d.config.Client.Caches().GuildCache().ForEach(func(guild discord.Guild) {
		if puser != nil || err != nil {
			return
		}

		member, ok := d.memberCache.Get(guild.ID, id)

		if ok {
			p, pOk := d.presenceCache.Get(guild.ID, id)

			var status = discord.OnlineStatusOffline
			if pOk {
				status = p.Status
			}

			puser = &dovetypes.PlatformUser{
				ID:          idStr,
				Username:    member.User.Username,
				Avatar:      member.User.EffectiveAvatarURL(),
				DisplayName: member.EffectiveName(),
				Bot:         member.User.Bot,
				Flags:       disgoFlagsToArray(&member.User),
				ExtraData: map[string]any{
					"nickname":        member.Nick,
					"mutual_guild":    guild.ID.String(),
					"preferred_guild": false,
					"public_flags":    member.User.PublicFlags,
				},
				Status: disgoPlatformStatus(status),
			}
			err = nil
		}
	})

	return puser, err
}

func (d *DisgoState) GetUser(ctx context.Context, idStr string) (*dovetypes.PlatformUser, error) {
	// Get from discord
	id, err := snowflake.Parse(idStr)

	if err != nil {
		return nil, err
	}

	user, err := d.config.Client.Rest().GetUser(id)

	if err != nil {
		return nil, err
	}

	return &dovetypes.PlatformUser{
		ID:          idStr,
		Username:    user.Username,
		Avatar:      user.EffectiveAvatarURL(),
		DisplayName: user.EffectiveName(),
		Bot:         user.Bot,
		Status:      dovetypes.PlatformStatusOffline,
		Flags:       disgoFlagsToArray(user),
	}, nil
}
