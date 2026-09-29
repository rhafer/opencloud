// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
)

type errorResponse struct {
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
	ShareID   string `json:"share_id"`
}

func writeError(w http.ResponseWriter, status int, body errorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeRedeemError(w http.ResponseWriter, err error) {
	var re *authguest.RedeemError
	if !errors.As(err, &re) {
		writeError(w, http.StatusInternalServerError, errorResponse{ErrorType: "internal_error", Message: "An internal error occurred."})
		return
	}

	status := http.StatusInternalServerError
	errorType := "internal_error"
	switch {
	case errors.Is(re.ErrorType, authguest.ErrExpired):
		status, errorType = http.StatusUnauthorized, "token_expired"
	case errors.Is(re.ErrorType, token.ErrInvalidToken):
		status, errorType = http.StatusUnauthorized, "token_invalid"
	case errors.Is(re.ErrorType, storage.ErrNotFound):
		status, errorType = http.StatusNotFound, "token_not_found"
	case errors.Is(re.ErrorType, storage.ErrInvalidHash):
		status, errorType = http.StatusUnauthorized, "token_invalid"
	case errors.Is(re.ErrorType, authguest.ErrAlreadyRedeemed):
		status, errorType = http.StatusConflict, "token_already_redeemed"
	case errors.Is(re.ErrorType, authguest.ErrShareNotFound):
		status, errorType = http.StatusNotFound, "share_not_found"
	case errors.Is(re.ErrorType, authguest.ErrShareExpired):
		status, errorType = http.StatusGone, "share_expired"
	}

	message := re.ErrorType.Error()
	if errorType == "internal_error" {
		message = "An internal error occurred."
	}

	writeError(w, status, errorResponse{ErrorType: errorType, Message: message, ShareID: re.ShareID})
}
