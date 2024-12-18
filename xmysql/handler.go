package xmysql

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/newbiediver/golib/exception"
	"github.com/newbiediver/golib/scheduler"
)

/*
## 사용법 ##
1. 연결
func setupDatabaseServer() {
	e := conf.Get()
	newHandler, err := xmysql.NewHandler(e.DataCenter.Server, e.DataCenter.UID, e.DataCenter.PWD, e.DataCenter.DataSource, e.DataCenter.Port, e.DataCenter.Count)
	if err != nil {
		panic(err)
	}

	xmysql.KeepHandler("main", newHandler)
}

2. 쿼리
func testCallDB() {
	handler := xmysql.GetHandler("main")

	newQuery := xmysql.QueryExecutor{
		SqlString: "SELECT * FROM someTable WHERE sid = 1;",
		OnQuery: func(rs *xmysql.RecordSet) error {
			if rs.NextRow() {
				var (
					index    int
					str      string
					floating float64
				)
				if err := rs.Scan(&index, &str, &floating); err != nil {
					return err
				}

				fmt.Printf("[Row %d] str: %s, floating: %f\n", index, str, floating)
			}

			return nil
		},
		OnError: func(err error) {
			if err != nil {
				fmt.Printf("TestCallError: %s\n", err)
			}
		},
	}

	handler.Query(&newQuery)
}

3. 트랜잭션
func testCallDB() {
	handler := xmysql.GetHandler("main")

	newQuery1 := xmysql.QueryExecutor{
		SqlString: "INSERT INTO (str1, float2) VALUES ('테스트', 1283971.1232);",
		OnQuery: func(rs *xmysql.RecordSet) error {
			return nil
		},
		OnError: func(err error) {
			if err != nil {
				fmt.Printf("TestCallError: %s\n", err)
			}
		},
	}

	newQuery2 := xmysql.QueryExecutor{
		SqlString: "INSERT INTO (str1, float2) VALUES ('테스트2', 891723.23);",
		OnQuery: func(rs *xmysql.RecordSet) error {
			return errors.New("just error")		// 트랜잭션에서 error 를 리턴하면 롤백됨
		},
		OnError: func(err error) {
			if err != nil {
				fmt.Printf("TestCallError: %s\n", err)
			}
		},
	}

	handler.Transaction(func() {
		fmt.Println("Committed")
	}, func(err error) {
		fmt.Printf("Rollbacked. Err: %s", err.Error())
	}, &newQuery1, &newQuery2)
}
*/

type Handler struct {
	sqlHandler *sql.DB
}

type RecordSet struct {
	curRows *sql.Rows
}

type ExecCallback func(int64, int64)
type QueryCallback func(*RecordSet) error
type ErrorCallback func(error)
type CommitCallback func()
type RollbackCallback func(error)

type QueryExecutor struct {
	SqlString string
	OnQuery   QueryCallback
	OnError   ErrorCallback
}

var (
	managedHandlers  map[string]*Handler
	backgroundObject *scheduler.Handler
)

// NewHandler 새연결
func NewHandler(server, uid, pwd, source string, port, io int) (*Handler, error) {
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&multiStatements=true&parseTime=true",
		uid, pwd, server, port, source)

	dbHandler, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	dbHandler.SetMaxIdleConns(io)
	err = dbHandler.Ping()
	if err != nil {
		return nil, err
	}

	newHandler := new(Handler)
	newHandler.sqlHandler = dbHandler

	if backgroundObject == nil {
		backgroundObject = new(scheduler.Handler)
		backgroundObject.Run(scheduler.PriorityVerySlow)

		obj := scheduler.CreateObjectByInterval(30000, func() {
			if err := newHandler.sqlHandler.Ping(); err != nil {
				fmt.Printf("SQL connections look like disconnected: %s", err.Error())
			}
		})

		backgroundObject.NewObject(obj)
	}

	return newHandler, nil
}

// KeepHandler 연결을 보관하고자 할 때
func KeepHandler(name string, newHandler *Handler) {
	if managedHandlers == nil {
		managedHandlers = make(map[string]*Handler)
	}
	managedHandlers[name] = newHandler
}

func FlushHandlers() {
	backgroundObject.Stop()
	for _, handler := range managedHandlers {
		_ = handler.sqlHandler.Close()
	}
}

func GetHandler(name string) *Handler {
	return managedHandlers[name]
}

func (s *Handler) SyncQuery(sqlString string) (*RecordSet, error) {
	result := RecordSet{}
	rows, err := s.sqlHandler.Query(sqlString)
	if err != nil {
		return nil, err
	}

	if rows != nil {
		result.curRows = rows
	}

	return &result, nil
}

func (s *Handler) Execute(queryString string, execCallback ExecCallback, errCallback ErrorCallback) {
	go func() {
		defer func() {
			if rcv := recover(); rcv != nil {
				if ex := exception.GetExceptionHandler(); ex != nil {
					ex.ExceptionCallbackFunctor()
				}
			}
		}()

		r, err := s.sqlHandler.Exec(queryString)
		if err != nil {
			errCallback(err)
			return
		}

		if execCallback != nil {
			affected, _ := r.RowsAffected()
			lastInsertedID, _ := r.LastInsertId()
			execCallback(affected, lastInsertedID)
		}
	}()

}

// Query 단일 쿼리 호출
func (s *Handler) Query(executor *QueryExecutor) {
	go func() {
		result := RecordSet{}
		rows, err := s.sqlHandler.Query(executor.SqlString)

		defer func() {
			if rows != nil {
				_ = rows.Close()
			}
			if r := recover(); r != nil {
				switch r.(type) {
				case string:
					executor.OnError(errors.New(r.(string)))
				case error:
					executor.OnError(r.(error))
				default:
					executor.OnError(errors.New("unknown error"))
				}

				if ex := exception.GetExceptionHandler(); ex != nil {
					ex.ExceptionCallbackFunctor()
				}
			}
		}()

		if err != nil && executor.OnError != nil {
			executor.OnError(err)
			return
		}

		if rows != nil {
			result.curRows = rows
		}

		err = executor.OnQuery(&result)

		if err != nil {
			if executor.OnError != nil {
				executor.OnError(err)
			}
		}
	}()
}

// Transaction 트랜잭션을 위한 쿼리 호출 (주의: procedure 를 사용할 경우 procedure 내에서 transaction 처리를 하면 안됨)
func (s *Handler) Transaction(onCommit CommitCallback, onRollback RollbackCallback, queries ...*QueryExecutor) {
	if onCommit == nil || onRollback == nil {
		panic("[DBTransaction] Commit or Rollback callback is nil")
	}
	go func() {
		_, _ = s.sqlHandler.Exec("SET AUTOCOMMIT = 0;")
		defer func() {
			_, _ = s.sqlHandler.Exec("SET AUTOCOMMIT = 1;")
		}()

		tx, err := s.sqlHandler.Begin()
		if err != nil {
			panic(err)
		}

		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback()

				switch r.(type) {
				case string:
					onRollback(errors.New(r.(string)))
				case error:
					onRollback(r.(error))
				default:
					onRollback(errors.New("unknown error"))
				}
				fmt.Println(r)
			}
		}()

		for _, executor := range queries {
			result := RecordSet{}
			rows, err := tx.Query(executor.SqlString)
			if err != nil {
				if executor.OnError != nil {
					executor.OnError(err)
				}
				panic(err)
			}

			if rows != nil {
				result.curRows = rows
			}

			err = executor.OnQuery(&result)

			if rows != nil {
				_ = rows.Close()
			}

			if err != nil {
				if executor.OnError != nil {
					executor.OnError(err)
				}
				panic(err)
			}
		}

		_ = tx.Commit()
		onCommit()
	}()
}

func (rs *RecordSet) NextRow() bool {
	if rs.curRows == nil {
		return false
	}
	return rs.curRows.Next()
}

func (rs *RecordSet) NextResultSet() bool {
	if rs.curRows == nil {
		return false
	}
	return rs.curRows.NextResultSet()
}

func (rs *RecordSet) Scan(fields ...any) error {
	if rs.curRows == nil {
		return errors.New("no recordset data")
	}
	return rs.curRows.Scan(fields...)
}

func (rs *RecordSet) Close() {
	if rs.curRows != nil {
		_ = rs.curRows.Close()
	}
}
