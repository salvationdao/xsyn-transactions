package transactor

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/puzpuzpuz/xsync"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
	"xsyn-transactions/gen/transactions/v1"
	"xsyn-transactions/storage"
)

var ErrTimeToClose = fmt.Errorf("closing")
var ErrQueueFull = fmt.Errorf("transaction queue is full")
var ErrUnableToFindAccount = fmt.Errorf("unable to find account")

type Transactor struct {
	Storage     *storage.Storage
	log         *zerolog.Logger
	runner      chan func() error
	broadcaster chan *transactionsv1.TransferCompleteSubscribeResponse

	userMap     map[string]map[transactionsv1.Ledger]*transactionsv1.Account // map[user_id]map[currency]account
	userMapLock deadlock.RWMutex

	// We use this cool package, meant to be faster than using mutex locks to ensure concurrency safeness
	// https://pkg.go.dev/github.com/puzpuzpuz/xsync#Map
	clients *xsync.MapOf[string, connect.StreamingHandlerConn]
}

type NewTransactorOpts struct {
	StorageOpts *storage.Opts
	Log         *zerolog.Logger
}

func NewTransactor(opts *NewTransactorOpts) (*Transactor, error) {
	var err error
	txr := &Transactor{
		runner:      make(chan func() error, 100),
		broadcaster: make(chan *transactionsv1.TransferCompleteSubscribeResponse, 1000),
		userMap:     make(map[string]map[transactionsv1.Ledger]*transactionsv1.Account),
		clients:     xsync.NewMapOf[connect.StreamingHandlerConn](),
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
	if len(accounts) == 0 {
		err = fmt.Errorf("accounts len is 0")
		txr.log.Error().Err(err).Msg("unexpected accounts length, possibly incorrect spin up")
		return nil, err
	}

	txr.userMapLock.Lock()
	for _, account := range accounts {
		ledger := account.Ledger
		// create user ledger account map if not exist
		if _, ok := txr.userMap[account.UserId]; !ok {
			txr.userMap[account.UserId] = make(map[transactionsv1.Ledger]*transactionsv1.Account)
		}
		// create account on the ledger map
		txr.userMap[account.UserId][ledger] = account
	}
	txr.userMapLock.Unlock()

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
		t.clients.Range(func(key string, conn connect.StreamingHandlerConn) bool {
			err := conn.Send(res)
			if err != nil {
				t.clients.Delete(key) // it is safe to modify the map while iterating it
				t.log.Error().Err(err).Msg("failed to send")
				return true
			}
			return true
		})
	}
}

func (t *Transactor) TransferCompleteSubscribe(
	ctx context.Context,
	req *connect.Request[transactionsv1.TransferCompleteSubscribeRequest],
	resp *connect.ServerStream[transactionsv1.TransferCompleteSubscribeResponse],
) error {
	t.log.Info().Str("clientID ", req.Msg.Id).Msg("new transfer complete subscriber")
	t.clients.Store(req.Msg.Id, resp.Conn())

	for {
		select {
		case <-ctx.Done():
			t.clients.Delete(req.Msg.Id)
			t.log.Debug().Str("clientID", req.Msg.Id).Msg("removing client")
			return nil
		}
	}
}
