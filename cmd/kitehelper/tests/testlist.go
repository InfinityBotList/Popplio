// Contains the list of tests
package tests

import (
	"embed"
)

//go:embed all:custom
var customTests embed.FS

var testList = testset{
	Tests: []test{
		{
			name:       "db_fields_check.py",
			cmd:        []string{"python3"},
			customTest: "db_fields_check.py",
		},
	},
}
