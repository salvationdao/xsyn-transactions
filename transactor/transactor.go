package transactor

import (
	"context"
	"errors"
	"fmt"
	connect_go "github.com/bufbuild/connect-go"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
	transactionsv1 "xsyn-transactions/gen/transactions/v1"
	"xsyn-transactions/storage"
)

var ErrTimeToClose = fmt.Errorf("closing")
var ErrQueueFull = fmt.Errorf("transaction queue is full")
var ErrUnableToFindAccount = fmt.Errorf("unable to find account")

type Transactor struct {
	Storage *storage.Storage
	log     *zerolog.Logger
	m       map[string]map[transactionsv1.Ledger]*transactionsv1.Account // map[user_id]map[currency]account
	runner  chan func() error
	deadlock.RWMutex

	broadcaster   chan *transactionsv1.TransferCompleteSubscribeResponse
	ClientStreams map[string]connect_go.StreamingHandlerConn
}

type NewTransactorOpts struct {
	StorageOpts *storage.Opts
	Log         *zerolog.Logger
}

func NewTransactor(opts *NewTransactorOpts) (*Transactor, error) {
	var err error

	txr := &Transactor{
		m:             make(map[string]map[transactionsv1.Ledger]*transactionsv1.Account),
		runner:        make(chan func() error, 100),
		broadcaster:   make(chan *transactionsv1.TransferCompleteSubscribeResponse, 1000),
		ClientStreams: make(map[string]connect_go.StreamingHandlerConn),
		RWMutex:       deadlock.RWMutex{},
	}

	if opts == nil {
		return nil, fmt.Errorf("transactor config is nil")
	}

	txr.log = opts.Log

	txr.Storage, err = storage.NewStorage(opts.StorageOpts)
	if err != nil {
		return nil, err
	}

	accounts, err := txr.Storage.GetAllAccounts()
	if err != nil {
		txr.log.Error().Err(err).Msg("unable to retrieve user account balances")
		return nil, err
	}

	txr.Lock()
	for _, account := range accounts {
		ledger := account.Ledger
		// create user ledger account map if not exist
		if _, ok := txr.m[account.UserId]; !ok {
			txr.m[account.UserId] = make(map[transactionsv1.Ledger]*transactionsv1.Account)
		}
		// create account on the ledger map
		txr.m[account.UserId][ledger] = account
	}
	txr.Unlock()

	go txr.run()
	go txr.broadcast()

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

func (t *Transactor) broadcast() {
	for res := range t.broadcaster {
		t.RLock()
		for id, conn := range t.ClientStreams {
			err := conn.Send(res)
			if err != nil {
				delete(t.ClientStreams, id)
				t.log.Error().Err(err).Msg("failed to send")
				return
			}
		}
		t.RUnlock()
	}
}

func (t *Transactor) TransferCompleteSubscribe(
	ctx context.Context,
	req *connect_go.Request[transactionsv1.TransferCompleteSubscribeRequest],
	resp *connect_go.ServerStream[transactionsv1.TransferCompleteSubscribeResponse],
) error {
	t.Lock()
	t.ClientStreams[req.Msg.Id] = resp.Conn()
	t.Unlock()

	for {
		select {
		case <-ctx.Done():
			t.Lock()
			delete(t.ClientStreams, req.Msg.Id)
			t.log.Debug().Str("client id", req.Msg.Id).Msg("removing client")
			t.Unlock()
			return nil
		}
	}
}
