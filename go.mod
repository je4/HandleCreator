module github.com/je4/HandleCreator/v2

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/je4/sshtunnel/v2 v2.0.0-20210324104725-ab38247e5ffa
	github.com/je4/utils/v2 v2.0.0-20210702125424-8c1cdd3f1ccc
	github.com/lib/pq v1.10.2
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
	golang.org/x/crypto v0.0.0-20210812204632-0ba0e8f03122 // indirect
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
)

replace github.com/je4/HandleCreator/v2 => ./
