package transactor

import (
	"context"
	"fmt"
	connect_go "github.com/bufbuild/connect-go"
	"github.com/friendsofgo/errors"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
)

func (t *Transactor) GetBalance(ctx context.Context, req *connect_go.Request[transactionsv1.GetBalanceRequest]) (*connect_go.Response[transactionsv1.GetBalanceResponse], error) {
	fmt.Println("here1")

	account, err := t.get(req.Msg.UserId, req.Msg.Ledger)
	if err != nil {
		if errors.Is(err, ErrUnableToFindAccount) && req.Msg.CreateIfNotExists {
			err = t.Storage.CreateAccount(req.Msg.UserId, transactionsv1.AccountCode_AccountUser, req.Msg.Ledger)
			if err != nil {
				return nil, connect_go.NewError(connect_go.CodeInternal, err)
			}
			account, err = t.get(req.Msg.UserId, req.Msg.Ledger)
			if err != nil {
				return nil, connect_go.NewError(connect_go.CodeInternal, err)
			}
		} else {
			return nil, connect_go.NewError(connect_go.CodeInternal, err)
		}
	}

	fmt.Println(account.Balance)

	return connect_go.NewResponse[transactionsv1.GetBalanceResponse](&transactionsv1.GetBalanceResponse{
		Balance: account.Balance,
	}), nil
}

func (t *Transactor) AccountGetViaUser(ctx context.Context, req *connect_go.Request[transactionsv1.AccountGetViaUserRequest]) (*connect_go.Response[transactionsv1.AccountGetViaUserResponse], error) {
	if req.Msg.UserId == "" {
		return nil, connect_go.NewError(connect_go.CodeInvalidArgument, fmt.Errorf("user id is empty"))
	}

	account, err := t.get(req.Msg.UserId, req.Msg.Ledger)
	if err != nil {
		return nil, connect_go.NewError(connect_go.CodeInternal, err)
	}

	return connect_go.NewResponse[transactionsv1.AccountGetViaUserResponse](&transactionsv1.AccountGetViaUserResponse{Account: account}), nil
}

func (t *Transactor) AccountsUser(ctx context.Context, req *connect_go.Request[transactionsv1.AccountsUserRequest]) (*connect_go.Response[transactionsv1.AccountsUserResponse], error) {
	accounts := []*transactionsv1.Account{}

	// loop over all ledgers and get the account
	// check if we want to create that ledger if not exist
	for _, l := range transactionsv1.Ledger_value {
		account, err := t.get(req.Msg.UserId, transactionsv1.Ledger(l))
		if err != nil {
			// if we cannot find account, check if we want to create it
			if errors.Is(err, ErrUnableToFindAccount) {
				create := false
				for _, ledgers := range req.Msg.CreateIfNotExist {
					if int32(ledgers) == l {
						create = true
					}
				}
				// create it
				if create {
					err = t.Storage.CreateAccount(req.Msg.UserId, transactionsv1.AccountCode_AccountUser, transactionsv1.Ledger(l))
					if err != nil {
						return nil, connect_go.NewError(connect_go.CodeInternal, err)
					}
					account, err := t.get(req.Msg.UserId, transactionsv1.Ledger(l))
					if err != nil {
						return nil, connect_go.NewError(connect_go.CodeInternal, err)
					}
					accounts = append(accounts, account)
					continue
				}
			}
			return nil, connect_go.NewError(connect_go.CodeInternal, err)
		}

		accounts = append(accounts, account)
	}

	return connect_go.NewResponse[transactionsv1.AccountsUserResponse](&transactionsv1.AccountsUserResponse{Accounts: accounts}), nil
}
