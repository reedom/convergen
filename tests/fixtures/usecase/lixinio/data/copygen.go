//go:build convergen

/* Specify the name of the generated file's package. */
package data

//go:generate /tmp/convergen/convergen -suffix transfer

import (
	"errors"
	_ "strings"

	"github.com/reedom/convergen/tests/fixtures/usecase/lixinio/biz"
)

func Int8(status biz.ClentStatus) int8 {
	return int8(status)
}

func ClientStatus(status int8) biz.ClentStatus {
	return biz.ClentStatus(status)
}

func prepareClientProvider(dst *ClientProvider, src *biz.ClientProvider) error {
	if src == nil {
		return errors.New("empty model")
	}

	return nil
}

func cleanUpClientProvider(dst *ClientProvider, src *biz.ClientProvider) error {
	if dst.ID == 0 {
		return errors.New("empty model id")
	}

	return nil
}

type Convergen interface {
	//:typecast
	// 使用成员函数, 函数名去掉前缀(Client, 就保留 ToBiz)
	// :recv client Client
	// 忽略字段
	// :skip ClientSecret
	// 也可以这样 :conv ClientStatus Status
	// 字段映射
	// :map CreateAt CreateTime
	// :map IntSlice IntSlice2
	// 调用成员函数
	// :method ToBiz Provider
	// :method ToBiz Provider2 Provider3
	// :method ToBiz Uris[]
	ClientToBiz(*Client) *biz.Client
	//:typecast
	// 忽略字段
	// :skip ClientSecret
	// 用自定义函数转换
	// :conv Int8 Status
	// 字段映射
	// :map CreateTime CreateAt
	// :method Local UpdateTime
	// :map IntSlice2 IntSlice
	// 转换
	// :conv NewClientProviderFromBiz Provider
	// :conv NewClientProviderFromBiz Provider3 Provider2
	// :conv NewClientRedirectUriFromBiz Uris[]
	NewClientFromBiz(*biz.Client) (*Client, error)

	// 映射并转换
	// :conv strings.ToUpper Url Uri
	// 使用成员函数, 函数名去掉前缀(ClientRedirectUri, 就保留 ToBiz)
	// :recv client ClientRedirectUri
	ClientRedirectUriToBiz(*ClientRedirectUri) *biz.ClientRedirectUri
	// 映射并转换
	// :conv strings.ToUpper Uri Url
	NewClientRedirectUriFromBiz(*biz.ClientRedirectUri) *ClientRedirectUri

	// 忽略特定字段
	// :skip ClientID
	// 转换
	// :conv strings.ToLower Uri
	// 使用成员函数, 函数名去掉前缀(ClientProvider, 就保留 ToBiz)
	// :recv client ClientProvider
	ClientProviderToBiz(*ClientProvider) *biz.ClientProvider
	// 转换
	// :conv strings.ToLower Uri
	// 前置/后置检查
	// :preprocess prepareClientProvider
	// :postprocess cleanUpClientProvider
	// :literal InternalFlag 123
	NewClientProviderFromBiz(*biz.ClientProvider) (*ClientProvider, error)
}
