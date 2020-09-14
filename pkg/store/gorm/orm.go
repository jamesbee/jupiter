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

	"github.com/douyu/jupiter/pkg/util/xdebug"
	"github.com/douyu/jupiter/pkg/xlog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// SQLCommon ...
type (
	// CallbackProcessor wrapper of gorm.processor
	CallbackProcessor interface {
		Get(name string) func(db *gorm.DB)
		Replace(name string, fn func(*DB)) error
	}
	// DB ...
	DB = gorm.DB
	// Model ...
	Model = gorm.Model
	// Association ...
	Association = gorm.Association
)

var (
	errSlowCommand = errors.New("mysql slow command")

	// ErrRecordNotFound record not found error
	ErrRecordNotFound = gorm.ErrRecordNotFound
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = gorm.ErrInvalidTransaction
	// ErrNotImplemented not implemented
	ErrNotImplemented = gorm.ErrNotImplemented
	// ErrMissingWhereClause missing where clause
	ErrMissingWhereClause = gorm.ErrMissingWhereClause
	// ErrUnsupportedRelation unsupported relations
	ErrUnsupportedRelation = gorm.ErrUnsupportedRelation
	// ErrPrimaryKeyRequired primary keys required
	ErrPrimaryKeyRequired = gorm.ErrPrimaryKeyRequired
	// ErrModelValueRequired model value required
	ErrModelValueRequired = gorm.ErrModelValueRequired
	// ErrInvalidData unsupported data
	ErrInvalidData = gorm.ErrInvalidData
	// ErrUnsupportedDriver unsupported driver
	ErrUnsupportedDriver = gorm.ErrUnsupportedDriver
	// ErrRegistered registered
	ErrRegistered = gorm.ErrRegistered
	// ErrInvalidField invalid field
	ErrInvalidField = gorm.ErrInvalidField
	// ErrEmptySlice empty slice found
	ErrEmptySlice = gorm.ErrEmptySlice
)

// IsRecordNotFoundError
func IsRecordNotFoundError(err error) bool {
	return errors.Is(err, ErrRecordNotFound)
}

// WithContext ...
func WithContext(ctx context.Context, db *DB) *DB {
	// db.InstantSet("_context", ctx)
	return db.WithContext(ctx)
}

// Open ...
func Open(options *Config) (*DB, error) {
	inner, err := gorm.Open(mysql.Open(options.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if options.Debug || xdebug.IsDevelopmentMode() {
		inner = inner.Debug()
	}
	// 设置默认连接配置
	db, err := inner.DB()
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(options.MaxIdleConns)
	db.SetMaxOpenConns(options.MaxOpenConns)

	if options.ConnMaxLifetime != 0 {
		db.SetConnMaxLifetime(options.ConnMaxLifetime)
	}

	replace := func(processor CallbackProcessor, callbackName string, interceptors ...Interceptor) {
		old := processor.Get(callbackName)
		var handler = old
		for _, inte := range interceptors {
			handler = inte(options.dsnCfg, callbackName, options)(handler)
		}
		err := processor.Replace(callbackName, handler)
		if err != nil {
			options.logger.Panic("failed to replace interceptor", xlog.FieldErr(err))
		}
	}

	replace(
		inner.Callback().Delete(),
		"gorm:delete",
		options.interceptors...,
	)
	replace(
		inner.Callback().Update(),
		"gorm:update",
		options.interceptors...,
	)
	replace(
		inner.Callback().Create(),
		"gorm:create",
		options.interceptors...,
	)
	replace(
		inner.Callback().Query(),
		"gorm:query",
		options.interceptors...,
	)
	replace(
		inner.Callback().Row(),
		"gorm:row",
		options.interceptors...,
	)

	return inner, err
}
