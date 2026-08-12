package notification

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"google.golang.org/grpc/metadata"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/collaboration"
)

type ServiceOptions struct {
	logger            log.Logger                                `validate:"required"`
	eventPublisher    events.Publisher                          `validate:"required"`
	gatewaySelector   pool.Selectable[gateway.GatewayAPIClient] `validate:"required"`
	machineAuthAPIKey string                                    `validate:"required,min=1"`
}

func (o ServiceOptions) WithLogger(logger log.Logger) ServiceOptions {
	o.logger = logger
	return o
}

func (o ServiceOptions) WithEventPublisher(eventPublisher events.Publisher) ServiceOptions {
	o.eventPublisher = eventPublisher
	return o
}

func (o ServiceOptions) WithMachineAuthAPIKey(key string) ServiceOptions {
	o.machineAuthAPIKey = key
	return o
}

func (o ServiceOptions) WithGatewaySelector(gws pool.Selectable[gateway.GatewayAPIClient]) ServiceOptions {
	o.gatewaySelector = gws
	return o
}

type Service struct {
	log               log.Logger
	eventPublisher    events.Publisher
	gatewaySelector   pool.Selectable[gateway.GatewayAPIClient]
	machineAuthAPIKey string
}

func NewService(options ServiceOptions) (Service, error) {
	if err := validate.Struct(options); err != nil {
		return Service{}, err
	}

	return Service{
		log:               options.logger,
		eventPublisher:    options.eventPublisher,
		gatewaySelector:   options.gatewaySelector,
		machineAuthAPIKey: options.machineAuthAPIKey,
	}, nil
}

func (s Service) HandleNotification(w http.ResponseWriter, r *http.Request) {
	ctx := metadata.AppendToOutgoingContext(r.Context(), revactx.TokenHeader, r.Header.Get(revactx.TokenHeader))

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	requestUser, canManage, err := collaboration.CheckPermissions(gatewayClient, ctx, collaboration.PermissionCollaborationPublishNotification)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case !canManage:
		w.WriteHeader(http.StatusForbidden)
		return
	}

	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var data = struct {
		UserIDs []string `json:"userIDs" validate:"required,min=1"`
		FileID  string   `json:"fileID" validate:"required"`
	}{}
	if err := json.Unmarshal(body, &data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resourceID, err := storagespace.ParseID(data.FileID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ref := &storageprovider.Reference{ResourceId: &resourceID}

	// accept the mention only if the requester can see the file himself
	statResponse, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: ref})
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case statResponse.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	event := ocEvents.ResourceMention{
		Executant: requestUser.GetId(),
		Ref:       &storageprovider.Reference{ResourceId: statResponse.GetInfo().GetId()},
		Timestamp: time.Now(),
	}

	seen := make(map[string]struct{}, len(data.UserIDs))
	for _, userID := range data.UserIDs {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}

		authResponse, err := gatewayClient.Authenticate(r.Context(), &gateway.AuthenticateRequest{
			Type:         "machine",
			ClientId:     "userid:" + userID,
			ClientSecret: s.machineAuthAPIKey,
		})
		if err != nil || authResponse.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			s.log.Debug().Str("userID", userID).Msg("could not authenticate the mentioned user, skipping")
			continue
		}

		statResponse, err := gatewayClient.Stat(
			metadata.AppendToOutgoingContext(r.Context(), revactx.TokenHeader, authResponse.GetToken()),
			&storageprovider.StatRequest{Ref: ref},
		)
		if err != nil || statResponse.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			s.log.Debug().Str("userID", userID).Msg("mentioned user has no access to the resource, skipping")
			continue
		}

		event.UserIDs = append(event.UserIDs, authResponse.GetUser().GetId())
	}

	if len(event.UserIDs) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := events.Publish(r.Context(), s.eventPublisher, event); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
