# Migrating the search index

A version that changes how resources are indexed works on a new index and
leaves the old one untouched. The service starts normally, but the new index is
empty: search finds nothing until it is filled. The old index stays around
until you remove it.

## v7.x.x to %%NEXT%%

### OpenSearch

The new index is `opencloud-resource-v4`. Fill it by indexing all spaces
again; copying the old index over would miss the search sibling fields the
service writes at index time, so copied documents would not be found.

```shell
# the service keeps running while it happens
opencloud search index --all-spaces
```

Once the new index is filled, remove the old one:

```shell
curl -X DELETE "https://os.example.com:9200/opencloud-resource"
```

### bleve

The new index is the `bleve-v4` directory next to the old `bleve` one, both in
`$OC_BASE_DATA_PATH/search` by default (`SEARCH_ENGINE_BLEVE_DATA_PATH`). A
bleve index cannot be copied, index all spaces again:

```shell
opencloud search index --all-spaces
```

Once the new index is filled, remove the old one:

```shell
rm -r "$OC_BASE_DATA_PATH/search/bleve"
```
