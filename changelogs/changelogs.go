package changelogs

import "popplio/types"

var Changelog = types.Changelog{
	Entries: []types.ChangelogEntry{
		{
			Version:          "4.3.0",
			Prerelease:       true,
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
