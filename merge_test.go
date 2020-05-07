package compiler

import (
	"strings"
	"testing"

	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
)

func TestMergeExtensions(t *testing.T) {
	testCases := []struct {
		Name  string
		Input string
		check func(t *testing.T, merged *ast.TypeSpec)
	}{
		{
			Name: "Schema",
			Input: `schema {}

extend schema {
	query: Query
}

extend schema {
	mutation: Mutation
}`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				schema, ok := merged.Type.(*ast.TypeSpec_Schema)
				if !ok {
					ct.Error("expected schema type")
					return
				}

				fields := schema.Schema.RootOps.List
				if len(fields) != 2 {
					ct.Fail()
					return
				}
			},
		},
		{
			Name: "Scalar",
			Input: `scalar Test

extend scalar Test @a`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				if len(merged.Directives) != 1 {
					ct.Fail()
					return
				}
			},
		},
		{
			Name: "Object",
			Input: `type Test {}

extend type Test implements One

extend type Test {
	a: A
}`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				obj := merged.Type.(*ast.TypeSpec_Object).Object

				if len(obj.Interfaces) != 1 {
					ct.Fail()
					return
				}

				if len(obj.Fields.List) != 1 {
					ct.Fail()
					return
				}
			},
		},
		{
			Name: "Interface",
			Input: `interface Test {}

extend interface Test {
	a: A
}`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				inter := merged.Type.(*ast.TypeSpec_Interface).Interface

				if len(inter.Fields.List) != 1 {
					ct.Fail()
					return
				}
			},
		},
		{
			Name: "Union",
			Input: `union Test

extend union Test = A | B`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				union := merged.Type.(*ast.TypeSpec_Union).Union

				if len(union.Members) != 2 {
					ct.Fail()
					return
				}
			},
		},
		{
			Name: "Enum",
			Input: `enum Test {}

extend enum Test {
	A
}`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				enum := merged.Type.(*ast.TypeSpec_Enum).Enum

				if len(enum.Values.List) != 1 {
					ct.Fail()
					return
				}
			},
		},
		{
			Name: "Input",
			Input: `input Test {}

extend input Test {
	a: A
}`,
			check: func(ct *testing.T, merged *ast.TypeSpec) {
				input := merged.Type.(*ast.TypeSpec_Input).Input

				if len(input.Fields.List) != 1 {
					ct.Fail()
					return
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(subT *testing.T) {
			doc, err := parser.ParseDoc(token.NewDocSet(), testCase.Name, strings.NewReader(testCase.Input), 0)
			if err != nil {
				subT.Error(err)
				return
			}

			docIR := MergeExtensions(toDeclMap(doc.Types))

			for _, decls := range docIR {
				if len(decls) > 1 {
					subT.Fail()
					return
				}

				ts, ok := decls[0].Spec.(*ast.TypeDecl_TypeSpec)
				if !ok {
					subT.Error("expected type spec but got ext instead")
					return
				}

				testCase.check(subT, ts.TypeSpec)
			}
		})
	}
}
