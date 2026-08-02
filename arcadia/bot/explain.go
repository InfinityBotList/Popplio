package bot

// explainText holds the explainme texts verbatim, including the leading and
// trailing whitespace the Rust raw strings carry. These are user-visible and
// frozen.
//
// Interpreted string literals are used rather than raw ones because the texts
// contain double backticks, which would terminate a Go raw string.
func explainText(option string) string {
	switch option {
	case "Claim":
		return "\n**Objective**\n" +
			"- Run ``/claim`` to claim the bot\n" +
			"\n**Explanation**       \n" +
			"Claiming bots will allow you to approve or deny bots after you review a bot! You should do this before you review a bot. Also, if you ever have to stop reviewing a bot. You can always unclaim the bot and ask another staff member, to help review the rest of this bot for you by doing ``/unclaim``\n" +
			"        "
	case "Testing":
		return "\n**Objective**\n" +
			"- Test the bot according to the rules stated in ``/staffguide``\n" +
			"\n**Explanation**\n" +
			"Test the bot according to the rules stated in ``/staffguide``. Note that you do not have to test every single command in the bot however *the majority* should work. During onboarding however, you must test each command! Once you have done this progress to using the ``/approve`` or ``/deny`` commands.\n" +
			"        "
	case "ApproveDeny":
		return "\n**Objective**\n" +
			"- Approve or deny the bot using either ``/approve`` or ``/deny``\n" +
			"\n**Explanation**\n" +
			"Approving or denying a bot will remove the bot from the queue and either approve it (it shows on the home page) or deny it (bot has to be resubmitted to the queue for restesting after issues are resolved).\n" +
			"        "
	case "Tips":
		return "\n1. If you are on mobile, consider setting ``embed`` to false in ``queue`` command.\n" +
			"\n*Explanation*\n" +
			"\nCopy-pasting embeds on mobile is not well supported.\n" +
			"\n2. Don't test *every* command, but test the main commands and functionality of the bot (outside of onboarding)\n" +
			"\n*Explanation*\n" +
			"\nTesting every command is very time-consuming. You should test the main commands and functionality of the bot (outside of onboarding where you should test every command and report on it in the feedback)\n" +
			"\n3. Check the bots description\n" +
			"\n*Explanation*\n" +
			"\nWhere possible, check the bots description to make sure it is readable and does not abuse character limits without good reason.\n" +
			"        "
	default:
		return "\nWelcome to IBL, ``explainme`` is a easy way to get an explanation on commands.\n" +
			"\n**The Steps**\n" +
			"\nThese are the steps *in order*\n" +
			"\n1. ``claim``: Claim the bot\n" +
			"2. ``testing``: Test the bot\n" +
			"3. ``approve/deny``: Approve or deny the bot\n" +
			"        "
	}
}
