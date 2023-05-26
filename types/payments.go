package types

type PaymentPlan struct {
	ID         string  `json:"id" validate:"required"`
	Name       string  `json:"name" validate:"required"`
	Benefit    string  `json:"benefit" validate:"required"`     // To be fixed
	TimePeriod int     `json:"time_period" validate:"required"` // In seconds
	Price      float32 `json:"price" validate:"required"`       // In USD
}

type PlanList struct {
	Plans []PaymentPlan `json:"plans"`
}
