package changelogs

import "popplio/types"

var Changelog = types.Changelog{
	Entries: []types.ChangelogEntry{
		{
			Version: "4.8.0",
			Added:   []string{},
			Updated: []string{
				"Voting has been rewritten from the ground up to be more generic and abstracted to all entities. This will allow easier addition of server listing coming in the near future",
				"Bot Packs layout has been improved to be more user friendly",
				"Analytics on bot pages is now better handled using a target query parameter",
				"Removed use of custom x-client header for privacy, openness and security reasons. This also fixes some CORS errors that max be experienced by some browsers",
			},
			Removed: []string{},
		},
		{
			Version: "4.7.0",
			Added:   []string{},
			Updated: []string{
				"Internal backend change: service discovery has been added to the backend to allow for easier scaling if required",
			},
			Removed: []string{},
		},
		{
			Version: "4.6.0",
			Added: []string{
				"Added rules to help center",
				"Webhook logs has been added",
			},
			Updated: []string{
				"Internal backend changes to better support webhooks",
				"Internal frontend changes",
				"General bug fixes and improvements",
				"Improvements to webhook testing",
			},
			Removed: []string{
				"Webhooks v1 has been mostly removed outside of a set of whitelisted bots (13 to be precise)",
			},
		},
		{
			Version: "4.5.2",
			Added:   []string{},
			Updated: []string{
				"Dummy release to fix up CI",
			},
			Removed: []string{},
		},
		{
			Version: "4.5.1",
			Added:   []string{},
			Updated: []string{
				"Hotfix to fix errors in fetching blog posts due to a regression in 4.4.0 and 4.5.0",
			},
			Removed: []string{},
		},
		{
			Version: "4.5.0",
			Added:   []string{},
			Updated: []string{
				"htmlsanitize no longer sends the entire long description for sanitization, but instead now uses RPC queries to minimize bandwidth usage",
				"New layout, the new layout wastes less space and is less hacky layout wise (less use of absolute positioning)",
				"General bug fixes and improvements",
			},
			Removed: []string{
				"Small snippets under servers and shards. They were a waste of space and broke the new layout as well",
			},
		},
		{
			Version: "4.4.0",
			Added: []string{
				"Team webhooks have been added to Team Settings",
				"Alerts can now be managed through User Settings",
			},
			Updated: []string{
				"Refactored/rewrote the data fetching code for the site to make it more performant with less hacks and issues",
				"User settings is now tabbed using next/dynamic to reduce load/download times",
				"TypeScript has been added to most of the site",
				"Reviews has been refactored internally, no user-facing changes",
				"General bug fixes and improvements",
			},
			Removed: []string{},
		},
		{
			Version:          "4.3.0",
			ExtraDescription: "This version is mostly revamping existing code and systems!",
			Added: []string{
				"Internal: Proper TypeScript coverage!!!",
				"Changelogs and versioning have been added to allow you to keep track of the changes we're making",
				"Staff: Onboarding Quizzes and Onboarding V2. As an aside: We're hiring :)",
				"Redirects: teams/ID -> team/ID to make it easier to link to teams",
			},
			Updated: []string{
				"Some useless code blocks have been removed to reduce bundle sizes slightly",
				"Rewrote lots of the site to use up-to-date practices including proper TypeScript coverage along with increased performance",
				"Increased usage of next/dynamic (nextjs dynamic) to reduce page sizes, especially in team settings and bot settings pages",
				"Bot settings is now a paned layout (bot settings v2) to reduce clutter, page size and avoid confusion with buttons",
				"The 'Bot JSON' feature in Bot Settings has been replaced by a custom user-friendly viewer called 'Tree View'. Other uses of JSON printing have also been replaced",
				"API: Webhooks should be more performant with bad intent handling to allow for authentication tests",
				"Adding a bot with a team ownership now only makes a single API call to avoid failing in a partial state if the team cannot be transferred to as well to decrease time taken to add bots",
				"Reviews has been refactored internally, no user-facing changes",
				"Add bot has been improved with better validation and error management to fix several crashes experienced during testing and QA",
				"An infinite loop when accessing bot settings and possibly other pages as a not-logged-in user has been resolved. The site will now redirect to the login page in such a case",
				"Updated nextjs to 13.4.4",
				"Broken sorting in the 'About Us' page due to a broken string comparison sort has been removed. Staff team members are now sorted by permissions instead",
			},
			Removed: []string{},
		},
	},
}
