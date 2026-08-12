package notification_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3Permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	cs3RPC "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	revaCtx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	revaEvents "github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/events"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/notification"
)

const (
	requesterToken = "requester-token"
	fileID         = "storage$space!opaque"
)

type publisher struct {
	events []any
}

func (p *publisher) Publish(_ string, event any, _ ...events.PublishOption) error {
	p.events = append(p.events, event)
	return nil
}

func (p *publisher) mentions() []ocEvents.ResourceMention {
	var mentions []ocEvents.ResourceMention
	for _, event := range p.events {
		if mention, ok := event.(ocEvents.ResourceMention); ok {
			mentions = append(mentions, mention)
		}
	}

	return mentions
}

func token(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}

	return strings.Join(md.Get(revaCtx.TokenHeader), "")
}

func statOK() *storageprovider.StatResponse {
	return &storageprovider.StatResponse{
		Status: &cs3RPC.Status{Code: cs3RPC.Code_CODE_OK},
		Info: &storageprovider.ResourceInfo{
			Id: &storageprovider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "opaque"},
		},
	}
}

func newService(t *testing.T, p revaEvents.Publisher, canStat ...string) notification.Service {
	t.Helper()

	gatewayAPIClient := &cs3mocks.GatewayAPIClient{}
	gatewayAPIClient.On("CheckPermission", mock.Anything, mock.Anything).Return(
		&cs3Permissions.CheckPermissionResponse{Status: &cs3RPC.Status{Code: cs3RPC.Code_CODE_OK}}, nil)

	gatewayAPIClient.On("Authenticate", mock.Anything, mock.Anything).Return(
		func(_ context.Context, req *gateway.AuthenticateRequest, _ ...grpc.CallOption) *gateway.AuthenticateResponse {
			userID := strings.TrimPrefix(req.GetClientId(), "userid:")

			return &gateway.AuthenticateResponse{
				Status: &cs3RPC.Status{Code: cs3RPC.Code_CODE_OK},
				Token:  userID + "-token",
				User:   &userpb.User{Id: &userpb.UserId{OpaqueId: userID}},
			}
		}, nil)

	allowed := map[string]struct{}{}
	for _, name := range canStat {
		allowed[name+"-token"] = struct{}{}
	}

	gatewayAPIClient.On("Stat", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, _ *storageprovider.StatRequest, _ ...grpc.CallOption) *storageprovider.StatResponse {
			if _, ok := allowed[token(ctx)]; !ok {
				return &storageprovider.StatResponse{Status: &cs3RPC.Status{Code: cs3RPC.Code_CODE_NOT_FOUND}}
			}

			return statOK()
		}, nil)

	svc, err := notification.NewService(
		notification.ServiceOptions{}.
			WithLogger(log.NopLogger()).
			WithEventPublisher(p).
			WithMachineAuthAPIKey("machine-auth-api-key").
			WithGatewaySelector(pool.GetSelector[gateway.GatewayAPIClient](
				"GatewaySelector"+t.Name(),
				"eu.opencloud.api.gateway",
				func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
					return gatewayAPIClient
				},
			)),
	)
	require.NoError(t, err)

	return svc
}

func newRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/collaboration/notify", strings.NewReader(body))
	req.Header.Set(revaCtx.TokenHeader, requesterToken)

	return req.WithContext(revaCtx.ContextSetUser(req.Context(), &userpb.User{
		Id: &userpb.UserId{OpaqueId: "requester"},
	}))
}

func TestHandleNotification_RequesterWithoutAccess(t *testing.T) {
	p := &publisher{}
	svc := newService(t, p, "alice")

	resp := httptest.NewRecorder()
	svc.HandleNotification(resp, newRequest(`{"userIDs":["alice"],"fileID":"`+fileID+`"}`))

	require.Equal(t, http.StatusNotFound, resp.Code)
	require.Empty(t, p.mentions())
}

func TestHandleNotification_SkipsRecipientWithoutAccess(t *testing.T) {
	p := &publisher{}
	svc := newService(t, p, "requester", "alice")

	resp := httptest.NewRecorder()
	svc.HandleNotification(resp, newRequest(`{"userIDs":["alice","carol"],"fileID":"`+fileID+`"}`))

	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, p.mentions(), 1)
	require.Len(t, p.mentions()[0].UserIDs, 1)
	require.Equal(t, "alice", p.mentions()[0].UserIDs[0].GetOpaqueId())
}

func TestHandleNotification_NoRecipientLeft(t *testing.T) {
	p := &publisher{}
	svc := newService(t, p, "requester")

	resp := httptest.NewRecorder()
	svc.HandleNotification(resp, newRequest(`{"userIDs":["alice"],"fileID":"`+fileID+`"}`))

	require.Equal(t, http.StatusOK, resp.Code)
	require.Empty(t, p.mentions())
}

func TestHandleNotification_DedupsUserIDs(t *testing.T) {
	p := &publisher{}
	svc := newService(t, p, "requester", "alice")

	resp := httptest.NewRecorder()
	svc.HandleNotification(resp, newRequest(`{"userIDs":["alice","alice","alice"],"fileID":"`+fileID+`"}`))

	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, p.mentions(), 1)
	require.Len(t, p.mentions()[0].UserIDs, 1)
}

func TestHandleNotification_BadRequest(t *testing.T) {
	for name, body := range map[string]string{
		"no json":       `not json`,
		"no userIDs":    `{"fileID":"` + fileID + `"}`,
		"empty userIDs": `{"userIDs":[],"fileID":"` + fileID + `"}`,
		"no fileID":     `{"userIDs":["alice"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := &publisher{}
			svc := newService(t, p, "requester", "alice")

			resp := httptest.NewRecorder()
			svc.HandleNotification(resp, newRequest(body))

			require.Equal(t, http.StatusBadRequest, resp.Code)
			require.Empty(t, p.mentions())
		})
	}
}
