package compiler

import (
	"fmt"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
	"strings"
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
	return fmt.Sprintf("compiler: encountered type error in %s:%s", e.Doc.Name, e.Msg)
}

// TypeChecker represents type checking functionality for a GraphQL Document.
type TypeChecker interface {
	// Check analyzes the types in a GraphQL Document and returns any
	// problems it has detected.
	//
	Check(doc *ast.Document) []*TypeError
}

// TypeCheckFn represents a single function behaving as a TypeChecker.
type TypeCheckerFn func(*ast.Document) []*TypeError

// Check calls the TypeCheckerFn given the GraphQL Document.
func (f TypeCheckerFn) Check(doc *ast.Document) []*TypeError {
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
			cerrs := checker.Check(doc)
			if len(cerrs) > 0 {
				errs = append(errs, cerrs...)
			}
		}
	}
	return
}

// Types contains the builtin types and any other user-defined types that
// should be included with the GraphQL Documents being passed to CheckTypes.
//
var Types = []*ast.TypeDecl{
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
				Name: &ast.Ident{Name: "Boolean"},
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
	},
}

// RegisterTypes registers pre-defined types with the compiler.
func RegisterTypes(decls ...*ast.TypeDecl) { Types = append(Types, decls...) }

// Validate applies the GraphQL type system validation rules, per the GraphQL spec.
func Validate(doc *ast.Document) (errs []*TypeError) {
	defer func() {
		for _, err := range errs {
			err.Doc = doc
		}
	}()

	// Create pqueue and item map
	itemMap := make(map[string]*ast.TypeDecl, len(doc.Types))

	// Populate item map
	for _, decl := range doc.Types {
		ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		name := "schema"
		if _, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Schema); !ok {
			name = ts.TypeSpec.Name.Name
		}
		itemMap[name] = decl
	}

	// Consume pqueue and validate each type decl
	for _, decl := range doc.Types {
		switch u := decl.Spec.(type) {
		case *ast.TypeDecl_TypeSpec:
			ts := u.TypeSpec

			// Validate type
			var typ token.Token
			loc := ast.DirectiveLocation_NoPos
			switch v := ts.Type.(type) {
			case *ast.TypeSpec_Schema:
				validateSchema(v.Schema, itemMap, &errs)

				typ = token.SCHEMA
				loc = ast.DirectiveLocation_SCHEMA
			case *ast.TypeSpec_Scalar:
				typ = token.SCALAR
				loc = ast.DirectiveLocation_SCALAR
			case *ast.TypeSpec_Enum:
				validateEnum(ts.Name.Name, v.Enum, itemMap, &errs)

				typ = token.ENUM
				loc = ast.DirectiveLocation_ENUM
			case *ast.TypeSpec_Union:
				validateUnion(ts.Name.Name, v.Union, itemMap, &errs)

				typ = token.UNION
				loc = ast.DirectiveLocation_UNION
			case *ast.TypeSpec_Interface:
				validateInterface(ts.Name.Name, v.Interface, itemMap, &errs)

				typ = token.INTERFACE
				loc = ast.DirectiveLocation_INTERFACE
			case *ast.TypeSpec_Input:
				validateInput(ts.Name.Name, v.Input, itemMap, &errs)

				typ = token.INPUT
				loc = ast.DirectiveLocation_INPUT_OBJECT
			case *ast.TypeSpec_Object:
				validateObject(ts.Name.Name, v.Object, itemMap, &errs)

				typ = token.TYPE
				loc = ast.DirectiveLocation_OBJECT
			case *ast.TypeSpec_Directive:
				validateDirective(ts.Name.Name, v.Directive, itemMap, &errs)

				typ = token.DIRECTIVE
			}

			// Check type name
			if loc != ast.DirectiveLocation_SCHEMA {
				checkName(typ, ts.Name, &errs)
			}

			// Validate applied directives
			if loc != ast.DirectiveLocation_NoPos {
				validateDirectives(ts.Directives, loc, itemMap, &errs)
			}
		case *ast.TypeDecl_TypeExtSpec:
			ts := u.TypeExtSpec.Type
			validateExtend(ts, itemMap, &errs)
		}
	}

	// Validate top-lvl directives
	validateDirectives(doc.Directives, ast.DirectiveLocation_DOCUMENT, itemMap, &errs)
	return
}

// validateSchema validates a schema declaration
func validateSchema(schema *ast.SchemaType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if schema.RootOps == nil {
		return
	}

	if len(schema.RootOps.List) == 0 {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("schema: at minimum query object must be provided"),
		})
		return
	}

	var hasQuery bool
	for _, f := range schema.RootOps.List {
		if f.Name.Name == "query" {
			hasQuery = true
		}

		var id *ast.Ident
		switch v := f.Type.(type) {
		case *ast.Field_Ident:
			id = v.Ident
		case *ast.Field_List:
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("schema:%s: root operation return type can not be a list type", f.Name.Name),
			})
			continue
		case *ast.Field_NonNull:
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("schema:%s: root operation return type can not be a non null type", f.Name.Name),
			})
			continue
		default:
			panic(fmt.Sprintf("compiler: schema:%s: must have type", f.Name.Name))
		}

		i, exists := items[id.Name]
		if !exists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("schema:%s: unknown type: %s", f.Name.Name, id.Name),
			})
			continue
		}

		ts, ok := i.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		if _, ok = ts.TypeSpec.Type.(*ast.TypeSpec_Object); !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("schema:%s: root operation return type must be an object type", f.Name.Name),
			})
		}
	}

	if !hasQuery {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("schema: query object must be provided"),
		})
	}
}

// validateEnum validates an enum declaration
func validateEnum(name string, enum *ast.EnumType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if enum.Values == nil {
		return
	}

	if len(enum.Values.List) == 0 {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s: enum type must define one or more unique enum values", name),
		})
		return
	}

	vMap := make(map[string]int, len(enum.Values.List))
	for _, v := range enum.Values.List {
		c := vMap[v.Name.Name]
		vMap[v.Name.Name] = c + 1

		validateDirectives(v.Directives, ast.DirectiveLocation_ENUM_VALUE, items, errs)
	}

	for v, c := range vMap {
		if c == 1 {
			continue
		}

		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s:%s: enum value must be unique", name, v),
		})
	}
}

// validateUnion validates a union declaration
func validateUnion(name string, union *ast.UnionType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if union.Members == nil {
		return
	}

	if len(union.Members) == 0 {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s: union type must include one or more unique member types", name),
		})
		return
	}

	vMap := make(map[string]int, len(union.Members))
	for _, v := range union.Members {
		c := vMap[v.Name]
		vMap[v.Name] = c + 1
	}

	for v, c := range vMap {
		i, exists := items[v]
		if !exists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: undefined type", name, v),
			})
			continue
		}

		ts, ok := i.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		if _, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Object); !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: member type must be an object type", name, v),
			})
		}

		if c > 1 {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: member type must be unique", name, v),
			})
		}
	}
}

// validateArgDefs validates a list of argument definitions
func validateArgDefs(name string, args []*ast.Field, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	aMap := make(map[string]struct {
		field *ast.Field
		count int
	})
	for _, f := range args {
		i, exists := aMap[f.Name.Name]
		if !exists {
			i = struct {
				field *ast.Field
				count int
			}{field: f}
			aMap[f.Name.Name] = i
		}

		i.count++
		aMap[f.Name.Name] = i
	}

	for aname, a := range aMap {
		// Ensure field uniqueness
		if a.count > 1 {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: argument must be unique", name, aname),
			})
		}

		// Check field name
		if strings.HasPrefix(aname, "__") {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: argument name cannot start with \"__\" (double underscore)", name, aname),
			})
		}

		// Validate field type is an InputType
		var id *ast.Ident
		var valType interface{}
		switch v := a.field.Type.(type) {
		case *ast.Field_Ident:
			valType = v.Ident
			id = v.Ident
		case *ast.Field_List:
			valType = v.List
			id = unwrapType(v.List)
		case *ast.Field_NonNull:
			valType = v.NonNull
			id = unwrapType(v.NonNull)
		default:
			panic(fmt.Sprintf("compiler: %s:%s: argument must have a type", name, aname))
		}

		if !isInputType(id, items) {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: argument type must be a valid input type, not: %s", name, aname, id.Name),
			})
		}

		// Validate any default value provided
		switch v := a.field.Default.(type) {
		case *ast.Field_BasicLit:
			validateValue(name, aname, a, v.BasicLit, valType, items, errs)
		case *ast.Field_CompositeLit:
			validateValue(name, aname, a, v.CompositeLit, valType, items, errs)
		}

		if len(a.field.Directives) > 0 {
			validateDirectives(a.field.Directives, ast.DirectiveLocation_ARGUMENT_DEFINITION, items, errs)
		}
	}
}

// validateFields validates a list of field definitions
func validateFields(name string, fields []*ast.Field, items map[string]*ast.TypeDecl, errs *[]*TypeError) map[string]struct {
	field *ast.Field
	count int
} {
	fMap := make(map[string]struct {
		field *ast.Field
		count int
	})
	for _, f := range fields {
		i, exists := fMap[f.Name.Name]
		if !exists {
			i = struct {
				field *ast.Field
				count int
			}{field: f}
			fMap[f.Name.Name] = i
		}

		i.count++
		fMap[f.Name.Name] = i
	}

	for fname, f := range fMap {
		// Ensure field uniqueness
		if f.count > 1 {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: field must be unique", name, fname),
			})
		}

		// Check field name
		if strings.HasPrefix(fname, "__") {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: field name cannot start with \"__\" (double underscore)", name, fname),
			})
		}

		// Validate args
		if args := f.field.Args; args != nil {
			validateArgDefs(fmt.Sprintf("%s:%s", name, fname), args.List, items, errs)
		}

		// Validate field type is an OutputType
		var id *ast.Ident
		var valType interface{}
		switch v := f.field.Type.(type) {
		case *ast.Field_Ident:
			valType = v.Ident
			id = v.Ident
		case *ast.Field_List:
			valType = v.List
			id = unwrapType(v.List)
		case *ast.Field_NonNull:
			valType = v.NonNull
			id = unwrapType(v.NonNull)
		default:
			panic(fmt.Sprintf("compiler: %s:%s: field must have a type", name, fname))
		}

		if !isOutputType(id, items) {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: field type must be a valid output type, not: %s", name, fname, id.Name),
			})
		}

		// Validate any default value provided
		switch v := f.field.Default.(type) {
		case *ast.Field_BasicLit:
			validateValue(name, fname, f, v.BasicLit, valType, items, errs)
		case *ast.Field_CompositeLit:
			validateValue(name, fname, f, v.CompositeLit, valType, items, errs)
		}

		if len(f.field.Directives) > 0 {
			validateDirectives(f.field.Directives, ast.DirectiveLocation_FIELD_DEFINITION, items, errs)
		}
	}

	return fMap
}

// validateInterface validates an interface declaration
func validateInterface(name string, inter *ast.InterfaceType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if inter.Fields == nil {
		return
	}

	if len(inter.Fields.List) == 0 {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s: interface type must one or more fields", name),
		})
		return
	}

	validateFields(name, inter.Fields.List, items, errs)
}

// validateInput validates an input object declaration
func validateInput(name string, input *ast.InputType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if input.Fields == nil {
		return
	}

	if len(input.Fields.List) == 0 {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s: input object type must define one or more input fields", name),
		})
		return
	}

	fMap := validateFields(name, input.Fields.List, items, errs)
	for fname, f := range fMap {
		if f.field.Args != nil {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: input object fields cannot have arguments", name, fname),
			})
		}
	}
}

// validateObject validates an object declaration
func validateObject(name string, object *ast.ObjectType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if object.Fields == nil {
		return
	}

	if len(object.Fields.List) == 0 {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s: an object type must define one or more fields", name),
		})
		return
	}

	// Validate fields
	fMap := validateFields(name, object.Fields.List, items, errs)

	// Check for interfaces
	if len(object.Interfaces) == 0 {
		return
	}

	// Validate interfaces
	for _, inter := range object.Interfaces {
		i, exists := items[inter.Name]
		if !exists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: undefined interface: %s", name, inter.Name),
			})
			continue
		}

		ts, ok := i.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		in, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Interface)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: non-interface type can not be used as interface", name, ts.TypeSpec.Name.Name),
			})
			continue
		}

		if in.Interface.Fields == nil {
			continue
		}

		validateInterfaceFields(name, inter.Name, fMap, in.Interface.Fields.List, items, errs)
	}
}

// validateInterfaceFields validates an objects field set satisfies an interfaces field set
func validateInterfaceFields(objName, interName string, objFields map[string]struct {
	field *ast.Field
	count int
}, interFields []*ast.Field, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	// The object fields must be a super-set of the interface fields
	for _, interField := range interFields {
		objField, exists := objFields[interField.Name.Name]
		if !exists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: object type must include field: %s", objName, interName, interField.Name.Name),
			})
			continue
		}
		fname := interField.Name.Name

		// Check for type existence so it doesn't have to be worried about when comparing types.
		var a, b interface{}
		switch v := objField.field.Type.(type) {
		case *ast.Field_Ident:
			a = v.Ident
		case *ast.Field_List:
			a = v.List
		case *ast.Field_NonNull:
			a = v.NonNull
		}
		switch v := interField.Type.(type) {
		case *ast.Field_Ident:
			b = v.Ident
		case *ast.Field_List:
			b = v.List
		case *ast.Field_NonNull:
			b = v.NonNull
		}

		oid, iid := unwrapType(a), unwrapType(b)
		_, oexists := items[oid.Name]
		if !oexists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: undefined return type: %s", objName, fname, oid.Name),
			})
		}
		_, iexists := items[iid.Name]
		if !iexists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: undefined return type: %s", interName, fname, iid.Name),
			})
		}
		if !oexists || !iexists {
			continue
		}

		// 1. The object field must be of a type which is equal to or a sub-type of the interface field.
		ok := compareTypes(a, b, items)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: object field type must be a sub-type of interface field type", objName, fname),
			})
		}

		// 2. The object field must include an argument of the same name for every argument defined in the
		//	  interface field
		if interField.Args == nil {
			continue
		}
		if objField.field.Args == nil {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: object field must include the same argument definitions that the interface field has", objName, fname),
			})
			continue
		}

		aMap := make(map[string]interface{}, len(objField.field.Args.List))
		for _, oa := range objField.field.Args.List {
			_, exists := aMap[oa.Name.Name]
			if exists {
				continue
			}

			switch v := oa.Type.(type) {
			case *ast.Field_Ident:
				a = v.Ident
			case *ast.Field_List:
				a = v.List
			case *ast.Field_NonNull:
				a = v.NonNull
			}

			aMap[oa.Name.Name] = a
		}

		for _, ia := range interField.Args.List {
			a, exists = aMap[ia.Name.Name]
			if !exists {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: object field is missing interface field argument: %s", objName, fname, ia.Name.Name),
				})
				continue
			}
			delete(aMap, ia.Name.Name)

			switch v := ia.Type.(type) {
			case *ast.Field_Ident:
				b = v.Ident
			case *ast.Field_List:
				b = v.List
			case *ast.Field_NonNull:
				b = v.NonNull
			}

			l := compareTypes(a, b, items)
			r := compareTypes(b, a, items)
			if l && r {
				continue
			}

			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s:%s: object argument and interface argument must be the same type", objName, fname, ia.Name.Name),
			})
		}

		// 3. The object field may include additional arguments not defined in the interface field, but any
		// 	  additional argument must not be required, i.e. must not be of a non‐nullable type.
		for oaName, oaType := range aMap {
			if _, ok := oaType.(*ast.NonNull); ok {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s:%s: additional arguments to interface field implementation must be non-null", objName, fname, oaName),
				})
			}
		}
	}
}

// compareTypes compares two types, a and b.
// It returns a <= b.
func compareTypes(a, b interface{}, items map[string]*ast.TypeDecl) bool {
	ai, _ := a.(*ast.Ident)
	bi, _ := b.(*ast.Ident)
	if ai != nil && bi != nil {
		if ai.Name == bi.Name {
			return true
		}

		// Check if a is a sub-type of b through interface implementation
		at := items[ai.Name].Spec.(*ast.TypeDecl_TypeSpec)
		bt := items[bi.Name].Spec.(*ast.TypeDecl_TypeSpec)

		aObj, ok := at.TypeSpec.Type.(*ast.TypeSpec_Object)
		if !ok {
			return false
		}

		switch v := bt.TypeSpec.Type.(type) {
		case *ast.TypeSpec_Interface:
			for _, i := range aObj.Object.Interfaces {
				if i.Name == bt.TypeSpec.Name.Name {
					return true
				}
			}
		case *ast.TypeSpec_Union:
			for _, m := range v.Union.Members {
				if m.Name == at.TypeSpec.Name.Name {
					return true
				}
			}
		}

		return false
	}

	al, _ := a.(*ast.List)
	bl, _ := b.(*ast.List)
	if al != nil && bl != nil {
		switch v := al.Type.(type) {
		case *ast.List_Ident:
			a = v.Ident
		case *ast.List_List:
			a = v.List
		case *ast.List_NonNull:
			a = v.NonNull
		}

		switch v := bl.Type.(type) {
		case *ast.List_Ident:
			b = v.Ident
		case *ast.List_List:
			b = v.List
		case *ast.List_NonNull:
			b = v.NonNull
		}

		return compareTypes(a, b, items)
	}

	an, _ := a.(*ast.NonNull)
	bn, _ := b.(*ast.NonNull)
	if an != nil {
		switch v := an.Type.(type) {
		case *ast.NonNull_Ident:
			a = v.Ident
		case *ast.NonNull_List:
			a = v.List
		}

		if bn != nil {
			switch v := bn.Type.(type) {
			case *ast.NonNull_Ident:
				b = v.Ident
			case *ast.NonNull_List:
				b = v.List
			}
		}

		return compareTypes(a, b, items)
	}

	return false
}

// validateExtend validates a type extension
func validateExtend(ts *ast.TypeSpec, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	name := "schema"
	if ts.Name != nil {
		name = ts.Name.Name
	}
	ogtd, exists := items[name]
	if !exists {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("extend:%s: no definition found for this type", name),
		})
		return
	}
	ogts := ogtd.Spec.(*ast.TypeDecl_TypeSpec).TypeSpec

	var loc ast.DirectiveLocation_Loc
	switch t := ts.Type.(type) {
	case *ast.TypeSpec_Schema:
		_, ok := ogts.Type.(*ast.TypeSpec_Schema)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:schema: original type defintion must be a schema"),
			})
			return
		}

		loc = ast.DirectiveLocation_SCHEMA
	case *ast.TypeSpec_Scalar:
		_, ok := ogts.Type.(*ast.TypeSpec_Scalar)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:scalar:%s: original type defintion must be a scalar", name),
			})
			return
		}

		loc = ast.DirectiveLocation_SCALAR
	case *ast.TypeSpec_Object:
		ogObj, ok := ogts.Type.(*ast.TypeSpec_Object)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:object:%s: original type defintion must be a object", name),
			})
			return
		}

		// Set name and directive loc
		name = fmt.Sprintf("extend:object:%s", name)
		loc = ast.DirectiveLocation_OBJECT
		ogFields := ogObj.Object.Fields
		if ogFields == nil {
			break
		}

		// Collect fields
		fMap := make(map[string]struct {
			field *ast.Field
			count int
		})
		for _, f := range ogFields.List {
			fMap[f.Name.Name] = struct {
				field *ast.Field
				count int
			}{field: f}
		}

		// Validate any new fields
		if t.Object.Fields != nil {
			efMap := validateFields(name, t.Object.Fields.List, items, errs)

			for efName, ef := range efMap {
				if _, ok := fMap[efName]; ok {
					*errs = append(*errs, &TypeError{
						Msg: fmt.Sprintf("%s:%s: field defintion already exists in original object definition", name, efName),
					})
					continue
				}

				fMap[efName] = struct {
					field *ast.Field
					count int
				}{field: ef.field}
			}
		}

		// Validate interfaces
		if len(t.Object.Interfaces) == 0 {
			break
		}

		// Validate interfaces
		for _, inter := range t.Object.Interfaces {
			i, exists := items[inter.Name]
			if !exists {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s: undefined interface: %s", name, inter.Name),
				})
				continue
			}

			ts, ok := i.Spec.(*ast.TypeDecl_TypeSpec)
			if !ok {
				continue
			}

			in, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Interface)
			if !ok {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: non-interface type can not be used as interface", name, ts.TypeSpec.Name.Name),
				})
				continue
			}

			if in.Interface.Fields == nil {
				continue
			}

			validateInterfaceFields(name, inter.Name, fMap, in.Interface.Fields.List, items, errs)
		}
	case *ast.TypeSpec_Interface:
		ogInter, ok := ogts.Type.(*ast.TypeSpec_Interface)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:interface:%s: original type defintion must be a interface", name),
			})
			return
		}

		name = fmt.Sprintf("extend:interface:%s", name)
		loc = ast.DirectiveLocation_INTERFACE
		if t.Interface.Fields == nil || ogInter.Interface.Fields == nil {
			break
		}

		validateInterface(name, t.Interface, items, errs)

		for _, of := range ogInter.Interface.Fields.List {
			for _, ef := range t.Interface.Fields.List {
				if of.Name.Name == ef.Name.Name {
					*errs = append(*errs, &TypeError{
						Msg: fmt.Sprintf("%s:%s: field already exists in original interface definition", name, of.Name.Name),
					})
				}
			}
		}

		// TODO: Any object type which implemented the original interface type must also be a super-set
		// 		 of the fields of the interface type extension (which may be due to object type extension)
	case *ast.TypeSpec_Union:
		ogUnion, ok := ogts.Type.(*ast.TypeSpec_Union)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:union:%s: original type defintion must be a union", name),
			})
			return
		}

		name = fmt.Sprintf("extend:union:%s", name)
		loc = ast.DirectiveLocation_UNION
		if t.Union.Members == nil || ogUnion.Union.Members == nil {
			break
		}

		validateUnion(name, t.Union, items, errs)

		for _, om := range ogUnion.Union.Members {
			for _, em := range t.Union.Members {
				if om.Name == em.Name {
					*errs = append(*errs, &TypeError{
						Msg: fmt.Sprintf("%s:%s: union member already exists in original union definition", name, om.Name),
					})
				}
			}
		}
	case *ast.TypeSpec_Enum:
		ogEnum, ok := ogts.Type.(*ast.TypeSpec_Enum)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:enum:%s: original type defintion must be a enum", name),
			})
			return
		}

		name = fmt.Sprintf("extend:enum:%s", name)
		loc = ast.DirectiveLocation_ENUM
		if t.Enum.Values == nil || ogEnum.Enum.Values == nil {
			break
		}

		validateEnum(name, t.Enum, items, errs)

		for _, oev := range ogEnum.Enum.Values.List {
			for _, eev := range t.Enum.Values.List {
				if oev.Name.Name == eev.Name.Name {
					*errs = append(*errs, &TypeError{
						Msg: fmt.Sprintf("%s:%s: enum value already exists in original enum definition", name, oev.Name.Name),
					})
				}
			}
		}
	case *ast.TypeSpec_Input:
		ogInput, ok := ogts.Type.(*ast.TypeSpec_Input)
		if !ok {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("extend:input:%s: original type defintion must be a input", name),
			})
			return
		}

		name = fmt.Sprintf("extend:input:%s", name)
		loc = ast.DirectiveLocation_INPUT_OBJECT
		if t.Input.Fields == nil || ogInput.Input.Fields == nil {
			break
		}

		// Validate fields and check that they don't have args
		validateInput(name, t.Input, items, errs)

		// Validate any new fields aren't already in og input def
		for _, of := range ogInput.Input.Fields.List {
			for _, ef := range t.Input.Fields.List {
				if of.Name.Name == ef.Name.Name {
					*errs = append(*errs, &TypeError{
						Msg: fmt.Sprintf("%s:%s: field defintion already exists in original input definition", name, of.Name.Name),
					})
				}
			}
		}
	default:
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("extend:%s: type extensions are not supported for this type", ts.Name.Name),
		})
		return
	}

	// Any directives applied to extension must not already be applied to the original type
	validateDirectives(ts.Directives, loc, items, errs)
	for _, od := range ogts.Directives {
		for _, ed := range ts.Directives {
			if od.Name == ed.Name {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: directive is already applied to original type definition", name, od.Name),
				})
			}
		}
	}
}

// validateDirective validates a directive declaration
func validateDirective(name string, directive *ast.DirectiveType, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	if directive.Args == nil {
		return
	}

	for _, f := range directive.Args.List {
		// 1. Check name of arg
		if strings.HasPrefix(f.Name.Name, "__") {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: argument name cannot start with \"__\" (double underscore)", name, f.Name.Name),
			})
		}

		// 2. Verify that the arg type is an input type
		var id *ast.Ident
		var valType interface{}
		switch v := f.Type.(type) {
		case *ast.Field_Ident:
			valType = v.Ident
			id = v.Ident
		case *ast.Field_List:
			valType = v.List
			id = unwrapType(v.List)
		case *ast.Field_NonNull:
			valType = v.NonNull
			id = unwrapType(v.NonNull)
		default:
			panic(fmt.Sprintf("compiler: %s:%s: directive argument must have a type", name, f.Name.Name))
		}

		if !isInputType(id, items) {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: directive argument must be a valid input type, not: %s", name, f.Name.Name, id.Name),
			})
		}

		// 3. Validate any default value provided
		switch v := f.Default.(type) {
		case *ast.Field_BasicLit:
			validateValue(name, f.Name.Name, f, v.BasicLit, valType, items, errs)
		case *ast.Field_CompositeLit:
			validateValue(name, f.Name.Name, f, v.CompositeLit, valType, items, errs)
		}

		// 4. Check that the arg directives don't reference this one
		for _, d := range f.Directives {
			if d.Name == name {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: directive argument cannont reference its own directive definition", name, f.Name.Name),
				})
			}
		}

		if len(f.Directives) > 0 {
			validateDirectives(f.Directives, ast.DirectiveLocation_ARGUMENT_DEFINITION, items, errs)
		}

		// TODO: 5. Check that the arg Type doesn't reference this directive
	}
}

// validateArgs validates a list of args. host can either be
func validateArgs(host string, argDefs []*ast.Field, args []*ast.Arg, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	argMap := make(map[string]struct {
		arg   *ast.Arg
		count int
	}, len(args))
	for _, arg := range args {
		if _, exists := argMap[arg.Name.Name]; !exists {
			argMap[arg.Name.Name] = struct {
				arg   *ast.Arg
				count int
			}{
				arg: arg,
			}
		}

		a := argMap[arg.Name.Name]
		a.count += 1
	}

	for _, argDef := range argDefs {
		// Args must be unique
		a, exists := argMap[argDef.Name.Name]
		if exists {
			delete(argMap, argDef.Name.Name)
		}

		if a.count > 1 {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: arg must be unique in: %s", argDef.Name.Name, host),
			})
			continue
		}
		delete(argMap, argDef.Name.Name)

		// Extract value and value type for arg
		var val, valType interface{}
		switch v := argDef.Type.(type) {
		case *ast.Field_Ident:
			valType = v.Ident
		case *ast.Field_List:
			valType = v.List
		case *ast.Field_NonNull:
			valType = v.NonNull
		}

		// 3: Non-null args are required and cannot have non value if defVal doesn't exist
		_, isNonNull := valType.(*ast.NonNull)
		switch a.arg == nil {
		case !isNonNull: // optional arg
			continue
		case isNonNull && argDef.Default != nil: // not required cuz it has a default value
			continue
		case isNonNull && argDef.Default == nil: // required cuz it doesn't hav a default value
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: non-null arg must be present in: %s", argDef.Name.Name, host),
			})
			continue
		}

		switch v := a.arg.Value.(type) {
		case *ast.Arg_BasicLit:
			val = v.BasicLit
		case *ast.Arg_CompositeLit:
			val = v.CompositeLit
		}

		validateValue(host, a.arg.Name.Name, a.arg, val, valType, items, errs)
	}

	// Args must exist
	for arg := range argMap {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s: undefined arg: %s", host, arg),
		})
	}
}

// validateValue validates a value
func validateValue(host, cName string, c interface{}, val, valType interface{}, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	switch u := valType.(type) {
	case *ast.Ident:
		// Check if its a composite
		var bLit *ast.BasicLit
		var cLit *ast.CompositeLit
		switch v := val.(type) {
		case *ast.BasicLit:
			bLit = v
		case *ast.CompositeLit:
			ccLit, ok := v.Value.(*ast.CompositeLit_BasicLit)
			if ok {
				bLit = ccLit.BasicLit
				break
			}

			cLit = v
		default:
			panic("compiler: validateValue can only be provided an ast.BasicLit or ast.CompositeLit val")
		}

		if cLit != nil {
			objLit, ok := cLit.Value.(*ast.CompositeLit_ObjLit)
			if !ok {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: input object must be provided", host, cName),
				})
				return
			}

			objDef, exists := items[u.Name]
			if !exists {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: undefined input object: %s", host, cName, u.Name),
				})
				return
			}

			objSpec, ok := objDef.Spec.(*ast.TypeDecl_TypeSpec)
			if !ok {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: could not find type spec for input object: %s", host, cName, u.Name),
				})
				return
			}

			inputType, ok := objSpec.TypeSpec.Type.(*ast.TypeSpec_Input)
			if !ok {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: %s is not an input object", host, cName, u.Name),
				})
				return
			}

			validateObj(host, cName, inputType.Input.Fields.List, objLit.ObjLit.Fields, items, errs)
			return
		}

		// Coerce builtin scalar types
		switch u.Name {
		case "Int":
			if bLit.Kind != int64(token.INT) {
				break
			}

			return
		case "Float":
			if bLit.Kind != int64(token.INT) && bLit.Kind != int64(token.FLOAT) {
				break
			}

			if bLit.Kind == int64(token.INT) {
				bLit.Value += ".0"
			}

			bLit.Kind = int64(token.FLOAT)
			return
		case "String":
			if bLit.Kind != int64(token.STRING) {
				break
			}

			return
		case "Boolean":
			if bLit.Kind != int64(token.BOOL) {
				break
			}

			return
		case "ID":
			if bLit.Kind != int64(token.STRING) && bLit.Kind != int64(token.INT) {
				break
			}

			return
		}

		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s:%s: %s is not coercible to: %s", host, cName, token.Token(bLit.Kind), u.Name),
		})
	case *ast.List:
		switch v := u.Type.(type) {
		case *ast.List_Ident:
			valType = v.Ident
		case *ast.List_List:
			valType = v.List
		case *ast.List_NonNull:
			valType = v.NonNull
		}

		var cLit *ast.CompositeLit
		switch v := val.(type) {
		case *ast.BasicLit:
		case *ast.CompositeLit:
			_, ok := v.Value.(*ast.CompositeLit_BasicLit)
			if ok {
				break
			}

			cLit = v
		default:
			panic("compiler: validateValue can only be provided an ast.BasicLit or ast.CompositeLit val")
		}

		if cLit != nil {
			switch v := cLit.Value.(type) {
			case *ast.CompositeLit_ListLit:
				var vals []interface{}
				switch w := v.ListLit.List.(type) {
				case *ast.ListLit_BasicList:
					for _, b := range w.BasicList.Values {
						vals = append(vals, b)
					}
				case *ast.ListLit_CompositeList:
					for _, c := range w.CompositeList.Values {
						vals = append(vals, c)
					}
				}

				for _, l := range vals {
					validateValue(host, cName, c, l, valType, items, errs)
				}

				return
			case *ast.CompositeLit_ObjLit:
				val = cLit
				break
			}
		}

		validateValue(host, cName, c, val, valType, items, errs)

		// Coerce single lit to list
		listLit := new(ast.ListLit)
		switch w := c.(type) {
		case *ast.Field:
			switch x := w.Default.(type) {
			case *ast.Field_BasicLit:
				listLit.List = &ast.ListLit_BasicList{
					BasicList: &ast.ListLit_Basic{
						Values: []*ast.BasicLit{x.BasicLit},
					},
				}
			case *ast.Field_CompositeLit:
				listLit.List = &ast.ListLit_CompositeList{
					CompositeList: &ast.ListLit_Composite{
						Values: []*ast.CompositeLit{x.CompositeLit},
					},
				}
			}

			w.Default = &ast.Field_CompositeLit{CompositeLit: &ast.CompositeLit{
				Value: &ast.CompositeLit_ListLit{
					ListLit: listLit,
				},
			}}
		case *ast.Arg:
			switch x := w.Value.(type) {
			case *ast.Arg_BasicLit:
				listLit.List = &ast.ListLit_BasicList{
					BasicList: &ast.ListLit_Basic{
						Values: []*ast.BasicLit{x.BasicLit},
					},
				}
			case *ast.Arg_CompositeLit:
				listLit.List = &ast.ListLit_CompositeList{
					CompositeList: &ast.ListLit_Composite{
						Values: []*ast.CompositeLit{x.CompositeLit},
					},
				}
			}

			w.Value = &ast.Arg_CompositeLit{CompositeLit: &ast.CompositeLit{
				Value: &ast.CompositeLit_ListLit{
					ListLit: listLit,
				},
			}}
		case *ast.ObjLit_Pair:
			switch x := w.Val.Value.(type) {
			case *ast.CompositeLit_BasicLit:
				listLit.List = &ast.ListLit_BasicList{
					BasicList: &ast.ListLit_Basic{
						Values: []*ast.BasicLit{x.BasicLit},
					},
				}
			case *ast.CompositeLit_ObjLit:
				listLit.List = &ast.ListLit_CompositeList{
					CompositeList: &ast.ListLit_Composite{
						Values: []*ast.CompositeLit{w.Val},
					},
				}
			}

			w.Val.Value = &ast.CompositeLit_ListLit{
				ListLit: listLit,
			}
		}
	case *ast.NonNull:
		switch v := u.Type.(type) {
		case *ast.NonNull_Ident:
			valType = v.Ident

			bLit, ok := val.(*ast.BasicLit)
			if !ok {
				break
			}

			if bLit.Kind == int64(token.NULL) {
				*errs = append(*errs, &TypeError{
					Msg: fmt.Sprintf("%s:%s: non-null arg cannot be the null value", host, cName),
				})
				return
			}
		case *ast.NonNull_List:
			valType = v.List
		}

		validateValue(host, cName, c, val, valType, items, errs)
	}
}

// validateObj validates an input value
func validateObj(host, arg string, fieldDefs []*ast.Field, objFields []*ast.ObjLit_Pair, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	objFieldMap := make(map[string]struct {
		objField *ast.ObjLit_Pair
		count    int
	}, len(objFields))
	for _, f := range objFields {
		if _, exists := objFieldMap[f.Key.Name]; !exists {
			objFieldMap[f.Key.Name] = struct {
				objField *ast.ObjLit_Pair
				count    int
			}{
				objField: f,
			}
		}

		o := objFieldMap[f.Key.Name]
		o.count += 1
	}

	for _, fieldDef := range fieldDefs {
		f, exists := objFieldMap[fieldDef.Name.Name]
		if exists {
			delete(objFieldMap, fieldDef.Name.Name)
		}

		// Fields must be unique
		if f.count > 1 {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s:%s: field must be unique: %s", host, arg, fieldDef.Name.Name),
			})
			continue
		}

		// Extract value and value type for arg
		var val, valType interface{}
		switch v := fieldDef.Type.(type) {
		case *ast.Field_Ident:
			valType = v.Ident
		case *ast.Field_List:
			valType = v.List
		case *ast.Field_NonNull:
			valType = v.NonNull
		}

		// 3: Non-null args are required and cannot have non value if defVal doesn't exist
		_, isNonNull := valType.(*ast.NonNull)
		switch f.objField == nil {
		case !isNonNull: // optional arg
			continue
		case isNonNull && fieldDef.Default != nil: // not required cuz it has a default value
			continue
		case isNonNull && fieldDef.Default == nil: // required cuz it doesn't hav a default value
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: non-null field must be present in: %s", fieldDef.Name.Name, host),
			})
			continue
		}

		switch v := f.objField.Val.Value.(type) {
		case *ast.CompositeLit_BasicLit:
			val = v.BasicLit
		case *ast.CompositeLit_ListLit:
			val = v.ListLit
		case *ast.CompositeLit_ObjLit:
			val = v.ObjLit
		}

		validateValue(fmt.Sprintf("%s:%s", host, arg), fieldDef.Name.Name, f.objField, val, valType, items, errs)
	}

	// Fields must exist
	for _, f := range objFieldMap {
		*errs = append(*errs, &TypeError{
			Msg: fmt.Sprintf("%s:%s: undefined field: %s", host, arg, f.objField.Key.Name),
		})
	}
}

// validateDirectives validates a list of applied directives
func validateDirectives(directives []*ast.DirectiveLit, loc ast.DirectiveLocation_Loc, items map[string]*ast.TypeDecl, errs *[]*TypeError) {
	dirMap := make(map[string]struct {
		dirLit *ast.DirectiveLit
		count  int
	}, len(directives))
	for _, dirLit := range directives {
		i, exists := dirMap[dirLit.Name]
		if !exists {
			i = struct {
				dirLit *ast.DirectiveLit
				count  int
			}{dirLit: dirLit}
			dirMap[dirLit.Name] = i
		}

		i.count++
		dirMap[dirLit.Name] = i
	}

	for name, d := range dirMap {
		// 1: Directive definition must exist
		dirDef, exists := items[name]
		if !exists {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: undefined directive", name),
			})
			continue
		}

		dirSpec, ok := dirDef.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		dirType := dirSpec.TypeSpec.Type.(*ast.TypeSpec_Directive).Directive

		// 2: Directive must be applied in proper location
		var validLoc bool
		for _, l := range dirType.Locs {
			if l.Loc == loc {
				validLoc = true
			}
		}
		if !validLoc {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: invalid location for directive: %s", name, loc),
			})
			continue
		}

		// 3: Directives must be unique per location
		if d.count > 1 {
			*errs = append(*errs, &TypeError{
				Msg: fmt.Sprintf("%s: directive cannot be applied more than once per location: %s", name, loc),
			})
		}

		// 4: Directive arguments must be valid
		if dirType.Args == nil || d.dirLit.Args == nil {
			continue
		}
		validateArgs(name, dirType.Args.List, d.dirLit.Args.Args, items, errs)
	}
}

// checkName enforces that no Ident starts with "__" (two underscores).
func checkName(typ token.Token, name *ast.Ident, errs *[]*TypeError) {
	if !strings.HasPrefix(name.Name, "__") {
		return
	}

	*errs = append(*errs, &TypeError{
		Msg: fmt.Sprintf("%s is an invalid name for type: %s", name.Name, typ),
	})
}

func isInputType(id *ast.Ident, items map[string]*ast.TypeDecl) bool {
	i, exists := items[id.Name]
	if !exists {
		return false
	}

	ts, ok := i.Spec.(*ast.TypeDecl_TypeSpec)
	if !ok {
		return false
	}

	switch ts.TypeSpec.Type.(type) {
	case *ast.TypeSpec_Scalar:
	case *ast.TypeSpec_Enum:
	case *ast.TypeSpec_Input:
	default:
		return false
	}

	return true
}

func isOutputType(id *ast.Ident, items map[string]*ast.TypeDecl) bool {
	i, exists := items[id.Name]
	if !exists {
		return false
	}

	ts, ok := i.Spec.(*ast.TypeDecl_TypeSpec)
	if !ok {
		return false
	}

	switch ts.TypeSpec.Type.(type) {
	case *ast.TypeSpec_Scalar:
	case *ast.TypeSpec_Object:
	case *ast.TypeSpec_Interface:
	case *ast.TypeSpec_Union:
	case *ast.TypeSpec_Enum:
	default:
		return false
	}

	return true
}
