// Package spec provides the types and validator as defined by the GraphQL spec.
package spec

import (
	"github.com/gqlc/compiler"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
)

// BuiltinTypes contains the builtin types as defined by the GraphQL spec.
var BuiltinTypes = []*ast.TypeDecl{
	{
		Tok: token.Token_SCALAR,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Int"},
				Type: &ast.TypeSpec_Scalar{
					Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "Int"}},
				},
			},
		},
	},
	{
		Tok: token.Token_SCALAR,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Float"},
				Type: &ast.TypeSpec_Scalar{
					Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "Float"}},
				},
			},
		},
	},
	{
		Tok: token.Token_SCALAR,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "String"},
				Type: &ast.TypeSpec_Scalar{
					Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "String"}},
				},
			},
		},
	},
	{
		Tok: token.Token_SCALAR,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "Boolean"},
				Type: &ast.TypeSpec_Scalar{
					Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "Boolean"}},
				},
			},
		},
	},
	{
		Tok: token.Token_SCALAR,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "ID"},
				Type: &ast.TypeSpec_Scalar{
					Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "ID"}},
				},
			},
		},
	},
	{
		Tok: token.Token_DIRECTIVE,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "skip"},
				Type: &ast.TypeSpec_Directive{
					Directive: &ast.DirectiveType{
						Args: &ast.InputValueList{
							List: []*ast.InputValue{
								{
									Name: &ast.Ident{Name: "if"},
									Type: &ast.InputValue_NonNull{
										NonNull: &ast.NonNull{
											Type: &ast.NonNull_Ident{
												Ident: &ast.Ident{Name: "Boolean"},
											},
										},
									},
								},
							},
						},
						Locs: []*ast.DirectiveLocation{
							{
								Loc: ast.DirectiveLocation_FIELD,
							},
							{
								Loc: ast.DirectiveLocation_FRAGMENT_SPREAD,
							},
							{
								Loc: ast.DirectiveLocation_INLINE_FRAGMENT,
							},
						},
					},
				},
			},
		},
	},
	{
		Tok: token.Token_DIRECTIVE,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "include"},
				Type: &ast.TypeSpec_Directive{
					Directive: &ast.DirectiveType{
						Args: &ast.InputValueList{
							List: []*ast.InputValue{
								{
									Name: &ast.Ident{Name: "if"},
									Type: &ast.InputValue_NonNull{
										NonNull: &ast.NonNull{
											Type: &ast.NonNull_Ident{
												Ident: &ast.Ident{Name: "Boolean"},
											},
										},
									},
								},
							},
						},
						Locs: []*ast.DirectiveLocation{
							{
								Loc: ast.DirectiveLocation_FIELD,
							},
							{
								Loc: ast.DirectiveLocation_FRAGMENT_SPREAD,
							},
							{
								Loc: ast.DirectiveLocation_INLINE_FRAGMENT,
							},
						},
					},
				},
			},
		},
	},
	{
		Tok: token.Token_DIRECTIVE,
		Spec: &ast.TypeDecl_TypeSpec{
			TypeSpec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "deprecated"},
				Type: &ast.TypeSpec_Directive{
					Directive: &ast.DirectiveType{
						Args: &ast.InputValueList{
							List: []*ast.InputValue{
								{
									Name: &ast.Ident{Name: "reason"},
									Type: &ast.InputValue_Ident{
										Ident: &ast.Ident{Name: "String"},
									},
									Default: &ast.InputValue_BasicLit{
										BasicLit: &ast.BasicLit{Kind: token.Token_STRING, Value: "No longer supported"},
									},
								},
							},
						},
						Locs: []*ast.DirectiveLocation{
							{
								Loc: ast.DirectiveLocation_FIELD_DEFINITION,
							},
							{
								Loc: ast.DirectiveLocation_ENUM_VALUE,
							},
						},
					},
				},
			},
		},
	},
}

func init() {
	compiler.RegisterTypes(BuiltinTypes...)
}
