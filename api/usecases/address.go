package usecases

import (
	"context"
)

type AddressUsecases struct {
}

func (u *AddressUsecases) CreateAddress(ctx context.Context, userID int64, currency string) (string, error) {
	return "", nil
}
