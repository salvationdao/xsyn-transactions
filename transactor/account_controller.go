package transactor

import (
	"github.com/shopspring/decimal"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

func (t *Transactor) GetBalance(userID string, ledger transactionsv1.Ledger) string {
	account, err := t.get(userID, ledger)
	if err != nil {
		return "0"
	}

	return account.Balance
}

func (t *Transactor) balanceUpdate(tx *transactionsv1.CompletedTransfer) {
	amount, err := decimal.NewFromString(tx.Amount)
	if err != nil {
		t.log.Error().Err(err).Str("tx.Amount", tx.Amount).Msg("failed convert tx amount to decimal")
	}

	debitAccount, err := t.get(tx.DebitUserId, tx.Ledger)
	if err != nil {
		t.log.Error().Err(err).Interface("tx", tx).Msg("error updating balance")
	} else {
		balance, err := decimal.NewFromString(debitAccount.Balance)
		if err != nil {
			t.log.Error().Err(err).Str("debitAccount.Balance", debitAccount.Balance).Msg("failed convert balance to decimal")
		}

		debitAccount.Balance = balance.Sub(amount).String()
		t.put(debitAccount)
		t.balanceUpdateFunction(debitAccount, tx)
	}

	creditAccount, err := t.get(tx.CreditAccountId, tx.Ledger)
	if err != nil {
		t.log.Error().Err(err).Interface("tx", tx).Msg("error updating balance")
	} else {
		balance, err := decimal.NewFromString(creditAccount.Balance)
		if err != nil {
			t.log.Error().Err(err).Str("creditAccount.Balance", debitAccount.Balance).Msg("failed convert balance to decimal")
		}

		creditAccount.Balance = balance.Add(amount).String()
		t.put(creditAccount)
		t.balanceUpdateFunction(creditAccount, tx)
	}
}

func (t *Transactor) getAndSet(userID string, ledger transactionsv1.Ledger) (*transactionsv1.Account, error) {
	accounts, err := t.Storage.GetAllUserAccounts(userID)
	if err != nil {
		return nil, err
	}

	if _, ok := t.m[userID]; !ok {
		t.m[userID] = make(map[transactionsv1.Ledger]*transactionsv1.Account)
	}

	for _, account := range accounts {
		t.m[account.UserId][ledger] = account
	}

	if account, ok := t.m[userID][ledger]; ok {
		return account, nil
	}

	return nil, ErrUnableToFindAccount
}

func (t *Transactor) get(userID string, ledger transactionsv1.Ledger) (*transactionsv1.Account, error) {
	t.RLock()
	defer t.RUnlock()

	userLedgerMap, ok := t.m[userID]
	if ok {
		if account, ok := userLedgerMap[ledger]; ok {
			return account, nil
		}
	}

	return t.getAndSet(userID, ledger)
}

func (t *Transactor) put(account *transactionsv1.Account) {
	t.Lock()
	t.m[account.UserId][account.Ledger] = account
	t.Unlock()
}
