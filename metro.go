package main

import (
	"github.com/MetroReviews/metro-integrase/types"
)

// Dummy adapter backend
type DummyAdapter struct {
}

func (adp DummyAdapter) GetConfig() types.ListConfig {
	return types.ListConfig{
		SecretKey:   "GE6DWnDfKEgDrjNcdKvOIDgmtJ1KNtzoAjXfNrKF5wU",
		ListID:      "008900c1-96c0-4fba-82f9-e4f0ba904d73",
		RequestLogs: true,
		StartupLogs: true,
		BindAddr:    ":8080",
		DomainName:  "https://spider.infinitybotlist.com",
	}
}

func (adp DummyAdapter) ClaimBot(bot *types.Bot) error {
	return nil
}

func (adp DummyAdapter) UnclaimBot(bot *types.Bot) error {
	return nil
}

func (adp DummyAdapter) ApproveBot(bot *types.Bot) error {
	return nil
}

func (adp DummyAdapter) DenyBot(bot *types.Bot) error {
	return nil
}

func (adp DummyAdapter) DataDelete(id string) error {
	return nil
}

func (adp DummyAdapter) DataRequest(id string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id": id,
	}, nil
}
