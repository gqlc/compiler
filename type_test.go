package compiler

import (
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Register built in types
	RegisterTypes([]*ast.TypeDecl{
		{
			Tok: int64(token.SCALAR),
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
			Tok: int64(token.SCALAR),
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
			Tok: int64(token.SCALAR),
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
			Tok: int64(token.SCALAR),
			Spec: &ast.TypeDecl_TypeSpec{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: "Float"},
					Type: &ast.TypeSpec_Scalar{
						Scalar: &ast.ScalarType{Name: &ast.Ident{Name: "Boolean"}},
					},
				},
			},
		},
		{
			Tok: int64(token.SCALAR),
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
			Tok: int64(token.DIRECTIVE),
			Spec: &ast.TypeDecl_TypeSpec{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: "skip"},
					Type: &ast.TypeSpec_Directive{
						Directive: &ast.DirectiveType{
							Args: &ast.FieldList{
								List: []*ast.Field{
									{
										Name: &ast.Ident{Name: "if"},
										Type: &ast.Field_NonNull{
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
			Tok: int64(token.DIRECTIVE),
			Spec: &ast.TypeDecl_TypeSpec{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: "include"},
					Type: &ast.TypeSpec_Directive{
						Directive: &ast.DirectiveType{
							Args: &ast.FieldList{
								List: []*ast.Field{
									{
										Name: &ast.Ident{Name: "if"},
										Type: &ast.Field_NonNull{
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
			Tok: int64(token.DIRECTIVE),
			Spec: &ast.TypeDecl_TypeSpec{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: "deprecated"},
					Type: &ast.TypeSpec_Directive{
						Directive: &ast.DirectiveType{
							Args: &ast.FieldList{
								List: []*ast.Field{
									{
										Name: &ast.Ident{Name: "reason"},
										Type: &ast.Field_Ident{
											Ident: &ast.Ident{Name: "String"},
										},
										Default: &ast.Field_BasicLit{
											BasicLit: &ast.BasicLit{Kind: int64(token.STRING), Value: "No longer supported"},
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
		}}...)

	os.Exit(m.Run())
}