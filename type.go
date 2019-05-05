// TODO: Investigate directives for by-passing type checking
// TODO: Investigate type checking directive applications

package compiler

import (
	"github.com/gqlc/graphql/ast"
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
	for _, checker := range checkers {
		ok, cerrs := checker(docs...)
		if !ok {
			return cerrs
		}
	}
	return nil
}
