// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 回调模块主文件
// Authors: jdh99 <jdh821@163.com>

package dcom

// CallbackFunc 注册DCOM服务回调函数
// 返回值是应答和错误码.错误码为0表示回调成功,否则是错误码
type CallbackFunc func(req []uint8) ([]uint8, ErrorCode)

var services = make(map[int]CallbackFunc)

// Register 注册服务回调函数
func Register(rid int, callback CallbackFunc) {
	services[rid] = callback
}

// gCallback 回调资源号rid对应的函数
func gCallback(rid int, req []uint8) ([]uint8, ErrorCode) {
	v, ok := services[rid]
	if ok == false {
		return nil, SystemErrorInvalidRid
	}
	return v(req)
}
