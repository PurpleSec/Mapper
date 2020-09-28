/*
Mapper

Golang Database Statement Mapper for managing multiple prepared statements.

Mapper is a Go package that allows for mapping multiple prepared SQL statements with name "keys". This can
be used to manage resources and close all statements when your program closes.

Using Mapper is easy.


package main

import (
    "database/sql"
    "fmt"

    "github.com/PurpleSec/mapper"

    _ "github.com/mattn/go-sqlite3"
)

func main() {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        panic(err)
    }

    m := &mapper.Map{Database: db}
    defer m.Close()

    err = m.Add(
        "create_table",
            `CREATE TABLE IF NOT EXISTS Testing1 (
            TestID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
            TestName VARCHAR(64) NOT NULL UNIQUE
        )`,
    )
    if err != nil {
        panic(err)
    }

    if _, err := m.Exec("create_table"); err != nil {
        panic(err)
    }

    err = m.Extend(
        map[string]string{
            "insert": "INSERT INTO Testing1(TestName) VALUES(?)",
            "select": "SELECT TestName FROM Testing1 WHERE TestID = ?",
        },
    )
    if err != nil {
        panic(err)
    }

    r, err := m.Exec("insert", "Hello World :D!")
    if err != nil {
        panic(err)
    }

    a, err := r.RowsAffected()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Rows Affected: %d\n", a)

    q, err := m.Query("select", 1)
    if err != nil {
        panic(err)
    }

    var s string
    q.Next()
    if err := q.Scan(&s); err != nil {
        panic(err)
    }

    fmt.Printf("got %q\n", s)
    q.Close()
}


*/

package mapper
