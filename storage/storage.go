package storage

import (
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog"
	"net/url"
	"xsyn-transactions/boiler"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

type Storage struct {
	*sql.DB
	log *zerolog.Logger
}

type Opts struct {
	DatabaseTxUser string
	DatabaseTxPass string
	DatabaseHost   string
	DatabasePort   int
	DatabaseName   string
	MaxIdle        int
	MaxOpen        int
	Log            *zerolog.Logger
}

func NewStorage(
	opts *Opts,
) (*Storage, error) {
	newStorage := &Storage{}

	if opts == nil {
		return nil, fmt.Errorf("storage config is nil")
	}

	newStorage.log = opts.Log

	params := url.Values{}
	params.Add("sslmode", "disable")
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?%s",
		opts.DatabaseTxUser,
		opts.DatabaseTxPass,
		opts.DatabaseHost,
		opts.DatabasePort,
		opts.DatabaseName,
		params.Encode(),
	)
	cfg, err := pgx.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	newStorage.DB = stdlib.OpenDB(*cfg)
	if err != nil {
		return nil, err
	}
	newStorage.DB.SetMaxIdleConns(opts.MaxIdle)
	newStorage.DB.SetMaxOpenConns(opts.MaxOpen)
	newStorage.log.Info().Msg("successfully initiated storage")

	return newStorage, nil
}

func (s *Storage) GetAllAccounts() ([]*transactionsv1.Account, error) {
	var results []*transactionsv1.Account

	accounts, err := boiler.Accounts().All(s)
	if err != nil {
		s.log.Error().Err(err).Msg("unable to retrieve user account balances")
		return nil, err
	}
	for _, acc := range accounts {
		results = append(results, &transactionsv1.Account{
			Id:            acc.ID,
			UserId:        acc.XsynUserID,
			Ledger:        transactionsv1.Ledger(acc.Ledger),
			Code:          transactionsv1.AccountCode(acc.AccountCode),
			DebitsPosted:  acc.DebitsPosted.String(),
			CreditsPosted: acc.CreditsPosted.String(),
			Balance:       acc.CreditsPosted.Sub(acc.DebitsPosted).String(),
			CreatedAt:     acc.CreatedAt.Unix(),
		})
	}

	return results, nil
}

func (s *Storage) GetAllUserAccounts(userID string) ([]*transactionsv1.Account, error) {
	var results []*transactionsv1.Account

	accounts, err := boiler.Accounts(
		boiler.AccountWhere.XsynUserID.EQ(userID),
	).All(s)
	if err != nil {
		s.log.Error().Err(err).Msg("unable to retrieve user account balances")
		return nil, err
	}
	for _, acc := range accounts {
		results = append(results, &transactionsv1.Account{
			Id:            acc.ID,
			UserId:        acc.XsynUserID,
			Ledger:        transactionsv1.Ledger(acc.Ledger),
			Code:          transactionsv1.AccountCode(acc.AccountCode),
			DebitsPosted:  acc.DebitsPosted.String(),
			CreditsPosted: acc.CreditsPosted.String(),
			Balance:       acc.CreditsPosted.Sub(acc.DebitsPosted).String(),
			CreatedAt:     acc.CreatedAt.Unix(),
		})
	}

	return results, nil
}
