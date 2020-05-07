package compiler

import (
	"fmt"

	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
)

// MergeExtensions merges type extensions with their original declaration.
func MergeExtensions(types map[string][]*ast.TypeDecl) map[string][]*ast.TypeDecl {
	for name, decls := range types {
		if len(decls) == 1 {
			continue
		}

		types[name] = mergeDecls(decls)
	}
	return types
}

func mergeDecls(decls []*ast.TypeDecl) []*ast.TypeDecl {
	f := func(_, _ *ast.TypeSpec) {}
	switch decls[0].Tok {
	case token.Token_SCHEMA:
		f = mergeSchema
	case token.Token_SCALAR:
	case token.Token_TYPE:
		f = mergeObject
	case token.Token_INTERFACE:
		f = mergeInterface
	case token.Token_UNION:
		f = mergeUnion
	case token.Token_ENUM:
		f = mergeEnum
	case token.Token_INPUT:
		f = mergeInput
	default:
		panic(fmt.Sprintf("type of: %s cannot be extended", decls[0].Tok))
	}

	def, ok := decls[0].Spec.(*ast.TypeDecl_TypeSpec)
	if !ok {
		panic("") // TODO: Should be panic?
	}

	for _, edecl := range decls[1:] {
		ext, ok := edecl.Spec.(*ast.TypeDecl_TypeExtSpec)
		if !ok {
			panic("") // TODO: Should be panic?
		}

		f(def.TypeSpec, ext.TypeExtSpec.Type)

		def.TypeSpec.Directives = append(def.TypeSpec.Directives, ext.TypeExtSpec.Type.Directives...)
	}

	return decls[:1]
}

func mergeSchema(def, ext *ast.TypeSpec) {
	schema := def.Type.(*ast.TypeSpec_Schema).Schema
	extSchema := ext.Type.(*ast.TypeSpec_Schema).Schema.RootOps

	if extSchema != nil {
		schema.RootOps.List = append(schema.RootOps.List, extSchema.List...)
	}
}

func mergeObject(def, ext *ast.TypeSpec) {
	obj := def.Type.(*ast.TypeSpec_Object).Object
	extObj := ext.Type.(*ast.TypeSpec_Object).Object

	obj.Interfaces = append(obj.Interfaces, extObj.Interfaces...)

	if extObj.Fields != nil {
		obj.Fields.List = append(obj.Fields.List, extObj.Fields.List...)
	}
}

func mergeInterface(def, ext *ast.TypeSpec) {
	inter := def.Type.(*ast.TypeSpec_Interface).Interface
	extInter := ext.Type.(*ast.TypeSpec_Interface).Interface

	if extInter.Fields != nil {
		inter.Fields.List = append(inter.Fields.List, extInter.Fields.List...)
	}
}

func mergeUnion(def, ext *ast.TypeSpec) {
	union := def.Type.(*ast.TypeSpec_Union).Union
	extUnion := ext.Type.(*ast.TypeSpec_Union).Union

	union.Members = append(union.Members, extUnion.Members...)
}

func mergeEnum(def, ext *ast.TypeSpec) {
	enum := def.Type.(*ast.TypeSpec_Enum).Enum
	extEnum := ext.Type.(*ast.TypeSpec_Enum).Enum

	if extEnum.Values != nil {
		enum.Values.List = append(enum.Values.List, extEnum.Values.List...)
	}
}

func mergeInput(def, ext *ast.TypeSpec) {
	input := def.Type.(*ast.TypeSpec_Input).Input
	extInput := ext.Type.(*ast.TypeSpec_Input).Input

	if extInput.Fields != nil {
		input.Fields.List = append(input.Fields.List, extInput.Fields.List...)
	}
}
