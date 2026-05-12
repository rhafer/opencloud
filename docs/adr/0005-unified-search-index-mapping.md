---
title: "5. Unified Search Index Mapping"
---

* Status: proposed
* Deciders: []
* Date: 2026-04-23

Reference: https://github.com/opencloud-eu/opencloud/pull/2659, a
proof-of-concept that demonstrates the proposal end-to-end. It is
not presumed accepted by this ADR; scope and APIs there will be
revisited once the decision here lands.

## Context and Problem Statement

The search service supports two backends, bleve (embedded) and
OpenSearch (external). Each backend currently carries its own,
independently maintained description of the index layout:

- The bleve backend hand-builds a document mapping that explicitly
  declares only Name, Tags, Favorites and Content. Everything else,
  including the entire facet block (audio, image, photo, location),
  is left to bleve's dynamic mapping.
- The OpenSearch backend ships a static JSON template that covers a
  similar but not identical subset, plus a few OpenSearch-specific
  primitives (path_hierarchy analyzer, wildcard MimeType). It does
  not list the facet sub-fields either; they are produced by
  OpenSearch's dynamic templating at first write.
- The graph DriveItem assembly path keeps its own private copy of a
  reflection-based walker to turn CS3 ArbitraryMetadata back into
  typed libregraph facets, parallel to the search service's
  reflection helpers but maintained separately.
- The bleve KQL compiler keeps a hand-maintained set of field names
  whose query values need to be pre-lowercased, with a comment that
  literally says "Keep in sync with index.go".

The current implementation has four concrete problems:

1. **The two backends do not behave the same.** Both rely on their
   own implicit defaults for fields that are not explicitly
   declared. The inferred shapes differ: bleve produces keyword-
   analyzed text, OpenSearch produces a `text + keyword` multi-field
   with auto-detected dates. Nobody has written down which behavior
   is the intended one.
2. **The indexed facet metadata is effectively unreachable.** The
   audio, image, photo and location sub-fields go into the index and
   take up space, but no caller of the search service can actually
   use them. The KQL compiler does not support the dot syntax needed
   to refer to `audio.artist` on the OpenSearch side, and until
   #2633 the bleve-side compiler unconditionally lowercased query
   values even against case-preserving fields. The hit and REPORT
   paths do not expose facet values back to consumers either.
3. **Drift risk.** The OpenSearch JSON template is a subset of what
   actually gets indexed. Even where it overlaps with the bleve
   mapping it diverges on analyzer choices. Because point 2 means
   nothing queries the diverging fields, the divergence has been
   invisible, but it would surface the moment the first working
   cross-backend facet query landed.
4. **Per-facet cost.** Adding a new facet (motionPhoto, etc.)
   requires coordinated edits across the proto message, both backend
   mappings, the bleve hit converters, the OpenSearch convert
   closures, the search service's metadata persistence, the graph
   DriveItem assembly, and the KQL compiler's lowercasing set. Most
   of those edits are boilerplate following a copy-paste pattern.
   Adding a genuinely new index capability (geopoint, wildcard,
   ...) means wiring it in at every one of those sites, and there
   is no single place to hook a type-specific adapter.

### A note on backwards compatibility

The unreachability above (problem 2) has a useful corollary for
this ADR: **changing the indexed shape of the facet fields cannot
break any existing client of the search service**, because no
client successfully reads them today. The behavior changes
discussed below are therefore additive in a literal sense; nothing
that works today stops working as a result.

## Decision Drivers

* **Predictable OpenCloud API behavior independent of backend.**
  Consumers of the search service should be able to rely on the
  documented behavior of the API, not on which backend happens to
  be configured. Today the same query can give different results
  depending on whether bleve or OpenSearch is wired in (bleve's
  dynamic default is `keyword`, exact match; OpenSearch's dynamic
  default is `text + keyword`, also matches sub-tokens of a
  string). That is backend-implementation leakage, and trying to
  keep the two implicit defaults synchronized has not worked.
* Single source of truth for the indexed schema, so the two backends
  cannot drift silently again.
* Reduce the per-facet cost so future facets (motionPhoto and
  whatever comes next) can be added with minimal boilerplate.
* Establish a single place to hook index-type-specific behavior, so
  a new capability needs to be implemented at most once per backend
  and then becomes available for any field uniformly.
* A one-time reindex is an acceptable upgrade path. Both bleve and
  OpenSearch store their mapping alongside the data; existing
  indexes keep serving queries against their stored shape without
  any automatic reshaping. Benefiting from the new behavior is done
  by creating a fresh index and re-ingesting, which is the normal
  reindex flow, rather than by inventing migration tooling.

## Considered Options

### Option 1: Do nothing, keep relying on implicit backend defaults

Accept that bleve and OpenSearch each fall back to their own
dynamic-mapping defaults for whatever is not explicitly declared,
and treat the observable search behavior of OpenCloud as "whatever
the configured backend happens to do". Adding a facet stays a
copy-paste coordination across half a dozen sites; the existing
divergence between bleve (keyword) and OpenSearch (`text + keyword`
multi-field plus auto-date detection) stays silently in place
until a working query actually reaches the diverging field and
returns different answers on the two backends.

Low upfront work, but it makes the OpenCloud API behavior a
function of the backend rather than a contract, and it keeps the
per-facet boilerplate cost for every new field.

### Option 2: Generate one backend's mapping from the other

Treat one backend as canonical (likely bleve, because Go types) and
derive the other. Partial answer; it still does not help the reader
path or the graph walker, and still leaves per-facet boilerplate in
non-mapping code.

### Option 3: A struct-driven mapping (chosen)

Let the Go struct that represents an indexed document, together
with a small overrides map, be the single source of truth. A
reflection-based helper walks the struct via json tags and emits
each backend's index mapping. The same definition drives the
write-time path, the hit-decoding path, and the query compiler's
case-folding rules. Any future field follows one declaration in
one place and falls through the whole pipeline consistently.

## Decision Outcome

Adopt Option 3. The Go struct that represents an indexed document,
together with a small overrides map, becomes the single source of
truth for the search index. The bleve and OpenSearch index
mappings, the write-time conversion, the hit-decoding path, and
the query compiler's case-folding rules are all derived from that
same definition. Drift between backends is prevented by
construction, because there is no second place to edit.

The overrides surface stays small. Each entry declares one of a
handful of things per field: a semantic type for fields whose
intent cannot be inferred from the Go type (for example a path-
analyzed field, a fulltext field, a wildcard field, a geopoint
field), an analyzer name where one is needed, or a backend-
specific inclusion knob. Any field that needs something beyond the
inferred defaults gets one line in the overrides map and that one
line flows through every derived piece. Overrides are validated at
startup so a typo fails loudly instead of silently disabling a
setting.

A practical consequence of having one place to hook things: when a
new capability is needed (a geopoint representation, a sibling
field for a different aggregation behavior, a different analyzer
for a class of fields, ...) it can be implemented once per backend
in the central pipeline. After that, turning the capability on for
a specific field is a single override entry, and both backends
adopt it the same way. This ADR does not decide which capabilities
to add, only that they will land in this uniform shape rather than
through coordinated per-site edits.

### Facet values are indexed as case-preserving keywords

All facet sub-fields, meaning any leaf inside `audio`, `photo`,
`image`, `location`, are indexed as plain keywords on both
backends. No tokenization, no case folding. The raw value the
extractor saw, or the CS3 ArbitraryMetadata string, is what lands
in the index.

This is the single intended semantic for facets across bleve and
OpenSearch, and it is driven by what aggregations need.
Aggregation buckets ("group all files by `audio.artist`", "list
distinct `photo.cameraMake`") return bucket keys drawn from the
indexed terms. If the indexing analyzer lowercases (OpenSearch's
default `text + keyword` multi-field against the text leg, or a
`lowercaseKeyword`-style analyzer), the buckets come back lower-
cased: a distinct-artists query would answer `motörhead` and
`queen` instead of the original display casings, and two tag
writers using `Motörhead` versus `MOTÖRHEAD` would collapse into a
single bucket labelled `motörhead`. For a metadata display use
case (thumbnails, facet filters in the UI, distinct lists) that
behavior is not what we want.

Keyword-indexed values preserve case and produce correctly-
labelled buckets on both backends. If, in the future, case-
insensitive search on a facet property is also wanted alongside
the case-preserving bucket view, that is a strict superset and
can be added on top. It is much easier to bolt onto one
consistent backend-agnostic state than to retrofit it onto the
current divergent, implicit per-backend behavior.

The query side lines up with this: the KQL parser preserves the
user's casing, and the per-backend compiler decides per field
whether to fold that value to lowercase before handing it to the
backend, based on the same overrides map the index mapping uses.
Fields whose override selects a lowercasing analyzer (Name, Tags,
Favorites, Content) get their query value folded so user input
matches the tokens the index produced; everything else, including
all facet sub-fields, compares case-sensitive on both sides. The
case-sensitivity alignment of the compiler with the index mapping
was started in #2633; this ADR completes it by deriving both
sides from the same source.

### Behavior changes, applied to newly-created indexes

Existing indexes keep their stored mapping; the new shape applies
only to indexes created after the proposal lands.

- OpenSearch `Tags` and `Favorites` change from dynamic `keyword`
  to `text + lowercaseKeyword`, matching the bleve mapping. The
  analyzer is registered in the new index settings.
- OpenSearch facet sub-strings (`audio.*`, `photo.*`, `image.*`,
  `location.*`) change from the dynamic `text + keyword` multi-field to
  `keyword`-only, implementing the principle above. As noted in
  the context section, the tokenized leg is not actually reachable
  from user queries today, so no working path regresses.

### Known trade-off

The write-time pipeline produces the document as a generic map via
a json round-trip. The OpenSearch write path already does the
equivalent today via the same json-based conversion helper, so
that path is unchanged. The bleve write path, which previously
handed the struct directly to bleve's reflective indexer, now goes
through the same map-producing step and pays roughly the same
cost. On hot paths (initial indexing of a large space) this is
measurable but not significant; if it ever matters, a direct
reflection walker can replace the json round-trip without changing
any call site.

### Follow-ups out of scope for this ADR

- **WebDAV REPORT facet exposure.** The current webdav search
  endpoint renders none of the facet fields back to the client.
  This is a missing feature, not a regression of the proposal;
  its natural resolution is to let the graph-search endpoint
  (not yet in tree) take over once graph search lands.
- **Graph search hit conversion.** Once graph search is wired up,
  it needs to translate proto hits back into libregraph DriveItems.
  The same facet-copy helper used internally on the search service
  side is reusable.
- **reva's PROPFIND facet listing** uses its own hand-maintained
  per-facet key lists. reva deliberately does not depend on the
  libregraph Go types, so unifying those key sets is a reva-side
  decision tracked separately.
- **Write-path performance.** The json round-trip in the bleve
  write path is an optional optimisation target with no call-site
  impact when it lands.
