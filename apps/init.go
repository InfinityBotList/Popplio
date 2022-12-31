package apps

import (
	"sort"

	"github.com/go-playground/validator/v10"
)

var Stable = true

func Setup() {
	// Validate the order of the apps
	currOrder := []int{}
	for _, app := range Apps {
		currOrder = append(currOrder, app.Order)
	}

	// Sort the order
	sort.Ints(currOrder)

	// Ensure every number is one more than the last
	for i := 0; i < len(currOrder); i++ {
		if i == 0 {
			continue
		}

		if currOrder[i] != currOrder[i-1]+1 {
			panic("Order of apps is not sequential")
		}
	}

	for _, app := range Apps {
		appValidator := validator.New().Struct(app)

		if appValidator != nil {
			panic("App validation failed: " + appValidator.Error())
		}
	}
}
