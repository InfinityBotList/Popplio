// This contains the staff/dev/certification apps
package apps

import (
	"popplio/state"
	"popplio/types"
	"popplio/validators/timex"

	"github.com/disgoorg/snowflake/v2"
	"github.com/infinitybotlist/eureka/uapi"
)

var Apps = []types.Position{
	{
		ID: "staff",
		Info: `Join the Infinity Staff Team and help us Approve, Deny and Certify Discord Bots. 

We are a welcoming and laid back team who is always willing to give new people an opportunity!`,
		Name:        "Staff Team",
		Tags:        []string{"Staff", "Bot Reviewing"},
		ReviewLogic: reviewLogicStaff,
		Cooldown:    48 * timex.Hour,
		Questions: []types.Question{
			{
				ID:          "experience",
				Question:    "Past server experience",
				Paragraph:   "Tell us any experience you have working for other servers or bot lists.",
				Placeholder: "I have worked at...",
				Short:       false,
			},
			{
				ID:          "strengths",
				Question:    "List some of your strengths",
				Paragraph:   "What are your strengths/abilities. How long can you be active?",
				Placeholder: "I am always online and active...",
				Short:       false,
			},
			{
				ID:          "situations",
				Question:    "Situation Examples",
				Paragraph:   "How would you handle: Mass Pings, Nukes and Raids etc.",
				Placeholder: "I would handle it by...",
				Short:       false,
			},
			{
				ID:          "reason",
				Question:    "Why do you want to join the staff team?",
				Paragraph:   "Why do you want to join the staff team? Be specific",
				Placeholder: "I want to join the staff team because...",
				Short:       false,
			},
			{
				ID:          "team-player",
				Question:    "What is a scenario in which you had to be a team player?",
				Paragraph:   "What is a scenario in which you had to be a team player? We want to know that you can collaborate effectively with us.",
				Placeholder: "I had to...",
				Short:       false,
			},
			{
				ID:          "about-you",
				Question:    "Tell us a little about yourself",
				Paragraph:   "Tell us a little about yourself. Its that simple!",
				Placeholder: "I am...",
				Short:       false,
			},
			{
				ID:          "other",
				Question:    "Anything else you want to add?",
				Paragraph:   "Anything else you want to add?",
				Placeholder: "Just state anything that doesn't hit anywhere else",
				Short:       true,
			},
		},
	},
	{
		ID: "qaqc",
		Info: `Join the Infinity Development's QAQC (Quality Assurance and Quality Control) Team! As being part of the QAQC team, you will help ensure that our services will always stay top-notch making everyone's experience with our services much better. We are a welcoming and laid back team who is always willing to give new people an opportunity!`,
		Name:        "QAQC Team",
		Tags:        []string{"Staff", "QAQC"},
		ReviewLogic: reviewLogicStaff,
		Cooldown:    120 * timex.Hour,
		Questions: []types.Question{
			{
				ID:          "experience",
				Question:    "Past server experience",
				Paragraph:   "Tell us any experience you have working for other servers or bot lists.",
				Placeholder: "I have worked at...",
				Short:       false,
			},
			{
				ID:          "strengths",
				Question:    "List some of your strengths",
				Paragraph:   "What are your strengths/abilities. How long can you be active?",
				Placeholder: "I am always online and active...",
				Short:       false,
			},
			{
				ID:          "situations",
				Question:    "Situation Examples",
				Paragraph:   "How would you handle: Mass Pings, Nukes and Raids etc.",
				Placeholder: "I would handle it by...",
				Short:       false,
			},
			{
				ID:          "reason",
				Question:    "Why do you want to join the staff team?",
				Paragraph:   "Why do you want to join the staff team? Be specific",
				Placeholder: "I want to join the staff team because...",
				Short:       false,
			},
			{
				ID:          "team-player",
				Question:    "What is a scenario in which you had to be a team player?",
				Paragraph:   "What is a scenario in which you had to be a team player? We want to know that you can collaborate effectively with us.",
				Placeholder: "I had to...",
				Short:       false,
			},
			{
				ID:          "about-you",
				Question:    "Tell us a little about yourself",
				Paragraph:   "Tell us a little about yourself. Its that simple!",
				Placeholder: "I am...",
				Short:       false,
			},
			{
				ID:          "other",
				Question:    "Anything else you want to add?",
				Paragraph:   "Anything else you want to add?",
				Placeholder: "Just state anything that doesn't hit anywhere else",
				Short:       true,
			},
		},
	},
	{
		ID: "dev",
		Info: `Join our Dev Team and help us update, manage and maintain all of the Infinity Services!.

Some experience in PostgreSQL and at least one of the below languages is required:

- Rust
- TypeScript (Javascript with type-safety)
- Go/Golang`,
		Name:     "Dev Team",
		Tags:     []string{"Golang", "Rust"},
		Cooldown: 48 * timex.Hour,
		Questions: []types.Question{
			{
				ID:          "sql-basics-1",
				Question:    "Write a SQL expression to select from a table named 'shop' the columns price (float) and quantity (integer) limited to 6 rows, ordered by the price in descending order",
				Paragraph:   "Answer the questions above using the most readable and (where possible) the most optimized SQL. Assume PostgreSQL 15 is being used and the 'pgxpool' (go) driver is being used.",
				Placeholder: "SQL Here",
				Short:       false,
			},
			{
				ID:          "sql-basics-2",
				Question:    "You now need to select all rows of the 'shop' table where rating (float) is above 5, the name (text) is similar (and case-insensitive) to $1 and categories (text[]) contains at least one element from $2 and all elements of $3 where $1, $2 and $3 are parameters of a parameterized query",
				Paragraph:   "Answer the questions above using the most readable and (where possible) the most optimized SQL. Assume PostgreSQL 15 is being used and the 'pgxpool' (go) driver is being used.",
				Placeholder: "SQL Here",
				Short:       false,
			},
			{
				ID:          "foobar",
				Question:    "Write a program that loops over all numbers from 1 to 7847 (inclusive). For every multiple of 7 and not 19, print 7 times the number and a uppercase A (on the same line), for every multiple of 19 and not 7, print a lowercase B and 5 more than the number divided by 4 and rounded (on the same line), for every multiple of both 7 and 19 print 'foobar'. Otherwise print 24 times the number",
				Paragraph:   "Answer the question above with the least amount of code. Use either Go 1.18 or the latest nightly version of Rust for all solutions. Your solution must NOT link to an external resource or library and you MUST justify all code with comments",
				Placeholder: "Code here...",
				Short:       false,
			},
			{
				ID:          "experience",
				Question:    "Do you have experience in Typescript, Rust and/or Go. Give examples of projects/code you have written",
				Paragraph:   "Do you have experience in Typescript, Rust and/or Go. Give examples of projects/code you have written.",
				Placeholder: "I have worked on...",
				Short:       false,
			},
			{
				ID:          "db",
				Question:    "Do you have Exprience with PostgreSQL. How much experience do you have?",
				Paragraph:   "Do you have Exprience with PostgreSQL",
				Placeholder: "I have used PostgreSQL for... and know...",
				Short:       false,
			},
			{
				ID:          "reason",
				Question:    "Why do you want to join the dev team?",
				Paragraph:   "Why do you want to join the dev team? Be specific",
				Placeholder: "I want to join the dev team because...",
				Short:       false,
			},
			{
				ID:          "team-player",
				Question:    "What is a scenario in which you had to be a team player?",
				Paragraph:   "What is a scenario in which you had to be a team player? We want to know that you can collaborate effectively with us.",
				Placeholder: "I had to...",
				Short:       false,
			},
			{
				ID:          "other",
				Question:    "Anything else you want to add?",
				Paragraph:   "Anything else you want to add?",
				Placeholder: "Just state anything that doesn't hit anywhere else",
				Short:       true,
			},
		},
	},
	{
		ID: "partners",
		Info: `Partner your Discord Bot, Discord Server or Business today! It's easier than ever before!

Some points to note:

- When you apply for a partnership, make sure that you are authorized to speak on the services behalf
- Infinity Development reserves the right to deny or cancel any partnership application at any time.
`,
		Name: "Partners",
		Tags: []string{"Advertising", "Business"},
		Questions: []types.Question{
			{
				ID:          "what",
				Question:    "What are you looking to partner with us for?",
				Paragraph:   "What are you looking to partner with us for? Be descriptive here",
				Placeholder: "I wish to partner a bot/website called Foobar because...",
				Short:       true,
			},
			{
				ID:          "why",
				Question:    "Why do you want to partner with us?",
				Paragraph:   "Why do you want to partner with us? Be specific",
				Placeholder: "I want to partner with Infinity Bot List because...",
				Short:       true,
			},
			{
				ID:          "how",
				Question:    "How will you promote us?",
				Paragraph:   "How will you promote Infinity Bot List? This could be a partner command or a link on your website!",
				Placeholder: "I will promote Infinity Bot List using...",
				Short:       true,
			},
			{
				ID:          "demo",
				Question:    "Do you have anything to showcase what you wish to partner with us?",
				Paragraph:   "Links to show us demos of what you're partnering or how many members your server or bot has.",
				Placeholder: "LINK 1 etc.",
				Short:       false,
			},
			{
				ID:          "other",
				Question:    "Anything else you want to add?",
				Paragraph:   "Anything else you want to add?",
				Placeholder: "Just state anything that doesn't hit anywhere else",
				Short:       true,
			},
		},
	},
	{
		ID:         "resubmit",
		Info:       `Resubmit your denied bot to the list!`,
		Name:       "Bot Resubmission",
		Hidden:     true, // Mostly done by ibl next
		ExtraLogic: extraLogicResubmit,
		Tags:       []string{"Resubmissions"},
		Questions: []types.Question{
			{
				ID:          "id",
				Question:    "Bot ID?",
				Paragraph:   "What is the bot ID?",
				Placeholder: "Bot ID",
				Short:       true,
			},
			{
				ID:          "reason",
				Question:    "Anything else?",
				Paragraph:   "Make sure you know why your bot was denied and that you have fixed the problem. If you don't know why your bot was denied, please contact us on Discord",
				Placeholder: "I believe.../I fixed.../The bot was offline because...",
				Short:       true,
			},
		},
	},
	{
		ID: "banappeal",
		Info: `<h3 class="text-2xl font-semibold">Hello There, Welcome</h3>
If you find yourself browsing or using this site, you should be disappointed. 

Here at Infinity we strive in providing our users a safe, curtious, drama free community and only ask that you follow a few simple rules. 

<span class="font-semibold">You have clearly done something to violate them or piss us off.</span>

Our Staff may approve or deny your ban appeal based on your actions and reason for ban and how much it may or may not have an effect on our community. 

We do not guarantee that your ban appeal will be approved and your ban be lifted. If you feel you have been banned for an unjust cause please Contact Us.

You can only have up to one ban appeal at any given point of time. Abusing the system will simply mean that you will not be unbanned and your ban appeal will be kept in queue.
		`,
		Name:        "Ban Appeal",
		Hidden:      true, // We don't want it to be prominently shown
		ReviewLogic: reviewLogicBanAppeal,
		Tags:        []string{"Ban Appeal"},
		Channel: func() snowflake.ID {
			return state.Config.Channels.BanAppeals
		},
		PositionDescription: func(d uapi.RouteData, p types.Position) string {
			return "User <@" + d.Auth.ID + "> wants to be unbanned now? :thinking:"
		},
		AllowedForBanned: true,
		BannedOnly:       true,
		Questions: []types.Question{
			{
				ID:          "reason",
				Question:    "Why were you banned?",
				Paragraph:   "Why were you banned? If you do not know, say so here and we will try to reach out.",
				Placeholder: "I was banned because...",
				Short:       false,
			},
			{
				ID:          "why",
				Question:    "Why do you feel you should be unbanned?",
				Paragraph:   "Why do you feel you should be unbanned from the list? Have you made any changes to your conduct. Have you reflected on what you did?",
				Placeholder: "I feel I should be unbanned because... and I have made changes to my conduct by... and I have reflected on what I did by...",
			},
			{
				ID:          "next-steps",
				Question:    "What will you do to avoid being banned in the future?",
				Paragraph:   "What will you do to avoid being banned in the future? Will you apologize if required?",
				Placeholder: "I will avoid being banned in the future by... and I...",
			},
		},
	},
	{
		ID:   "certification",
		Name: "Certification",
		Info: `So you want to apply for certification, eh?

Firstly, read the <a class="text-themable-400" href="/help/programs/certification" target="_blank">rules</a> to see if you qualify.

Then fill out the form below and wait for a staff member to review your application. If you are accepted, you're in the club!

You can only have one certification application at a time. If you have already applied, please wait for a staff member to review your application before applying again.
		`,
		ExtraLogic:  extraLogicCert,
		ReviewLogic: reviewLogicCert,
		Tags:        []string{"Certification"},
		Questions: []types.Question{
			{
				ID:          "id",
				Question:    "Bot ID?",
				Paragraph:   "What is the bot ID?",
				Placeholder: "Bot ID",
				Short:       true,
			},
			{
				ID:          "why",
				Question:    "Why do you want to be certified?",
				Paragraph:   "Why do you want to be certified? What special functionality does your bot provide that others dont?",
				Placeholder: "I want to be certified because...",
				Short:       false,
			},
			{
				ID:          "what",
				Question:    "What does your bot do?",
				Paragraph:   "What does your bot do? Does it have NSFW functionality?",
				Placeholder: "My bot does...",
				Short:       false,
			},
		},
	},
}
