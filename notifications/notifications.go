package notifications

func Setup() {
	go webPush()

	startTaskMgr()
}
