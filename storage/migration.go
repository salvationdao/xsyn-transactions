package storage

import (
	"github.com/shopspring/decimal"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"time"
	"xsyn-transactions/boiler"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

/*
These functions should be used for migrating accounts and transactions
*/

func (s *Storage) MigrateInsertAccounts(accounts []*transactionsv1.Account) error {
	tx, err := s.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, account := range accounts {
		newAccount := &boiler.Account{
			ID:          account.Id,
			XsynUserID:  account.UserId,
			AccountCode: int(account.Code),
			Ledger:      int(account.Ledger),
			CreatedAt:   time.Unix(int64(account.CreatedAt), 0),
		}

		// convert balance
		newAccount.DebitsPosted, err = decimal.NewFromString(account.DebitsPosted)
		if err != nil {
			return err
		}
		newAccount.CreditsPosted, err = decimal.NewFromString(account.CreditsPosted)
		if err != nil {
			return err
		}

		err = newAccount.Insert(tx, boil.Infer())
		if err != nil {
			s.log.Error().Err(err).Interface("newAccount", newAccount).Msg("failed to insert new account")
			return err
		}
		s.log.Info().Str("user_id", newAccount.XsynUserID).Int("ledger", newAccount.Ledger).Msg("inserted new account")

	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) MigrateInsertTransfers(txes []*transactionsv1.MigrationTransfer) error {
	tx, err := s.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// disable trigger
	//  hypertables do not support  enabling or disabling triggers (so we just delete it?)
	_, err = tx.Exec("DROP TRIGGER trigger_check_balance ON transactions;")
	if err != nil {
		return err
	}

	for _, transaction := range txes {
		newTx := &boiler.Transaction{
			ID:              transaction.Id,
			DebitAccountID:  transaction.DebitAccountId,
			CreditAccountID: transaction.CreditAccountId,
			Ledger:          int(transaction.Ledger),
			TransferCode:    int(transaction.Code),
			CreatedAt:       time.Unix(int64(transaction.Timestamp), 0),
		}

		newTx.Amount, err = decimal.NewFromString(transaction.Amount)
		if err != nil {
			return err
		}

		err = newTx.Insert(tx, boil.Infer())
		if err != nil {
			return err
		}
		s.log.Info().Str("id", newTx.ID).Str("amount", newTx.Amount.String()).Msg("inserted new tx")
	}

	// enable trigger
	_, err = tx.Exec(`
		CREATE TRIGGER trigger_check_balance
			BEFORE INSERT
			ON transactions
			FOR EACH ROW
		EXECUTE PROCEDURE check_balances();`)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
