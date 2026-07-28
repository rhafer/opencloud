package announcement

import (
	"encoding/json"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	permissionsapi "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/pkg/errors"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// _writePermission is the settings permission required to manage the announcement.
const _writePermission = "Announcement.Write"

// _maxBodySize caps the announcement request body. The info text is Markdown and ends up in
// the public config.json that every client loads on bootstrap, so it must stay small.
const _maxBodySize = 50 << 10 // 50 KiB

// ServiceOptions defines the options to configure the Service.
type ServiceOptions struct {
	store           *Store
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// WithStore sets the announcement store.
func (o ServiceOptions) WithStore(s *Store) ServiceOptions {
	o.store = s
	return o
}

// WithGatewaySelector sets the gateway selector.
func (o ServiceOptions) WithGatewaySelector(gws pool.Selectable[gateway.GatewayAPIClient]) ServiceOptions {
	o.gatewaySelector = gws
	return o
}

// validate validates the input parameters.
func (o ServiceOptions) validate() error {
	if o.store == nil {
		return errors.New("store is required")
	}

	if o.gatewaySelector == nil {
		return errors.New("gatewaySelector is required")
	}

	return nil
}

// Service exposes the http handlers to manage the announcement.
type Service struct {
	store           *Store
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// NewService initializes a new Service.
func NewService(options ServiceOptions) (Service, error) {
	if err := options.validate(); err != nil {
		return Service{}, err
	}

	return Service{
		store:           options.store,
		gatewaySelector: options.gatewaySelector,
	}, nil
}

// Get returns the full stored announcement (including disabled ones) for management.
func (s Service) Get(w http.ResponseWriter, r *http.Request) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	rsp, err := gatewayClient.CheckPermission(r.Context(), &permissionsapi.CheckPermissionRequest{
		Permission: _writePermission,
		SubjectRef: &permissionsapi.SubjectReference{
			Spec: &permissionsapi.SubjectReference_UserId{
				UserId: user.GetId(),
			},
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if rsp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	a, err := s.store.Get()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(a); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Set persists the announcement provided in the request body.
func (s Service) Set(w http.ResponseWriter, r *http.Request) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	rsp, err := gatewayClient.CheckPermission(r.Context(), &permissionsapi.CheckPermissionRequest{
		Permission: _writePermission,
		SubjectRef: &permissionsapi.SubjectReference{
			Spec: &permissionsapi.SubjectReference_UserId{
				UserId: user.GetId(),
			},
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if rsp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var body Announcement
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, _maxBodySize)).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// an announcement without a banner text is nothing to show, so remove it entirely
	if body.BannerText == "" {
		if err := s.store.Delete(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if err := s.store.Set(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
