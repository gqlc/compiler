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
		{
			Name: "NotTransitive",
			Docs: map[string]io.Reader{
				"a": strings.NewReader(`@import(paths: ["b"])
scalar String

type Msg {
	text: String!
	time: Time!
}
`),
				"b": strings.NewReader(`@import(paths: ["c"])`),
				"c": strings.NewReader("scalar Time"),
			},
			Err: "compiler: encountered type error in a:unimported type: Time",
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
			if len(errs) == 0 && testCase.Err == "" {
				return
			}

			if errs[0].Error() == testCase.Err {
				return
			}

			subT.Logf("expected error: %s, but got: %s", testCase.Err, errs[0])
			subT.Fail()
		})
	}
}
