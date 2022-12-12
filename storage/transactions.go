package storage

import (
	"fmt"
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

func ValidTransactionColumn(column string) bool {
	switch column {
	case boiler.TransactionColumns.ID,
		boiler.TransactionColumns.Amount,
		boiler.TransactionColumns.CreatedAt,
		boiler.TransactionColumns.DebitAccountID,
		boiler.TransactionColumns.CreditAccountID,
		boiler.TransactionColumns.Ledger,
		boiler.TransactionColumns.TransferCode:
		return true
	default:
		return false
	}
}

func (s *Storage) TransactionsGetByAccountID(
	accountID string,
	offSet int,
	pageSize int,
	sortBy string,
	sortDir string,
) (int64, []*transactionsv1.CompletedTransfer, error) {
	results := []*transactionsv1.CompletedTransfer{}

	queryMods := []qm.QueryMod{
		boiler.TransactionWhere.DebitAccountID.EQ(accountID),
		qm.Or2(
			boiler.TransactionWhere.CreditAccountID.EQ(accountID),
		),
	}

	count, err := boiler.Transactions(queryMods...).Count(s)
	if err != nil {
		return 0, nil, err
	}
	if count == 0 {
		return 0, results, nil
	}

	queryMods = append(queryMods,
		qm.Limit(pageSize),
		qm.Load(boiler.TransactionRels.CreditAccount),
		qm.Load(boiler.TransactionRels.DebitAccount),
	)

	if offSet > 0 {
		queryMods = append(queryMods, qm.Offset(offSet))
	}
	if ValidTransactionColumn(sortBy) {
		if sortDir == "desc" {
			queryMods = append(queryMods, qm.OrderBy(fmt.Sprintf("%s DESC", sortBy)))
		} else {
			queryMods = append(queryMods, qm.OrderBy(sortBy))
		}
	} else {
		queryMods = append(queryMods, qm.OrderBy(fmt.Sprintf("%s DESC", boiler.TransactionColumns.CreatedAt)))
	}

	transaction, err := boiler.Transactions(
		queryMods...,
	).All(s)
	if err != nil {
		return 0, nil, err
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

	return count, results, nil
}
