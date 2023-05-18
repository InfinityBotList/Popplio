package stafftemplates

import "popplio/types"

var StaffTemplates = types.StaffTemplateList{
	Templates: []types.StaffTemplateMeta{
		{
			Name:        "Approval Templates",
			Icon:        "material-symbols:check",
			Description: "Choose the best approval reason based on the tags, title and text!",
			Templates: []types.StaffTemplate{
				{
					Name:  "Cool Bot",
					Emoji: "👏",
					Tags:  []string{"Excellent", "Perfect"},
					Description: `
Thank you for applying.


Your bot is cool and all the commands seem to be working as they should. <br />

Welcome to {listName}.
					`,
				},
				{
					Name:  "Nice Bot",
					Emoji: "😁",
					Tags:  []string{"Great", "Minor Errors"},
					Description: `
Very nice bot with a huge variety of features.


I have experienced little to no errors while testing this bots features.


Keep up the amazing work and welcome to {listName}
					`,
				},
				{
					Name:  "Minor Issues",
					Emoji: "🤨",
					Tags:  []string{"Good", "Minor Errors"},
					Description: `
Your bot seems to be having a few issues with {commands}.<br />

Everything else appears to work as intended. Welcome to {listName}
					`,
				},
				{
					Name:  "Passable",
					Emoji: "😐",
					Tags:  []string{"Passable", "Errors"},
					Description: `
Your bot seems to be having several issues with {commands} but works and is passable.


Welcome to {listName}`,
				},
			},
		},
		{
			Name:        "Denial Templates",
			Icon:        "material-symbols:close",
			Description: "Choose the best denial reason based on the tags, title and text!",
			Templates: []types.StaffTemplate{
				{
					Name:  "Bot Offline",
					Emoji: "📡",
					Tags:  []string{"Offline"},
					Description: `
Your bot was offline when we tried to review it.


Please get your bot online and re-apply.
					`,
				},
				{
					Name:  "Bot Offline",
					Emoji: "📡",
					Tags:  []string{"Offline"},
					Description: `Your bot was offline when we tried to review it.

Please get your bot online and re-apply.`,
				},
				{
					Name:  "API Abuse",
					Emoji: "🚫",
					Tags:  []string{"API Abuse"},
					Description: `Your bot has feature/commands that spams or abuses Discord's API.

This can cause your bot to get rate-limited frequently and can be considered Discord API Abuse.

Please fix the issue and re-apply.`,
				},
				{
					Name:  "Requires Admin",
					Emoji: "🛑",
					Tags:  []string{"Admin", "Permission"},
					Description: `Some of your bot's features require the bot itself to have the ADMINISTRATOR permission.

NO bot (outside of special exemptions decided on a case-by-case basis) requires administrator permissions to function securely.

You can dispute this in a ticket if you have valid reason though: e.g. Wick.

Please change your bot to only require the permissions it truly needs and re-apply.`,
				},
				{
					Name:  "Poor Page",
					Emoji: "🔑",
					Tags:  []string{"Low-Quality", "Description"},
					Description: `Your bot description or links is/are highly cryptic (source code, spam/low-quality content etc.)

Your long description should not consist of your bots code, advertising or low-quality spam, it should be about what your bot does, a command list etc.

Please rewrite your description to include more useful information about your bot and ensure all links added are also high-quality.

Friendly reminder to NEVER share your bots token with *anyone*`,
				},
				{
					Name:  "Abusive Page",
					Emoji: "👾",
					Tags:  []string{"Abusive", "Spam", "Junk", "Invisible Characters"},
					Description: `Your bot's long description has been found to be abusive:

- It is filled out with spam, junk or invisible characters

- It contains malicious links (such as phishing links or links to malware)

- It contains hate speech or other offensive content`,
				},
				{
					Name:  "Unresponsive",
					Emoji: "📵",
					Tags:  []string{"Offline", "Unresponsive"},
					Description: `Your bot has stopped responding during testing and due to this we are unable to continue testing it.

Friendly reminder that using repl.it and other shared hosting services is not reliable and is often a cause of this issue.`,
				},
				{
					Name:  "Open DM Commands",
					Emoji: "🛂",
					Tags:  []string{"DM Abuse", "Direct Messages", "DM Command"},
					Description: `Your bot has a DM command/function which allows anyone to DM a user which can be used maliciously. The following conditions must be met for such commands:

- The message your bot sends in DMs must state the author or that its from an anonymous user

- It must have a block/opt-out feature

Otherwise, remove this command entirely before resubmitting.`,
				},
				{
					Name:  "Presence/Status Abuse",
					Emoji: "🌀",
					Tags:  []string{"Presence Abuse", "Status Abuse", "Gateway Abuse"},
					Description: `Your bots presence changes every few seconds/too quickly and is as such considered Discord API abuse.

The maximum frequency your bot can change its status is *5 times per 20 seconds.* although we implore you to change it to something more reasonable, such as every 120 seconds.`,
				},
				{
					Name:  "No Entrypoint",
					Emoji: "❌",
					Tags:  []string{"No Help Command", "No Point of Entry", "Unresponsive"},
					Description: `Your bot doesnt have a (working) help command or obvious point of entry.

Please make sure your bot has a help command or has an explanation in the bot description.

Note that if you are using slash commands, then you do not need a help command`,
				},
				{
					Name:  "Suicide/Gore",
					Emoji: "🔪",
					Tags:  []string{"Death", "Suicide", "Gore"},
					Description: `Your bot has a suicide command which is considered as glorification/promotion of suicide, which is against Discord ToS.

Please remove this command entirely.`,
				},
				{
					Name:  "Cross Promotion",
					Emoji: "👀",
					Tags:  []string{"Promotion", "Spam", "Low-Quality Description"},
					Description: `Your bots page only contains a link/button to another bot listing website without much substance here!

Please improve your long description and resubmit your bot!!`,
				},
				{
					Name:  "Seizure Risk",
					Emoji: "🏨",
					Tags:  []string{"Seizure", "Health", "Flashy"},
					Description: `Your bots commands have emojis or gifs that could cause epileptic seizures due to its flashy and flickering nature.

Please remove all content of such nature in your commands.`,
				},
				{
					Name:  "NSFW",
					Emoji: "🔞",
					Tags:  []string{"NSFW", "Not-gated"},
					Description: `Your bot does not properly gate NSFW commands to NSFW channels and this is against Discord ToS.

Please make sure that all the NSFW functions are locked for [NSFW channels](https://support.discord.com/hc/en-us/articles/115000084051-NSFW-Channels-and-Content).`,
				},
				{
					Name:  "Blatant Fork",
					Emoji: "🤖",
					Tags:  []string{"BDFD", "Autocode", "Red Bot"},
					Description: `Your bot seems to be an unmodified instance of {linkToBot}.

We don't allow unmodified clones of other bots or bot creation services. Please note for BotGhost/BDFD/RedBot clones in particular, we require five or more custom commands.`,
				},
				{
					Name:  "Third Party Ads",
					Emoji: "🔖",
					Tags:  []string{"J4J", "Join4Join", "Ads"},
					Description: `Your bot is promoting sponsors/partners/servers.

Bots are not allowed to use the Discord API to advertise/promote third-party services and as stated in their [Terms](https://i.imgur.com/eTLRu3m.png), you may not use the APIs in any way to target users with advertisements or marketing.

If your bot has a Join4Join feature, please read Discord's stance regarding bots of this nature here or join the [Discord Developers](https://discord.gg/discord-developers) server for more information.`,
				},
				{
					Name:  "Flagged Bot",
					Emoji: "⚠️",
					Tags:  []string{"Flagged", "Banned", "Unverified"},
					Description: `Your bot has been flagged by Discord for one of the following reasons (spam, abusive behaviour, reaching guild limit without verification, verified successfully by Discord but using privileged intents that your application was not whitelisted for.) which results in us being unable to test your bot.

For more information, please reach out Discord directly [here](https://dis.gd/contact) if you have questions.

Please resolve this issue before reapplying.`,
				},
				{
					Name:  "No Functions",
					Emoji: "🤷‍♂️",
					Tags:  []string{"Nonfunctional"},
					Description: `Your bot doesn't have any actual (functioning) features/commands.

We require bots to have a minimum of at least seven working commands not including the help and about commands. Please add some features and/or commands to your bot before re-applying!`,
				},
				{
					Name:  "Banned Owner",
					Emoji: "⚖️",
					Tags:  []string{"Banned", "Banned Owner"},
					Description: `The primary owner of this bot was banned from our discord server. As such they are prohibited from adding any bots.

They can however appeal any and all bans by contacting our support team (or asking a friend who can)`,
				},
				{
					Name:  "Admin Only Bots",
					Emoji: "⚔️",
					Tags:  []string{"Admin", "Permissions"},
					Description: `Your bot is asking for the admin permission on invite. No bot should require this kind of permission to function correctly (excluding some exceptions decided on a case-by-case basis).

Please properly setup command permissions on your bots commands to what they *actually need* and reapply.`,
				},
				{
					Name:  "Bad Invite",
					Emoji: "🤔",
					Tags:  []string{"Unknown", "Invite"},
					Description: `There is an unknown application/insert error error when trying to invite your bot.

Please make sure that the application ID you entered is correct, you have a bot user with your application and your bot application wasn't deleted (most likely cause).

You must fix this issue before reapplying.`,
				},
				{
					Name:  "Broken Commands",
					Emoji: "💻",
					Tags:  []string{"Commands", "Broken"},
					Description: `The majority of your commands listed on your bots page, or help command do not provide a response, or do not seem to function/work.

Please resolve this issue and resubmit!`,
				},
				{
					Name:  "Sensitive Commands",
					Emoji: "🔓",
					Tags:  []string{"Dev Only", "Sensitive"},
					Description: `Your bot has an owner only command(s) that allows users to access potential vulnerabilities or features that should be locked to developers.

Please lock these commands for developers/owners only and re-apply.`,
				},
				{
					Name:  "Multiple Instances",
					Emoji: "👨‍👩‍👧‍👦",
					Tags:  []string{"Duplicate"},
					Description: `Your bot application seems to be running multiple instances, which could cause unhandled ratelimits and api abuse, as well as spam.


Please be sure that your bot isnt running on multiple instances prior to resubmitting.`,
				},
				{
					Name:  "Youtube",
					Emoji: "📺",
					Tags:  []string{"Youtube", "Music"},
					Description: `Alright so, pretty much everyone got to express their opinion recently regarding YouTube based music bots and we decided on not allowing them anymore period!

Our rules have been updated both on the website and in our main server to reflect this decision. Youtube bots are no longer allowed and is a good way to get your bot unverified/denied verification by Discord

Exceptions:

Note that copyright must still be followed

- Spotify API usage is probably fine
- SoundCloud API usage is fine`,
				},
			},
		},
	},
}
