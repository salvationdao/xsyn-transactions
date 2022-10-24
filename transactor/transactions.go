package transactor

import (
	"context"
	"fmt"
	connect_go "github.com/bufbuild/connect-go"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"sync"
	"xsyn-transactions/boiler"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

func (t *Transactor) Hello(ctx context.Context, req *connect_go.Request[transactionsv1.HelloRequest]) (*connect_go.Response[transactionsv1.HelloResponse], error) {
	fmt.Println(req.Msg.Name)
	return connect_go.NewResponse[transactionsv1.HelloResponse](&transactionsv1.HelloResponse{Message: fmt.Sprintf("hello there %s", req.Msg.Name)}), nil
}

// Transact makes a transaction using user id and ledger code
func (t *Transactor) Transact(ctx context.Context, req *connect_go.Request[transactionsv1.TransactRequest]) (*connect_go.Response[transactionsv1.TransactResponse], error) {

	creditorAccount, err := t.get(req.Msg.CreditUserId, req.Msg.Ledger)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}
	debitorAccount, err := t.get(req.Msg.DebitUserId, req.Msg.Ledger)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	amount, err := decimal.NewFromString(req.Msg.Amount)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	tx, err := t.transact(&NewTransaction{
		CreditUserID:    req.Msg.CreditUserId,
		CreditAccountID: creditorAccount.Id,
		DebitAccountID:  debitorAccount.Id,
		DebitUserID:     req.Msg.DebitUserId,
		Amount:          amount,
		Ledger:          req.Msg.Ledger,
		TransferCode:    req.Msg.Code,
	})
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	return connect_go.NewResponse[transactionsv1.TransactResponse](&transactionsv1.TransactResponse{Transfer: tx}), nil
}

// TransactWithID makes a transaction using user id and ledger code but takes a pre-generated tx id
func (t *Transactor) TransactWithID(ctx context.Context, req *connect_go.Request[transactionsv1.TransactWithIDRequest]) (*connect_go.Response[transactionsv1.TransactWithIDResponse], error) {
	creditorAccount, err := t.get(req.Msg.CreditUserId, req.Msg.Ledger)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}
	debitorAccount, err := t.get(req.Msg.DebitUserId, req.Msg.Ledger)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	amount, err := decimal.NewFromString(req.Msg.Amount)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	uid, err := uuid.FromString(req.Msg.TxId)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	tx, err := t.transact(&NewTransaction{
		ID:              uid,
		CreditUserID:    req.Msg.CreditUserId,
		CreditAccountID: creditorAccount.Id,
		DebitAccountID:  debitorAccount.Id,
		DebitUserID:     req.Msg.DebitUserId,
		Amount:          amount,
		Ledger:          req.Msg.Ledger,
		TransferCode:    req.Msg.Code,
	})
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	return connect_go.NewResponse[transactionsv1.TransactWithIDResponse](&transactionsv1.TransactWithIDResponse{Transfer: tx}), nil
}

type NewTransaction struct {
	ID              uuid.UUID
	CreditUserID    string
	CreditAccountID string
	DebitAccountID  string
	DebitUserID     string
	Amount          decimal.Decimal
	Ledger          transactionsv1.Ledger
	TransferCode    transactionsv1.TransferCode
}

func (t *Transactor) transact(nt *NewTransaction) (*transactionsv1.CompletedTransfer, error) {
	var transactionError error = nil
	var completedTx *transactionsv1.CompletedTransfer = nil
	transactionID := uuid.Must(uuid.NewV4())
	if !nt.ID.IsNil() {
		transactionID = nt.ID
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	fn := func() error {
		tx := &boiler.Transaction{
			ID:              transactionID.String(),
			CreditAccountID: nt.CreditAccountID,
			DebitAccountID:  nt.DebitAccountID,
			Amount:          nt.Amount,
			TransferCode:    int(nt.TransferCode),
			Ledger:          int(nt.Ledger),
		}

		transactionError = tx.Insert(t.Storage, boil.Infer())
		if transactionError != nil {
			log.Error().Err(transactionError).Str("from", tx.DebitAccountID).Str("to", tx.CreditAccountID).Str("id", tx.ID).Str("amount", tx.Amount.String()).Msg("transaction failed")
			wg.Done()
			return transactionError
		}

		completedTx = &transactionsv1.CompletedTransfer{
			Id:              tx.ID,
			CreditUserId:    nt.CreditUserID,
			CreditAccountId: nt.CreditAccountID,
			DebitUserId:     nt.DebitUserID,
			DebitAccountId:  nt.DebitAccountID,
			Amount:          nt.Amount.String(),
			Ledger:          nt.Ledger,
			Code:            nt.TransferCode,
			Timestamp:       tx.CreatedAt.Unix(),
		}

		//t.balanceUpdate(completedTx)
		wg.Done()
		return nil
	}
	select {
	case t.runner <- fn: //put in channel
	default: //unless it's full!
		log.Error().Msg("Transaction queue is blocked! 100 transactions waiting to be processed.")
		return completedTx, ErrQueueFull
	}
	wg.Wait()

	return completedTx, transactionError
}

func (t *Transactor) Close() {
	wg := sync.WaitGroup{}
	wg.Add(1)

	fn := func() error {
		wg.Done()
		return ErrTimeToClose
	}

	select {
	case t.runner <- fn: //queue close
	default: //unless it's full!
		log.Error().Msg("Transaction queue is blocked! Exiting.")
		return
	}
	wg.Wait()
}
