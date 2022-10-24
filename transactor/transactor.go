package transactor

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
	"xsyn-transactions/storage"
)

var ErrTimeToClose = fmt.Errorf("closing")
var ErrQueueFull = fmt.Errorf("transaction queue is full")
var ErrUnableToFindAccount = fmt.Errorf("unable to find account")

type Transactor struct {
	Storage               *storage.Storage
	log                   *zerolog.Logger
	m                     map[string]map[transactionsv1.Ledger]*transactionsv1.Account // map[user_id]map[currency]account
	runner                chan func() error
	balanceUpdateFunction func(account *transactionsv1.Account, transaction *transactionsv1.CompletedTransfer)
	deadlock.RWMutex
}

type NewTransactorOpts struct {
	StorageOpts           *storage.Opts
	Log                   *zerolog.Logger
	BalanceUpdateFunction func(account *transactionsv1.Account, transaction *transactionsv1.CompletedTransfer)
}

func NewTransactor(opts *NewTransactorOpts) (*Transactor, error) {
	var err error

	txr := &Transactor{
		m:       make(map[string]map[transactionsv1.Ledger]*transactionsv1.Account),
		runner:  make(chan func() error, 100),
		RWMutex: deadlock.RWMutex{},
	}

	if opts == nil {
		return nil, fmt.Errorf("transactor config is nil")
	}

	txr.log = opts.Log

	txr.Storage, err = storage.NewStorage(opts.StorageOpts)
	if err != nil {
		return nil, err
	}

	txr.balanceUpdateFunction = func(account *transactionsv1.Account, transaction *transactionsv1.CompletedTransfer) {
		go func() {
			if opts != nil && opts.BalanceUpdateFunction != nil {
				opts.BalanceUpdateFunction(account, transaction)
			}
		}()
	}

	accounts, err := txr.Storage.GetAllAccounts()
	if err != nil {
		txr.log.Error().Err(err).Msg("unable to retrieve user account balances")
		return nil, err
	}

	txr.Lock()
	for _, account := range accounts {
		ledger := transactionsv1.Ledger(account.Ledger)
		// create user ledger account map if not exist
		if _, ok := txr.m[account.UserId]; !ok {
			txr.m[account.UserId] = make(map[transactionsv1.Ledger]*transactionsv1.Account)
		}
		// create account on the ledger map
		txr.m[account.UserId][ledger] = account
	}
	txr.Unlock()

	go txr.run()
	txr.log.Info().Msg("successfully initiated transactor")
	return txr, nil
}

func (t *Transactor) run() {
	for {
		select {
		case fn := <-t.runner:
			if fn == nil {
				return
			}
			err := fn()
			if errors.Is(err, ErrTimeToClose) {
				return
			}
		}
	}
}
