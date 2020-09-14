// Copyright 2020 Douyu
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gorm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/douyu/jupiter/pkg/metric"

	"github.com/douyu/jupiter/pkg/trace"
	"github.com/douyu/jupiter/pkg/util/xcolor"
	"github.com/douyu/jupiter/pkg/xlog"
)

// Handler ...
type Handler func(db *DB)

// Interceptor ...
type Interceptor func(*DSN, string, *Config) func(next Handler) Handler

func debugInterceptor(dsn *DSN, op string, options *Config) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(scope *DB) {
			fmt.Printf("%-50s[%s] => %s\n", xcolor.Green(dsn.Addr+"/"+dsn.DBName), time.Now().Format("04:05.000"), xcolor.Green("Send: "+logSQL(scope.Statement.SQL.String(), scope.Statement.Vars, true)))
			next(scope)
			if scope.Error != nil {
				fmt.Printf("%-50s[%s] => %s\n", xcolor.Red(dsn.Addr+"/"+dsn.DBName), time.Now().Format("04:05.000"), xcolor.Red("Erro: "+scope.Error.Error()))
			} else {
				fmt.Printf("%-50s[%s] => %s\n", xcolor.Green(dsn.Addr+"/"+dsn.DBName), time.Now().Format("04:05.000"), xcolor.Green("Affected: "+strconv.Itoa(int(scope.RowsAffected))))
			}
		}
	}
}

func metricInterceptor(dsn *DSN, op string, options *Config) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(scope *DB) {
			beg := time.Now()
			next(scope)
			cost := time.Since(beg)

			// error metric
			if scope.Error != nil {
				metric.LibHandleCounter.WithLabelValues(metric.TypeGorm, dsn.DBName+"."+scope.Statement.Table, dsn.Addr, "ERR").Inc()
				// todo sql语句，需要转换成脱密状态才能记录到日志
				if errors.Is(scope.Error, ErrRecordNotFound) {
					options.logger.Error("mysql err", xlog.FieldErr(scope.Error), xlog.FieldName(dsn.DBName+"."+scope.Statement.Table), xlog.FieldMethod(op))
				} else {
					options.logger.Warn("record not found", xlog.FieldErr(scope.Error), xlog.FieldName(dsn.DBName+"."+scope.Statement.Table), xlog.FieldMethod(op))
				}
			} else {
				metric.LibHandleCounter.Inc(metric.TypeGorm, dsn.DBName+"."+scope.Statement.Table, dsn.Addr, "OK")
			}

			metric.LibHandleHistogram.WithLabelValues(metric.TypeGorm, dsn.DBName+"."+scope.Statement.Table, dsn.Addr).Observe(cost.Seconds())

			if options.SlowThreshold > time.Duration(0) && options.SlowThreshold < cost {
				options.logger.Error(
					"slow",
					xlog.FieldErr(errSlowCommand),
					xlog.FieldMethod(op),
					xlog.FieldExtMessage(logSQL(scope.Statement.SQL.String(), scope.Statement.Vars, options.DetailSQL)),
					xlog.FieldAddr(dsn.Addr),
					xlog.FieldName(dsn.DBName+"."+scope.Statement.Table),
					xlog.FieldCost(cost),
				)
			}
		}
	}
}

func logSQL(sql string, args []interface{}, containArgs bool) string {
	if containArgs {
		return bindSQL(sql, args)
	}
	return sql
}

func traceInterceptor(dsn *DSN, op string, options *Config) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(scope *DB) {
			if val, ok := scope.InstanceGet("_context"); ok {
				if ctx, ok := val.(context.Context); ok {
					span, _ := trace.StartSpanFromContext(
						ctx,
						"GORM", // TODO this op value is op or GORM
						trace.TagComponent("mysql"),
						trace.TagSpanKind("client"),
					)
					defer span.Finish()

					// 延迟执行 scope.CombinedConditionSql() 避免sqlVar被重复追加
					next(scope)

					span.SetTag("sql.inner", dsn.DBName)
					span.SetTag("sql.addr", dsn.Addr)
					span.SetTag("span.kind", "client")
					span.SetTag("peer.service", "mysql")
					span.SetTag("db.instance", dsn.DBName)
					span.SetTag("peer.address", dsn.Addr)
					span.SetTag("peer.statement", logSQL(scope.Statement.SQL.String(), scope.Statement.Vars, options.DetailSQL))
					return
				}
			}

			next(scope)
		}
	}
}
