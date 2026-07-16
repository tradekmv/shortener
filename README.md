# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этой репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Итерация 17: Бенчмарки и профилирование памяти

В проекте добавлены бенчмарки для ключевых компонентов сервиса сокращения URL.

### Запуск бенчмарков

```bash
# Бенчмарки для пакета service
go test -bench=. -benchmem ./internal/service/

# Бенчмарки для storage (включая Postgres)
go test -bench=. -benchmem ./internal/repository/storage/
```

### Снятие профиля памяти (pprof)

```bash
# Снять базовый профиль памяти
go test -bench=. -benchmem -memprofile=profiles/base.pprof -run=^$ ./internal/service/

# После оптимизаций снять итоговый профиль
go test -bench=. -benchmem -memprofile=profiles/result.pprof -run=^$ ./internal/service/

# Сравнить профили
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

### Результаты оптимизации

После оптимизации кода (устранение двойной аллокации в `SaveBatch`, таблица предвычислений для `generateID`) получено следующее снижение потребления памяти:

```
File: service.test
Type: alloc_space
Showing nodes accounting for -47.19MB, 4.20% of 1123.52MB total
flat  flat%   sum%        cum   cum%
-71.89MB  6.40%  6.40%   -71.89MB  6.40%  github.com/tradekmv/shortener.git/internal/repository/storage.(*MemoryStorage).SaveBatch
   16.71MB  1.49%  4.91%    16.71MB  1.49%  github.com/tradekmv/shortener.git/internal/repository/storage.(*MemoryStorage).Save
   12.50MB  1.11%  3.80%    12.50MB  1.11%  github.com/tradekmv/shortener.git/internal/service.generateID
    -5.50MB  0.49%  4.29%    -5.50MB  0.49%  internal/strconv.FormatInt
    -1.51MB  0.13%  4.16%   -71.40MB  6.35%  github.com/tradekmv/shortener.git/internal/service.(*Service).SaveBatch
```

**Главный результат:** `MemoryStorage.SaveBatch` — снижение потребления памяти на **71.89 MB** за счёт устранения двойной аллокации (storage больше не создаёт промежуточный слайс).

### Сравнение бенчмарков (до/после оптимизации)

| Бенчмарк | До (ns/op) | После (ns/op) | До (B/op) | После (B/op) |
|----------|-----------|---------------|-----------|--------------|
| `BenchmarkGenerateID` | 326.5 | 314.9 | 8 | 8 |
| `BenchmarkSave` | 766.5 | 770.9 | 165 | 164 |
| `BenchmarkSaveBatch` | 59755 | 67461 | 23117 | 23592 |
| `BenchmarkGet` | 19.53 | 19.66 | 0 | 0 |

### Покрытие тестами

По состоянию на iter17 покрытие тестами ключевых пакетов:

| Пакет | Покрытие |
|-------|----------|
| `internal/audit` | 83.7% |
| `internal/auth` | 83.3% |
| `internal/config` | 84.6% |
| `internal/handler` | 43.2% |
| `internal/middleware` | 96.9% |
| `internal/service` | 57.4% |
| `internal/repository/storage` | 27.5% |
