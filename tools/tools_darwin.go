//go:build tools && !windows && !plan9
// +build tools,!windows,!plan9

package server

//go:generate go build -o ../bin/migrate -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate
//go:generate go build -o ../bin/air github.com/cosmtrek/air
//go:generate go build -o ../bin/sqlboiler github.com/volatiletech/sqlboiler/v4
//go:generate go build -o ../bin/sqlboiler-psql github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql

import (
	_ "github.com/cosmtrek/air"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/volatiletech/sqlboiler/v4"
	_ "github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql"
)
