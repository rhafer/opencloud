// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest/mocks"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newRedeemHandler(t *testing.T, svc authguest.AuthGuest) http.HandlerFunc {
	t.Helper()
	cfg := &config.Config{
		JWT: config.JWT{
			CookieName: "__Host-oc_guest_session",
			TTL:        time.Hour,
		},
	}
	return RedeemHandler(log.NopLogger(), svc, cfg)
}

func TestRedeemHandler(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("Redeem", mock.Anything, "valid-token").Return("session-token", nil)

	body, err := json.Marshal(RedeemRequest{Token: "valid-token"})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	newRedeemHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

	assert.Equal(t, http.StatusOK, rr.Code)

	var cookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "__Host-oc_guest_session" {
			cookie = c
		}
	}
	require.NotNil(t, cookie)
	assert.Equal(t, "session-token", cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, "/", cookie.Path)
}

func TestRedeemHandlerErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantType   string
		wantShare  string
	}{
		{
			name:       "token expired",
			err:        &authguest.RedeemError{ErrorType: authguest.ErrExpired, ShareID: "share-1"},
			wantStatus: http.StatusUnauthorized,
			wantType:   "token_expired",
			wantShare:  "share-1",
		},
		{
			name:       "token invalid",
			err:        &authguest.RedeemError{ErrorType: token.ErrInvalidToken},
			wantStatus: http.StatusUnauthorized,
			wantType:   "token_invalid",
		},
		{
			name:       "token not found",
			err:        &authguest.RedeemError{ErrorType: storage.ErrNotFound},
			wantStatus: http.StatusNotFound,
			wantType:   "token_not_found",
		},
		{
			name:       "token already redeemed",
			err:        &authguest.RedeemError{ErrorType: authguest.ErrAlreadyRedeemed},
			wantStatus: http.StatusConflict,
			wantType:   "token_already_redeemed",
		},
		{
			name:       "share not found",
			err:        &authguest.RedeemError{ErrorType: authguest.ErrShareNotFound},
			wantStatus: http.StatusNotFound,
			wantType:   "share_not_found",
		},
		{
			name:       "share expired",
			err:        &authguest.RedeemError{ErrorType: authguest.ErrShareExpired},
			wantStatus: http.StatusGone,
			wantType:   "share_expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := mocks.NewAuthGuest(t)
			svcMock.On("Redeem", mock.Anything, "token").Return("", tt.err)

			body, err := json.Marshal(RedeemRequest{Token: "token"})
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			newRedeemHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

			assert.Equal(t, tt.wantStatus, rr.Code)

			var resp errorResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Equal(t, tt.wantType, resp.ErrorType)
			assert.Equal(t, tt.wantShare, resp.ShareID)
		})
	}
}

func TestRedeemHandlerMalformedBody(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	rr := httptest.NewRecorder()
	newRedeemHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json")))

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invalid_request", resp.ErrorType)
}
