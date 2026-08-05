package bot

func registerCommands() {
	register(
		cmdSetup(),
		cmdUpdate(),
		cmdDelete(),
		cmdLeaderboard(),
		cmdStats(),
	)
}
