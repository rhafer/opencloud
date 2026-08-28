package parity

import (
	"context"
	"fmt"
	"testing"

	bleveSearch "github.com/blevesearch/bleve/v2"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	bleveEngine "github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

type testEngine struct {
	name        string
	backend     search.Engine
	settle      func()
	unavailable string
}

func newEngines(t *testing.T, index string, fixtures []search.Resource) []testEngine {
	t.Helper()

	return []testEngine{
		newBleve(t, fixtures),
		newOpenSearch(t, index, fixtures),
	}
}

func newBleve(t *testing.T, fixtures []search.Resource) testEngine {
	t.Helper()

	mapping, err := bleveEngine.NewMapping()
	require.NoError(t, err, "failed to build the bleve mapping")

	idx, err := bleveSearch.NewMemOnly(mapping)
	require.NoError(t, err, "failed to create the bleve index")
	t.Cleanup(func() { _ = idx.Close() })

	backend := bleveEngine.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
	for _, doc := range fixtures {
		require.NoError(t, backend.Upsert(doc.ID, doc), "bleve upsert %s", doc.ID)
	}

	return testEngine{name: "bleve", backend: backend, settle: func() {}}
}

func newOpenSearch(t *testing.T, index string, fixtures []search.Resource) testEngine {
	t.Helper()

	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	versioned := opensearch.IndexName(index)

	if err := tc.IndicesReset(t.Context(), []string{versioned}); err != nil {
		return testEngine{name: "opensearch", unavailable: err.Error()}
	}

	backend, err := opensearch.NewBackend(index, tc.Client())
	if err != nil {
		return testEngine{name: "opensearch", unavailable: err.Error()}
	}

	t.Cleanup(func() {
		require.NoError(t, tc.IndicesDelete(context.Background(), []string{versioned}))
	})

	settle := func() { tc.Require.IndicesRefresh([]string{versioned}, nil) }
	for _, doc := range fixtures {
		require.NoError(t, backend.Upsert(doc.ID, doc), "opensearch upsert %s", doc.ID)
	}
	settle()

	return testEngine{name: "opensearch", backend: backend, settle: settle}
}

func badRequestAnswer(err error) []string {
	if err == nil {
		return []string{"no error"}
	}

	return []string{"bad request"}
}

func reads(read func(*searchMessage.Match) string) func(*searchService.SearchIndexResponse) []string {
	return readsMany(func(m *searchMessage.Match) []string { return []string{read(m)} })
}

func readsMany(read func(*searchMessage.Match) []string) func(*searchService.SearchIndexResponse) []string {
	return func(resp *searchService.SearchIndexResponse) []string {
		if len(resp.Matches) != 1 {
			return []string{fmt.Sprintf("%d matches", len(resp.Matches))}
		}

		return read(resp.Matches[0])
	}
}

func resourceID(id *searchMessage.ResourceID) string {
	return storagespace.FormatResourceID(&sprovider.ResourceId{
		StorageId: id.GetStorageId(),
		SpaceId:   id.GetSpaceId(),
		OpaqueId:  id.GetOpaqueId(),
	})
}

func hits(t *testing.T, e search.Engine, request *searchService.SearchIndexRequest) []string {
	t.Helper()

	resp, err := e.Search(context.Background(), request)
	require.NoError(t, err, "the query has to answer, an empty result is an answer")

	names := make([]string, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		names = append(names, m.Entity.GetName())
	}

	return names
}
