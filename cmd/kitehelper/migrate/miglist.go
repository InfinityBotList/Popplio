package migrate

// Contains the list of migrations

var (
// statusBoldErr = color.New(color.Bold, color.FgRed).PrintlnFunc()
)

var migs = []migration{
	{
		name: "Example migration",
		function: func() {
			if !colExists("bots", "id") {
				alrMigrated()
				return
			}

			if tableExists("foobar") {
				alrMigrated()
				return
			}

			// DO SOMETHING HERE
			_, err := pgpool.Exec(ctx, "ALTER TABLE bots DROP COLUMN id")

			if err != nil {
				panic(err)
			}
		},
	},
}
