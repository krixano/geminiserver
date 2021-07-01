package migration

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"database/sql"

	"github.com/krixano/ponixserver/src/db"
	"github.com/krixano/ponixserver/src/migration/migrations"
	"github.com/krixano/ponixserver/src/migration/types"
	"github.com/krixano/ponixserver/src/gemini"
	//"github.com/nakagami/firebirdsql"
	"github.com/spf13/cobra"
)

var listMigrations bool

func init() {
	migrateCommand := &cobra.Command{
		Use:   "migrate [Database] [target migration id]",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) <= 0 {
				fmt.Printf("Error: No database specified\n")
				os.Exit(1)
			}

			db := db.StringToDB(args[0])
			if listMigrations {
				ListMigrations(db)
				return
			}

			targetVersion := time.Time{}
			if len(args) > 1 {
				var err error
				targetVersion, err = time.Parse(time.RFC3339, args[1])
				if err != nil {
					fmt.Printf("ERROR: bad version string: %v", err)
					os.Exit(1)
				}
			}
			Migrate(db, types.MigrationVersion(targetVersion))
		},
	}
	migrateCommand.Flags().BoolVar(&listMigrations, "list", false, "List available migrations")

	makeMigrationCommand := &cobra.Command{
		Use:   "makemigration <name> <description>...",
		Short: "Create a new database migration file",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Println("You must provide a name and a description.")
				os.Exit(1)
			}

			name := args[0]
			description := strings.Join(args[1:], " ")

			MakeMigration(name, description)
		},
	}

	gemini.GeminiCommand.AddCommand(migrateCommand)
	gemini.GeminiCommand.AddCommand(makeMigrationCommand)
}

func getSortedMigrationVersions(All map[types.MigrationVersion]types.Migration) []types.MigrationVersion { // TODO
	var allVersions []types.MigrationVersion
	for migrationTime, _ := range All {
		allVersions = append(allVersions, migrationTime)
	}
	sort.Slice(allVersions, func(i, j int) bool {
		return allVersions[i].Before(allVersions[j])
	})

	return allVersions
}

func getCurrentVersion(conn *sql.DB) (types.MigrationVersion, error) {
	var currentVersion time.Time
	row := conn.QueryRowContext(context.Background(), "SELECT version FROM migration")
	err := row.Scan(&currentVersion)
	if err != nil {
		return types.MigrationVersion{}, err
	}
	currentVersion = currentVersion.UTC()

	return types.MigrationVersion(currentVersion), nil
}

func ListMigrations(database db.DBType) {
	conn := db.NewConn(database)
	defer conn.Close(/*context.Background()*/)

	All := migrations.GetMap(database)

	currentVersion, _ := getCurrentVersion(conn)
	for _, version := range getSortedMigrationVersions(All) {
		migration := All[version]
		indicator := "  "
		if version.Equal(currentVersion) {
			indicator = "✔ "
		}
		fmt.Printf("%s%v %s (%s: %s)\n", indicator, version, migration.DB(), migration.Name(), migration.Description())
	}
}

func Migrate(database db.DBType, targetVersion types.MigrationVersion) {
	conn := db.NewConn(database)
	defer conn.Close(/*context.Background()*/)

	All := migrations.GetMap(database)

	// create migration table
	_, err := conn.ExecContext(context.Background(), `
EXECUTE BLOCK AS BEGIN
if (not exists(select 1 from rdb$relations where rdb$relation_name = 'MIGRATION')) then
execute statement 'create table migration ( version TIMESTAMP );';
END
	`)
	if err != nil {
		panic(fmt.Errorf("failed to create migration table: %w", err))
	}

	// ensure there is a row
	row := conn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM migration")
	var numRows int
	err = row.Scan(&numRows)
	if err != nil {
		panic(err)
	}
	if numRows < 1 {
		_, err := conn.ExecContext(context.Background(), "INSERT INTO migration (version) VALUES (?)", time.Time{})
		if err != nil {
			panic(fmt.Errorf("failed to insert initial migration row: %w", err))
		}
	}

	// run migrations
	currentVersion, err := getCurrentVersion(conn)
	if err != nil {
		panic(fmt.Errorf("failed to get current version: %w", err))
	}
	if currentVersion.IsZero() {
		fmt.Println("This is the first time you have run database migrations.")
	} else {
		fmt.Printf("Current version: %s\n", currentVersion.String())
	}

	allVersions := getSortedMigrationVersions(All)
	if targetVersion.IsZero() {
		targetVersion = allVersions[len(allVersions)-1]
	}

	currentIndex := -1
	targetIndex := -1
	for i, version := range allVersions {
		if currentVersion.Equal(version) {
			currentIndex = i
		}
		if targetVersion.Equal(version) {
			targetIndex = i
		}
	}

	if targetIndex < 0 {
		fmt.Printf("ERROR: Could not find migration with version %v\n", targetVersion)
		return
	}

	if currentIndex < targetIndex {
		// roll forward
		for i := currentIndex + 1; i <= targetIndex; i++ {
			version := allVersions[i]
			fmt.Printf("Applying migration %v\n", version)
			migration := All[version]

			tx, err := conn.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				panic(fmt.Errorf("failed to start transaction: %w", err))
			}
			defer tx.Rollback()

			err = migration.Up(tx)
			if err != nil {
				fmt.Printf("MIGRATION FAILED for migration %v.\n", version)
				fmt.Printf("Error: %v\n", err)
				return
			}

			_, err = tx.ExecContext(context.Background(), "UPDATE migration SET version = ?", time.Time(version))
			if err != nil {
				panic(fmt.Errorf("failed to update version in migrations table: %w", err))
			}

			err = tx.Commit()
			if err != nil {
				panic(fmt.Errorf("failed to commit transaction: %w", err))
			}
		}
	} else if currentIndex > targetIndex {
		// roll back
		for i := currentIndex; i > targetIndex; i-- {
			version := allVersions[i]
			previousVersion := types.MigrationVersion{}
			if i > 0 {
				previousVersion = allVersions[i-1]
			}

			tx, err := conn.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				panic(fmt.Errorf("failed to start transaction: %w", err))
			}
			defer tx.Rollback()

			fmt.Printf("Rolling back migration %v\n", version)
			migration := All[version]
			err = migration.Down(tx)
			if err != nil {
				fmt.Printf("MIGRATION FAILED for migration %v.\n", version)
				fmt.Printf("Error: %v\n", err)
				return
			}

			_, err = tx.ExecContext(context.Background(), "UPDATE migration SET version = ?", time.Time(previousVersion))
			if err != nil {
				panic(fmt.Errorf("failed to update version in migrations table: %w", err))
			}

			err = tx.Commit()
			if err != nil {
				panic(fmt.Errorf("failed to commit transaction: %w", err))
			}
		}
	} else {
		fmt.Println("Already migrated; nothing to do.")
	}
}

//go:embed migrationTemplate.txt
var migrationTemplate string

func MakeMigration(name, description string) {
	result := migrationTemplate
	result = strings.ReplaceAll(result, "%NAME%", name)
	result = strings.ReplaceAll(result, "%DESCRIPTION%", fmt.Sprintf("%#v", description))

	now := time.Now().UTC()
	nowConstructor := fmt.Sprintf("time.Date(%d, %d, %d, %d, %d, %d, 0, time.UTC)", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	result = strings.ReplaceAll(result, "%DATE%", nowConstructor)

	safeVersion := strings.ReplaceAll(types.MigrationVersion(now).String(), ":", "")
	filename := fmt.Sprintf("%v_%v.go", safeVersion, name)
	path := filepath.Join("migration", "migrations", filename)

	err := os.WriteFile(path, []byte(result), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write migration file: %w", err))
	}

	fmt.Println("Successfully created migration file:")
	fmt.Println(path)
}
