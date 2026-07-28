package validatetable

import (
	"context"
	"fmt"
	"kitehelper/common"
	"os"
	"slices"
	"strings"

	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_pool *pgxpool.Pool
	Ctx   = context.Background()
)

func createUser(sp *common.SandboxPool, id string) error {
	return sp.Exec(Ctx, "INSERT INTO users (user_id, api_token, extra_links) VALUES ($1, $2, '{}')", id, crypto.RandString(256))
}

func ValidateTable(progname string, args []string) {
	if len(args) != 2 {
		fmt.Println("usage: validate-table <target/ref_column> <backer/column>")
		fmt.Println("example: validate-table reviews/author users/user_id")
		os.Exit(1)
	}

	target := args[0]
	backer := args[1]

	tgtSplit := strings.Split(target, "/")

	if len(tgtSplit) != 2 {
		fmt.Println("invalid target, not in format <target/ref_column>")
		os.Exit(1)
	}

	backSplit := strings.Split(backer, "/")

	if len(backSplit) != 2 {
		fmt.Println("invalid backer, not in format <backer/column>")
		os.Exit(1)
	}

	var err error
	_pool, err = pgxpool.New(Ctx, "postgres:///infinity")

	if err != nil {
		panic(err)
	}

	sp := common.NewSandboxPool(_pool)

	rows, err := sp.Query(Ctx, "SELECT "+tgtSplit[1]+" FROM "+tgtSplit[0]+" WHERE "+tgtSplit[1]+" IS NOT NULL")

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	sp.AllowCommit = true

	delIds := []string{}
	badIds := []string{}

	for rows.Next() {
		var id string

		err := rows.Scan(&id)

		if err != nil {
			panic(err)
		}

		if slices.Contains(delIds, id) {
			fmt.Println("ID", id, "already deleted")
			continue
		}

		// Ensure that the field also exists in the backer table
		var exists bool

		err = sp.QueryRow(Ctx, "SELECT EXISTS (SELECT 1 FROM "+backSplit[0]+" WHERE "+backSplit[1]+" = $1)", id).Scan(&exists)

		if err != nil {
			panic(err)
		}

		if !exists {
			fmt.Println("ID", id, "does not exist in", backSplit[0])

			if backSplit[0] == "users" && os.Getenv("CREATE_USERS") == "1" {
				fmt.Println("Creating user", id, "in users table")
				err = createUser(sp, id)

				if err != nil {
					panic(err)
				}

				fmt.Println("Created user", id, "in users table")
				continue
			}

			badIds = append(badIds, id)

			var ask bool

			if os.Getenv("S") == "" {
				ask = common.UserInputBoolean("Delete ID " + id + " from " + tgtSplit[0] + "?")
			} else {
				ask = os.Getenv("S") == "y"
			}

			if ask {
				err = sp.Exec(Ctx, "DELETE FROM "+tgtSplit[0]+" WHERE "+tgtSplit[1]+" = $1", id)

				if err != nil {
					panic(err)
				}

				fmt.Println("Deleted ID", id, "from", tgtSplit[0])
				delIds = append(delIds, id)
			}
		}
	}

	fmt.Println("Bad IDs:", badIds, "| len:", len(badIds))
}
