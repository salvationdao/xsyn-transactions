package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
	"xsyn-transactions/legacy_storage"
	"xsyn-transactions/storage"
)

const envPrefix = "XSYN_TRANSACTIONS_DATA_MIGRATOR"

func main() {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	log.Logger = log.Level(zerolog.InfoLevel)
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: io.Discard})

	app := &cli.App{
		Name:  "migrate transactions between databases",
		Usage: "moves transactions from postgres to tigerbeetle",
		Flags: []cli.Flag{
			// old db details
			&cli.StringFlag{Name: "from_db_user", Value: "passport", Usage: "The user for postgres", EnvVars: []string{envPrefix + "_FROM_DB_USER"}},
			&cli.StringFlag{Name: "from_db_pass", Value: "dev", Usage: "The pass for postgres", EnvVars: []string{envPrefix + "_FROM_DB_PASS"}},
			&cli.StringFlag{Name: "from_db_host", Value: "localhost", Usage: "The host for postgres", EnvVars: []string{envPrefix + "_FROM_DB_HOST"}},
			&cli.IntFlag{Name: "from_db_port", Value: 5432, Usage: "The port for postgres", EnvVars: []string{envPrefix + "_FROM_DB_PORT"}},
			&cli.StringFlag{Name: "from_db_name", Value: "passport", Usage: "The db name for postgres", EnvVars: []string{envPrefix + "_FROM_DB_NAME"}},

			// new db details
			&cli.StringFlag{Name: "to_db_user", Value: "xsyn-transactions-db", Usage: "The user for postgres", EnvVars: []string{envPrefix + "_TO_DB_USER"}},
			&cli.StringFlag{Name: "to_db_pass", Value: "dev", Usage: "The pass for postgres", EnvVars: []string{envPrefix + "_TO_DB_PASS"}},
			&cli.StringFlag{Name: "to_db_host", Value: "localhost", Usage: "The host for postgres", EnvVars: []string{envPrefix + "_TO_DB_HOST"}},
			&cli.IntFlag{Name: "to_db_port", Value: 5433, Usage: "The port for postgres", EnvVars: []string{envPrefix + "_TO_DB_PORT"}},
			&cli.StringFlag{Name: "to_db_name", Value: "xsyn-transactions-db", Usage: "The db name for postgres", EnvVars: []string{envPrefix + "_TO_DB_NAME"}},
			&cli.IntFlag{Name: "to_db_max_idle_conns", Value: 40, EnvVars: []string{envPrefix + "_TO_DB_MAX_IDLE_CONNS"}, Usage: "Database max idle conns"},
			&cli.IntFlag{Name: "to_db_max_open_conns", Value: 50, EnvVars: []string{envPrefix + "_TO_DB_MAX_OPEN_CONNS"}, Usage: "Database max open conns"},
		},
		Action:   RunMigrator,
		Commands: []*cli.Command{},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("run")
	}

}

func RunMigrator(c *cli.Context) error {
	fromDbUser := c.String("from_db_user")
	fromDbPass := c.String("from_db_pass")
	fromDbHost := c.String("from_db_host")
	fromDbPort := c.Int("from_db_port")
	fromDbName := c.String("from_db_name")

	toDbUser := c.String("to_db_user")
	toDbPass := c.String("to_db_pass")
	toDbHost := c.String("to_db_host")
	toDbPort := c.Int("to_db_port")
	toDbName := c.String("to_db_name")
	toDbMaxIdleConns := c.Int("to_db_max_idle_conns")
	toDbMaxOpenConns := c.Int("to_db_max_open_conns")
	log.Info().Msg("starting transaction migration tool")

	newStorage, err := storage.NewStorage(&storage.Opts{
		DatabaseTxUser: toDbUser,
		DatabaseTxPass: toDbPass,
		DatabaseHost:   toDbHost,
		DatabasePort:   toDbPort,
		DatabaseName:   toDbName,
		MaxIdle:        toDbMaxIdleConns,
		MaxOpen:        toDbMaxOpenConns,
		Log:            &log.Logger,
	})
	if err != nil {
		return fmt.Errorf("create new storage instance: %w", err)
	}

	dataExists, err := newStorage.DataExists()
	if err != nil {
		return fmt.Errorf("checking if data exists: %w", err)
	}
	// if data already exists, skip data migration
	// this is used because you cannot run a service a single time in docker compose
	if dataExists {
		return nil
	}

	newLegacyStorage, err := legacy_storage.NewLegacyStorage(&legacy_storage.StorageOpts{
		DatabaseTxUser: fromDbUser,
		DatabaseTxPass: fromDbPass,
		DatabaseHost:   fromDbHost,
		DatabasePort:   fromDbPort,
		DatabaseName:   fromDbName,
	},
	)
	if err != nil {
		return fmt.Errorf("create legacy_storage storage instance: %w", err)
	}

	migrator := &Migrator{From: newLegacyStorage, To: newStorage}
	err = migrator.MigrateAccounts()
	if err != nil {
		return fmt.Errorf("migrate accounts: %w", err)
	}
	err = migrator.MigrateTransactions()
	if err != nil {
		return fmt.Errorf("migrate transactions: %w", err)
	}
	return nil
}

type Migrator struct {
	From MigrateFromService
	To   MigrateToService
}

type MigrateFromService interface {
	GetAccounts() ([]*transactionsv1.Account, error)
	GetTransactions() ([]*transactionsv1.MigrationTransfer, error)
}

type MigrateToService interface {
	MigrateInsertAccounts([]*transactionsv1.Account) error
	MigrateInsertTransfers([]*transactionsv1.MigrationTransfer) error
	DataExists() (bool, error)
}

func (c *Migrator) MigrateAccounts() error {
	accounts, err := c.From.GetAccounts()
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	err = c.To.MigrateInsertAccounts(accounts)
	if err != nil {
		return fmt.Errorf("failed to insert accounts: %w", err)
	}

	return nil
}

func (c *Migrator) MigrateTransactions() error {
	txs, err := c.From.GetTransactions()
	if err != nil {
		return fmt.Errorf("failed to get transactions: %w", err)
	}

	err = c.To.MigrateInsertTransfers(txs)
	if err != nil {
		return fmt.Errorf("failed to insert transactions: %w", err)
	}

	return nil
}
