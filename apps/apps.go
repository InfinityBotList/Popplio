// This contains the staff/dev/certification apps
package apps

var Apps = map[string]Position{
	"staff": {
		Order: 1,
		Info: `Join the Infinity Staff Team and help us Approve, Deny and Certify Discord Bots. 

We are a welcoming and laid back team who is always willing to give new people an opportunity!`,
		Name: "Staff Team",
		Tags: []string{"Staff", "Bot Reviewing"},
		Interview: []Question{
			{
				ID:          "motive",
				Question:    "Why did you apply for the staff team position?",
				Paragraph:   "Why did you apply for this role? Be specific. We want to know how you can help Infinity Bot List and why you wish to",
				Placeholder: "I applied because...",
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
		},
		Questions: []Question{
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
				Paragraph:   "Tell us a little bit about yourself.",
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
				ID:          "other",
				Question:    "Anything else you want to add?",
				Paragraph:   "Anything else you want to add?",
				Placeholder: "Just state anything that doesn't hit anywhere else",
				Short:       true,
			},
		},
	},
	"dev": {
		Order: 2,
		Info: `Join our Dev Team and help us update, manage and maintain all of the Infinity Services!.

Some experience in PostgreSQL and at least one of the below languages is required:

- Rust
- TypeScript (Javascript with type-safety)
- Go/Golang`,
		Name: "Dev Team",
		Tags: []string{"Golang", "Rust"},
		Interview: []Question{
			{
				ID:          "motive",
				Question:    "Why did you apply for the dev team position?",
				Paragraph:   "Why did you apply for this role? Be specific. We want to know how you can help Infinity Bot List and why you wish to",
				Placeholder: "I applied because...",
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
				ID:          "some-work",
				Question:    "What is some of the projects you have done? Can you share some links with us?",
				Paragraph:   "What is some of the projects you have done? Can you share some links with us? We want to see your finest works",
				Placeholder: "Some work I did...",
				Short:       false,
			},
			{
				ID:          "about-you",
				Question:    "Tell us a little about yourself",
				Paragraph:   "Tell us a little about yourself. Its that simple!",
				Placeholder: "I am...",
				Short:       false,
			},
		},
		Questions: []Question{
			{
				ID:          "sql-basic",
				Question:    "Write a SQL expression to select from a table named 'shop' the columns price (float) and quantity (integer) limited to 6 rows, ordered by the price in descending order",
				Paragraph:   "Answer the questions above using the most readable and (where possible) the most optimized SQL. Assume PostgreSQL 15 is being used and the 'pgxpool' (go) driver is being used.",
				Placeholder: "SQL Here",
				Short:       false,
			},
			{
				ID:          "sql-basic",
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
				ID:          "strengths",
				Question:    "What are your strengths in coding",
				Paragraph:   "What are your strengths in coding",
				Placeholder: "I am good at...",
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
		},
	},
	"partners": {
		Order: 3,
		Info: `Partner your Discord Bot, Discord Server or Business today! It's easier than ever before!

Some points to note:

- When you apply for a partnership, make sure that you are authorized to speak on the services behalf
- Infinity Development reserves the right to deny or cancel any partnership application at any time.
`,
		Name: "Partners",
		Tags: []string{"Advertising", "Business"},
		Questions: []Question{
			{
				ID:          "what",
				Question:    "What are you looking to partner with us for?",
				Paragraph:   "What are you looking to partner with us for? Be descriptive here",
				Placeholder: "I wish to partner a bot/website called Foobar because...",
				Short:       false,
			},
			{
				ID:          "why",
				Question:    "Why do you want to partner with us?",
				Paragraph:   "Why do you want to partner with us? Be specific",
				Placeholder: "I want to partner with Infinity Bot List because...",
				Short:       false,
			},
			{
				ID:          "how",
				Question:    "How will you promote us?",
				Paragraph:   "How will you promote Infinity Bot List? This could be a partner command or a link on your website!",
				Placeholder: "I will promote Infinity Bot List using...",
				Short:       false,
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
	"resubmit": {
		Order:      4,
		Info:       `Resubmit your denied bot to the list!`,
		Name:       "Bot Resubmission",
		Hidden:     true, // Mostly done by ibl next
		ExtraLogic: extraLogicResubmit,
		Tags:       []string{"Resubmissions"},
		Questions: []Question{
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
}
