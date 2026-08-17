package svc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	settings "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("SendActivityNotification", func() {
	var (
		svc               service.Graph
		cfg               *config.Config
		gatewayClient     *cs3mocks.GatewayAPIClient
		eventsPublisher   mocks.Publisher
		permissionService *mocks.Permissions

		currentUser = &userpb.User{Id: &userpb.UserId{OpaqueId: "executant"}}

		mention = `{"topic":{"source":"text","value":"storage$space!item"},` +
			`"activityType":"mentioned","teamsAppId":"` + service.WebOfficeAppID + `"}`
	)

	// the recipient is the user in the path, so every request names one
	request := func(userID, body string) *http.Request {
		ctx := revactx.ContextSetUser(context.Background(), currentUser)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userID", userID)

		return httptest.NewRequest(
			http.MethodPost,
			"/graph/v1.0/users/"+userID+"/teamwork/sendActivityNotification",
			strings.NewReader(body),
		).WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	}

	mentions := func() []ocEvents.ResourceMention {
		var mentions []ocEvents.ResourceMention
		for _, call := range eventsPublisher.Calls {
			if mention, ok := call.Arguments[1].(ocEvents.ResourceMention); ok {
				mentions = append(mentions, mention)
			}
		}

		return mentions
	}

	statAs := func(tokens ...string) {
		allowed := make(map[string]struct{}, len(tokens))
		for _, token := range tokens {
			allowed[token] = struct{}{}
		}

		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(
			func(ctx context.Context, _ *provider.StatRequest, _ ...grpc.CallOption) *provider.StatResponse {
				var token string
				if md, ok := grpcmetadata.FromOutgoingContext(ctx); ok {
					token = strings.Join(md.Get(revactx.TokenHeader), "")
				}

				if _, ok := allowed[token]; !ok {
					return &provider.StatResponse{Status: status.NewNotFound(ctx, "not found")}
				}

				return &provider.StatResponse{
					Status: status.NewOK(ctx),
					Info: &provider.ResourceInfo{
						Id: &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "item"},
					},
				}
			}, nil)
	}

	BeforeEach(func() {
		eventsPublisher = mocks.Publisher{}
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		permissionService = &mocks.Permissions{}
		permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).
			Return(&settings.GetPermissionByIDResponse{}, nil)

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(
			func(_ context.Context, req *gateway.AuthenticateRequest, _ ...grpc.CallOption) *gateway.AuthenticateResponse {
				userID := strings.TrimPrefix(req.GetClientId(), "userid:")
				if userID == "nobody" {
					return &gateway.AuthenticateResponse{Status: status.NewNotFound(context.Background(), "not found")}
				}

				return &gateway.AuthenticateResponse{
					Status: status.NewOK(context.Background()),
					Token:  userID + "-token",
					User:   &userpb.User{Id: &userpb.UserId{OpaqueId: userID}},
				}
			}, nil)

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = ""
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
		cfg.MachineAuthAPIKey = "machine-auth-api-key"

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.PermissionService(permissionService),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("notifies the user from the path", func() {
		statAs("", "alice-token")

		rr := httptest.NewRecorder()
		svc.SendActivityNotification(rr, request("alice", mention))

		Expect(rr.Code).To(Equal(http.StatusAccepted))
		Expect(mentions()).To(HaveLen(1))
		Expect(mentions()[0].Executant.GetOpaqueId()).To(Equal("executant"))
		Expect(mentions()[0].UserIDs).To(HaveLen(1))
		Expect(mentions()[0].UserIDs[0].GetOpaqueId()).To(Equal("alice"))
		Expect(mentions()[0].Ref.GetResourceId().GetOpaqueId()).To(Equal("item"))
	})

	It("is unavailable without an events publisher", func() {
		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(pool.GetSelector[gateway.GatewayAPIClient](
				"GatewaySelector",
				"eu.opencloud.api.gateway",
				func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
					return gatewayClient
				},
			)),
			service.PermissionService(permissionService),
		)
		Expect(err).ToNot(HaveOccurred())
		statAs("", "alice-token")

		rr := httptest.NewRecorder()
		svc.SendActivityNotification(rr, request("alice", mention))

		Expect(rr.Code).To(Equal(http.StatusServiceUnavailable))
		Expect(mentions()).To(BeEmpty())
	})

	It("denies a caller without the publish permission", func() {
		permissionService.ExpectedCalls = nil
		permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).
			Return(nil, errors.New("not found"))
		statAs("", "alice-token")

		rr := httptest.NewRecorder()
		svc.SendActivityNotification(rr, request("alice", mention))

		Expect(rr.Code).To(Equal(http.StatusForbidden))
		Expect(mentions()).To(BeEmpty())
	})

	It("hides the item from a caller who cannot see it", func() {
		statAs("alice-token")

		rr := httptest.NewRecorder()
		svc.SendActivityNotification(rr, request("alice", mention))

		Expect(rr.Code).To(Equal(http.StatusNotFound))
		Expect(mentions()).To(BeEmpty())
	})

	// recipient failures look like success so the sender cannot probe access
	It("silently drops a mention for a recipient who cannot see the item", func() {
		statAs("")

		rr := httptest.NewRecorder()
		svc.SendActivityNotification(rr, request("alice", mention))

		Expect(rr.Code).To(Equal(http.StatusAccepted))
		Expect(mentions()).To(BeEmpty())
	})

	It("drop a mention for a recipient that does not exist", func() {
		statAs("", "alice-token")

		rr := httptest.NewRecorder()
		svc.SendActivityNotification(rr, request("nobody", mention))

		Expect(rr.Code).To(Equal(http.StatusAccepted))
		Expect(mentions()).To(BeEmpty())
	})

	DescribeTable("rejects a malformed body",
		func(body string) {
			statAs("", "alice-token")

			rr := httptest.NewRecorder()
			svc.SendActivityNotification(rr, request("alice", body))

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			Expect(mentions()).To(BeEmpty())
		},
		Entry("no json", `not json`),
		Entry("unknown field", `{"topic":{"source":"text","value":"storage$space!item"},"activityType":"mentioned","teamsAppId":"`+service.WebOfficeAppID+`","chainId":1}`),
		Entry("template parameters", `{"topic":{"source":"text","value":"storage$space!item"},"activityType":"mentioned","teamsAppId":"`+service.WebOfficeAppID+`","templateParameters":[{"name":"actor","value":"someone else"}]}`),
		Entry("no topic", `{"activityType":"mentioned","teamsAppId":"`+service.WebOfficeAppID+`"}`),
		Entry("topic source entityUrl", `{"topic":{"source":"entityUrl","value":"https://cloud.opencloud.test/f/item"},"activityType":"mentioned","teamsAppId":"`+service.WebOfficeAppID+`"}`),
		Entry("topic value is no resource id", `{"topic":{"source":"text","value":"not-an-id"},"activityType":"mentioned","teamsAppId":"`+service.WebOfficeAppID+`"}`),
		Entry("unknown activityType", `{"topic":{"source":"text","value":"storage$space!item"},"activityType":"reactedTo","teamsAppId":"`+service.WebOfficeAppID+`"}`),
		Entry("unknown teamsAppId", `{"topic":{"source":"text","value":"storage$space!item"},"activityType":"mentioned","teamsAppId":"14a4bd3a-1e0f-4a2e-9f30-1cc1f0d0a1cd"}`),
		Entry("no teamsAppId", `{"topic":{"source":"text","value":"storage$space!item"},"activityType":"mentioned"}`),
	)
})
