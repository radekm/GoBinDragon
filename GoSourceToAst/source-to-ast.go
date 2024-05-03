package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"reflect"
)

func translateType(w io.Writer, t ast.Expr) {
	switch t := t.(type) {
	case *ast.Ident:
		fmt.Fprintf(w, `{"Case":"Ident","Fields":["%s"]}`, t.Name)
	case *ast.SelectorExpr:
		fmt.Fprintf(w, `{"Case":"Selector","Fields":[`)
		translateType(w, t.X)
		fmt.Printf(`,"%s"]}`, t.Sel.Name)
	case *ast.StarExpr:
		fmt.Fprintf(w, `{"Case":"Star","Fields":[`)
		translateType(w, t.X)
		fmt.Printf(`]}`)
	case *ast.FuncType:
		translateFuncType(w, t)
	case *ast.MapType:
		fmt.Fprintf(w, `{"Case":"Map","Fields":[`)
		translateType(w, t.Key)
		fmt.Fprintf(w, `,`)
		translateType(w, t.Value)
		fmt.Printf(`]}`)
	case *ast.ArrayType:
		fmt.Fprintf(w, `{"Case":"Array","Fields":[`)
		if t.Len == nil {
			fmt.Fprintf(w, "null,")
		} else {
			fmt.Fprintf(w, "%s,", t.Len.(*ast.BasicLit).Value)
		}
		translateType(w, t.Elt)
		fmt.Printf(`]}`)
	case *ast.StructType:
		firstField := true
		beforeField := func() {
			if !firstField {
				fmt.Fprintln(w, ",")
			}
			firstField = false
		}

		fmt.Fprintf(w, `{"Case":"Struct","Fields":[%t,[`, t.Incomplete)
		for _, field := range t.Fields.List {
			if len(field.Names) == 0 {
				log.Panicf("Structures with nameless field are not supported\n")
			}
			for _, name := range field.Names {
				beforeField()
				translateField(w, name.Name, field.Type)
			}
		}
		fmt.Printf(`]]}`)
	case *ast.InterfaceType:
		firstMethod := true
		beforeMethod := func() {
			if !firstMethod {
				fmt.Fprintln(w, ",")
			}
			firstMethod = false
		}

		fmt.Fprintf(w, `{"Case":"Interface","Fields":[%t,[`, t.Incomplete)
		for _, field := range t.Methods.List {
			if len(field.Names) == 0 {
				log.Panicf("Interfaces with nameless method are not supported\n")
			}
			for _, name := range field.Names {
				beforeMethod()
				translateField(w, name.Name, field.Type)
			}
		}
		fmt.Printf(`]]}`)
	case *ast.UnaryExpr:
		if t.Op == token.TILDE {
			log.Fatalf("Unsupported unary operator%v\n", t.Op)
		} else {
			log.Fatalf("Unknown unary operator%v\n", t.Op)
		}
	case *ast.BinaryExpr:
		if t.Op == token.OR {
			log.Fatalf("Unsupported binary operator%v\n", t.Op)
		} else {
			log.Fatalf("Unknown binary operator%v\n", t.Op)
		}
	default:
		log.Fatalf("Unknown type %v\n", reflect.TypeOf(t))
	}
}

func translateFuncType(w io.Writer, t *ast.FuncType) {
	if t.TypeParams != nil {
		fmt.Fprintf(os.Stderr, "Ignoring type parameters for function\n")
	}

	firstParam := true
	beforeParam := func() {
		if !firstParam {
			fmt.Fprintln(w, ",")
		}
		firstParam = false
	}
	fmt.Fprint(w, `{"Params":[`)
	for _, param := range t.Params.List {
		if len(param.Names) == 0 {
			log.Panicf("Unexpected function parameter without name\n")
		}
		for _, name := range param.Names {
			beforeParam()
			translateField(w, name.Name, param.Type)
		}
	}

	firstResult := true
	beforeResult := func() {
		if !firstResult {
			fmt.Fprintln(w, ",")
		}
		firstResult = false
	}
	fmt.Fprint(w, `],"Results":[`)
	if t.Results != nil {
		for _, result := range t.Results.List {
			if len(result.Names) == 0 {
				beforeResult()
				translateFieldWithoutName(w, result.Type)
			} else {
				for _, name := range result.Names {
					beforeResult()
					translateField(w, name.Name, result.Type)
				}
			}
		}
	}
	fmt.Fprint(w, `]}`)
}

func translateField(w io.Writer, name string, t ast.Expr) {
	fmt.Fprintf(w, `{"Name":"%s","Type":`, name)
	translateType(w, t)
	fmt.Fprint(w, `}`)
}

func translateFieldWithoutName(w io.Writer, t ast.Expr) {
	fmt.Fprintf(w, `{"Type":`)
	translateType(w, t)
	fmt.Fprint(w, `}`)
}

func translateDeclarations(w io.Writer, declarations []ast.Decl) {
	firstDeclaration := true
	beforeDeclaration := func() {
		if !firstDeclaration {
			fmt.Fprintln(w, ",")
		}
		firstDeclaration = false
	}

	fmt.Fprintln(w, "[")
	for _, decl := range declarations {

		switch decl := decl.(type) {
		case *ast.GenDecl:
			switch decl.Tok {
			case token.IMPORT:
				beforeDeclaration()
				fmt.Fprintf(w, `{"Case":"Import"}`)
			case token.TYPE:
				for _, spec := range decl.Specs {
					beforeDeclaration()
					spec := spec.(*ast.TypeSpec)
					fmt.Fprintf(w, `{"Case":"Type","Fields":["%s",`, spec.Name)
					if spec.TypeParams != nil {
						fmt.Fprintf(os.Stderr, "Ignoring type parameters for type %s\n", spec.Name)
					}
					translateType(w, spec.Type)
					fmt.Fprint(w, `]}`)
				}
			case token.CONST:
				for _, spec := range decl.Specs {
					beforeDeclaration()
					spec := spec.(*ast.ValueSpec)
					for _, name := range spec.Names {
						fmt.Fprintf(w, `{"Case":"Const","Fields":["%s"]}`, name)
					}
				}
			case token.VAR:
				for _, spec := range decl.Specs {
					beforeDeclaration()
					spec := spec.(*ast.ValueSpec)
					for _, name := range spec.Names {
						fmt.Fprintf(w, `{"Case":"Var","Fields":["%s"]}`, name)
					}
				}
			default:
				log.Panicf("Unexpected GenDecl with token %v\n", decl.Tok)
			}
		case *ast.FuncDecl:
			beforeDeclaration()
			fmt.Fprintf(w, `{"Case":"Func","Fields":["%s",`, decl.Name.Name)

			if decl.Recv != nil {
				if len(decl.Recv.List) != 1 || len(decl.Recv.List[0].Names) != 1 {
					log.Panicf("Function %s has unexpected number of receivers\n", decl.Name.Name)
				}
				field := decl.Recv.List[0]
				translateField(w, field.Names[0].Name, field.Type)
			} else {
				fmt.Fprintf(w, "null")
			}

			fmt.Fprint(w, ",")
			translateFuncType(w, decl.Type)
			fmt.Fprint(w, "]}")
		default:
			fmt.Fprintf(os.Stderr, "Ignoring declaration %v\n", reflect.TypeOf(decl))
		}
	}
	fmt.Fprintln(w, "\n]")
}

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		fmt.Fprintf(os.Stderr, "This program will create JSON with declarations from Go source file.\n")
		fmt.Fprintf(os.Stderr, "Expecting Go source file\n")
		os.Exit(1)
	}

	file, err := os.Open(argsWithoutProg[0])
	if err != nil {
		log.Fatalf("File not opened: %v\n", err)
	}
	defer file.Close()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", file, 0)
	if err != nil {
		log.Fatalf("Parsing error: %v\n", err)
	}

	translateDeclarations(os.Stdout, f.Decls)
}
