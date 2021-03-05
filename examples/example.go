// Copyright 2021 PurpleSec Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
