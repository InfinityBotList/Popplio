package partners

import "popplio/types"

var Partners = PartnerList{
	Featured: []*Partner{
		{
			ID:     "discord-bot-constructor",
			Name:   "DBC",
			Short:  "Get A Free Custom Discord Bot - No Ads, No Paywalls, And 200 Feature Selection",
			UserID: "503591563046420483",
			Image:  "https://cdn.infinitybots.xyz/images/png/discordbotconstructor-partner.png",
			Links: []types.Link{
				{
					Name:  "Discord",
					Value: "https://discord.com/invite/4YG4PwQg63",
				},
			},
		},
		{
			ID:     "discord-israel-hub",
			Name:   "Discord Israel Hub",
			Short:  "Welcome! \"Discord Israel Hub\" is Your place to unite with the Discord community and managers and grow together!",
			UserID: "952562403751641178",
			Image:  "https://cdn.infinitybots.xyz/images/gif/discordisraelhub-partner.gif",
			Links: []types.Link{
				{
					Name:  "Discord",
					Value: "https://discord.com/invite/pAZ4FHpyXf",
				},
			},
		},
	},
	BotPartners: []*Partner{
		{
			ID:     "trivia-bot",
			Name:   "Trivia Bot",
			Short:  "A Trivia/Quiz Discord Bot with over 90000 questions, nitro as prizes, teams, leaderboards, dashboard, and can give winners roles as rewards!",
			UserID: "189759562910400512",
			Image:  "https://cdn.infinitybots.xyz/images/png/triviabot-partner.png",
			Links: []types.Link{
				{
					Name:  "View Trivia",
					Value: "https://triviabot.co.uk/?source=infinitybotlist",
				},
				{
					Name:  "Discord",
					Value: "https://discord.com/invite/brainbox",
				},
			},
		},
		{
			ID:     "mystic-warrior-bot",
			Name:   "Mystic Warrior Bot",
			Short:  "Mystic Warrior is a bot with over features like moderation, economy, games, utilities, and more! We are also constantly evolving to meet the server requirements. We have 200+ commands.",
			UserID: "756786481808408576",
			Image:  "https://cdn.infinitybots.xyz/images/png/mysticwarriorbot-partner.png",
			Links: []types.Link{
				{
					Name:  "View Mystic",
					Value: "https://dashboard.mystic-development.repl.co/?source=infinitybotlist",
				},
				{
					Name:  "Discord",
					Value: "https://discord.com/invite/FcykbqrW8X",
				},
			},
		},
		{
			ID:     "anti-raid-bot",
			Name:   "Anti-Raid Bot",
			Short:  "One of the mostly good free and always improving moderation bots on Discord with over 500 servers!",
			UserID: "775855009421066262",
			Image:  "https://cdn.infinitybots.xyz/images/png/antiraid-partner.png",
			Links: []types.Link{
				{
					Name:  "View Website",
					Value: "https://antiraid.xyz/?source=infinitybotlist",
				},
				{
					Name:  "Discord",
					Value: "https://discord.gg/k3rcRBc8",
				},
			},
		},
		{
			ID:     "melon",
			Name:   "Melon",
			Short:  "Melon is a bot designed to keep your server safe and engage your members with limitless capabilities. Melon comes inbuilt with an advanced giveaway module, custom commands, starboard, reminders, suggestions, moderation, logging, donation logging and much more!",
			UserID: "759180080328081450",
			Image:  "https://cdn.infinitybots.xyz/images/png/melon-partner.png",
			Links: []types.Link{
				{
					Name:  "Website",
					Value: "https://www.melonbot.io/?source=infinitybotlist",
				},
				{
					Name:  "Discord",
					Value: "https://discord.com/invite/mXfYuMy92r",
				},
			},
		},
	},
	BotListPartners: []*Partner{
		{
			ID:     "topic-bot-list",
			Name:   "Topic Bot List",
			Short:  "Do you want to expand and improve your Discord bot? Topic Bot List is here for you!",
			UserID: "787241442770419722",
			Image:  "https://cdn.infinitybots.xyz/images/png/topiclist-partner.png",
			Links: []types.Link{
				{
					Name:  "Website",
					Value: "https://topiclist.xyz/?source=infinitybotlist",
				},
				{
					Name:  "Twitter",
					Value: "https://twitter.com/topicbotlist",
				},
			},
		},
	},
}
