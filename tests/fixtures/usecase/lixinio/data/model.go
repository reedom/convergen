package data

import (
	"time"

	"github.com/reedom/convergen/tests/fixtures/usecase/lixinio/biz"
)

type (
	StructField2 biz.StructField
	String       string
	Int32        int32
)

// @table clients
type Client struct {
	ID           int64
	Status       int8
	StatusPtr    *int32
	StructPtr    *StructField2
	StringPtr    *String
	IntPtr       *Int32
	ClientID     string
	ClientSecret string
	TokenExpire  int
	CreateAt     time.Time            // 字段不一致
	UpdateTime   time.Time            // Local
	Provider     *ClientProvider      // 字段一致 (ToBiz)
	Provider2    *ClientProvider      // 字段不一致
	Uris         []*ClientRedirectUri // 数组 ToBiz
	StringSlice  []string             // 数组拷贝(基本类型)
	IntSlice     []int                // 字段不一致, 数组拷贝(基本类型), 直接赋值了, 没有copy
}

// @table client_redirect_uris
type ClientRedirectUri struct {
	ID       int64
	ClientID int64
	Url      string // 字段不一致
}

type ClientProvider struct {
	ID           int64
	ClientID     int64
	Uri          string
	InternalFlag int
}
