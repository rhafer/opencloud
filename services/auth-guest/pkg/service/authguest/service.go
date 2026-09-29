// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	"context"
	"errors"
	"fmt"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var ErrExpired = errors.New("token expired")
var ErrAlreadyRedeemed = errors.New("token already redeemed")
var ErrShareNotFound = errors.New("share not found")
var ErrShareExpired = errors.New("share expired")

const invitationTokenTTL = 30 * time.Minute

// RedeemError wraps a redeem failure together with the share id. The HTTP
// transport inspects ErrorType to choose a status code and message.
type RedeemError struct {
	ErrorType error
	ShareID   string
}

func (e *RedeemError) Error() string { return e.ErrorType.Error() }

// AuthGuest is the domain service used by the transport and event layers.
type AuthGuest interface {
	CreateToken(ctx context.Context, shareID string) (*token.Token, error)
	Redeem(ctx context.Context, tokenString string) (string, error)
	CleanupShare(shareID string) error
}

var _ AuthGuest = (*AuthGuestService)(nil)

// AuthGuestService contains the business logic shared by auth-guest transport services.
type AuthGuestService struct {
	tokenSvc        *token.TokenService
	store           storage.Manager
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	serviceAccount  config.ServiceAccount
	jwtService      *jwt.JwtService
}

func NewAuthGuestService(tokenSvc *token.TokenService, store storage.Manager, opts ...Option) *AuthGuestService {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	return &AuthGuestService{
		tokenSvc:        tokenSvc,
		store:           store,
		gatewaySelector: o.GatewaySelector,
		serviceAccount:  o.ServiceAccount,
		jwtService:      o.JWT,
	}
}

func (s *AuthGuestService) CreateToken(ctx context.Context, shareID string) (*token.Token, error) {
	tok, err := s.tokenSvc.Generate(shareID)
	if err != nil {
		return nil, err
	}

	if err := s.store.Add(storage.Record{
		ShareID:     shareID,
		ShareIDHash: tok.ShareIDHash,
		SecretHash:  tok.SecretHash(),
		Expiry:      time.Now().Add(invitationTokenTTL),
		Redeemed:    false,
	}); err != nil {
		return nil, err
	}

	return tok, nil
}

// Redeem validates a token and its share and exchanges them for a session token.
func (s *AuthGuestService) Redeem(ctx context.Context, tokenString string) (string, error) {
	rec, err := s.verifyToken(tokenString)
	if err != nil {
		return "", err
	}

	if _, err := s.validateShare(ctx, rec.ShareID); err != nil {
		return "", err
	}

	if err := s.store.Redeem(rec.ShareIDHash); err != nil {
		if errors.Is(err, storage.ErrAlreadyRedeemed) {
			return "", &RedeemError{ErrorType: ErrAlreadyRedeemed, ShareID: rec.ShareID}
		}
		return "", err
	}

	return s.jwtService.Sign(rec.ShareID)
}

// CleanupShare removes a share's token record from storage. Missing records are ignored.
func (s *AuthGuestService) CleanupShare(shareID string) error {
	shareIDHash := token.Hash(shareID)
	err := s.store.Remove(shareIDHash)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	return nil
}

// VerifyToken validates a token and returns its stored record.
func (s *AuthGuestService) verifyToken(tokenString string) (*storage.Record, error) {
	tok, err := s.tokenSvc.Parse(tokenString)
	if err != nil {
		return nil, &RedeemError{ErrorType: err}
	}

	rec, err := s.store.Get(tok.ShareIDHash)
	if err != nil {
		return nil, &RedeemError{ErrorType: err}
	}

	if err := s.tokenSvc.Verify(*tok, rec.SecretHash); err != nil {
		return nil, &RedeemError{ErrorType: err, ShareID: rec.ShareID}
	}

	if !rec.Expiry.IsZero() && rec.Expiry.Before(time.Now()) {
		return nil, &RedeemError{ErrorType: ErrExpired, ShareID: rec.ShareID}
	}

	if rec.Redeemed {
		return nil, &RedeemError{ErrorType: ErrAlreadyRedeemed, ShareID: rec.ShareID}
	}

	return &rec, nil
}

// validateShare extracts the share information from the gateway and checks its existence and expiration.
func (s *AuthGuestService) validateShare(ctx context.Context, shareID string) (*collaboration.Share, error) {
	share, err := s.getShare(ctx, shareID)
	if err != nil {
		return nil, &RedeemError{ErrorType: err, ShareID: shareID}
	}

	if exp := utils.TSToTime(share.GetExpiration()); !exp.IsZero() && exp.Before(time.Now()) {
		return nil, &RedeemError{ErrorType: ErrShareExpired, ShareID: shareID}
	}

	return share, nil
}

// getShare fetches a share from the gateway.
func (s *AuthGuestService) getShare(ctx context.Context, shareID string) (*collaboration.Share, error) {
	gwc, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	ctx, err = utils.GetServiceUserContextWithContext(ctx, gwc, s.serviceAccount.ServiceAccountID, s.serviceAccount.ServiceAccountSecret)
	if err != nil {
		return nil, err
	}

	resp, err := gwc.GetShare(ctx, &collaboration.GetShareRequest{
		Ref: &collaboration.ShareReference{
			Spec: &collaboration.ShareReference_Id{
				Id: &collaboration.ShareId{
					OpaqueId: shareID,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	switch resp.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
	case rpc.Code_CODE_NOT_FOUND:
		return nil, ErrShareNotFound
	default:
		return nil, fmt.Errorf("could not get share %s: %s", shareID, resp.GetStatus().GetMessage())
	}

	share := resp.GetShare()
	if share == nil {
		return nil, ErrShareNotFound
	}

	return share, nil
}
