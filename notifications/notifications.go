package notifications

import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Setup() {
	startTaskMgr()
}
