package opensearch_test

import (
	"fmt"
	"strings"
	"testing"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func TestNewBackend(t *testing.T) {
	t.Run("fails to create if the cluster is not healthy", func(t *testing.T) {
		client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
			Client: opensearchgo.Config{
				Addresses: []string{"http://localhost:1025"},
			},
		})
		require.NoError(t, err, "failed to create OpenSearch client")

		backend, err := opensearch.NewBackend("test-engine-new-engine", client)
		require.Nil(t, backend)
		require.ErrorIs(t, err, opensearch.ErrUnhealthyCluster)
	})
}

func TestEngine_Search(t *testing.T) {
	indexName := "opencloud-test-engine-search"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	document := opensearchtest.Testdata.Resources.File
	tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
	tc.Require.IndicesCount([]string{indexName}, nil, 1)

	t.Run("most simple search", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})

	t.Run("ignores files that are marked as deleted", func(t *testing.T) {
		deletedDocument := opensearchtest.Testdata.Resources.File
		deletedDocument.ID = "1$2!4"
		deletedDocument.Deleted = true

		tc.Require.DocumentCreate(indexName, deletedDocument.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, deletedDocument)))
		tc.Require.IndicesCount([]string{indexName}, nil, 2)

		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})
}

func TestEngine_Upsert(t *testing.T) {
	indexName := "opencloud-test-engine-upsert"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("upsert with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		require.NoError(t, backend.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{indexName}, nil, 1)
	})
}

func TestEngine_Move(t *testing.T) {
	indexName := "opencloud-test-engine-move"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("moves the document to a new path", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		body := opensearchtest.JSONMustMarshal(t, map[string]any{
			"query": map[string]any{
				"ids": map[string]any{
					"values": []string{document.ID},
				},
			},
		})

		resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](t, tc.Require.Search(indexName, strings.NewReader(body)).Hits)
		require.Len(t, resources, 1)
		require.Equal(t, document.Path, resources[0].Path)

		document.Path = "./new/path/to/resource"
		require.NoError(t, backend.Move(document.ID, document.ParentID, document.Path))

		resources = opensearchtest.SearchHitsMustBeConverted[search.Resource](t, tc.Require.Search(indexName, strings.NewReader(body)).Hits)
		require.Len(t, resources, 1)
		require.Equal(t, document.Path, resources[0].Path)
	})
}

func TestEngine_Delete(t *testing.T) {
	indexName := "opencloud-test-engine-delete"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		body := opensearchtest.JSONMustMarshal(t, map[string]any{
			"query": map[string]any{
				"term": map[string]any{
					"Deleted": map[string]any{
						"value": true,
					},
				},
			},
		})

		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 0)

		require.NoError(t, backend.Delete(document.ID))
		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 1)
	})
}

func TestEngine_Restore(t *testing.T) {
	indexName := "opencloud-test-engine-restore"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as not deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		document.Deleted = true
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		body := opensearchtest.JSONMustMarshal(t, map[string]any{
			"query": map[string]any{
				"term": map[string]any{
					"Deleted": map[string]any{
						"value": true,
					},
				},
			},
		})

		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 1)

		require.NoError(t, backend.Restore(document.ID))
		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 0)
	})
}

func TestEngine_Purge(t *testing.T) {
	indexName := "opencloud-test-engine-purge"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("purge with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		require.NoError(t, backend.Purge(document.ID, false))

		tc.Require.IndicesCount([]string{indexName}, nil, 0)
	})

	t.Run("purge resource trees", func(t *testing.T) {
		resourceFolder := opensearchtest.Testdata.Resources.Folder
		tc.Require.DocumentCreate(indexName, resourceFolder.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFolder)))

		resourceFile := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, resourceFile.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFile)))

		tc.Require.IndicesCount([]string{indexName}, nil, 2)

		require.NoError(t, backend.Purge(resourceFolder.ID, false))

		tc.Require.IndicesCount([]string{indexName}, nil, 0)
	})

	t.Run("purge resource trees and ignores undeleted resources", func(t *testing.T) {
		resourceFolder := opensearchtest.Testdata.Resources.Folder
		tc.Require.DocumentCreate(indexName, resourceFolder.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFolder)))

		resourceFile := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, resourceFile.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFile)))

		tc.Require.IndicesCount([]string{indexName}, nil, 2)

		require.NoError(t, backend.Delete(resourceFile.ID))
		tc.Require.IndicesRefresh([]string{indexName}, nil)
		require.NoError(t, backend.Purge(resourceFolder.ID, true))

		tc.Require.IndicesCount([]string{indexName}, nil, 1)
	})
}

func TestEngine_DocCount(t *testing.T) {
	indexName := "opencloud-test-engine-doc-count"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("ignore deleted documents", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		count, err := backend.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)

		tc.Require.Update(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, map[string]any{
			"doc": map[string]any{
				"Deleted": true,
			},
		})))

		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		count, err = backend.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(0), count)
	})
}

// resourceByID fetches a single indexed resource by its document ID.
func resourceByID(t *testing.T, tc *opensearchtest.TestClient, index, id string) search.Resource {
	t.Helper()

	body := opensearchtest.JSONMustMarshal(t, map[string]any{
		"query": map[string]any{
			"ids": map[string]any{
				"values": []string{id},
			},
		},
	})

	resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](t, tc.Require.Search(index, strings.NewReader(body)).Hits)
	require.Len(t, resources, 1)
	return resources[0]
}

// otherRoot returns a copy of the given resource that lives in a different root (space)
// while keeping the same path, so it can be used to assert that cross-root updates do
// not affect identically-named resources in other roots.
func otherRoot(r search.Resource) search.Resource {
	r.ID = "2$2!3"
	r.RootID = "2$2!1"
	r.ParentID = "2$2!2"
	return r
}

// newRootScopeBackend resets the index, applies the resource mapping via NewBackend and
// indexes the given resources. It is used by the root-scope tests below.
func newRootScopeBackend(t *testing.T, indexName string, resources ...search.Resource) (*opensearch.Backend, *opensearchtest.TestClient) {
	t.Helper()

	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	for _, r := range resources {
		tc.Require.DocumentCreate(indexName, r.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, r)))
	}
	tc.Require.IndicesCount([]string{indexName}, nil, len(resources))

	return backend, tc
}

// TestEngine_UpdateSelfAndDescendants_RootScope ensures that updates which affect a
// resource and its descendants (Delete, Restore, Move) are scoped to the root (space)
// of the target resource. Two resources living in different roots may share the exact
// same path, so matching by path alone would incorrectly update the wrong resource.
func TestEngine_UpdateSelfAndDescendants_RootScope(t *testing.T) {
	t.Run("delete only affects the resource in the target root", func(t *testing.T) {
		indexName := "opencloud-test-engine-root-scope-delete"

		target := opensearchtest.Testdata.Resources.File
		other := otherRoot(target)

		backend, tc := newRootScopeBackend(t, indexName, target, other)
		defer tc.Require.IndicesDelete([]string{indexName})

		require.NoError(t, backend.Delete(target.ID))

		require.True(t, resourceByID(t, tc, indexName, target.ID).Deleted, "target resource should be marked as deleted")
		require.False(t, resourceByID(t, tc, indexName, other.ID).Deleted, "resource in a different root must not be affected")
	})

	t.Run("restore only affects the resource in the target root", func(t *testing.T) {
		indexName := "opencloud-test-engine-root-scope-restore"

		target := opensearchtest.Testdata.Resources.File
		target.Deleted = true
		other := otherRoot(target)

		backend, tc := newRootScopeBackend(t, indexName, target, other)
		defer tc.Require.IndicesDelete([]string{indexName})

		require.NoError(t, backend.Restore(target.ID))

		require.False(t, resourceByID(t, tc, indexName, target.ID).Deleted, "target resource should be restored")
		require.True(t, resourceByID(t, tc, indexName, other.ID).Deleted, "resource in a different root must not be affected")
	})

	t.Run("move only affects the resource in the target root", func(t *testing.T) {
		indexName := "opencloud-test-engine-root-scope-move"

		target := opensearchtest.Testdata.Resources.File
		other := otherRoot(target)

		backend, tc := newRootScopeBackend(t, indexName, target, other)
		defer tc.Require.IndicesDelete([]string{indexName})

		require.NoError(t, backend.Move(target.ID, target.ParentID, "./new/path/to/resource"))

		require.Equal(t, "./new/path/to/resource", resourceByID(t, tc, indexName, target.ID).Path, "target resource should be moved")
		require.Equal(t, other.Path, resourceByID(t, tc, indexName, other.ID).Path, "resource in a different root must not be moved")
	})
}
