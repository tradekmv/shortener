// Package noexit реализует анализатор, запрещающий использование os.Exit
// непосредственно в функции main пакета main.
//
// Этот анализатор предотвращает использование os.Exit в функции main,
// так как это препятствует корректному выполнению deferred функций,
// которые могут быть необходимы для освобождения ресурсов.
//
// Механизм работы:
//
// Анализатор обходит AST-дерево и ищет функцию main в пакете main.
// При обнаружении прямого вызова os.Exit в теле функции main
// (не внутри горутин, локальных функций или defer) выдаётся предупреждение.
//
// Запуск:
//
//	go run ./cmd/staticlint ./...
//
// При сборке:
//
//	go install ./cmd/staticlint
//	staticlint ./...
package noexit

import (
	"go/ast"
	"path/filepath"
	"strings"

	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer — экспортируемый анализатор для использования в multichecker.
var Analyzer = &analysis.Analyzer{
	Name:             "noexit",
	Doc:              "check that os.Exit is not called directly in main function of main package",
	URL:              "https://pkg.go.dev/github.com/tradekmv/shortener.git/cmd/staticlint/noexit",
	Requires:         []*analysis.Analyzer{inspect.Analyzer},
	RunDespiteErrors: true,
	Run:              run,
}

// isExitCall checks if the call is os.Exit using TypesInfo.
func isExitCall(call *ast.CallExpr, info *types.Info) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Use TypesInfo.Uses to get the actual object regardless of alias
	obj := info.Uses[ident]
	if obj == nil {
		return false
	}

	// Check if it's from os package
	if pkg := obj.Pkg(); pkg != nil && pkg.Path() == "os" && sel.Sel.Name == "Exit" {
		return true
	}

	return false
}

// findDirectExitCalls finds direct os.Exit calls in function body.
// Ignores calls inside goroutines, local functions, and defer.
func findDirectExitCalls(body *ast.BlockStmt, info *types.Info) bool {
	for _, stmt := range body.List {
		if hasDirectExit(stmt, info) {
			return true
		}
	}
	return false
}

// hasDirectExit checks for os.Exit in statement, ignoring nested functions and goroutines.
func hasDirectExit(stmt ast.Stmt, info *types.Info) bool {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok && isExitCall(call, info) {
			return true
		}

	case *ast.GoStmt:
		return false

	case *ast.DeferStmt:
		return false

	case *ast.AssignStmt:
		for _, expr := range s.Rhs {
			if hasExitInExpr(expr, info) {
				return true
			}
		}

	case *ast.IfStmt:
		if hasDirectExit(s.Body, info) {
			return true
		}
		if s.Else != nil {
			return hasDirectExit(s.Else, info)
		}

	case *ast.ForStmt:
		return hasDirectExit(s.Body, info)

	case *ast.RangeStmt:
		return hasDirectExit(s.Body, info)

	case *ast.BlockStmt:
		for _, inner := range s.List {
			if hasDirectExit(inner, info) {
				return true
			}
		}

	case *ast.LabeledStmt:
		return hasDirectExit(s.Stmt, info)
	}

	return false
}

// hasExitInExpr checks for os.Exit in expression.
func hasExitInExpr(expr ast.Expr, info *types.Info) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isExitCall(call, info)
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl := node.(*ast.FuncDecl)

		// Проверяем только функцию main в пакете main
		if pass.Pkg.Name() != "main" || funcDecl.Name.Name != "main" {
			return
		}

		// Пропускаем cmd/staticlint и его подпакеты (инструменты анализа)
		pkgPath := pass.Pkg.Path()
		if strings.HasPrefix(pkgPath, "github.com/tradekmv/shortener.git/cmd/staticlint") {
			return
		}

		// Также проверяем, что пакет находится в cmd/shortener или cmd/staticlint директории
		for _, f := range pass.Files {
			if f.Name != nil {
				pos := pass.Fset.Position(f.Pos())
				// Исключаем файлы из директории инструментов анализа
				dir := filepath.Dir(pos.Filename)
				if strings.Contains(dir, "/cmd/staticlint") || strings.Contains(dir, "\\cmd\\staticlint") {
					return
				}
				// Исключаем файлы из кэша сборки
				if strings.Contains(dir, ".cache") || strings.Contains(dir, ".build") {
					return
				}
				break
			}
		}

		if funcDecl.Body == nil {
			return
		}

		info := pass.TypesInfo

		if findDirectExitCalls(funcDecl.Body, info) {
			pass.Report(analysis.Diagnostic{
				Pos:     funcDecl.Pos(),
				Message: "os.Exit called directly in main function of main package",
			})
		}
	})

	return nil, nil
}
