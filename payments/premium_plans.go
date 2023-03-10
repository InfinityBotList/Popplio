package payments

type Plan struct {
	ID         string  `json:"id" validate:"required"`
	Name       string  `json:"name" validate:"required"`
	Benefit    string  `json:"benefit" validate:"required"`     // To be fixed
	TimePeriod int     `json:"time_period" validate:"required"` // In seconds
	Price      float32 `json:"price" validate:"required"`       // In USD
}

var Plans = []Plan{
	{
		"bronze",
		"Bronze Plan",
		"1 month of premium",
		24 * 30,
		1.99,
	},
	{
		"silver",
		"Silver Plan",
		"6 months of premium",
		24 * 30 * 6,
		4.99,
	},
	{
		"gold",
		"Gold Plan",
		"1 year of premium",
		24 * 30 * 12,
		7.99,
	},
}
