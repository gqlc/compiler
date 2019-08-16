package compiler

import (
	"github.com/gqlc/graphql/ast"
)

// CommandLine provides a clean and concise way to implement
// CLIs for compiling the GraphQL IDL.
//
type CommandLine interface {
	// RegisterGenerator registers a language generator with the CLI
	// flagDetails can be either two, three, or more than three strings.
	//		Case two:
	//			first - flag name
	//			second - flag help text
	//		Case three:
	//			first - flag name
	//			second - flag option name
	//			third - flag help text
	//		Case more than three:
	//			Same as Case three but ignores extras
	//
	RegisterGenerator(gen Generator, flagDetails ...string)

	// AllowPlugins enables "plugins". If a command-line flag ends with "_out"
	// but does not match any register code generator, the compiler will
	// attempt to find the "plugin" to implement the generator. Plugins are
	// just executables. They should reside in your PATH.
	//
	// The compiler determines the executable name to search for by concatenating
	// exe_name_prefix with the unrecognized flag name, removing "_out".  So, for
	// example, if exe_name_prefix is "gqlc-" and you pass the flag --foo_out,
	// the compiler will try to run the program "gqlc-foo".
	//
	AllowPlugins(exeNamePrefix string)

	// Run the compiler with the given command-line parameters.
	Run(args []string) error
}

// ToIR converts a GraphQL Document to a intermediate
// representation for the compiler internals.
//
func ToIR(types []*ast.TypeDecl) map[string][]*ast.TypeDecl {
	ir := make(map[string][]*ast.TypeDecl, len(types))

	var ts *ast.TypeSpec
	for _, decl := range types {
		switch v := decl.Spec.(type) {
		case *ast.TypeDecl_TypeSpec:
			ts = v.TypeSpec
		case *ast.TypeDecl_TypeExtSpec:
			ts = v.TypeExtSpec.Type
		}

		name := "schema"
		if ts.Name != nil {
			name = ts.Name.Name
		}

		l := ir[name]
		l = append(l, decl)
		ir[name] = l
	}

	return ir
}
