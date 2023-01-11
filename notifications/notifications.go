package notifications

func Setup() {
	go webPush()
	go premium()

	startTaskMgr()
}
