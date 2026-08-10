package bleve

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/blevesearch/bleve/v2"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

// The following quoted string enumerates the characters which may be escaped: "+-=&|><!(){}[]^\"~*?:\\/ "
// based on bleve docs https://blevesearch.com/docs/Query-String-Query/
// Wildcards * and ? are excluded
var bleveEscaper = strings.NewReplacer(
	`+`, `\+`,
	`-`, `\-`,
	`=`, `\=`,
	`&`, `\&`,
	`|`, `\|`,
	`>`, `\>`,
	`<`, `\<`,
	`!`, `\!`,
	`(`, `\(`,
	`)`, `\)`,
	`{`, `\{`,
	`}`, `\}`,
	`[`, `\[`,
	`]`, `\]`,
	`^`, `\^`,
	`"`, `\"`,
	`~`, `\~`,
	`:`, `\:`,
	`\`, `\\`,
	`/`, `\/`,
	` `, `\ `,
)

// Compiler represents a KQL query search string to the bleve query formatter.
type Compiler struct{}

// Compile implements the query formatter which converts the KQL query search string to the bleve query.
func (c Compiler) Compile(givenAst *ast.Ast) (bleveQuery.Query, error) {
	q, err := compile(givenAst)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func compile(a *ast.Ast) (bleveQuery.Query, error) {
	q, _, err := walk(0, a.Nodes)
	if err != nil {
		return nil, err
	}
	switch q.(type) {
	case *bleveQuery.ConjunctionQuery, *bleveQuery.DisjunctionQuery:
		return q, nil
	}
	return bleve.NewConjunctionQuery(q), nil
}

func walk(offset int, nodes []ast.Node) (bleveQuery.Query, int, error) {
	var prev, next bleveQuery.Query
	var operator *ast.OperatorNode
	var isGroup bool
	for i := offset; i < len(nodes); i++ {
		switch n := nodes[i].(type) {
		case *ast.StringNode:
			// keys are resolved and media-type expanded by normalize; MimeType
			// values are literal MIME types, so they skip the escaper.
			k := n.Key
			v := n.Value
			if k != "ID" && k != "Size" && k != "MimeType" {
				v = bleveEscaper.Replace(n.Value)
			}
			if n.CaseInsensitive {
				k += mapping.LowercaseSuffix
				v = strings.ToLower(v)
			}

			var q bleveQuery.Query = bleveQuery.NewQueryStringQuery(k + ":" + v)
			if searchQuery.FieldIsPath(n.Key) {
				// bleve has no path hierarchy analyzer, unlike OpenSearch: match the
				// folder itself and its descendants (`\/*`). A BooleanQuery keeps
				// this atomic; a DisjunctionQuery would be redistributed by an
				// enclosing AND (mapBinary treats a left disjunction as an OR-chain).
				bq := bleve.NewBooleanQuery()
				bq.AddShould(q, bleveQuery.NewQueryStringQuery(k+":"+v+`\/*`))
				bq.SetMinShould(1)
				q = bq
			}

			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.DateTimeNode:
			q := &bleveQuery.DateRangeQuery{
				Start:          bleveQuery.BleveQueryTime{},
				End:            bleveQuery.BleveQueryTime{},
				InclusiveStart: nil,
				InclusiveEnd:   nil,
				FieldVal:       n.Key,
			}

			if n.Operator == nil {
				continue
			}

			switch n.Operator.Value {
			case ">":
				q.Start.Time = n.Value
				q.InclusiveStart = &[]bool{false}[0]
			case ">=":
				q.Start.Time = n.Value
				q.InclusiveStart = &[]bool{true}[0]
			case "<":
				q.End.Time = n.Value
				q.InclusiveEnd = &[]bool{false}[0]
			case "<=":
				q.End.Time = n.Value
				q.InclusiveEnd = &[]bool{true}[0]
			default:
				continue
			}

			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.NumberNode:
			var q bleveQuery.Query
			if field := n.Key; slices.Contains([]string{"Size", "Type"}, field) {
				q = numberRange(field, n.Operator, n.Value)
			} else {
				// same answer as the OpenSearch backend: unknown numeric keys
				// match nothing instead of querying an arbitrary field
				q = bleveQuery.NewMatchNoneQuery()
			}
			if q == nil {
				continue
			}

			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.BooleanNode:
			q := bleveQuery.NewBoolFieldQuery(n.Value)
			q.SetField(n.Key)
			if prev == nil {
				prev = q
			} else {
				next = q
			}
		case *ast.GroupNode:
			// keys resolved and grouping property propagated in normalize
			q, _, err := walk(0, n.Nodes)
			if err != nil {
				return nil, 0, err
			}
			if prev == nil {
				prev = q
				isGroup = true
			} else {
				next = q
			}
		case *ast.OperatorNode:
			if n.Value == kql.BoolAND || n.Value == kql.BoolOR {
				operator = n
			} else if n.Value == kql.BoolNOT {
				var err error
				next, offset, err = nextNode(i+1, nodes)
				if err != nil {
					return nil, 0, err
				}
				q := bleve.NewBooleanQuery()
				q.AddMustNot(next)
				if prev == nil {
					// unary in the beginning
					prev = q
				} else {
					next = q
				}
			}
		}
		if prev != nil && next != nil && operator != nil {
			prev = mapBinary(operator, prev, next, isGroup)
			isGroup = false
			operator = nil
			next = nil
		}
		if i < offset {
			i = offset
		}
	}
	if prev == nil {
		return nil, 0, fmt.Errorf("can not compile the query")
	}
	return prev, offset, nil
}

func nextNode(offset int, nodes []ast.Node) (bleveQuery.Query, int, error) {
	if n, ok := nodes[offset].(*ast.GroupNode); ok {
		if n.Key != "" {
			n = normalizeGroupingProperty(n)
		}

		gq, _, err := walk(0, n.Nodes)
		if err != nil {
			return nil, 0, err
		}
		return gq, offset + 1, nil
	}
	if n, ok := nodes[offset].(*ast.OperatorNode); ok {
		if n.Value == kql.BoolNOT {
			return walk(offset, nodes)
		}
	}
	one := nodes[:offset+1]
	return walk(offset, one)
}

func mapBinary(operator *ast.OperatorNode, ln, rn bleveQuery.Query, leftIsGroup bool) bleveQuery.Query {
	if operator.Value == kql.BoolOR {
		right, ok := rn.(*bleveQuery.DisjunctionQuery)
		switch left := ln.(type) {
		case *bleveQuery.DisjunctionQuery:
			if ok {
				left.AddQuery(right.Disjuncts...)
			} else {
				left.AddQuery(rn)
			}
			return left
		case *bleveQuery.ConjunctionQuery:
			return bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{ln, rn})
		default:
			if ok {
				left := bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{ln})
				left.AddQuery(right.Disjuncts...)
				return left
			}
			return bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{ln, rn})
		}
	}
	if operator.Value == kql.BoolAND {
		switch left := ln.(type) {
		case *bleveQuery.ConjunctionQuery:
			left.AddQuery(rn)
			return left
		case *bleveQuery.DisjunctionQuery:
			if !leftIsGroup {
				last := left.Disjuncts[len(left.Disjuncts)-1]
				rn = bleveQuery.NewConjunctionQuery([]bleveQuery.Query{
					last,
					rn,
				})
				dj := bleveQuery.NewDisjunctionQuery(left.Disjuncts[:len(left.Disjuncts)-1])
				dj.AddQuery(rn)
				return dj
			}
		}
	}
	return bleveQuery.NewConjunctionQuery([]bleveQuery.Query{
		ln,
		rn,
	})
}

func numberRange(field string, operator *ast.OperatorNode, value float64) bleveQuery.Query {
	if operator == nil {
		return nil
	}

	inclusive, exclusive := true, false

	var q *bleveQuery.NumericRangeQuery
	switch operator.Value {
	case ">":
		q = bleveQuery.NewNumericRangeInclusiveQuery(&value, nil, &exclusive, nil)
	case ">=":
		q = bleveQuery.NewNumericRangeInclusiveQuery(&value, nil, &inclusive, nil)
	case "<":
		q = bleveQuery.NewNumericRangeInclusiveQuery(nil, &value, nil, &exclusive)
	case "<=":
		q = bleveQuery.NewNumericRangeInclusiveQuery(nil, &value, nil, &inclusive)
	default:
		return nil
	}

	q.SetField(field)

	return q
}

func pathAndBelow(field, path string) bleveQuery.Query {
	path = strings.TrimSuffix(path, "/")

	self := bleveQuery.NewTermQuery(path)
	self.SetField(field)

	below := bleveQuery.NewPrefixQuery(path + "/")
	below.SetField(field)

	return closed(bleveQuery.NewDisjunctionQuery([]bleveQuery.Query{self, below}))
}

func closed(q bleveQuery.Query) bleveQuery.Query {
	// a bare disjunction reads as an open OR chain to mapBinary, a later OR
	// would merge into it and widen the group
	return bleveQuery.NewConjunctionQuery([]bleveQuery.Query{q})
}

func phrase(field, value string) bleveQuery.Query {
	q := bleveQuery.NewMatchPhraseQuery(value)
	q.SetField(field)

	return q
}

func normalizeGroupingProperty(group *ast.GroupNode) *ast.GroupNode {
	for _, n := range group.Nodes {
		if onode, ok := n.(*ast.StringNode); ok {
			onode.Key = group.Key
		}
	}
	return group
}

func resourceType(value string) string {
	switch strings.ToLower(value) {
	case "file":
		return strconv.FormatUint(uint64(provider.ResourceType_RESOURCE_TYPE_FILE), 10)
	case "folder":
		return strconv.FormatUint(uint64(provider.ResourceType_RESOURCE_TYPE_CONTAINER), 10)
	default:
		return value
	}
}

func mimeType(k, v string) (bleveQuery.Query, bool) {
	switch v {
	case "file":
		q := bleve.NewBooleanQuery()
		q.AddMustNot(bleveQuery.NewQueryStringQuery(k + ":httpd/unix-directory"))
		return q, false
	case "folder":
		return bleveQuery.NewQueryStringQuery(k + ":httpd/unix-directory"), false
	case "document":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.form",
			"application/vnd.oasis.opendocument.text",
			"text/plain",
			"text/markdown",
			"application/rtf",
			"application/vnd.apple.pages",
		)), true
	case "spreadsheet":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/vnd.ms-excel",
			"application/vnd.oasis.opendocument.spreadsheet",
			"text/csv",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.apple.numbers",
		)), true
	case "presentation":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
			"application/vnd.oasis.opendocument.presentation",
			"application/vnd.ms-powerpoint",
			"application/vnd.apple.keynote",
		)), true
	case "pdf":
		return bleveQuery.NewQueryStringQuery(k + ":application/pdf"), false
	case "image":
		return bleveQuery.NewQueryStringQuery(k + ":image/*"), false
	case "video":
		return bleveQuery.NewQueryStringQuery(k + ":video/*"), false
	case "audio":
		return bleveQuery.NewQueryStringQuery(k + ":audio/*"), false
	case "archive":
		return bleveQuery.NewDisjunctionQuery(newQueryStringQueryList(k,
			"application/zip",
			"application/gzip",
			"application/x-gzip",
			"application/x-7z-compressed",
			"application/x-rar-compressed",
			"application/x-tar",
			"application/x-bzip2",
			"application/x-bzip",
			"application/x-tgz",
		)), true
	default:
		return bleveQuery.NewQueryStringQuery(k + ":" + v), false
	}
}

func newQueryStringQueryList(k string, v ...string) []bleveQuery.Query {
	list := make([]bleveQuery.Query, len(v))
	for i := 0; i < len(v); i++ {
		list[i] = bleveQuery.NewQueryStringQuery(k + ":" + v[i])
	}
	return list
}
