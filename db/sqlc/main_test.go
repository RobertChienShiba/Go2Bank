package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/RobertChienShiba/simplebank/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testStore Store

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal(err)
	}

	testStore = NewStore(pool)
	os.Exit(m.Run())
}
