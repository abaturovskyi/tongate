package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/abaturovskyi/tongate/api/types"
	"github.com/abaturovskyi/tongate/api/usecases"
	"github.com/uptrace/bunrouter"
)

type Handler struct {
	AddressUsecase usecases.AddressUsecase
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(g *bunrouter.Group) {
	g.POST("/address", h.createAddress)
}

func (h *Handler) createAddress(w http.ResponseWriter, req bunrouter.Request) error {
	var data = types.CreateAddressRequest{}

	err := json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		writeHttpError(w, http.StatusBadRequest, fmt.Sprintf("decode payload data err: %v", err))
		return err
	}

	addr, err := h.AddressUsecase.CreateAddress(req.Context(), data.UserID, data.Currency)
	if err != nil {
		writeHttpError(w, http.StatusInternalServerError, fmt.Sprintf("generate address err: %v", err))
		return nil
	}
	res := types.CreateAddressResponse{
		Address: addr,
	}
	w.WriteHeader(http.StatusOK)
	return bunrouter.JSON(w, res)
}

func writeHttpError(resp http.ResponseWriter, status int, comment string) {
	body := struct {
		Error string `json:"error"`
	}{
		Error: comment,
	}
	resp.WriteHeader(status)
	err := json.NewEncoder(resp).Encode(body)
	if err != nil {
		// log.Errorf("json encode error: %v", err)
	}
}
