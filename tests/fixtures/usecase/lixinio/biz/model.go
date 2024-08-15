package biz

import "time"

type ClentStatus int32

type StructField struct {
	Test int32
}

type Client struct {
	ID           int64
	Status       ClentStatus
	StatusPtr    *ClentStatus
	StructPtr    *StructField
	StringPtr    *string
	IntPtr       *int32
	ClientID     string
	ClientSecret string
	TokenExpire  int
	CreateTime   time.Time
	UpdateTime   time.Time
	Provider     *ClientProvider
	Provider3    *ClientProvider
	Uris         []*ClientRedirectUri
	StringSlice  []string
	IntSlice2    []int
}

type ClientRedirectUri struct {
	ID       int64
	ClientID int64
	Uri      string
}

type ClientProvider struct {
	ID       int64
	ClientID int64
	Uri      string
}
