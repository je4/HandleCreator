module github.com/je4/HandleCreator/v2

go 1.21

toolchain go1.22.0

replace github.com/je4/HandleCreator/v2 => ./

//replace github.com/je4/utils/v2 => ../utils/

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/go-sql-driver/mysql v1.7.1
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/je4/utils/v2 v2.0.23
	github.com/lib/pq v1.10.2
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.30.0
)

require (
	emperror.dev/errors v0.8.1 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
)
