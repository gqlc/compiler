// Package doc contains a Documentation generator for GraphQL Documents.
package doc

import (
	"context"
	"github.com/gqlc/compiler"
	"github.com/gqlc/graphql/ast"
	"sort"
	"strings"
	"sync"
	"text/template"
)

// Generator generates Documentation for a GraphQL Document(s).
type Generator struct {
	sync.Once
}

// Generate generates documentation for the given document.
func (gen *Generator) Generate(ctx context.Context, doc *ast.Document, opts string) error {
	// Initialize templates here so they don't occur when doc gen isn't used
	gen.Do(func() {
		docTmpl.Funcs(map[string]interface{}{
			"add":     func(a, b int) int { return a + b },
			"Title":   strings.Title,
			"ToLower": strings.ToLower,
			"ToMembers": func(t ast.Expr) (mems []string) {
				unt := t.(*ast.UnionType)
				for _, mem := range unt.Members {
					mems = append(mems, mem.Name)
				}
				return
			},
			"ToFieldData": ToFieldData,
			"ToObjData":   ToObjData,
			"Trim":        func(s string) string { return strings.Trim(strings.TrimSpace(s), "\"") },
			"PrintDir": func(d *ast.DirectiveLit) string {
				var s strings.Builder
				s.WriteRune('@')
				s.WriteString(d.Name)

				if d.Args == nil {
					return s.String()
				} else if len(d.Args.Args) == 0 {
					return s.String()
				}

				s.WriteRune('(')
				aLen := len(d.Args.Args)
				for i, arg := range d.Args.Args {
					s.WriteString(arg.Name.Name)
					s.WriteString(": ")
					switch v := arg.Value.(type) {
					case *ast.BasicLit:
						s.WriteString(v.Value)
					case *ast.ListLit:
					case *ast.ObjLit:
					}

					if i < aLen-1 {
						s.WriteString(", ")
					}
				}
				s.WriteRune(')')
				return s.String()
			},
		})
		template.Must(docTmpl.Parse(mdTmpl))
		template.Must(docTmpl.New("objTmpl").Parse(objTmpl))
		template.Must(docTmpl.New("fieldListTmpl").Parse(fieldListTmpl))
	})

	// Extract types from doc into an mdData
	tmplData := extractTypes(doc)
	// TODO: Extract title from opts

	// Lexicographically sort types from document in mdData
	sort.Sort(tmplData.Scalars)
	sort.Sort(tmplData.Objects)
	sort.Sort(tmplData.Interfaces)
	sort.Sort(tmplData.Unions)
	sort.Sort(tmplData.Enums)
	sort.Sort(tmplData.Inputs)
	sort.Sort(tmplData.Directives)

	// Extract generator context
	gCtx := compiler.Context(ctx)
	if gCtx.Out == nil {
		panic("compiler: doc: nil output provided")
	}

	err := docTmpl.Execute(gCtx.Out, tmplData)
	if err != nil {
		return compiler.Error{
			DocName: doc.Name,
			GenName: "doc",
			Msg:     err.Error(),
		}
	}

	// TODO: Pass markdown source through html renderer if option is passed
	return nil
}

// GenerateAll generates documentation for all the given documents.
func (gen *Generator) GenerateAll(ctx context.Context, doc []*ast.Document, opts string) error {
	return nil
}

func extractTypes(doc *ast.Document) (tmplData *mdData) {
	tmplData = &mdData{
		Title: doc.Name,
	}

	for _, gd := range doc.Types {
		ts, ok := gd.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}

		switch ts.Type.(type) {
		case *ast.SchemaType:
			tmplData.Schema = ts
		case *ast.ScalarType:
			tmplData.Scalars = append(tmplData.Scalars, ts)
		case *ast.ObjectType:
			tmplData.Objects = append(tmplData.Objects, ts)
		case *ast.InterfaceType:
			tmplData.Interfaces = append(tmplData.Interfaces, ts)
		case *ast.UnionType:
			tmplData.Unions = append(tmplData.Unions, ts)
		case *ast.EnumType:
			tmplData.Enums = append(tmplData.Enums, ts)
		case *ast.InputType:
			tmplData.Inputs = append(tmplData.Inputs, ts)
		case *ast.DirectiveType:
			tmplData.Directives = append(tmplData.Directives, ts)
		}
	}

	// Remove schema ops from objects list
	if tmplData.Schema == nil {
		return
	}

	for _, op := range tmplData.Schema.Type.(*ast.SchemaType).Fields.List {
		for i, obj := range tmplData.Objects {
			if strings.ToLower(obj.Name.Name) != op.Name.Name {
				continue
			}

			tmplData.Objects = append(tmplData.Objects[:i], tmplData.Objects[i+1:]...)
			tmplData.RootTypes = append(tmplData.RootTypes, obj)
		}
	}
	return
}
