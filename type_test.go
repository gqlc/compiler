package compiler

import (
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/parser"
	"github.com/gqlc/graphql/token"
	"strings"
	"testing"
)

func TestCheckTypes(t *testing.T) {
	s := `schema {
	query: Query
	mutation: Mutation
}

type Query {}

type Mutation {}`

	dset := token.NewDocSet()
	doc, err := parser.ParseDoc(dset, "perfect", strings.NewReader(s), 0)
	if err != nil {
		t.Errorf("unexpected error when parsing perfect schema: %s", err)
	}

	errs := CheckTypes([]*ast.Document{doc})
	if errs != nil {
		t.Fail()
	}
}

func TestVerifySchema(t *testing.T) {
	t.Run("perfect", func(subT *testing.T) {
		s := `schema {
	query: Query
	mutation: Mutation
}

type Query {}

type Mutation {}`

		dset := token.NewDocSet()
		doc, err := parser.ParseDoc(dset, "perfect", strings.NewReader(s), 0)
		if err != nil {
			subT.Errorf("unexpected error when parsing perfect schema: %s", err)
		}

		ok, _ := verifySchema([]*ast.Document{doc})
		if !ok {
			subT.Fail()
		}
	})

	t.Run("moreThanOne", func(subT *testing.T) {
		s := `schema {
	query: Query
	mutation: Mutation
}

type Query {}

type Mutation {}

schema {
	query: Query
}`

		dset := token.NewDocSet()
		doc, err := parser.ParseDoc(dset, "moreThanOne", strings.NewReader(s), 0)
		if err != nil {
			subT.Errorf("unexpected error when parsing moreThanOne schema: %s", err)
		}

		ok, _ := verifySchema([]*ast.Document{doc})
		if ok {
			subT.Fail()
		}
	})

	t.Run("invalidRootOps", func(subT *testing.T) {
		s := `schema {
	query: Query
}

scalar Query`

		dset := token.NewDocSet()
		doc, err := parser.ParseDoc(dset, "perfect", strings.NewReader(s), 0)
		if err != nil {
			subT.Errorf("unexpected error when parsing perfect schema: %s", err)
		}

		ok, _ := verifySchema([]*ast.Document{doc})
		if ok {
			subT.Fail()
		}
	})
}
