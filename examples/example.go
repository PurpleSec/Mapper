// Copyright (C) 2020 PurpleSec Team
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//

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
