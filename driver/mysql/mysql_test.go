package mysql

import (
	"database/sql"
	"os"
	"testing"

	"github.com/3dsinteractive/migration"
	"github.com/3dsinteractive/migration/parser"
	"github.com/go-sql-driver/mysql"
)

func TestMySQLDriver(t *testing.T) {

	mysqlHost := os.Getenv("MYSQL_HOST")

	database := "migrationtest"

	// prepare clean database
	connection, err := sql.Open("mysql", "root:@tcp("+mysqlHost+")/")

	if err != nil {
		t.Fatal(err)
	}

	defer connection.Close()

	_, err = connection.Exec("CREATE DATABASE IF NOT EXISTS " + database)

	if err != nil {
		t.Fatal(err)
	}

	_, err = connection.Exec("USE " + database)

	if err != nil {
		t.Fatal(err)
	}

	driver, err := New("root:@tcp(" + mysqlHost + ")/" + database + "?multiStatements=true")

	if err != nil {
		t.Errorf("Unable to open connection to mysql server: %s", err)
	}

	defer driver.Close()

	defer func() {
		connection.Exec("DROP DATABASE IF EXISTS " + database)
	}()

	migrations := []*migration.PlannedMigration{
		{
			Migration: &migration.Migration{
				ID: "201610041422_init",
				Up: &parser.ParsedMigration{
					Statements: []string{
						`CREATE TABLE test_table1 (id integer not null primary key);

				   		 CREATE TABLE test_table2 (id integer not null primary key)`,
					},
					UseTransaction: false,
				},
			},
			Direction: migration.Up,
		},
		{
			Migration: &migration.Migration{
				ID: "201610041425_drop_unused_table",
				Up: &parser.ParsedMigration{
					Statements: []string{
						"DROP TABLE test_table2",
					},
					UseTransaction: false,
				},
				Down: &parser.ParsedMigration{
					Statements: []string{
						"CREATE TABLE test_table2(id integer not null primary key)",
					},
					UseTransaction: false,
				},
			},
			Direction: migration.Up,
		},
		{
			Migration: &migration.Migration{
				ID: "201610041422_invalid_sql",
				Up: &parser.ParsedMigration{
					Statements: []string{
						"CREATE TABLE test_table3 (some error",
					},
					UseTransaction: false,
				},
			},
			Direction: migration.Up,
		},
	}

	err = driver.Migrate(migrations[0])

	if err != nil {
		t.Errorf("Unexpected error while running migration: %s", err)
	}

	_, err = connection.Exec("INSERT INTO test_table1 (id) values (1)")

	if err != nil {
		t.Errorf("Unexpected error while testing if migration succeeded: %s", err)
	}

	_, err = connection.Exec("INSERT INTO test_table2 (id) values (1)")

	if err != nil {
		t.Errorf("Unexpected error while testing if migration succeeded: %s", err)
	}

	err = driver.Migrate(migrations[1])

	if err != nil {
		t.Errorf("Unexpected error while running migration: %s", err)
	}

	if _, err = connection.Exec("INSERT INTO test_table2 (id) values (1)"); err != nil {
		if err.(*mysql.MySQLError).Number != 1146 {
			t.Errorf("Received an error while inserting into a non-existent table, but it was not a table_undefined error: %s", err)
		}
	} else {
		t.Error("Expected an error while inserting into non-existent table, but did not receive any.")
	}

	err = driver.Migrate(migrations[2])

	if err == nil {
		t.Error("Expected an error while executing invalid statement, but did not receive any.")
	}

	versions, err := driver.Versions()

	if err != nil {
		t.Errorf("Unexpected error while retriving version information: %s", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected %d versions to be applied, %d was actually applied.", 2, len(versions))
	}

	migrations[1].Direction = migration.Down

	err = driver.Migrate(migrations[1])

	if err != nil {
		t.Errorf("Unexpected error while running migration: %s", err)
	}

	versions, err = driver.Versions()

	if err != nil {
		t.Errorf("Unexpected error while retriving version information: %s", err)
	}

	if len(versions) != 1 {
		t.Errorf("Expected %d versions to be applied, %d was actually applied.", 2, len(versions))
	}
}

func TestCreateDriverUsingInvalidDBInstance(t *testing.T) {

	db, _, err := sqlmock.New()

	if err != nil {
		t.Fatalf("Error opening stub database connection: %s", err)
	}

	_, err = NewFromDB(db)

	if err == nil {
		t.Error("Expected error when creating MySQL driver with a non-MySQL database instance, but there was no error")
	}
}

func TestCreateDriverUsingDBInstance(t *testing.T) {
	mysqlHost := os.Getenv("MYSQL_HOST")

	database := "migrationtest"

	// prepare clean database
	connection, err := sql.Open("mysql", "root:@tcp("+mysqlHost+")/")

	if err != nil {
		t.Fatal(err)
	}

	_, err = connection.Exec("CREATE DATABASE IF NOT EXISTS " + database)

	if err != nil {
		t.Fatal(err)
	}

	defer connection.Close()

	defer func() {
		connection.Exec("DROP DATABASE IF EXISTS " + database)
	}()

	db, err := sql.Open("mysql", "root:@tcp("+mysqlHost+")/"+database+"?multiStatements=true")

	if err != nil {
		t.Fatalf("Could not open MySQL connection: %s", err)
	}

	driver, err := NewFromDB(db)

	if err != nil {
		t.Errorf("Unable to create MySQL driver: %s", err)
	}

	defer driver.Close()
}
