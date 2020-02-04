package gobatis

import (
	"context"
	"database/sql"
	"errors"
)

type ResultType string

const (
	resultTypeMap     ResultType = "map"     // result set is a map: map[string]interface{}
	resultTypeMaps    ResultType = "maps"    // result set is a slice, item is map: []map[string]interface{}
	resultTypeStruct  ResultType = "struct"  // result set is a struct
	resultTypeStructs ResultType = "structs" // result set is a slice, item is struct
	resultTypeSlice   ResultType = "slice"   // result set is a value slice, []interface{}
	resultTypeSlices  ResultType = "slices"  // result set is a value slice, item is value slice, []interface{}
	resultTypeArray   ResultType = "array"   //
	resultTypeArrays  ResultType = "arrays"  // result set is a value slice, item is value slice, []interface{}
	resultTypeValue   ResultType = "value"   // result set is single value
)

type GoBatis interface {
	Select(stmt string, param interface{}) func(res interface{}) error
	Insert(stmt string, param interface{}) (int64, int64, error)
	Update(stmt string, param interface{}) (int64, error)
	Delete(stmt string, param interface{}) (int64, error)
}

// reference from https://github.com/yinshuwei/osm/blob/master/osm.go start
type dbRunner interface {
	Prepare(query string) (*sql.Stmt, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

func Get(datasource string) *DB {
	if nil == conf {
		panic(errors.New("DB config no init, please invoke DB.ConfInit() to init db config!"))
	}

	if nil == db {
		panic(errors.New("DB init err, db == nil!"))
	}

	ds, ok := db[datasource]
	if !ok {
		panic(errors.New("Datasource:" + datasource + " not exists!"))
	}

	dbType := ds.dbType
	if dbType != DBTypeMySQL {
		panic(errors.New("No support to this driver name!"))
	}

	gb := &DB{
		gbBase{
			db:     ds.db,
			dbType: ds.dbType,
			config: conf,
		},
	}

	return gb
}

type gbBase struct {
	db     dbRunner
	dbType DBType
	config *config
}

// DB
type DB struct {
	gbBase
}

// TX
type TX struct {
	gbBase
}

// Begin TX
//
// ps：
//  TX, err := this.Begin()
func (d *DB) Begin() (*TX, error) {
	if nil == d.db {
		return nil, errors.New("db no opened")
	}

	sqlDB, ok := d.db.(*sql.DB)
	if !ok {
		return nil, errors.New("db no opened")
	}

	db, err := sqlDB.Begin()
	if nil != err {
		return nil, err
	}

	t := &TX{
		gbBase{
			dbType: d.dbType,
			config: d.config,
			db:     db,
		},
	}
	return t, nil
}

// Begin TX with ctx & opts
//
// ps：
//  TX, err := this.BeginTx(ctx, ops)
func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TX, error) {
	if nil == d.db {
		return nil, errors.New("db no opened")
	}

	sqlDb, ok := d.db.(*sql.DB)
	if !ok {
		return nil, errors.New("db no opened")
	}

	db, err := sqlDb.BeginTx(ctx, opts)
	if nil != err {
		return nil, err
	}

	t := &TX{
		gbBase{
			dbType: d.dbType,
			config: d.config,
			db:     db,
		},
	}
	return t, nil
}

// Close db
//
// ps：
//  err := this.Close()
func (g *gbBase) Close() error {
	if nil == g.db {
		return errors.New("db no opened")
	}

	sqlDb, ok := g.db.(*sql.DB)
	if !ok {
		return errors.New("db no opened")
	}

	err := sqlDb.Close()
	g.db = nil
	return err
}

// Commit TX
//
// ps：
//  err := TX.Commit()
func (t *TX) Commit() error {
	if nil == t.db {
		return errors.New("TX no running")
	}

	sqlTx, ok := t.db.(*sql.Tx)
	if !ok {
		return errors.New("TX no running")

	}

	return sqlTx.Commit()
}

// Rollback TX
//
// ps：
//  err := TX.Rollback()
func (t *TX) Rollback() error {
	if nil == t.db {
		return errors.New("TX no running")
	}

	sqlTx, ok := t.db.(*sql.Tx)
	if !ok {
		return errors.New("TX no running")
	}

	return sqlTx.Rollback()
}

// reference from https://github.com/yinshuwei/osm/blob/master/osm.go end

func (g *gbBase) Select(stmt string, param interface{}) func(res interface{}) error {
	ms := g.config.mapperConf.getMappedStmt(stmt)
	if nil == ms {
		return func(res interface{}) error {
			return errors.New("Mapped statement not found:" + stmt)
		}
	}
	ms.dbType = g.dbType

	params := paramProcess(param)

	return func(res interface{}) error {
		executor := &executor{
			gb: g,
		}
		err := executor.query(ms, params, res)
		return err
	}
}

// insert(stmt string, param interface{})
func (g *gbBase) Insert(stmt string, param interface{}) (int64, int64, error) {
	ms := g.config.mapperConf.getMappedStmt(stmt)
	if nil == ms {
		return 0, 0, errors.New("Mapped statement not found:" + stmt)
	}
	ms.dbType = g.dbType

	params := paramProcess(param)

	executor := &executor{
		gb: g,
	}

	lastInsertId, affected, err := executor.update(ms, params)
	if nil != err {
		return 0, 0, err
	}

	return lastInsertId, affected, nil
}

// update(stmt string, param interface{})
func (g *gbBase) Update(stmt string, param interface{}) (int64, error) {
	ms := g.config.mapperConf.getMappedStmt(stmt)
	if nil == ms {
		return 0, errors.New("Mapped statement not found:" + stmt)
	}
	ms.dbType = g.dbType

	params := paramProcess(param)

	executor := &executor{
		gb: g,
	}

	_, affected, err := executor.update(ms, params)
	if nil != err {
		return 0, err
	}

	return affected, nil
}

// delete(stmt string, param interface{})
func (g *gbBase) Delete(stmt string, param interface{}) (int64, error) {
	return g.Update(stmt, param)
}
