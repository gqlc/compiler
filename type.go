package compiler

import (
	"github.com/gqlc/graphql/ast"
)

// TypeError represents a type error.
type TypeError struct {
	// Document where type error was discovered
	Doc *ast.Document

	// Type error message
	Msg string
}

// Error returns a string representation of a TypeError.
func (e *TypeError) Error() string {
	return e.Msg
}

// TypeChecker represents type checking functionality for a GraphQL Document.
type TypeChecker interface {
	// Check analyzes the types in a GraphQL Document and returns any
	// problems it has detected. Errors may be returned along side a
	// "true" ok value. This signifies that the errors are more
	// along the lines of warnings.
	//
	Check(doc *ast.Document) (ok bool, errs []*TypeError)
}

// TypeCheckFn represents a single function behaving as a TypeChecker.
type TypeCheckerFn func(*ast.Document) (bool, []*TypeError)

// Check calls the TypeCheckerFn given the GraphQL Document.
func (f TypeCheckerFn) Check(doc *ast.Document) (bool, []*TypeError) {
	return f(doc)
}

// CheckTypes is a helper function for running a suite of
// type checking on several GraphQL Documents. Any TypeDecls
// passed to RegisterTypes will be appended to each Documents' Type list.
//
// All errors encountered will be appended into the return slice: errs
//
func CheckTypes(docs []*ast.Document, checkers ...TypeChecker) (errs []*TypeError) {
	for _, doc := range docs {
		doc.Types = append(doc.Types, Types...)

		for _, checker := range checkers {
			ok, cerrs := checker.Check(doc)
			if !ok {
				errs = append(errs, cerrs...)
			}
		}
	}
	return
}

// Types contains any builtin types that should be included with
// the GraphQL Documents being passed to CheckTypes.
//
var Types []*ast.TypeDecl

// RegisterTypes registers pre-defined types with the compiler.
func RegisterTypes(decls ...*ast.TypeDecl) { Types = append(Types, decls...) }
