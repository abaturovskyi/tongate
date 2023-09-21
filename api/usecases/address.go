package usecases

import (
	"context"
)

type AddressUsecase struct {
}

func (u *AddressUsecase) CreateAddress(ctx context.Context, userID int64, currency string) (string, error) {
	return "", nil
}
