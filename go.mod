module github.com/je4/HandleCreator/v2

go 1.17

replace github.com/je4/HandleCreator/v2 => ./
//replace github.com/je4/utils/v2 => ../utils/

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/je4/utils/v2 v2.0.3
	github.com/lib/pq v1.10.2
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
)

require (
	github.com/blend/go-sdk v1.20211025.3 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/machinebox/progress v0.2.0 // indirect
	github.com/pkg/sftp v1.13.4 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/sys v0.0.0-20211102061401-a2f17f7b995c // indirect
)
