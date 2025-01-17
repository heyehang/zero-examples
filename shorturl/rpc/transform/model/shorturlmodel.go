package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/builder"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/stringx"
)

var (
	shorturlFieldNames          = builder.RawFieldNames(&Shorturl{})
	shorturlRows                = strings.Join(shorturlFieldNames, ",")
	shorturlRowsExpectAutoSet   = strings.Join(stringx.Remove(shorturlFieldNames, "create_time", "update_time"), ",")
	shorturlRowsWithPlaceHolder = strings.Join(stringx.Remove(shorturlFieldNames, "shorten", "create_time", "update_time"), "=?,") + "=?"

	cacheShorturlShortenPrefix = "cache#Shorturl#shorten#"
)

type (
	ShorturlModel struct {
		sqlc.CachedConn
		table string
	}

	Shorturl struct {
		Shorten string `db:"shorten"` // shorten key
		Url     string `db:"url"`     // original url
	}
)

func NewShorturlModel(conn sqlx.SqlConn, c cache.CacheConf, table string) *ShorturlModel {
	return &ShorturlModel{
		CachedConn: sqlc.NewConn(conn, c),
		table:      table,
	}
}

func (m *ShorturlModel) Insert(ctx context.Context, data Shorturl) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + shorturlRowsExpectAutoSet + `) values (?, ?)`
	return m.ExecNoCacheCtx(ctx, query, data.Shorten, data.Url)
}

func (m *ShorturlModel) FindOne(ctx context.Context, shorten string) (*Shorturl, error) {
	shorturlShortenKey := fmt.Sprintf("%s%v", cacheShorturlShortenPrefix, shorten)
	var resp Shorturl
	err := m.QueryRowCtx(ctx, &resp, shorturlShortenKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := `select ` + shorturlRows + ` from ` + m.table + ` where shorten = ? limit 1`
		return conn.QueryRowCtx(ctx, v, query, shorten)
	})
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *ShorturlModel) Update(ctx context.Context, data Shorturl) error {
	shorturlShortenKey := fmt.Sprintf("%s%v", cacheShorturlShortenPrefix, data.Shorten)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + shorturlRowsWithPlaceHolder + ` where shorten = ?`
		return conn.ExecCtx(ctx, query, data.Url, data.Shorten)
	}, shorturlShortenKey)
	return err
}

func (m *ShorturlModel) Delete(ctx context.Context, shorten string) error {
	_, err := m.FindOne(ctx, shorten)
	if err != nil {
		return err
	}

	shorturlShortenKey := fmt.Sprintf("%s%v", cacheShorturlShortenPrefix, shorten)
	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := `delete from ` + m.table + ` where shorten = ?`
		return conn.ExecCtx(ctx, query, shorten)
	}, shorturlShortenKey)
	return err
}
