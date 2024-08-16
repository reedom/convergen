//go:build convergen

/* Specify the name of the generated file's package. */
package data

//go:generate /tmp/convergen/convergen -suffix transfer

import (
	_ "strings"

	"github.com/reedom/convergen/tests/fixtures/usecase/lixinio/biz"
)

// :convergen
// :style arg
type ConvergenArg interface {
	// :typecast
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
	ClientToBiz2(*Client) *biz.Client
	// :typecast
	// 忽略字段
	// :skip ClientSecret
	// 用自定义函数转换
	// :conv Int8 Status
	// 字段映射
	// :map CreateTime CreateAt
	// :method Local UpdateTime
	// :map IntSlice2 IntSlice
	// 转换
	// :skip Provider Provider2
	// :conv NewClientRedirectUriFromBiz Uris[]
	NewClientArgFromBiz(*biz.Client) (*Client, error)

	// 映射并转换
	// :conv strings.ToUpper Url Uri
	// 使用成员函数, 函数名去掉前缀(ClientRedirectUri, 就保留 ToBiz)
	// :recv client ClientRedirectUri
	ClientRedirectUriToBiz2(*ClientRedirectUri) *biz.ClientRedirectUri
	// 映射并转换
	// :conv strings.ToUpper Uri Url
	// :preprocess prepareEmpty
	NewClientArgRedirectUriFromBiz(*biz.ClientRedirectUri) *ClientRedirectUri

	// 忽略特定字段
	// :skip ClientID
	// 转换
	// :conv strings.ToLower Uri
	// 使用成员函数, 函数名去掉前缀(ClientProvider, 就保留 ToBiz)
	// :recv client ClientProvider
	ClientProviderToBiz2(*ClientProvider) *biz.ClientProvider
	// 转换
	// :conv strings.ToLower Uri
	// 前置/后置检查
	// :preprocess prepareClientProvider
	// :postprocess cleanUpClientProvider
	// :literal InternalFlag 123
	NewClientArgProviderFromBiz(*biz.ClientProvider) (*ClientProvider, error)
}
