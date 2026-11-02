package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

const (
	exitClean      = 0
	exitDuplicates = 1
	exitFailed     = 2
)

const usage = `usage: seatduplicates -instance <name> [-mount <name>] [-schema <name>] [-dialect postgres|sqlite]
reads POSTGRES_URL (postgres) or SQLITE_PATH (sqlite), never an argument
exits 0 with no duplicate live seat, 1 with at least one, 2 when it cannot look`

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("seatduplicates", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	instance := flags.String("instance", "", "the instance whose seat table to read")
	mount := flags.String("mount", "", "the mount within that instance; empty is the default mount")
	schema := flags.String("schema", "", "the schema that holds the tables, when it is not on the search path")
	dialect := flags.String("dialect", "postgres", "postgres or sqlite")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, usage)
		return exitFailed
	}
	if *instance == "" || flags.NArg() > 0 {
		fmt.Fprintln(stderr, usage)
		return exitFailed
	}

	variable, err := variableFor(*dialect)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailed
	}
	location := getenv(variable)
	if location == "" {
		fmt.Fprintf(stderr, "%s is not set\n", variable)
		return exitFailed
	}

	fail := func(err error) int {
		fmt.Fprintln(stderr, redact(err.Error(), location))
		return exitFailed
	}

	s, err := mwanachamaactor.SpecForMount(*instance, *mount)
	if err != nil {
		return fail(err)
	}
	db, closeDB, err := openReadOnly(*dialect, location)
	if err != nil {
		return fail(err)
	}
	defer closeDB()

	duplicates, err := mwanachamaactor.FindSeatDuplicates(context.Background(), db, s, *schema)
	if err != nil {
		return fail(err)
	}

	for _, d := range duplicates {
		fmt.Fprintf(stdout, "actor=%s group=%s group_name=%q kind=%s kind_name=%q seats=%d ids=%s oldest=%s newest=%s\n",
			d.ActorID, d.GroupID, d.GroupName, d.KindID, d.KindName, d.Seats,
			strings.Join(d.SeatIDs, ","), orDash(d.OldestGrantedAt), orDash(d.NewestGrantedAt))
	}
	fmt.Fprintf(stdout, "duplicates: %d\n", len(duplicates))
	if len(duplicates) > 0 {
		return exitDuplicates
	}
	return exitClean
}

func variableFor(dialect string) (string, error) {
	switch dialect {
	case "postgres":
		return "POSTGRES_URL", nil
	case "sqlite":
		return "SQLITE_PATH", nil
	}
	return "", fmt.Errorf("the dialect is postgres or sqlite, not %q", dialect)
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
