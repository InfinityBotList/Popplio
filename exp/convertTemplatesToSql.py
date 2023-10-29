import asyncpg

data = {
    "templates": [
      {
        "name": "Approval Templates",
        "icon": "material-symbols:check",
        "description": "Choose the best approval reason based on the tags, title and text!",
        "templates": [
          {
            "name": "Cool Bot",
            "emoji": "👏",
            "tags": [
              "Excellent",
              "Perfect"
            ],
            "description": "\nThank you for applying.\n\n\nYour bot is cool and all the commands seem to be working as they should. <br />\n\nWelcome to {listName}.\n\t\t\t\t\t"
          },
          {
            "name": "Nice Bot",
            "emoji": "😁",
            "tags": [
              "Great",
              "Minor Errors"
            ],
            "description": "\nVery nice bot with a huge variety of features.\n\n\nI have experienced little to no errors while testing this bots features.\n\n\nKeep up the amazing work and welcome to {listName}\n\t\t\t\t\t"
          },
          {
            "name": "Minor Issues",
            "emoji": "🤨",
            "tags": [
              "Good",
              "Minor Errors"
            ],
            "description": "\nYour bot seems to be having a few issues with {commands}.<br />\n\nEverything else appears to work as intended. Welcome to {listName}\n\t\t\t\t\t"
          },
          {
            "name": "Passable",
            "emoji": "😐",
            "tags": [
              "Passable",
              "Errors"
            ],
            "description": "\nYour bot seems to be having several issues with {commands} but works and is passable.\n\n\nWelcome to {listName}"
          }
        ]
      },
      {
        "name": "Denial Templates",
        "icon": "material-symbols:close",
        "description": "Choose the best denial reason based on the tags, title and text!",
        "templates": [
          {
            "name": "Bot Offline",
            "emoji": "📡",
            "tags": [
              "Offline"
            ],
            "description": "\nYour bot was offline when we tried to review it.\n\n\nPlease get your bot online and re-apply.\n\t\t\t\t\t"
          },
          {
            "name": "Bot Offline",
            "emoji": "📡",
            "tags": [
              "Offline"
            ],
            "description": "Your bot was offline when we tried to review it.\n\nPlease get your bot online and re-apply."
          },
          {
            "name": "API Abuse",
            "emoji": "🚫",
            "tags": [
              "API Abuse"
            ],
            "description": "Your bot has feature/commands that spams or abuses Discord's API.\n\nThis can cause your bot to get rate-limited frequently and can be considered Discord API Abuse.\n\nPlease fix the issue and re-apply."
          },
          {
            "name": "Requires Admin",
            "emoji": "🛑",
            "tags": [
              "Admin",
              "Permission"
            ],
            "description": "Some of your bot's features require the bot itself to have the ADMINISTRATOR permission.\n\nNO bot (outside of special exemptions decided on a case-by-case basis) requires administrator permissions to function securely.\n\nYou can dispute this in a ticket if you have valid reason though: e.g. Wick.\n\nPlease change your bot to only require the permissions it truly needs and re-apply."
          },
          {
            "name": "Poor Page",
            "emoji": "🔑",
            "tags": [
              "Low-Quality",
              "Description"
            ],
            "description": "Your bot description or links is/are highly cryptic (source code, spam/low-quality content etc.)\n\nYour long description should not consist of your bots code, advertising or low-quality spam, it should be about what your bot does, a command list etc.\n\nPlease rewrite your description to include more useful information about your bot and ensure all links added are also high-quality.\n\nFriendly reminder to NEVER share your bots token with *anyone*"
          },
          {
            "name": "Abusive Page",
            "emoji": "👾",
            "tags": [
              "Abusive",
              "Spam",
              "Junk",
              "Invisible Characters"
            ],
            "description": "Your bot's long description has been found to be abusive:\n\n- It is filled out with spam, junk or invisible characters\n\n- It contains malicious links (such as phishing links or links to malware)\n\n- It contains hate speech or other offensive content"
          },
          {
            "name": "Unresponsive",
            "emoji": "📵",
            "tags": [
              "Offline",
              "Unresponsive"
            ],
            "description": "Your bot has stopped responding during testing and due to this we are unable to continue testing it.\n\nFriendly reminder that using repl.it and other shared hosting services is not reliable and is often a cause of this issue."
          },
          {
            "name": "Open DM Commands",
            "emoji": "🛂",
            "tags": [
              "DM Abuse",
              "Direct Messages",
              "DM Command"
            ],
            "description": "Your bot has a DM command/function which allows anyone to DM a user which can be used maliciously. The following conditions must be met for such commands:\n\n- The message your bot sends in DMs must state the author or that its from an anonymous user\n\n- It must have a block/opt-out feature\n\nOtherwise, remove this command entirely before resubmitting."
          },
          {
            "name": "Presence/Status Abuse",
            "emoji": "🌀",
            "tags": [
              "Presence Abuse",
              "Status Abuse",
              "Gateway Abuse"
            ],
            "description": "Your bots presence changes every few seconds/too quickly and is as such considered Discord API abuse.\n\nThe maximum frequency your bot can change its status is *5 times per 20 seconds.* although we implore you to change it to something more reasonable, such as every 120 seconds."
          },
          {
            "name": "No Entrypoint",
            "emoji": "❌",
            "tags": [
              "No Help Command",
              "No Point of Entry",
              "Unresponsive"
            ],
            "description": "Your bot doesnt have a (working) help command or obvious point of entry.\n\nPlease make sure your bot has a help command or has an explanation in the bot description.\n\nNote that if you are using slash commands, then you do not need a help command"
          },
          {
            "name": "Suicide/Gore",
            "emoji": "🔪",
            "tags": [
              "Death",
              "Suicide",
              "Gore"
            ],
            "description": "Your bot has a suicide command which is considered as glorification/promotion of suicide, which is against Discord ToS.\n\nPlease remove this command entirely."
          },
          {
            "name": "Cross Promotion",
            "emoji": "👀",
            "tags": [
              "Promotion",
              "Spam",
              "Low-Quality Description"
            ],
            "description": "Your bots page only contains a link/button to another bot listing website without much substance here!\n\nPlease improve your long description and resubmit your bot!!"
          },
          {
            "name": "Seizure Risk",
            "emoji": "🏨",
            "tags": [
              "Seizure",
              "Health",
              "Flashy"
            ],
            "description": "Your bots commands have emojis or gifs that could cause epileptic seizures due to its flashy and flickering nature.\n\nPlease remove all content of such nature in your commands."
          },
          {
            "name": "NSFW",
            "emoji": "🔞",
            "tags": [
              "NSFW",
              "Not-gated"
            ],
            "description": "Your bot does not properly gate NSFW commands to NSFW channels and this is against Discord ToS.\n\nPlease make sure that all the NSFW functions are locked for [NSFW channels](https://support.discord.com/hc/en-us/articles/115000084051-NSFW-Channels-and-Content)."
          },
          {
            "name": "Blatant Fork",
            "emoji": "🤖",
            "tags": [
              "BDFD",
              "Autocode",
              "Red Bot"
            ],
            "description": "Your bot seems to be an unmodified instance of {linkToBot}.\n\nWe don't allow unmodified clones of other bots or bot creation services. Please note for BotGhost/BDFD/RedBot clones in particular, we require five or more custom commands."
          },
          {
            "name": "Third Party Ads",
            "emoji": "🔖",
            "tags": [
              "J4J",
              "Join4Join",
              "Ads"
            ],
            "description": "Your bot is promoting sponsors/partners/servers.\n\nBots are not allowed to use the Discord API to advertise/promote third-party services and as stated in their [Terms](https://i.imgur.com/eTLRu3m.png), you may not use the APIs in any way to target users with advertisements or marketing.\n\nIf your bot has a Join4Join feature, please read Discord's stance regarding bots of this nature here or join the [Discord Developers](https://discord.gg/discord-developers) server for more information."
          },
          {
            "name": "Flagged Bot",
            "emoji": "⚠️",
            "tags": [
              "Flagged",
              "Banned",
              "Unverified"
            ],
            "description": "Your bot has been flagged by Discord for one of the following reasons (spam, abusive behaviour, reaching guild limit without verification, verified successfully by Discord but using privileged intents that your application was not whitelisted for.) which results in us being unable to test your bot.\n\nFor more information, please reach out Discord directly [here](https://dis.gd/contact) if you have questions.\n\nPlease resolve this issue before reapplying."
          },
          {
            "name": "No Functions",
            "emoji": "🤷‍♂️",
            "tags": [
              "Nonfunctional"
            ],
            "description": "Your bot doesn't have any actual (functioning) features/commands.\n\nWe require bots to have a minimum of at least seven working commands not including the help and about commands. Please add some features and/or commands to your bot before re-applying!"
          },
          {
            "name": "Banned Owner",
            "emoji": "⚖️",
            "tags": [
              "Banned",
              "Banned Owner"
            ],
            "description": "The primary owner of this bot was banned from our discord server. As such they are prohibited from adding any bots.\n\nThey can however appeal any and all bans by contacting our support team (or asking a friend who can)"
          },
          {
            "name": "Admin Only Bots",
            "emoji": "⚔️",
            "tags": [
              "Admin",
              "Permissions"
            ],
            "description": "Your bot is asking for the admin permission on invite. No bot should require this kind of permission to function correctly (excluding some exceptions decided on a case-by-case basis).\n\nPlease properly setup command permissions on your bots commands to what they *actually need* and reapply."
          },
          {
            "name": "Bad Invite",
            "emoji": "🤔",
            "tags": [
              "Unknown",
              "Invite"
            ],
            "description": "There is an unknown application/insert error error when trying to invite your bot.\n\nPlease make sure that the application ID you entered is correct, you have a bot user with your application and your bot application wasn't deleted (most likely cause).\n\nYou must fix this issue before reapplying."
          },
          {
            "name": "Broken Commands",
            "emoji": "💻",
            "tags": [
              "Commands",
              "Broken"
            ],
            "description": "The majority of your commands listed on your bots page, or help command do not provide a response, or do not seem to function/work.\n\nPlease resolve this issue and resubmit!"
          },
          {
            "name": "Sensitive Commands",
            "emoji": "🔓",
            "tags": [
              "Dev Only",
              "Sensitive"
            ],
            "description": "Your bot has an owner only command(s) that allows users to access potential vulnerabilities or features that should be locked to developers.\n\nPlease lock these commands for developers/owners only and re-apply."
          },
          {
            "name": "Multiple Instances",
            "emoji": "👨‍👩‍👧‍👦",
            "tags": [
              "Duplicate"
            ],
            "description": "Your bot application seems to be running multiple instances, which could cause unhandled ratelimits and api abuse, as well as spam.\n\n\nPlease be sure that your bot isnt running on multiple instances prior to resubmitting."
          },
          {
            "name": "Youtube",
            "emoji": "📺",
            "tags": [
              "Youtube",
              "Music"
            ],
            "description": "Youtube bots are no longer allowed and is a good way to get your bot unverified/denied verification by Discord.\n\nOur rules have been updated both on the website and in our main server to reflect this decision\n\nExceptions:\n\nNote that copyright must still be followed\n\n- Spotify API usage is probably fine\n- SoundCloud API usage is fine"
          }
        ]
      }
    ]
  }

async def load():
    client = await asyncpg.create_pool("postgresql:///infinity")

    if not client:
        print("Failed to connect to database [pool is None]")
        return

    for templates in data["templates"]:
        if templates["name"] == "Approval Templates":
            type = "approval"
        else:
            type = "denial"
        
        for template in templates["templates"]:
            await client.execute("INSERT INTO staff_templates (name, emoji, tags, description, type) VALUES ($1, $2, $3, $4, $5)", template["name"], template["emoji"], template["tags"], template["description"], type)

    await client.close()

if __name__ == "__main__":
    import asyncio
    asyncio.run(load())