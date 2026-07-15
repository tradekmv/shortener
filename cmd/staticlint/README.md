# Staticlint — Multichecker для статического анализа

Инструмент статического анализа, объединяющий стандартные анализаторы Go и собственный анализатор noexit.

## Запуск

```bash
# Установка и использование
go install ./cmd/staticlint
staticlint ./...

# Или запуск напрямую
go run ./cmd/staticlint ./...
```

## Включённые анализаторы

### golang.org/x/tools (33 анализатора)

| Анализатор | Назначение |
|------------|------------|
| asmdecl | Согласованность объявлений ассемблерных функций |
| assign | Неиспользуемые присваивания |
| atomic | Атомарные операции |
| bools | Булевы выражения |
| buildtag | Build-теги |
| cgocall | Вызовы cgo |
| composite | Составные литералы без ключей |
| copylock | Копирование мьютексов |
| ctrlflow | Анализ потока управления |
| deepequalerrors | deep.Equal с ошибками |
| errorsas | errors.As |
| findcall | Поиск вызовов функций |
| framepointer | Указатели кадров |
| httpresponse | HTTP-ответы |
| ifaceassert | Подавление ifaceassert |
| loopclosure | Замыкания в циклах |
| lostcancel | Отмена контекста |
| nilfunc | Сравнения с nil |
| nilness | Указатели на nil |
| pkgfact | Факты о пакетах |
| printf | Формат printf |
| shift | Сдвиг битов |
| sigchanyzer | Сигнальные каналы |
| slog | log/slog |
| stdmethods | Стандартные методы |
| stdversion | Версии Go |
| stringintconv | Строки в int |
| structtag | Теги структур |
| tests | Тесты |
| unmarshal | Unmarshal |
| unreachable | Недостижимый код |
| unsafeptr | unsafe.Pointer |
| unusedresult | Неиспользуемые результаты |
| usesgenerics | Дженерики |

### noexit (собственный анализатор)

**Назначение:** Запрещает `os.Exit` непосредственно в функции `main` пакета `main`.

**Почему:** `os.Exit` блокирует выполнение `defer`, что препятствует корректному освобождению ресурсов.

**Пример сообщения:**
```
cmd/shortener/main.go:85: os.Exit вызван непосредственно в функции main
```

**Исключения:** Горутины, локальные функции, defer.

## Дополнительные проверки

Для расширенного анализа (SA, ST, SC, QF):

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```
