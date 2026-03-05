// Archivo main.go
package main

import (
	"database/sql"
	"fmt"
	"net"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "file:Movies.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	fmt.Println("Listening on :8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go get(conn, db)
	}
}
