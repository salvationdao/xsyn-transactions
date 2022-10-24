package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
	"os"
	"time"
	"xsyn-transactions/gen/transactions/v1/transactionsv1connect"
	"xsyn-transactions/storage"
	"xsyn-transactions/transactor"
)

const envPrefix = "XSYN_TRANSACTIONS"

func main() {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	log.Logger = log.Level(zerolog.InfoLevel)

	app := &cli.App{
		Compiled: time.Now(),
		Name:     "xsyn transaction service",
		Usage:    "handles all transactions for xsyn services",
		Authors: []*cli.Author{
			{
				Name:  "Ninja Syndicate",
				Email: "hello@supremacy.game",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "run xsyn transaction service",
				Flags: []cli.Flag{
					// db details
					&cli.StringFlag{Name: "db_user", Value: "xsyn-transactions", Usage: "The user for postgres", EnvVars: []string{envPrefix + "_DB_USER"}},
					&cli.StringFlag{Name: "db_pass", Value: "dev", Usage: "The pass for postgres", EnvVars: []string{envPrefix + "_DB_PASS"}},
					&cli.StringFlag{Name: "db_host", Value: "localhost", Usage: "The host for postgres", EnvVars: []string{envPrefix + "_DB_HOST"}},
					&cli.IntFlag{Name: "db_port", Value: 5433, Usage: "The port for postgres", EnvVars: []string{envPrefix + "_DB_PORT"}},
					&cli.StringFlag{Name: "db_name", Value: "xsyn-transactions", Usage: "The db name for postgres", EnvVars: []string{envPrefix + "DB_NAME"}},
					&cli.IntFlag{Name: "db_max_idle_conns", Value: 40, EnvVars: []string{envPrefix + "_DB_MAX_IDLE_CONNS"}, Usage: "Database max idle conns"},
					&cli.IntFlag{Name: "db_max_open_conns", Value: 50, EnvVars: []string{envPrefix + "_DB_MAX_OPEN_CONNS"}, Usage: "Database max open conns"},

					// api details
					&cli.IntFlag{Name: "api_port", Value: 8087, EnvVars: []string{envPrefix + "_API_PORT", "API_PORT"}, Usage: "port to run the API"},
				},
				Action: RunService,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("run")
	}

}

func RunService(c *cli.Context) error {
	toDbUser := c.String("db_user")
	toDbPass := c.String("db_pass")
	toDbHost := c.String("db_host")
	toDbPort := c.Int("db_port")
	toDbName := c.String("db_name")
	toDbMaxIdleConns := c.Int("db_max_idle_conns")
	toDbMaxOpenConns := c.Int("db_max_open_conns")

	apiPort := c.Int("api_port")

	newTransactor, err := transactor.NewTransactor(
		&transactor.NewTransactorOpts{
			StorageOpts: &storage.Opts{
				DatabaseTxUser: toDbUser,
				DatabaseTxPass: toDbPass,
				DatabaseHost:   toDbHost,
				DatabasePort:   toDbPort,
				DatabaseName:   toDbName,
				MaxIdle:        toDbMaxIdleConns,
				MaxOpen:        toDbMaxOpenConns,
				Log:            &log.Logger,
			},
			Log: &log.Logger,
			//BalanceUpdateFunction: nil,
		},
	)
	if err != nil {
		return fmt.Errorf("create new storage instance: %w", err)
	}

	mux := http.NewServeMux()
	path, handler := transactionsv1connect.NewTransactorHandler(newTransactor)
	mux.Handle(path, handler)

	hostAddr := fmt.Sprintf("localhost:%d", apiPort)

	log.Info().Msgf("serving transactor on %s", hostAddr)
	err = http.ListenAndServe(
		hostAddr,
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		return err
	}

	return nil
}
