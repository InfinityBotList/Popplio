package state

import "github.com/plutov/paypal/v4"

func CreatePaypalClient() (*paypal.Client, error) {
	c, err := paypal.NewClient(Config.Meta.PaypalClientID, Config.Meta.PaypalSecret, func() string {
		if Config.Meta.PaypalUseSandbox {
			return paypal.APIBaseSandBox
		} else {
			return paypal.APIBaseLive
		}
	}())

	if err != nil {
		return nil, err
	}

	return c, nil
}
