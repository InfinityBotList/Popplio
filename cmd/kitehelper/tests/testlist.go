// Contains the list of tests
package tests

import (
	"embed"
	"kitehelper/tests/gotests"
)

//go:embed custom
var customTests embed.FS

/*
EXAMPLE

	var testList = testset{
		Tests: []test{
			{
				name:       "strings-ts-enums-sync.py",
				cmd:        []string{"python3"},
				customTest: "strings-ts-enums-sync.py",
			},
			{
				name:       "route_no_httpexception.py",
				cmd:        []string{"python3"},
				customTest: "route_no_httpexception.py",
			},
			{
				name:       "enums_no_pydantic.py",
				cmd:        []string{"python3"},
				customTest: "enums_no_pydantic.py",
			},
			{
				name:       "blacklisted_imports.py",
				cmd:        []string{"python3"},
				customTest: "blacklisted_imports.py",
			},
			{
				name:       "docstring_ensure.py",
				cmd:        []string{"python3"},
				customTest: "docstring_ensure.py",
			},
			{
				name: "sunbeam (format)",
				cmd:  []string{"npm", "run", "format"},
				cwd:  "sunbeam",
			},
			{
				name: "sunbeam (lint)",
				cmd:  []string{"npm", "run", "lint-fix"},
				cwd:  "sunbeam",
			},
			{
				name: "fates (black)",
				cmd:  []string{"black", "fates"},
			},
			{
				name: "silverpelt (black)",
				cmd:  []string{"black", "silverpelt"},
			},
			{
				name: "libcommon (black)",
				cmd:  []string{"black", "libcommon"},
			},
		},
	}
*/
var testList = testset{
	Tests: []test{
		{
			name:       "team_perms_check.py",
			cmd:        []string{"python3"},
			customTest: "team_perms_check.py",
		},
		{
			name:   "team_perms_checks.go",
			goFunc: gotests.TestCheck,
		},
	},
}
