package storage

import (
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"xsyn-transactions/boiler"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

func (s *Storage) TransactionGetByID(transactionID string) (*transactionsv1.CompletedTransfer, error) {
	transaction, err := boiler.Transactions(
		boiler.TransactionWhere.ID.EQ(transactionID),
		qm.Load(boiler.TransactionRels.CreditAccount),
		qm.Load(boiler.TransactionRels.DebitAccount),
	).One(s)
	if err != nil {
		return nil, err
	}

	return &transactionsv1.CompletedTransfer{
			Id:              transaction.ID,
			CreditUserId:    transaction.R.CreditAccount.XsynUserID,
			CreditAccountId: transaction.CreditAccountID,
			DebitUserId:     transaction.R.DebitAccount.XsynUserID,
			DebitAccountId:  transaction.DebitAccountID,
			Amount:          transaction.Amount.String(),
			Ledger:          transactionsv1.Ledger(transaction.Ledger),
			Code:            transactionsv1.TransferCode(transaction.TransferCode),
			Timestamp:       transaction.CreatedAt.Unix(),
		},
		nil
}

func (s *Storage) TransactionsGetByAccountID(accountID string) ([]*transactionsv1.CompletedTransfer, error) {
	results := []*transactionsv1.CompletedTransfer{}
	transaction, err := boiler.Transactions(
		boiler.TransactionWhere.DebitAccountID.EQ(accountID),
		qm.Or2(
			boiler.TransactionWhere.CreditAccountID.EQ(accountID),
		),
		qm.Load(boiler.TransactionRels.CreditAccount),
		qm.Load(boiler.TransactionRels.DebitAccount),
	).All(s)
	if err != nil {
		return nil, err
	}

	for _, tx := range transaction {
		results = append(results, &transactionsv1.CompletedTransfer{
			Id:              tx.ID,
			CreditUserId:    tx.R.CreditAccount.XsynUserID,
			CreditAccountId: tx.CreditAccountID,
			DebitUserId:     tx.R.DebitAccount.XsynUserID,
			DebitAccountId:  tx.DebitAccountID,
			Amount:          tx.Amount.String(),
			Ledger:          transactionsv1.Ledger(tx.Ledger),
			Code:            transactionsv1.TransferCode(tx.TransferCode),
			Timestamp:       tx.CreatedAt.Unix(),
		})
	}

	return results, nil
}
