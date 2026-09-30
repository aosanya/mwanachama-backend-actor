package main

import (
	"flag"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func main() {
	instance := flag.String("instance", "", "the instance whose tables to provision")
	mount := flag.String("mount", "", "the mount within that instance; empty is the default mount")
	flag.Parse()

	if *instance == "" {
		fmt.Fprintln(os.Stderr, "usage: provision -instance <name> [-mount <name>]")
		fmt.Fprintln(os.Stderr, "reads POSTGRES_URL")
		os.Exit(2)
	}

	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "POSTGRES_URL is not set")
		os.Exit(2)
	}

	s, err := mwanachamaactor.SpecForMount(*instance, *mount)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := mwanachamaactor.Provision(db, s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, o := range s.Objects {
		fmt.Println(s.TableFor(o))
	}
}
