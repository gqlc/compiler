// TODO: Investigate directives for by-passing type checking
// TODO: Investigate type checking directive applications

package compiler

import (
	"fmt"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
)

// TypeError represents a type error.
type TypeError struct {
	// Document where type error was discovered
	Doc *ast.Doc

	// Type error message
	Msg string
}

// Error returns a string representation of a TypeError.
func (e *TypeError) Error() string {
	return e.Msg
}

// TypeChecker represents type checking functionality for a
// GraphQL Document Set. Errors may be returned along side a
// "true" ok value. This signifies that the errors are more
// along the lines of warnings.
//
type TypeChecker func(docs ...*ast.Document) (ok bool, errs []*TypeError)

// CheckTypes type checks a set of GraphQL documents.
// Only one schema is allowed in a set of GraphQL documents.
//
func CheckTypes(docs []*ast.Document, checkers ...TypeChecker) []*TypeError {
	validSchema, errs := verifySchema(docs)
	if !validSchema {
		return errs
	}

	for _, checker := range checkers {
		ok, cerrs := checker(docs...)
		if !ok {
			return cerrs
		}
	}
	return nil
}

// verifySchema verifies that only one schema is contained in a GraphQL Document Set.
func verifySchema(docs []*ast.Document) (ok bool, errs []*TypeError) {
	// First, verify only one schema and collect all type declarations
	var s []ast.Spec
	tdecls := make(map[string]*ast.TypeSpec)
	for _, doc := range docs {
		if len(doc.Schemas) == 0 {
			continue
		}

		for _, ss := range doc.Schemas {
			s = append(s, ss.Spec)
		}

		for _, gtd := range doc.Types {
			if gtd.Tok == token.SCHEMA {
				continue
			}

			ts := gtd.Spec.(*ast.TypeSpec)
			switch v := ts.Name.(type) {
			case *ast.Ident:
				tdecls[v.Name] = ts
			case *ast.SelectorExpr:
				tdecls[v.Sel.Name] = ts
			}
		}
	}
	if len(s) == 0 {
		return
	}
	if len(s) > 1 {
		errs = append(errs, &TypeError{Msg: "document set contains more than one GraphQL schema"})
		return
	}

	// Next, verify schema root operations
	sts := s[0].(*ast.TypeSpec)
	st := sts.Type.(*ast.SchemaType)
	rootOps := st.Fields.List
	for _, rootOp := range rootOps {
		fieldTyp := rootOp.Type.(*ast.Ident)
		rootT := tdecls[fieldTyp.Name]

		if _, tok := rootT.Type.(*ast.ObjectType); !tok {
			errs = append(errs, &TypeError{Msg: fmt.Sprintf("schema field: %s must return an Object type", rootOp.Name.Name)})
			return
		}
	}

	return true, nil
}

// UnusedTypes checks a GraphQL Document(s) for any unused types.
func UnusedTypes(docs ...*ast.Document) (bool, []*TypeError) {
	// TODO
	return false, nil
}
