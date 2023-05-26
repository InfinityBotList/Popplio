package assets

import "popplio/types"

var Plans = []types.PaymentPlan{
	{
		ID:         "bronze",
		Name:       "Bronze Plan",
		Benefit:    "1 month of premium",
		TimePeriod: 24 * 30,
		Price:      1.99,
	},
	{
		ID:         "silver",
		Name:       "Silver Plan",
		Benefit:    "6 months of premium",
		TimePeriod: 24 * 30 * 6,
		Price:      4.99,
	},
	{
		ID:         "gold",
		Name:       "Gold Plan",
		Benefit:    "1 year of premium",
		TimePeriod: 24 * 30 * 12,
		Price:      7.99,
	},
}
