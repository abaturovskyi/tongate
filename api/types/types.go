package types

type CreateAddressRequest struct {
	UserID   int64  `json:"user_id"`
	Currency string `json:"currency"`
}

type CreateAddressResponse struct {
	Address string `json:"address"`
}

type WalletAddress struct {
	Address  string `json:"address"`
	Currency string `json:"currency"`
}
