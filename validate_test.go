package compiler

import (
	"io"
	"strings"
	"testing"

	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
)

func TestImportValidator(t *testing.T) {
	testCases := []struct {
		Name string
		Docs map[string]io.Reader
		Err  string
	}{
		{
			Name: "NoImports",
			Docs: map[string]io.Reader{"a": strings.NewReader("scalar Msg")},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(subT *testing.T) {
			docs, err := parser.ParseDocs(token.NewDocSet(), testCase.Docs, 0)
			if err != nil {
				subT.Error(err)
				return
			}

			ir := ToIR(docs)

			errs := CheckTypes(ir, ImportValidator)
			if len(errs) == 0 {
				return
			}

			for _, err := range errs {
				subT.Log(err)
			}
			subT.Fail()
		})
	}
}
