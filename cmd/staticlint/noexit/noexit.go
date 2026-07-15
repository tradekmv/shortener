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

// isExitCall проверяет, является ли выражение вызовом os.Exit.
func isExitCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "os" && sel.Sel.Name == "Exit"
}

// findDirectExitCalls находит прямые вызовы os.Exit в теле функции.
// Игнорирует вызовы внутри горутин, локальных функций и defer.
func findDirectExitCalls(body *ast.BlockStmt) bool {
	for _, stmt := range body.List {
		if hasDirectExit(stmt) {
			return true
		}
	}
	return false
}

// hasDirectExit проверяет наличие os.Exit в операторе,
// игнорируя вложенные функции и горутины.
func hasDirectExit(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok && isExitCall(call) {
			return true
		}

	case *ast.GoStmt:
		return false

	case *ast.DeferStmt:
		return false

	case *ast.AssignStmt:
		for _, expr := range s.Rhs {
			if hasExitInExpr(expr) {
				return true
			}
		}

	case *ast.IfStmt:
		if hasDirectExit(s.Body) {
			return true
		}
		if s.Else != nil {
			return hasDirectExit(s.Else)
		}

	case *ast.ForStmt:
		return hasDirectExit(s.Body)

	case *ast.RangeStmt:
		return hasDirectExit(s.Body)

	case *ast.BlockStmt:
		for _, inner := range s.List {
			if hasDirectExit(inner) {
				return true
			}
		}

	case *ast.LabeledStmt:
		return hasDirectExit(s.Stmt)
	}

	return false
}

// hasExitInExpr проверяет наличие os.Exit в выражении.
func hasExitInExpr(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isExitCall(call)
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

		if findDirectExitCalls(funcDecl.Body) {
			pass.Report(analysis.Diagnostic{
				Pos:     funcDecl.Pos(),
				Message: "os.Exit called directly in main function of main package",
			})
		}
	})

	return nil, nil
}
