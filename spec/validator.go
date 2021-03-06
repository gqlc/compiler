package spec

import (
	"fmt"
	"strings"

	"github.com/gqlc/compiler"
	"github.com/gqlc/graphql/ast"
	"github.com/gqlc/graphql/token"
)

// Validator uses the rules defined in the GraphQL spec to validates types.
var Validator = compiler.TypeCheckerFn(validate)

type typeDecls struct {
	ir    compiler.IR
	types map[string][]*ast.TypeDecl
}

func (decls typeDecls) lookup(name string) []*ast.TypeDecl {
	decl, ok := decls.types[name]
	if ok {
		return decl
	}

	_, decl = compiler.Lookup(name, decls.ir)
	return decl
}

func validate(ir compiler.IR) (errs []error) {
	for doc, types := range ir {
		typeDecl := typeDecls{types: types, ir: ir}

		for name, decls := range types {
			decl := decls[0]

			// Make sure the front is a TypeSpec and not an TypeExt
			ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
			if !ok {
				errs = append(errs, fmt.Errorf("missing type declaration for: %s", name))
				continue
			}

			typ, loc := validateType(ts.TypeSpec, typeDecl, &errs)

			// Check type name
			if loc != ast.DirectiveLocation_SCHEMA {
				checkName(typ, ts.TypeSpec.Name, &errs)
			}

			// Validate applied directives
			if loc != ast.DirectiveLocation_NoPos {
				validateDirectives(ts.TypeSpec.Directives, loc, typeDecl, &errs)
			}

			for _, decl = range decls[1:] {
				exts, ok := decl.Spec.(*ast.TypeDecl_TypeExtSpec)
				if !ok {
					errs = append(errs, fmt.Errorf("cannot have more than one type definition for: %s", name))
					continue
				}

				validateExtend(ts.TypeSpec, exts.TypeExtSpec.Type, typeDecl, &errs)
			}
		}

		// Validate top-lvl directives
		validateDirectives(doc.Directives, ast.DirectiveLocation_DOCUMENT, typeDecl, &errs)
	}

	return
}

func validateType(ts *ast.TypeSpec, decls typeDecls, errs *[]error) (typ token.Token, loc ast.DirectiveLocation_Loc) {
	switch v := ts.Type.(type) {
	case *ast.TypeSpec_Schema:
		validateSchema(v.Schema, decls, errs)

		typ = token.Token_SCHEMA
		loc = ast.DirectiveLocation_SCHEMA
	case *ast.TypeSpec_Scalar:
		typ = token.Token_SCALAR
		loc = ast.DirectiveLocation_SCALAR
	case *ast.TypeSpec_Enum:
		validateEnum(ts.Name.Name, v.Enum, decls, errs)

		typ = token.Token_ENUM
		loc = ast.DirectiveLocation_ENUM
	case *ast.TypeSpec_Union:
		validateUnion(ts.Name.Name, v.Union, decls, errs)

		typ = token.Token_UNION
		loc = ast.DirectiveLocation_UNION
	case *ast.TypeSpec_Interface:
		validateInterface(ts.Name.Name, v.Interface, decls, errs)

		typ = token.Token_INTERFACE
		loc = ast.DirectiveLocation_INTERFACE
	case *ast.TypeSpec_Input:
		validateInput(ts.Name.Name, v.Input, decls, errs)

		typ = token.Token_INPUT
		loc = ast.DirectiveLocation_INPUT_OBJECT
	case *ast.TypeSpec_Object:
		validateObject(ts.Name.Name, v.Object, decls, errs)

		typ = token.Token_TYPE
		loc = ast.DirectiveLocation_OBJECT
	case *ast.TypeSpec_Directive:
		validateDirective(ts.Name.Name, v.Directive, decls, errs)

		typ = token.Token_DIRECTIVE
	}
	return
}

// validateSchema validates a schema declaration
func validateSchema(schema *ast.SchemaType, items typeDecls, errs *[]error) {
	if schema.RootOps == nil {
		return
	}

	if len(schema.RootOps.List) == 0 {
		*errs = append(*errs, fmt.Errorf("schema: at minimum query object must be provided"))
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
			*errs = append(*errs, fmt.Errorf("schema:%s: root operation return type can not be a list type", f.Name.Name))
			continue
		case *ast.Field_NonNull:
			*errs = append(*errs, fmt.Errorf("schema:%s: root operation return type can not be a non null type", f.Name.Name))
			continue
		default:
			panic(fmt.Sprintf("spec: schema:%s: must have type", f.Name.Name))
		}

		decls := items.lookup(id.Name)
		if decls == nil {
			*errs = append(*errs, fmt.Errorf("schema:%s: unknown type: %s", f.Name.Name, id.Name))
			continue
		}

		decl := decls[0]

		ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		if _, ok = ts.TypeSpec.Type.(*ast.TypeSpec_Object); !ok {
			*errs = append(*errs, fmt.Errorf("schema:%s: root operation return type must be an object type", f.Name.Name))
		}
	}

	if !hasQuery {
		*errs = append(*errs, fmt.Errorf("schema: query object must be provided"))
	}
}

// validateEnum validates an enum declaration
func validateEnum(name string, enum *ast.EnumType, items typeDecls, errs *[]error) {
	if enum.Values == nil {
		return
	}

	if len(enum.Values.List) == 0 {
		*errs = append(*errs, fmt.Errorf("%s: enum type must define one or more unique enum values", name))
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

		*errs = append(*errs, fmt.Errorf("%s:%s: enum value must be unique", name, v))
	}
}

// validateUnion validates a union declaration
func validateUnion(name string, union *ast.UnionType, items typeDecls, errs *[]error) {
	if union.Members == nil {
		return
	}

	if len(union.Members) == 0 {
		*errs = append(*errs, fmt.Errorf("%s: union type must include one or more unique member types", name))
		return
	}

	vMap := make(map[string]int, len(union.Members))
	for _, v := range union.Members {
		c := vMap[v.Name]
		vMap[v.Name] = c + 1
	}

	for v, c := range vMap {
		decls := items.lookup(v)
		if decls == nil {
			*errs = append(*errs, fmt.Errorf("%s:%s: undefined type", name, v))
			continue
		}

		decl := decls[0]

		ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		if _, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Object); !ok {
			*errs = append(*errs, fmt.Errorf("%s:%s: member type must be an object type", name, v))
		}

		if c > 1 {
			*errs = append(*errs, fmt.Errorf("%s:%s: member type must be unique", name, v))
		}
	}
}

// validateArgDefs validates a list of argument definitions
func validateArgDefs(name string, args []*ast.InputValue, items typeDecls, errs *[]error) {
	aMap := make(map[string]struct {
		field *ast.InputValue
		count int
	})
	for _, f := range args {
		i, exists := aMap[f.Name.Name]
		if !exists {
			i = struct {
				field *ast.InputValue
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
			*errs = append(*errs, fmt.Errorf("%s:%s: argument must be unique", name, aname))
		}

		// Check field name
		if strings.HasPrefix(aname, "__") {
			*errs = append(*errs, fmt.Errorf("%s:%s: argument name cannot start with \"__\" (double underscore)", name, aname))
		}

		// Validate field type is an InputType
		var id *ast.Ident
		var valType interface{}
		switch v := a.field.Type.(type) {
		case *ast.InputValue_Ident:
			valType = v.Ident
			id = v.Ident
		case *ast.InputValue_List:
			valType = v.List
			id = unwrapType(v.List)
		case *ast.InputValue_NonNull:
			valType = v.NonNull
			id = unwrapType(v.NonNull)
		default:
			panic(fmt.Sprintf("spec: %s:%s: argument must have a type", name, aname))
		}

		if !isInputType(id, items) {
			*errs = append(*errs, fmt.Errorf("%s:%s: argument type must be a valid input type, not: %s", name, aname, id.Name))
		}

		// Validate any default value provided
		switch v := a.field.Default.(type) {
		case *ast.InputValue_BasicLit:
			validateValue(name, aname, a, v.BasicLit, valType, items, errs)
		case *ast.InputValue_CompositeLit:
			validateValue(name, aname, a, v.CompositeLit, valType, items, errs)
		}

		if len(a.field.Directives) > 0 {
			validateDirectives(a.field.Directives, ast.DirectiveLocation_ARGUMENT_DEFINITION, items, errs)
		}
	}
}

// validateFields validates a list of field definitions
func validateFields(name string, fields []*ast.Field, items typeDecls, errs *[]error) map[string]struct {
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
			*errs = append(*errs, fmt.Errorf("%s:%s: field must be unique", name, fname))
		}

		// Check field name
		if strings.HasPrefix(fname, "__") {
			*errs = append(*errs, fmt.Errorf("%s:%s: field name cannot start with \"__\" (double underscore)", name, fname))
		}

		// Validate args
		if args := f.field.Args; args != nil {
			validateArgDefs(fmt.Sprintf("%s:%s", name, fname), args.List, items, errs)
		}

		// Validate field type is an OutputType
		var id *ast.Ident
		switch v := f.field.Type.(type) {
		case *ast.Field_Ident:
			id = v.Ident
		case *ast.Field_List:
			id = unwrapType(v.List)
		case *ast.Field_NonNull:
			id = unwrapType(v.NonNull)
		default:
			panic(fmt.Sprintf("compiler: %s:%s: field must have a type", name, fname))
		}

		if !isOutputType(id, items) {
			*errs = append(*errs, fmt.Errorf("%s:%s: field type must be a valid output type, not: %s", name, fname, id.Name))
		}

		if len(f.field.Directives) > 0 {
			validateDirectives(f.field.Directives, ast.DirectiveLocation_FIELD_DEFINITION, items, errs)
		}
	}

	return fMap
}

// validateInterface validates an interface declaration
func validateInterface(name string, inter *ast.InterfaceType, items typeDecls, errs *[]error) {
	if inter.Fields == nil {
		return
	}

	if len(inter.Fields.List) == 0 {
		*errs = append(*errs, fmt.Errorf("%s: interface type must one or more fields", name))
		return
	}

	validateFields(name, inter.Fields.List, items, errs)
}

// validateInput validates an input object declaration
func validateInput(name string, input *ast.InputType, items typeDecls, errs *[]error) {
	if input.Fields == nil {
		return
	}

	if len(input.Fields.List) == 0 {
		*errs = append(*errs, fmt.Errorf("%s: input object type must define one or more input fields", name))
		return
	}

	validateArgDefs(name, input.Fields.List, items, errs)
}

// validateObject validates an object declaration
func validateObject(name string, object *ast.ObjectType, items typeDecls, errs *[]error) {
	if object.Fields == nil {
		return
	}

	if len(object.Fields.List) == 0 {
		*errs = append(*errs, fmt.Errorf("%s: an object type must define one or more fields", name))
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
		decls := items.lookup(inter.Name)
		if decls == nil {
			*errs = append(*errs, fmt.Errorf("%s: undefined interface: %s", name, inter.Name))
			continue
		}

		decl := decls[0]

		ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		in, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Interface)
		if !ok {
			*errs = append(*errs, fmt.Errorf("%s:%s: non-interface type can not be used as interface", name, ts.TypeSpec.Name.Name))
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
}, interFields []*ast.Field, items typeDecls, errs *[]error) {
	// The object fields must be a super-set of the interface fields
	for _, interField := range interFields {
		objField, exists := objFields[interField.Name.Name]
		if !exists {
			*errs = append(*errs, fmt.Errorf("%s:%s: object type must include field: %s", objName, interName, interField.Name.Name))
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
		odecls := items.lookup(oid.Name)
		if odecls == nil {
			*errs = append(*errs, fmt.Errorf("%s:%s: undefined return type: %s", objName, fname, oid.Name))
		}
		idecls := items.lookup(iid.Name)
		if idecls == nil {
			*errs = append(*errs, fmt.Errorf("%s:%s: undefined return type: %s", interName, fname, iid.Name))
		}
		if odecls == nil || idecls == nil {
			continue
		}

		// 1. The object field must be of a type which is equal to or a sub-type of the interface field.
		ok := compareTypes(a, b, items)
		if !ok {
			*errs = append(*errs, fmt.Errorf("%s:%s: object field type must be a sub-type of interface field type", objName, fname))
		}

		// 2. The object field must include an argument of the same name for every argument defined in the
		//	  interface field
		if interField.Args == nil {
			continue
		}
		if objField.field.Args == nil {
			*errs = append(*errs, fmt.Errorf("%s:%s: object field must include the same argument definitions that the interface field has", objName, fname))
			continue
		}

		aMap := make(map[string]interface{}, len(objField.field.Args.List))
		for _, oa := range objField.field.Args.List {
			_, exists := aMap[oa.Name.Name]
			if exists {
				continue
			}

			switch v := oa.Type.(type) {
			case *ast.InputValue_Ident:
				a = v.Ident
			case *ast.InputValue_List:
				a = v.List
			case *ast.InputValue_NonNull:
				a = v.NonNull
			}

			aMap[oa.Name.Name] = a
		}

		for _, ia := range interField.Args.List {
			a, exists = aMap[ia.Name.Name]
			if !exists {
				*errs = append(*errs, fmt.Errorf("%s:%s: object field is missing interface field argument: %s", objName, fname, ia.Name.Name))
				continue
			}
			delete(aMap, ia.Name.Name)

			switch v := ia.Type.(type) {
			case *ast.InputValue_Ident:
				b = v.Ident
			case *ast.InputValue_List:
				b = v.List
			case *ast.InputValue_NonNull:
				b = v.NonNull
			}

			l := compareTypes(a, b, items)
			r := compareTypes(b, a, items)
			if l && r {
				continue
			}

			*errs = append(*errs, fmt.Errorf("%s:%s:%s: object argument and interface argument must be the same type", objName, fname, ia.Name.Name))
		}

		// 3. The object field may include additional arguments not defined in the interface field, but any
		// 	  additional argument must not be required, i.e. must not be of a non‐nullable type.
		for oaName, oaType := range aMap {
			if _, ok := oaType.(*ast.NonNull); ok {
				*errs = append(*errs, fmt.Errorf("%s:%s:%s: additional arguments to interface field implementation must be non-null", objName, fname, oaName))
			}
		}
	}
}

// compareTypes compares two types, a and b.
// It returns a <= b.
func compareTypes(a, b interface{}, items typeDecls) bool {
	ai, _ := a.(*ast.Ident)
	bi, _ := b.(*ast.Ident)
	if ai != nil && bi != nil {
		if ai.Name == bi.Name {
			return true
		}

		ad := items.lookup(ai.Name)[0]
		bd := items.lookup(bi.Name)[0]

		// Check if a is a sub-type of b through interface implementation
		at := ad.Spec.(*ast.TypeDecl_TypeSpec)
		bt := bd.Spec.(*ast.TypeDecl_TypeSpec)

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
func validateExtend(ogts, exts *ast.TypeSpec, items typeDecls, errs *[]error) {
	name := "schema"
	if exts.Name != nil {
		name = exts.Name.Name
	}

	var loc ast.DirectiveLocation_Loc
	switch t := exts.Type.(type) {
	case *ast.TypeSpec_Schema:
		_, ok := ogts.Type.(*ast.TypeSpec_Schema)
		if !ok {
			*errs = append(*errs, fmt.Errorf("extend:schema: original type definition must be a schema"))
			return
		}

		loc = ast.DirectiveLocation_SCHEMA
	case *ast.TypeSpec_Scalar:
		_, ok := ogts.Type.(*ast.TypeSpec_Scalar)
		if !ok {
			*errs = append(*errs, fmt.Errorf("extend:scalar:%s: original type definition must be a scalar", name))
			return
		}

		loc = ast.DirectiveLocation_SCALAR
	case *ast.TypeSpec_Object:
		ogObj, ok := ogts.Type.(*ast.TypeSpec_Object)
		if !ok {
			*errs = append(*errs, fmt.Errorf("extend:object:%s: original type definition must be a object", name))
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
					*errs = append(*errs, fmt.Errorf("%s:%s: field definition already exists in original object definition", name, efName))
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
			decls := items.lookup(inter.Name)
			if decls == nil {
				*errs = append(*errs, fmt.Errorf("%s: undefined interface: %s", name, inter.Name))
				continue
			}

			decl := decls[0]

			ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
			if !ok {
				continue
			}

			in, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Interface)
			if !ok {
				*errs = append(*errs, fmt.Errorf("%s:%s: non-interface type can not be used as interface", name, ts.TypeSpec.Name.Name))
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
			*errs = append(*errs, fmt.Errorf("extend:interface:%s: original type definition must be a interface", name))
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
					*errs = append(*errs, fmt.Errorf("%s:%s: field already exists in original interface definition", name, of.Name.Name))
				}
			}
		}

		// TODO: Any object type which implemented the original interface type must also be a super-set
		// 		 of the fields of the interface type extension (which may be due to object type extension)
	case *ast.TypeSpec_Union:
		ogUnion, ok := ogts.Type.(*ast.TypeSpec_Union)
		if !ok {
			*errs = append(*errs, fmt.Errorf("extend:union:%s: original type definition must be a union", name))
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
					*errs = append(*errs, fmt.Errorf("%s:%s: union member already exists in original union definition", name, om.Name))
				}
			}
		}
	case *ast.TypeSpec_Enum:
		ogEnum, ok := ogts.Type.(*ast.TypeSpec_Enum)
		if !ok {
			*errs = append(*errs, fmt.Errorf("extend:enum:%s: original type definition must be a enum", name))
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
					*errs = append(*errs, fmt.Errorf("%s:%s: enum value already exists in original enum definition", name, oev.Name.Name))
				}
			}
		}
	case *ast.TypeSpec_Input:
		ogInput, ok := ogts.Type.(*ast.TypeSpec_Input)
		if !ok {
			*errs = append(*errs, fmt.Errorf("extend:input:%s: original type definition must be a input", name))
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
					*errs = append(*errs, fmt.Errorf("%s:%s: field definition already exists in original input definition", name, of.Name.Name))
				}
			}
		}
	default:
		*errs = append(*errs, fmt.Errorf("extend:%s: type extensions are not supported for this type", exts.Name.Name))
		return
	}

	// Any directives applied to extension must not already be applied to the original type
	validateDirectives(exts.Directives, loc, items, errs)
	for _, od := range ogts.Directives {
		for _, ed := range exts.Directives {
			if od.Name == ed.Name {
				*errs = append(*errs, fmt.Errorf("%s:%s: directive is already applied to original type definition", name, od.Name))
			}
		}
	}
}

// validateDirective validates a directive declaration
func validateDirective(name string, directive *ast.DirectiveType, items typeDecls, errs *[]error) {
	if directive.Args == nil {
		return
	}

	for _, f := range directive.Args.List {
		// 1. Check name of arg
		if strings.HasPrefix(f.Name.Name, "__") {
			*errs = append(*errs, fmt.Errorf("%s:%s: argument name cannot start with \"__\" (double underscore)", name, f.Name.Name))
		}

		// 2. Verify that the arg type is an input type
		var id *ast.Ident
		var valType interface{}
		switch v := f.Type.(type) {
		case *ast.InputValue_Ident:
			valType = v.Ident
			id = v.Ident
		case *ast.InputValue_List:
			valType = v.List
			id = unwrapType(v.List)
		case *ast.InputValue_NonNull:
			valType = v.NonNull
			id = unwrapType(v.NonNull)
		default:
			panic(fmt.Sprintf("spec: %s:%s: directive argument must have a type", name, f.Name.Name))
		}

		if !isInputType(id, items) {
			*errs = append(*errs, fmt.Errorf("%s:%s: directive argument must be a valid input type, not: %s", name, f.Name.Name, id.Name))
		}

		// 3. Validate any default value provided
		switch v := f.Default.(type) {
		case *ast.InputValue_BasicLit:
			validateValue(name, f.Name.Name, f, v.BasicLit, valType, items, errs)
		case *ast.InputValue_CompositeLit:
			validateValue(name, f.Name.Name, f, v.CompositeLit, valType, items, errs)
		}

		// 4. Check that the arg directives don't reference this one
		for _, d := range f.Directives {
			if d.Name == name {
				*errs = append(*errs, fmt.Errorf("%s:%s: directive argument cannont reference its own directive definition", name, f.Name.Name))
			}
		}

		if len(f.Directives) > 0 {
			validateDirectives(f.Directives, ast.DirectiveLocation_ARGUMENT_DEFINITION, items, errs)
		}

		// TODO: 5. Check that the arg Type doesn't reference this directive
	}
}

// validateArgs validates a list of args. host can either be
func validateArgs(host string, argDefs []*ast.InputValue, args []*ast.Arg, items typeDecls, errs *[]error) {
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
			*errs = append(*errs, fmt.Errorf("%s: arg must be unique in: %s", argDef.Name.Name, host))
			continue
		}
		delete(argMap, argDef.Name.Name)

		// Extract value and value type for arg
		var val, valType interface{}
		switch v := argDef.Type.(type) {
		case *ast.InputValue_Ident:
			valType = v.Ident
		case *ast.InputValue_List:
			valType = v.List
		case *ast.InputValue_NonNull:
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
			*errs = append(*errs, fmt.Errorf("%s: non-null arg must be present in: %s", argDef.Name.Name, host))
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
		*errs = append(*errs, fmt.Errorf("%s: undefined arg: %s", host, arg))
	}
}

// validateValue validates a value
func validateValue(host, cName string, c interface{}, val, valType interface{}, items typeDecls, errs *[]error) {
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
			panic("spec: validateValue can only be provided an ast.BasicLit or ast.CompositeLit val")
		}

		if cLit != nil {
			objLit, ok := cLit.Value.(*ast.CompositeLit_ObjLit)
			if !ok {
				*errs = append(*errs, fmt.Errorf("%s:%s: input object must be provided", host, cName))
				return
			}

			decls := items.lookup(u.Name)
			if decls == nil {
				*errs = append(*errs, fmt.Errorf("%s:%s: undefined input object: %s", host, cName, u.Name))
				return
			}

			decl := decls[0]

			objSpec, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
			if !ok {
				*errs = append(*errs, fmt.Errorf("%s:%s: could not find type spec for input object: %s", host, cName, u.Name))
				return
			}

			inputType, ok := objSpec.TypeSpec.Type.(*ast.TypeSpec_Input)
			if !ok {
				*errs = append(*errs, fmt.Errorf("%s:%s: %s is not an input object", host, cName, u.Name))
				return
			}

			validateObj(host, cName, inputType.Input.Fields.List, objLit.ObjLit.Fields, items, errs)
			return
		}

		// Coerce builtin scalar types
		switch u.Name {
		case "Int":
			if bLit.Kind != token.Token_INT {
				break
			}

			return
		case "Float":
			if bLit.Kind != token.Token_INT && bLit.Kind != token.Token_FLOAT {
				break
			}

			if bLit.Kind == token.Token_INT {
				bLit.Value += ".0"
			}

			bLit.Kind = token.Token_FLOAT
			return
		case "String":
			if bLit.Kind != token.Token_STRING {
				break
			}

			return
		case "Boolean":
			if bLit.Kind != token.Token_BOOL {
				break
			}

			return
		case "ID":
			if bLit.Kind != token.Token_STRING && bLit.Kind != token.Token_INT {
				break
			}

			return
		default:
			decls := items.lookup(u.Name)
			if decls == nil {
				break
			}

			decl := decls[0]

			ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
			if !ok {
				break
			}

			enum, ok := ts.TypeSpec.Type.(*ast.TypeSpec_Enum)
			if !ok {
				break
			}

			if enum.Enum.Values == nil {
				break
			}

			exists := true
			for _, eval := range enum.Enum.Values.List {
				if eval.Name.Name == bLit.Value {
					exists = false
				}
			}

			if !exists {
				return
			}

			*errs = append(*errs, fmt.Errorf("%s:%s: enum: %s has no value named: %s", host, cName, u.Name, bLit.Value))
			return
		}

		*errs = append(*errs, fmt.Errorf("%s:%s: %s is not coercible to: %s", host, cName, token.Token(bLit.Kind), u.Name))
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
		case *ast.InputValue:
			switch x := w.Default.(type) {
			case *ast.InputValue_BasicLit:
				listLit.List = &ast.ListLit_BasicList{
					BasicList: &ast.ListLit_Basic{
						Values: []*ast.BasicLit{x.BasicLit},
					},
				}
			case *ast.InputValue_CompositeLit:
				listLit.List = &ast.ListLit_CompositeList{
					CompositeList: &ast.ListLit_Composite{
						Values: []*ast.CompositeLit{x.CompositeLit},
					},
				}
			}

			w.Default = &ast.InputValue_CompositeLit{CompositeLit: &ast.CompositeLit{
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

			if bLit.Kind == token.Token_NULL {
				*errs = append(*errs, fmt.Errorf("%s:%s: non-null arg cannot be the null value", host, cName))
				return
			}
		case *ast.NonNull_List:
			valType = v.List
		}

		validateValue(host, cName, c, val, valType, items, errs)
	}
}

// validateObj validates an input value
func validateObj(host, arg string, fieldDefs []*ast.InputValue, objFields []*ast.ObjLit_Pair, items typeDecls, errs *[]error) {
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
			*errs = append(*errs, fmt.Errorf("%s:%s: field must be unique: %s", host, arg, fieldDef.Name.Name))
			continue
		}

		// Extract value and value type for arg
		var val, valType interface{}
		switch v := fieldDef.Type.(type) {
		case *ast.InputValue_Ident:
			valType = v.Ident
		case *ast.InputValue_List:
			valType = v.List
		case *ast.InputValue_NonNull:
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
			*errs = append(*errs, fmt.Errorf("%s: non-null field must be present in: %s", fieldDef.Name.Name, host))
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
		*errs = append(*errs, fmt.Errorf("%s:%s: undefined field: %s", host, arg, f.objField.Key.Name))
	}
}

// validateDirectives validates a list of applied directives
func validateDirectives(directives []*ast.DirectiveLit, loc ast.DirectiveLocation_Loc, items typeDecls, errs *[]error) {
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
		decls := items.lookup(name)
		if decls == nil {
			*errs = append(*errs, fmt.Errorf("%s: undefined directive", name))
			continue
		}

		decl := decls[0]

		dirSpec, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
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
			*errs = append(*errs, fmt.Errorf("%s: invalid location for directive: %s", name, loc))
			continue
		}

		// 3: Directives must be unique per location
		if d.count > 1 {
			*errs = append(*errs, fmt.Errorf("%s: directive cannot be applied more than once per location: %s", name, loc))
		}

		// 4: Directive arguments must be valid
		if dirType.Args == nil || d.dirLit.Args == nil {
			continue
		}
		validateArgs(name, dirType.Args.List, d.dirLit.Args.Args, items, errs)
	}
}

// checkName enforces that no Ident starts with "__" (two underscores).
func checkName(typ token.Token, name *ast.Ident, errs *[]error) {
	if !strings.HasPrefix(name.Name, "__") {
		return
	}

	*errs = append(*errs, fmt.Errorf("%s is an invalid name for type: %s", name.Name, typ))
}

func isInputType(id *ast.Ident, items typeDecls) bool {
	decls := items.lookup(id.Name)
	if decls == nil {
		return false
	}

	decl := decls[0]

	ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
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

func isOutputType(id *ast.Ident, items typeDecls) bool {
	decls := items.lookup(id.Name)
	if decls == nil {
		return false
	}

	decl := decls[0]

	ts, ok := decl.Spec.(*ast.TypeDecl_TypeSpec)
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

func unwrapType(i interface{}) *ast.Ident {
	switch v := i.(type) {
	case *ast.Ident:
		return v
	case *ast.List:
		switch u := v.Type.(type) {
		case *ast.List_Ident:
			return u.Ident
		case *ast.List_List:
			return unwrapType(u.List)
		case *ast.List_NonNull:
			return unwrapType(u.NonNull)
		}
	case *ast.NonNull:
		switch u := v.Type.(type) {
		case *ast.NonNull_Ident:
			return u.Ident
		case *ast.NonNull_List:
			return unwrapType(u.List)
		}
	}

	return nil
}
