// Package doc contains a Documentation generator for GraphQL Documents.
package doc

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gqlc/compiler"
	"github.com/gqlc/graphql/ast"
	"gitlab.com/golang-commonmark/markdown"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
)

// Generator generates Documentation for a GraphQL Document(s).
type Generator struct {
	sync.Once
	md *markdown.Markdown
}

func (gen *Generator) initTmpls() {
	gen.md = markdown.New()
	docTmpl.Funcs(map[string]interface{}{
		"add":         func(a, b int) int { return a + b },
		"Title":       strings.Title,
		"ToLower":     strings.ToLower,
		"ToFieldData": ToFieldData,
		"ToObjData":   ToObjData,
		"ToMembers": func(i interface{}) []*ast.Ident {
			return i.(*ast.TypeSpec_Union).Union.Members
		},
		"Trim": func(s string) string { return strings.Trim(strings.TrimSpace(s), "\"") },
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
				case *ast.Arg_BasicLit:
					s.WriteString(v.BasicLit.Value)
				case *ast.Arg_CompositeLit:
					// TODO: Print composite literal
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
}

// Generate generates documentation for the given document.
func (gen *Generator) Generate(ctx context.Context, doc *ast.Document, opts string) (err error) {
	defer func() {
		if err != nil {
			err = compiler.Error{
				DocName: doc.Name,
				GenName: "doc",
				Msg:     err.Error(),
			}
		}
	}()

	// Initialize templates here so they don't occur when doc gen isn't used
	gen.Do(gen.initTmpls)

	// Unmarshal options
	var optData struct {
		Title string `json:"title"`
		HTML  bool   `json:"html"`
	}
	if len(opts) > 0 {
		err = json.Unmarshal(json.RawMessage(opts), &optData)
		if err != nil {
			return
		}
	}

	// Extract types from doc into an mdData
	tmplData := extractTypes(doc)
	if optData.Title != "" {
		tmplData.Title = optData.Title
	}

	// Lexicographically sort types from document in mdData
	sort.Sort(tmplData.Scalars)
	sort.Sort(tmplData.Objects)
	sort.Sort(tmplData.Interfaces)
	sort.Sort(tmplData.Unions)
	sort.Sort(tmplData.Enums)
	sort.Sort(tmplData.Inputs)
	sort.Sort(tmplData.Directives)

	// Generate Markdown
	var b bytes.Buffer
	err = docTmpl.Execute(&b, tmplData)
	if err != nil {
		return
	}

	// Extract generator context
	gCtx := compiler.Context(ctx)

	// Open file to write markdown to
	base := doc.Name[:len(doc.Name)-len(filepath.Ext(doc.Name))]
	mdFile, err := gCtx.Open(base + ".md")
	defer mdFile.Close()
	if err != nil {
		return
	}

	// Check for HTML option
	if !optData.HTML {
		_, err = io.Copy(mdFile, &b)
		return
	}

	// Write markdown but make sure to keep bytes for HTML rendering
	_, err = io.Copy(mdFile, bytes.NewReader(b.Bytes()))
	if err != nil {
		return
	}

	// Open HTML file
	htmlFile, err := gCtx.Open(base + ".html")
	defer htmlFile.Close()
	if err != nil {
		return
	}

	err = gen.md.Render(htmlFile, b.Bytes())
	return
}

func extractTypes(doc *ast.Document) (tmplData *mdData) {
	tmplData = &mdData{
		DocName: doc.Name[:len(doc.Name)-len(filepath.Ext(doc.Name))],
		Title:   doc.Name[:len(doc.Name)-len(filepath.Ext(doc.Name))],
	}

	for _, gd := range doc.Types {
		ts, ok := gd.Spec.(*ast.TypeDecl_TypeSpec)
		if !ok {
			continue
		}

		switch ts.TypeSpec.Type.(type) {
		case *ast.TypeSpec_Schema:
			tmplData.Schema = ts.TypeSpec
		case *ast.TypeSpec_Scalar:
			tmplData.Scalars = append(tmplData.Scalars, ts.TypeSpec)
		case *ast.TypeSpec_Object:
			tmplData.Objects = append(tmplData.Objects, ts.TypeSpec)
		case *ast.TypeSpec_Interface:
			tmplData.Interfaces = append(tmplData.Interfaces, ts.TypeSpec)
		case *ast.TypeSpec_Union:
			tmplData.Unions = append(tmplData.Unions, ts.TypeSpec)
		case *ast.TypeSpec_Enum:
			tmplData.Enums = append(tmplData.Enums, ts.TypeSpec)
		case *ast.TypeSpec_Input:
			tmplData.Inputs = append(tmplData.Inputs, ts.TypeSpec)
		case *ast.TypeSpec_Directive:
			tmplData.Directives = append(tmplData.Directives, ts.TypeSpec)
		}
	}

	// Remove schema ops from objects list
	if tmplData.Schema == nil {
		return
	}

	for _, op := range tmplData.Schema.Type.(*ast.TypeSpec_Schema).Schema.RootOps.List {
		for i, obj := range tmplData.Objects {
			name := obj.Name.Name
			if strings.ToLower(name) != op.Name.Name {
				continue
			}

			tmplData.Objects = append(tmplData.Objects[:i], tmplData.Objects[i+1:]...)
			tmplData.RootTypes = append(tmplData.RootTypes, obj)
		}
	}
	return
}
