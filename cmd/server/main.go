package main

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
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
					&cli.StringFlag{Name: "db_user", Value: "xsyn-transactions-db", Usage: "The user for postgres", EnvVars: []string{envPrefix + "_DB_USER"}},
					&cli.StringFlag{Name: "db_pass", Value: "dev", Usage: "The pass for postgres", EnvVars: []string{envPrefix + "_DB_PASS"}},
					&cli.StringFlag{Name: "db_host", Value: "localhost", Usage: "The host for postgres", EnvVars: []string{envPrefix + "_DB_HOST"}},
					&cli.IntFlag{Name: "db_port", Value: 5433, Usage: "The port for postgres", EnvVars: []string{envPrefix + "_DB_PORT"}},
					&cli.StringFlag{Name: "db_name", Value: "xsyn-transactions-db", Usage: "The db name for postgres", EnvVars: []string{envPrefix + "DB_NAME"}},
					&cli.IntFlag{Name: "db_max_idle_conns", Value: 40, EnvVars: []string{envPrefix + "_DB_MAX_IDLE_CONNS"}, Usage: "Database max idle conns"},
					&cli.IntFlag{Name: "db_max_open_conns", Value: 50, EnvVars: []string{envPrefix + "_DB_MAX_OPEN_CONNS"}, Usage: "Database max open conns"},

					// api details
					&cli.IntFlag{Name: "api_port", Value: 8087, EnvVars: []string{envPrefix + "_API_PORT", "API_PORT"}, Usage: "port to run the API"},

					&cli.StringFlag{Name: "auth_key", Value: "d21f0c89-567e-4b4f-928f-68679e48df6c", EnvVars: []string{envPrefix + "_AUTH_KEY"}, Usage: "Auth key for clients to connect to xsyn-transactions"},
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
	authKey := c.String("auth_key")

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
		},
	)
	if err != nil {
		return fmt.Errorf("create new storage instance: %w", err)
	}

	mux := http.NewServeMux()
	path, handler := transactionsv1connect.NewTransactorHandler(newTransactor, connect.WithInterceptors(newAuthInterceptor(authKey)))
	mux.Handle(path, handler)
	path, handler = transactionsv1connect.NewAccountsHandler(newTransactor, connect.WithInterceptors(newAuthInterceptor(authKey)))
	mux.Handle(path, handler)

	hostAddr := fmt.Sprintf("0.0.0.0:%d", apiPort)

	log.Info().Msgf("serving transactor on %s", hostAddr)
	err = http.ListenAndServe(
		hostAddr,
		// Use h2c, so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		return err
	}

	return nil
}

// newAuthInterceptor created the rpc middleware for our auth
func newAuthInterceptor(authKey string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				// Send a token with client requests.
				req.Header().Set("xsyn-transaction-auth-key", authKey)
			} else if req.Header().Get("xsyn-transaction-auth-key") != authKey {
				// Check token in handlers.
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					fmt.Errorf("invalid token provided"),
				)
			}
			return next(ctx, req)
		}
	}
	return interceptor
}
