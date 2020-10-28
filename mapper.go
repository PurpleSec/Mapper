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

package mapper

import (
	"context"
	"database/sql"
	"sync"
)

// ErrInvalidDB is an error returned when the Database property of the Map is nil.
var ErrInvalidDB = &errval{s: "database cannot be nil"}

// Map is a struct that is used to track and manage multiple database *Stmt structs. Each statement can be mapped
// to a name that can be used again to recall or execute the statement.
//
// This struct is safe for multiple co-current goroutine usage.
type Map struct {
	Database *sql.DB

	lock    sync.RWMutex
	entries map[string]*sql.Stmt
}
type errval struct {
	e error
	s string
}

// Len returns the size of the internal mapping.
func (m *Map) Len() int {
	if m.entries == nil {
		return 0
	}
	return len(m.entries)
}

// Close will attempt to close all the contained database statements. This will bail on any errors that occur.
// Multiple calls to close can be used to make sure that all statements are closed successfully. Note: this will
// also attempt to close the connected database if all statement closures are successful.
func (m *Map) Close() error {
	var err error
	m.lock.Lock()
	if m.entries != nil {
		for k, v := range m.entries {
			if v == nil {
				continue
			}
			if err = v.Close(); err != nil {
				err = &errval{e: err, s: `error closing mapping "` + k + `"`}
				break
			}
			m.entries[k] = nil
		}
	}
	if m.lock.Unlock(); err != nil {
		return err
	}
	return m.Database.Close()
}
func (e errval) Error() string {
	if e.e == nil {
		return e.s
	}
	return e.s + ": " + e.e.Error()
}
func (e errval) Unwrap() error {
	return e.e
}

// Remove will attempt to remove the statement with the provided name. This function will return True if the
// mapping was found and removed. Otherwise the function will return false.
func (m *Map) Remove(name string) bool {
	if m.entries == nil {
		return false
	}
	s, ok := m.entries[name]
	if !ok {
		return false
	}
	m.lock.Lock()
	s.Close()
	delete(m.entries, name)
	m.lock.Unlock()
	return true
}

// Contains returns True if the name provided has an associated statement.
func (m *Map) Contains(name string) bool {
	if len(m.entries) == 0 {
		return false
	}
	m.lock.RLock()
	s, ok := m.entries[name]
	m.lock.RUnlock()
	return ok && s != nil
}

// Add will prepare and add the specified query to the Map with the provided name. This will only add the mapping
// if the 'Prepare' function is successful. Otherwise the prepare error will be returned. This function does not
// allow for adding a mapping when one already exists. If a mapping with an overlapping name is attempted, an
// error will be returned before attempting to prepare the query.
func (m *Map) Add(name, query string) error {
	return m.AddContext(context.Background(), name, query)
}

// Batch is a function that can be used to perform execute statements in a specific order. This function will
// execute all the stataments in the provided string array and will stop and return any errors that occur. The
// passed query results will not be returned or parsed.
func (m *Map) Batch(queries []string) error {
	return m.BatchContext(context.Background(), queries)
}

// Get will attempt to return the statement that is associated with the provided name. This function will return the
// statement and True if the mapping exists. Otherwise, the statement will be nil and the boolean will be False.
func (m *Map) Get(name string) (*sql.Stmt, bool) {
	if len(m.entries) == 0 {
		return nil, false
	}
	m.lock.RLock()
	s, ok := m.entries[name]
	m.lock.RUnlock()
	return s, ok
}

// Extend will prepare and add all the specified queries in the provided map to the Map. This will only add each
// mapping if the 'Prepare' function is successful. Otherwise the prepare error will be returned. This function does
// not allow for adding a mapping when one already exists. If a mapping with an overlapping name is attempted, an
// error will be returned before attempting to prepare the query.
func (m *Map) Extend(data map[string]string) error {
	return m.ExtendContext(context.Background(), data)
}

// AddContext will prepare and add the specified query to the Map with the provided name. This will only add the
// mapping if the 'Prepare' function is successful. Otherwise the prepare error will be returned. This function does
// not allow for adding a mapping when one already exists. If a mapping with an overlapping name is attempted, an
// error will be returned before attempting to prepare the query. This function specifies a Context that can be used
// to interrupt and cancel the prepare calls.
func (m *Map) AddContext(x context.Context, name, query string) error {
	if m.Database == nil {
		return ErrInvalidDB
	}
	if m.entries == nil {
		m.entries = make(map[string]*sql.Stmt, 1)
	}
	m.lock.Lock()
	if s, ok := m.entries[name]; ok && s != nil {
		m.lock.Unlock()
		return &errval{s: `statement with name "` + name + `" already exists`}
	}
	s, err := m.Database.PrepareContext(x, query)
	if err == nil {
		m.entries[name] = s
	} else {
		err = &errval{e: err, s: `error adding mapping "` + name + `"`}
	}
	m.lock.Unlock()
	return err
}

// BatchContext is a function that can be used to perform execute statements in a specific order. This function will
// execute all the stataments in the provided string array and will stop and return any errors that occur. The
// passed query results will not be returned or parsed. This function specifies a Context that can be used to
// interrupt and cancel the execute calls.
func (m *Map) BatchContext(x context.Context, queries []string) error {
	if len(queries) == 0 {
		return nil
	}
	if m.Database == nil {
		return ErrInvalidDB
	}
	var err error
	m.lock.Lock()
	for i := range queries {
		select {
		case <-x.Done():
			err = x.Err()
		default:
		}
		if err != nil {
			break
		}
		if _, err = m.Database.ExecContext(x, queries[i]); err != nil {
			err = &errval{e: err, s: `error executing Init statement mapping "` + queries[i] + `"`}
			break
		}
	}
	m.lock.Unlock()
	return err
}

// Exec will attempt to get the statement with the provided name and then attempt to call the 'Exec' function on
// the statement. This provides the results of the Exec function.
func (m *Map) Exec(name string, args ...interface{}) (sql.Result, error) {
	return m.ExecContext(context.Background(), name, args...)
}

// Query will attempt to get the statement with the provided name and then attempt to call the 'Query' function on
// the statement. This provides the results of the Query function.
func (m *Map) Query(name string, args ...interface{}) (*sql.Rows, error) {
	return m.QueryContext(context.Background(), name, args...)
}

// QueryRow will attempt to get the statement with the provided name and then attempt to call the 'QueryRow' function
// on the statement. This function differs from the original 'QueryRow' statement as this provides a boolean to
// indicate if the provided named statement was found. If the returned boolean is True, the result is not-nil and
// safe to use.
func (m *Map) QueryRow(name string, args ...interface{}) (*sql.Row, bool) {
	return m.QueryRowContext(context.Background(), name, args...)
}

// ExtendContext will prepare and add all the specified queries in the provided map to the Map. This will only add
// each mapping if the 'Prepare' function is successful. Otherwise the prepare error will be returned. This function
// does not allow for adding a mapping when one already exists. If a mapping with an overlapping name is attempted, an
// error will be returned before attempting to prepare the query. This function specifies a Context that can be used
// to interrupt and cancel the prepare calls.
func (m *Map) ExtendContext(x context.Context, data map[string]string) error {
	if data == nil {
		return nil
	}
	if m.Database == nil {
		return ErrInvalidDB
	}
	if m.entries == nil {
		m.entries = make(map[string]*sql.Stmt, len(data))
	}
	var (
		s   *sql.Stmt
		err error
	)
	m.lock.Lock()
	for k, v := range data {
		select {
		case <-x.Done():
			err = x.Err()
		default:
		}
		if err != nil {
			break
		}
		if s, ok := m.entries[k]; ok && s != nil {
			err = &errval{s: `statement with name "` + k + `" already exists`}
			break
		}
		if s, err = m.Database.PrepareContext(x, v); err != nil {
			err = &errval{e: err, s: `error adding mapping "` + k + `"`}
			break
		}
		m.entries[k] = s
	}
	m.lock.Unlock()
	return err
}

// ExecContext will attempt to get the statement with the provided name and then attempt to call the 'Exec' function
// on the statement. This provides the results of the Exec function. This provides the results of the Exec function.
// This function specifies a Context that can be used to interrupt and cancel the Exec function.
func (m *Map) ExecContext(x context.Context, name string, args ...interface{}) (sql.Result, error) {
	if m.Database == nil {
		return nil, ErrInvalidDB
	}
	if len(m.entries) == 0 {
		return nil, &errval{s: `statement with name "` + name + `" does not exist`}
	}
	m.lock.RLock()
	s, ok := m.entries[name]
	if m.lock.RUnlock(); !ok || s == nil {
		return nil, &errval{s: `statement with name "` + name + `" does not exist`}
	}
	return s.ExecContext(x, args...)
}

// QueryContext will attempt to get the statement with the provided name and then attempt to call the 'Query'
// function on the statement. This provides the results of the Query function. This function specifies a Context
// that can be used to interrupt and cancel the Query function.
func (m *Map) QueryContext(x context.Context, name string, args ...interface{}) (*sql.Rows, error) {
	if m.Database == nil {
		return nil, ErrInvalidDB
	}
	if len(m.entries) == 0 {
		return nil, &errval{s: `statement with name "` + name + `" does not exist`}
	}
	m.lock.RLock()
	s, ok := m.entries[name]
	if m.lock.RUnlock(); !ok || s == nil {
		return nil, &errval{s: `statement with name "` + name + `" does not exist`}
	}
	return s.QueryContext(x, args...)
}

// QueryRowContext will attempt to get the statement with the provided name and then attempt to call the 'QueryRow'
// function on the statement. This function differs from the original 'QueryRow' statement as this provides a boolean
// to indicate if the provided named statement was found. If the returned boolean is True, the result is not-nil and
// safe to use. This function specifies a Context that can be used to interrupt and cancel the Query function.
func (m *Map) QueryRowContext(x context.Context, name string, args ...interface{}) (*sql.Row, bool) {
	if m.Database == nil {
		return nil, false
	}
	if len(m.entries) == 0 {
		return nil, false
	}
	m.lock.RLock()
	s, ok := m.entries[name]
	if m.lock.RUnlock(); !ok || s == nil {
		return nil, false
	}
	return s.QueryRowContext(x, args...), true
}
