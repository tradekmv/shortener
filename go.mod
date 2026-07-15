module github.com/tradekmv/shortener.git

go 1.26.0

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/lib/pq v1.12.3
	github.com/rs/zerolog v1.35.1
	go.uber.org/mock v0.6.0
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/gostaticanalysis/comment v1.5.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools/go/expect v0.1.1-deprecated // indirect
)

require (
	github.com/gostaticanalysis/nilerr v0.1.2
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/tools v0.44.0
	honnef.co/go/tools v0.6.1
)

replace github.com/tradekmv/shortener.git => ./
