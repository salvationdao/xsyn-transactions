package legacy_storage

import (
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"net/url"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

type Storage struct {
	*sql.DB
}

type StorageOpts struct {
	DatabaseTxUser string
	DatabaseTxPass string
	DatabaseHost   string
	DatabasePort   int
	DatabaseName   string
}

func NewLegacyStorage(
	opts *StorageOpts,
) (*Storage, error) {
	newStorage := &Storage{}

	if opts == nil {
		return nil, fmt.Errorf("storage config is nil")
	}

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

	return newStorage, nil
}

func (s *Storage) GetAccounts() ([]*transactionsv1.Account, error) {
	results := []*transactionsv1.Account{}

	q := `SELECT 	id,
					user_id,
					code,
					ledger,
					debits_posted,
					credits_posted,
					TRUNC(EXTRACT(EPOCH FROM created_at)::NUMERIC)
			FROM accounts;`
	rows, err := s.Query(q)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		result := &transactionsv1.Account{}
		err := rows.Scan(
			&result.Id,
			&result.UserId,
			&result.Code,
			&result.Ledger,
			&result.DebitsPosted,
			&result.CreditsPosted,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *Storage) GetTransactions() ([]*transactionsv1.MigrationTransfer, error) {
	results := []*transactionsv1.MigrationTransfer{}

	q := `SELECT 	id,
					amount,
					debit_account_id,
					credit_account_id,
					ledger,
					code,
					 TRUNC(EXTRACT(EPOCH FROM created_at)::NUMERIC)
			FROM transactions ORDER BY created_at;`
	rows, err := s.Query(q)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		result := &transactionsv1.MigrationTransfer{}
		err := rows.Scan(
			&result.Id,
			&result.Amount,
			&result.DebitAccountId,
			&result.CreditAccountId,
			&result.Ledger,
			&result.Code,
			&result.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}
