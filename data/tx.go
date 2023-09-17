package data

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

var (
	ErrNotFound = errors.New("not found")
)

type ITx interface {
	RunInTx(context.Context, func(ctx context.Context, tx Tx) error) error
}

type useTx struct {
	client bun.IDB
}

type Tx interface {
	bun.IDB
}

func (t *useTx) RunInTx(ctx context.Context, f func(ctx context.Context, tx Tx) error) error {
	return t.client.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return f(ctx, tx)
	})
}
