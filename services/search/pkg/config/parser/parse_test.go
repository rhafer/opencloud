package parser

import "testing"

func TestValidateQueryFilter(t *testing.T) {
	for query, valid := range map[string]bool{
		"": true,
		`NOT path:"./.space" AND NOT id:{{.SPACE_ROOT_PERSONAL}}`:            true,
		`NOT id:{{.SPACE_ROOT_PERSONAL}} AND NOT id:{{.SPACE_ROOT_PROJECT}}`: true,
		`NOT mediatype:folder`:           true,
		`NOT name:"a(b"`:                 true,
		`NOT id:{{.SPACE_ROOT_GRANT}}`:   false,
		`NOT id:{{.SPACE_ROOT_PERSONAL}`: false,
		`(NOT path:x`:                    false,
		`NOT path:x) OR (name:y`:         false,
		`AND AND`:                        false,
		`NOT`:                            false,
	} {
		if err := validateQueryFilter(query); (err == nil) != valid {
			t.Errorf("validateQueryFilter(%q) = %v, want valid %v", query, err, valid)
		}
	}
}
