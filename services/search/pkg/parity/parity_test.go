package parity

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"

	"github.com/opencloud-eu/opencloud/pkg/kql"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var defaultConfig *config.Config

func TestMain(m *testing.M) {
	// the fixtures are stamped with fixtureNow, holding the clock still keeps
	// "today" on their side of midnight for the whole run
	kql.PatchTimeNow(func() time.Time { return fixtureNow })

	cfg, done, err := opensearchtest.SetupTests(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to setup tests:", err)
		os.Exit(1)
		return
	}

	defaultConfig = cfg
	code := m.Run()
	writeMatrix()
	done()
	os.Exit(code)
}

type queryCase struct {
	id             int
	query          string
	want           []string
	limit          int32
	wantCount      *int
	ref            *searchMessage.Reference
	wantBadRequest bool
}

func (c queryCase) label(group string) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(group), c.id)
}

func (c queryCase) name(group string) string {
	return c.label(group) + " " + c.query
}

type queryGroup struct {
	name     string
	fixtures []search.Resource
	cases    []queryCase
}

func queryGroups() []queryGroup {
	return []queryGroup{
		nameGroup(),
		extensionGroup(),
		tagsGroup(),
		titleGroup(),
		contentGroup(),
		favoritesGroup(),
		mediatypeGroup(),
		pathGroup(),
		fieldsGroup(),
		deletedGroup(),
		visibilityGroup(),
		booleanGroup(),
		samenameGroup(),
		stressGroup(),
		everythingGroup(),
		rangeGroup(),
		scopeGroup(),
		invalidGroup(),
	}
}

type responseCase struct {
	id    int
	do    func(e search.Engine) error
	after string
	query string
	reads string
	read  func(*searchService.SearchIndexResponse) []string
	want  []string
}

func (c responseCase) label(group string) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(group), c.id)
}

func (c responseCase) name(group string) string {
	return c.label(group) + " " + c.reads
}

type responseGroup struct {
	name     string
	fixtures []search.Resource
	cases    []responseCase
}

func responseGroups() []responseGroup {
	return []responseGroup{
		entityGroup(),
		metadataGroup(),
	}
}

type lifecycleCase struct {
	id      int
	title   string
	do      func(e search.Engine) error
	expect  []expectation
	wantErr bool

	fixtures []search.Resource

	wantDocCount *uint64
}

type expectation struct {
	query string
	want  []string
}

func (c lifecycleCase) label(group string) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(group), c.id)
}

func (c lifecycleCase) name(group string) string {
	return c.label(group) + " " + c.title
}

type lifecycleGroup struct {
	name     string
	fixtures []search.Resource
	cases    []lifecycleCase
}

func lifecycleGroups() []lifecycleGroup {
	return []lifecycleGroup{
		deleteLifecycle(),
		restoreLifecycle(),
		purgeLifecycle(),
		purgeSpaceLifecycle(),
		moveLifecycle(),
		rootScopeLifecycle(),
		casePathLifecycle(),
		hiddenLifecycle(),
		upsertLifecycle(),
		idempotencyLifecycle(),
		batchLifecycle(),
	}
}

func TestEngineParity(t *testing.T) {
	for groupAt, group := range queryGroups() {
		t.Run(group.name, func(t *testing.T) {
			t.Parallel()

			engines := newEngines(t, "opencloud-test-engine-parity-"+group.name, group.fixtures)

			for caseAt, c := range group.cases {
				row := matrixRow{
					section: "Queries", group: group.name, id: c.label(group.name), query: c.query, want: c.want, wantCount: c.wantCount, wantBadRequest: c.wantBadRequest, limit: c.limit, ref: c.ref,
					groupAt: groupAt, caseAt: caseAt,
				}

				t.Run(c.name(group.name), func(t *testing.T) {
					for _, e := range engines {
						t.Run(e.name, func(t *testing.T) {
							if e.unavailable != "" {
								recordSkip(row, e.name)
								t.Skip(e.unavailable)
							}

							request := &searchService.SearchIndexRequest{Query: c.query, PageSize: c.limit, Ref: c.ref}

							if c.wantBadRequest {
								_, err := e.backend.Search(context.Background(), request)
								recordAnswer(row, e.name, badRequestAnswer(err))
								require.IsType(t, errtypes.BadRequest(""), err)

								return
							}

							answer := hits(t, e.backend, request)
							recordAnswer(row, e.name, answer)

							if c.wantCount != nil {
								require.Len(t, answer, *c.wantCount)
								return
							}

							require.ElementsMatch(t, c.want, answer)
						})
					}
				})
			}
		})
	}
}

func TestEngineParityResponse(t *testing.T) {
	offset := len(queryGroups()) + len(lifecycleGroups())

	for groupAt, group := range responseGroups() {
		t.Run(group.name, func(t *testing.T) {
			t.Parallel()

			shared := newEngines(t, "opencloud-test-engine-parity-"+group.name, group.fixtures)

			for caseAt, c := range group.cases {
				row := matrixRow{
					section: "Response", group: group.name, id: c.label(group.name), query: c.query, reads: c.reads, context: c.after, want: c.want,
					groupAt: offset + groupAt, caseAt: caseAt,
				}

				t.Run(c.name(group.name), func(t *testing.T) {
					engines := shared
					if c.do != nil {
						engines = newEngines(t, fmt.Sprintf("opencloud-test-engine-parity-%s-%d", group.name, c.id), group.fixtures)
					}

					for _, e := range engines {
						t.Run(e.name, func(t *testing.T) {
							if e.unavailable != "" {
								recordSkip(row, e.name)
								t.Skip(e.unavailable)
							}

							if c.do != nil {
								require.NoError(t, c.do(e.backend), "the operation under test failed")
								e.settle()
							}

							resp, err := e.backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: c.query})
							require.NoError(t, err, "the query has to answer, an empty result is an answer")

							answer := c.read(resp)
							recordAnswer(row, e.name, answer)
							require.Equal(t, c.want, answer, c.reads)
						})
					}
				})
			}
		})
	}
}

func TestEngineParityLifecycle(t *testing.T) {
	for groupAt, group := range lifecycleGroups() {
		t.Run(group.name, func(t *testing.T) {
			t.Parallel()

			for caseAt, c := range group.cases {
				t.Run(c.name(group.name), func(t *testing.T) {
					t.Parallel()

					fixtures := c.fixtures
					if fixtures == nil {
						fixtures = group.fixtures
					}

					index := fmt.Sprintf("opencloud-test-engine-parity-%s-%d", group.name, c.id)
					rows := c.matrixRows(group.name, len(queryGroups())+groupAt, caseAt)

					for _, e := range newEngines(t, index, fixtures) {
						t.Run(e.name, func(t *testing.T) {
							if e.unavailable != "" {
								for _, row := range rows {
									recordSkip(row, e.name)
								}

								t.Skip(e.unavailable)
							}

							err := c.do(e.backend)
							if c.wantErr {
								require.Error(t, err, "the operation had to report that it did not find the resource")
							} else {
								require.NoError(t, err, "the operation under test failed")
							}

							e.settle()

							for i, expect := range c.expect {
								answer := hits(t, e.backend, &searchService.SearchIndexRequest{Query: expect.query})
								recordAnswer(rows[i], e.name, answer)
								require.ElementsMatch(t, expect.want, answer, expect.query)
							}

							if c.wantDocCount != nil {
								count, err := e.backend.DocCount()
								require.NoError(t, err)
								recordAnswer(rows[len(rows)-1], e.name, []string{fmt.Sprint(count)})
								require.Equal(t, *c.wantDocCount, count, "documents left in the index")
							}
						})
					}
				})
			}
		})
	}
}
