// Package main реализует multichecker — инструмент статического анализа,
// объединяющий множество анализаторов для комплексной проверки кода.
//
// Multichecker включает:
//
//   - Стандартные анализаторы пакета golang.org/x/tools/go/analysis/passes
//   - Все анализаторы класса SA (ошибки и баги) из staticcheck.io
//   - Анализаторы других классов staticcheck.io (ST — стиль, QF — исправления, simple)
//   - Публичные анализаторы: nilerr
//   - Собственные анализаторы: noexit
//
// Multichecker собирается и запускается следующим образом:
//
// Установка:
//
//	go install ./cmd/staticlint
//
// Запуск:
//
//	staticlint ./...
//
// Или без установки:
//
//	go run ./cmd/staticlint ./...
//
// Каждый анализатор выполняет определённую проверку:
//
// Стандартные анализаторы (golang.org/x/tools):
//   - asmdecl    — согласование объявлений ассемблерных функций
//   - assign     — неиспользуемые присваивания
//   - atomic     — корректность атомарных операций
//   - bools      — оптимизация булевых выражений
//   - buildtag   — корректность build-тегов
//   - cgocall    — вызовы cgo
//   - composite  — составные литералы без ключей
//   - copylock   — копирование мьютексов
//   - ctrlflow   — анализ потока управления
//   - deepequalerrors — deep.Equal с ошибками
//   - errorsas   — корректность errors.As
//   - findcall   — поиск вызовов функций
//   - framepointer — указатели кадров
//   - httpresponse — HTTP-ответы без закрытия тела
//   - ifaceassert — подавление ifaceassert
//   - loopclosure — замыкания в циклах
//   - lostcancel — отмена контекста
//   - nilfunc    — сравнения с nilfunc
//   - nilness    — указатели на nil
//   - pkgfact    — факты о пакетах
//   - printf     — формат printf
//   - shift      — сдвиг битов
//   - sigchanyzer — сигнальные каналы
//   - slog       — log/slog
//   - stdmethods — стандартные методы
//   - stdversion — версии Go
//   - stringintconv — строки в int
//   - structtag  — теги структур
//   - tests      — тесты
//   - unmarshal  — json.Unmarshal
//   - unreachable — недостижимый код
//   - unsafeptr  — unsafe.Pointer
//   - unusedresult — неиспользуемые результаты
//   - usesgenerics — дженерики
//
// Анализаторы staticcheck.io:
//   - SA (SA1xxx-SA9xxx) — ошибки и баги в коде
//   - ST (ST1xxx) — стилистические проверки
//   - QF (QF1xxx) — быстрые исправления
//   - simple (S1xxx) — упрощения кода
//
// Публичные анализаторы:
//   - nilerr    — проверка return nil при ошибке
//
// Собственные анализаторы:
//   - noexit    — запрет os.Exit в функции main
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"

	// Анализаторы staticcheck (SA — ошибки, ST — стиль, QF — исправления, simple)
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	// Публичные анализаторы
	"github.com/gostaticanalysis/nilerr" // проверка return nil при ошибке

	// Собственные анализаторы
	"github.com/tradekmv/shortener.git/cmd/staticlint/noexit"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// === Стандартные анализаторы golang.org/x/tools ===
		asmdecl.Analyzer,         // согласование объявлений ассемблерных функций
		assign.Analyzer,          // неиспользуемые присваивания
		atomic.Analyzer,          // атомарные операции
		bools.Analyzer,           // булевы выражения
		buildtag.Analyzer,        // build-теги
		cgocall.Analyzer,         // вызовы cgo
		composite.Analyzer,       // составные литералы без ключей
		copylock.Analyzer,        // копирование мьютексов
		ctrlflow.Analyzer,        // анализ потока управления
		deepequalerrors.Analyzer, // deep.Equal с ошибками
		errorsas.Analyzer,        // errors.As
		findcall.Analyzer,        // поиск вызовов функций
		framepointer.Analyzer,    // указатели кадров
		httpresponse.Analyzer,    // HTTP-ответы
		ifaceassert.Analyzer,     // подавление ifaceassert
		loopclosure.Analyzer,     // замыкания в циклах
		lostcancel.Analyzer,      // отмена контекста
		nilfunc.Analyzer,         // сравнения с nil
		nilness.Analyzer,         // указатели на nil
		pkgfact.Analyzer,         // факты о пакетах
		printf.Analyzer,          // формат printf
		shift.Analyzer,           // сдвиг битов
		sigchanyzer.Analyzer,     // сигнальные каналы
		slog.Analyzer,            // log/slog
		stdmethods.Analyzer,      // стандартные методы
		stdversion.Analyzer,      // версии Go
		stringintconv.Analyzer,   // строки в int
		structtag.Analyzer,       // теги структур
		tests.Analyzer,           // тесты
		unmarshal.Analyzer,       // unmarshal
		unreachable.Analyzer,     // недостижимый код
		unsafeptr.Analyzer,       // unsafe.Pointer
		unusedresult.Analyzer,    // неиспользуемые результаты
		usesgenerics.Analyzer,    // дженерики
	}

	// === Анализаторы класса SA из staticcheck ===
	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// === Анализаторы класса ST из staticcheck ===
	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// === Анализаторы класса QF из staticcheck ===
	for _, a := range quickfix.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// === Анализаторы simple из staticcheck (S1xxx) ===
	for _, a := range simple.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// === Публичные анализаторы ===
	analyzers = append(analyzers,
		nilerr.Analyzer, // проверка return nil при ошибке
	)

	// === Собственные анализаторы ===
	analyzers = append(analyzers, noexit.Analyzer) // запрет os.Exit в main

	multichecker.Main(analyzers...)
}
