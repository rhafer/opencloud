package parity

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const matrixFile = "README.md"

type matrixRow struct {
	section        string
	wantCount      *int
	wantBadRequest bool
	limit          int32
	ref            *searchMessage.Reference
	group          string
	id             string
	query          string
	reads          string
	context        string
	want           []string
	answered       map[string][]string
	skipped        map[string]bool

	groupAt, caseAt, queryAt int
}

var (
	matrixMu   sync.Mutex
	matrixRows = map[string]*matrixRow{}
)

func recordAnswer(row matrixRow, engine string, answer []string) {
	matrixMu.Lock()
	defer matrixMu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", row.group, row.id, row.query)
	known, ok := matrixRows[key]
	if !ok {
		row.answered = map[string][]string{}
		row.skipped = map[string]bool{}
		known = &row
		matrixRows[key] = known
	}

	if answer == nil {
		known.skipped[engine] = true
		return
	}

	known.answered[engine] = answer
}

func recordSkip(row matrixRow, engine string) {
	recordAnswer(row, engine, nil)
}

func writeMatrix() {
	matrixMu.Lock()
	defer matrixMu.Unlock()

	rows := make([]*matrixRow, 0, len(matrixRows))
	skipped, incomplete := false, false
	for _, row := range matrixRows {
		rows = append(rows, row)
		skipped = skipped || len(row.skipped) > 0
		incomplete = incomplete || len(row.answered)+len(row.skipped) != 2
	}

	switch {
	case len(rows) != matrixExpectedRows():
		fmt.Fprintf(os.Stderr, "%s left alone, %d of %d cases ran\n", matrixFile, len(rows), matrixExpectedRows())
		return
	case skipped:
		fmt.Fprintf(os.Stderr, "%s left alone, an engine was not reachable\n", matrixFile)
		return
	case incomplete:
		fmt.Fprintf(os.Stderr, "%s left alone, an engine died before it answered\n", matrixFile)
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case a.groupAt != b.groupAt:
			return a.groupAt < b.groupAt
		case a.caseAt != b.caseAt:
			return a.caseAt < b.caseAt
		default:
			return a.queryAt < b.queryAt
		}
	})

	out := &strings.Builder{}

	group, section := "", ""
	for _, row := range rows {
		if row.section != section {
			section = row.section
			if out.Len() > 0 {
				out.WriteString("\n")
			}

			fmt.Fprintf(out, "## %s\n", section)
		}

		if row.group != group {
			group = row.group
			fmt.Fprintf(out, "\n### %s\n\n%s\n\n", group, matrixFixtures(group))
			out.WriteString("| Case | Query | expected | bleve | OpenSearch | same? |\n")
			out.WriteString("|---|---|---|---|---|---|\n")
		}

		query := "`" + shortenQuery(row.query) + "`"
		if row.ref != nil {
			query += " " + matrixScope(row.ref)
		}

		if row.limit != 0 {
			query = fmt.Sprintf("%s with a limit of %d", query, row.limit)
		}

		if row.reads != "" {
			query += " reads `" + row.reads + "`"
		}

		if row.context != "" {
			query = row.context + ", then " + query
		}

		expected := matrixNames(row.want)
		switch {
		case row.wantBadRequest:
			expected = "bad request"
		case row.wantCount != nil:
			expected = fmt.Sprintf("%d items", *row.wantCount)
		}

		fmt.Fprintf(out, "| %s | %s | %s | %s | %s | %s |\n",
			row.id, query, expected,
			matrixNames(row.answered["bleve"]), matrixNames(row.answered["opensearch"]), matrixVerdict(row))
	}

	if err := os.WriteFile(matrixFile, []byte(out.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", matrixFile, err)
	}
}

func matrixExpectedRows() int {
	rows := 0
	for _, g := range queryGroups() {
		rows += len(g.cases)
	}

	for _, g := range lifecycleGroups() {
		for _, c := range g.cases {
			rows += len(c.expect)
			if c.wantDocCount != nil {
				rows++
			}
		}
	}

	for _, g := range responseGroups() {
		rows += len(g.cases)
	}

	return rows
}

func matrixFixtures(group string) string {
	var (
		fixtures []search.Resource
		withIDs  bool
	)

	for _, g := range queryGroups() {
		if g.name != group {
			continue
		}

		fixtures = g.fixtures
		for _, c := range g.cases {
			withIDs = withIDs || strings.Contains(c.query, "id:")
		}
	}

	for _, g := range lifecycleGroups() {
		if g.name != group {
			continue
		}

		withIDs = true
		fixtures = g.fixtures
		for _, c := range g.cases {
			fixtures = append(fixtures, c.fixtures...)
		}
	}

	for _, g := range responseGroups() {
		if g.name != group {
			continue
		}

		withIDs = true
		fixtures = g.fixtures
	}

	if len(fixtures) == 0 {
		return "Fixtures: none"
	}

	seen := map[string]bool{}
	lines := []string{"Fixtures:", ""}
	for _, f := range fixtures {
		line := "- `" + shorten(f.Name) + "`"
		if fields := fixtureFields(f, withIDs); fields != "" {
			line += ", " + fields
		}

		if seen[line] {
			continue
		}

		seen[line] = true
		lines = append(lines, line)
	}

	if listed := len(lines) - 2; listed > 12 {
		lines = append(lines[:12], fmt.Sprintf("- ... and %d more of the same", listed-10))
	}

	return strings.Join(lines, "\n")
}

func fixtureFields(f search.Resource, withID bool) string {
	var fields []string

	add := func(format string, args ...any) {
		fields = append(fields, fmt.Sprintf(format, args...))
	}

	if withID {
		add("ID = %s", f.ID)
	}

	if f.Type == uint64(sprovider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		add("folder")
	} else if f.MimeType != "text/plain" {
		add("MimeType = %s", f.MimeType)
	}

	if f.Path != "./"+f.Name {
		add("Path = %s", f.Path)
	}

	if f.Title != "" {
		add("Title = %q", f.Title)
	}

	if f.Content != "" {
		add("Content = %q", f.Content)
	}

	if len(f.Tags) > 0 {
		add("Tags = %s", strings.Join(f.Tags, ", "))
	}

	if len(f.Favorites) > 0 {
		add("Favorites = %s", strings.Join(f.Favorites, ", "))
	}

	if f.Size != 1000 {
		add("Size = %d", f.Size)
	}

	if !strings.HasPrefix(f.Mtime, fixtureNow.Format("2006-01-02")) {
		add("Mtime = %s", f.Mtime)
	}

	if f.Hidden {
		add("hidden")
	}

	if f.Deleted {
		add("deleted")
	}

	return strings.Join(fields, ", ")
}

func matrixScope(ref *searchMessage.Reference) string {
	space := storagespace.FormatResourceID(&sprovider.ResourceId{
		StorageId: ref.GetResourceId().GetStorageId(),
		SpaceId:   ref.GetResourceId().GetSpaceId(),
		OpaqueId:  ref.GetResourceId().GetOpaqueId(),
	})

	if path := ref.GetPath(); path != "" {
		return "in " + space + " under " + path
	}

	return "in " + space
}

func matrixNames(names []string) string {
	if len(names) == 0 {
		return "no match"
	}

	shortened := make([]string, 0, len(names))
	for _, name := range names {
		shortened = append(shortened, shorten(name))
	}
	sort.Strings(shortened)

	return strings.Join(shortened, ", ")
}

func shorten(name string) string {
	if len(name) <= 24 {
		return name
	}

	return name[:10] + "..." + name[len(name)-8:]
}

func shortenQuery(q string) string {
	if len(q) <= 64 {
		return q
	}

	return q[:32] + "..." + q[len(q)-24:]
}

func matrixVerdict(row *matrixRow) string {
	var off []string
	for _, engine := range []string{"bleve", "opensearch"} {
		if row.wantBadRequest {
			if matrixNames(row.answered[engine]) != "bad request" {
				off = append(off, engine)
			}

			continue
		}

		if row.wantCount != nil {
			if len(row.answered[engine]) != *row.wantCount {
				off = append(off, engine)
			}

			continue
		}

		if matrixNames(row.answered[engine]) != matrixNames(row.want) {
			off = append(off, engine)
		}
	}

	if len(off) == 0 {
		return "✅"
	}

	return "❌"
}

func (c lifecycleCase) matrixRows(group string, groupAt, caseAt int) []matrixRow {
	rows := make([]matrixRow, 0, len(c.expect)+1)
	for i, expect := range c.expect {
		rows = append(rows, matrixRow{
			section: "Operations", group: group, id: c.label(group), query: expect.query, context: c.title, want: expect.want,
			groupAt: groupAt, caseAt: caseAt, queryAt: i,
		})
	}

	if c.wantDocCount != nil {
		rows = append(rows, matrixRow{
			section: "Operations", group: group, id: c.label(group), query: "DocCount()", context: c.title,
			want:    []string{fmt.Sprint(*c.wantDocCount)},
			groupAt: groupAt, caseAt: caseAt, queryAt: len(c.expect),
		})
	}

	return rows
}
