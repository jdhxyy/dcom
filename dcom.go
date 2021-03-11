// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// dcom接口文件
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
	"bytes"
	"encoding/binary"
)

// 对外参数
const (
	// 版本
	Version = "1.0"
)

const (
	// 正确值
	SystemOK = 0
	// 接收超时
	SystemErrorRxTimeout = 0x10
	// 发送超时
	SystemErrorTxTimeout = 0x11
	// 内存不足
	SystemErrorNotEnoughMemory = 0x12
	// 没有对应的资源ID
	SystemErrorInvalidRid = 0x13
	// 块传输校验错误
	SystemErrorWrongBlockCheck = 0x14
	// 块传输偏移地址错误
	SystemErrorWrongBlockOffset = 0x15
	// 参数错误
	SystemErrorParamInvalid = 0x16
)

// 模块内参数
const (
	// CODE码
	gCodeCon  = 0
	gCodeNon  = 1
	gCodeAck  = 2
	gCodeRst  = 3
	gCodeBack = 4

	// 单帧最大字节数.超过此字节数需要块传输
	gSingleFrameSizeMax = 255

	// 控制字字节数
	gControlWordLen = 4
	// 块传输头部长度
	gBlockHeaderLen = 6

	// 运行间隔.单位:us.子模块运行函数执行间隔
	gInterval = 100000
)

// tControlWord 控制字
type tControlWord struct {
	payloadLen int
	token      int
	rid        int
	blockFlag  int
	code       int
}

// tFrame dcom帧
type tFrame struct {
	controlWord tControlWord
	payload     []uint8
}

// tBlocHeader 块传输头部
type tBlocHeader struct {
	crc16  uint16
	total  int
	offset int
}

// tBlockFrame 块传输帧.重定义了dcom帧的载荷
// 此时控制字中的载荷长度为本帧长度.块传输中的总字节数指示了整个块的字节数
type tBlockFrame struct {
	controlWord tControlWord
	blockHeader tBlocHeader
	payload     []uint8
}

// IsAllowSendFuncByPortFunc 某端口是否允许发送函数类型
type IsAllowSendFuncByPortFunc func(port uint64) bool

// SendByPortFunc 向指定端口发送函数类型
type SendByPortFunc func(protocol int, port uint64, dstIA uint64, bytes []uint8)

// LoadParam 载入参数
type LoadParam struct {
	// 块传输帧重试间隔.单位:ms
	BlockRetryInterval int
	// 块传输帧重试最大次数
	BlockRetryMaxNum int

	// API接口
	// 是否允许发送
	IsAllowSend IsAllowSendFuncByPortFunc
	// 发送的是DCOM协议数据
	Send SendByPortFunc
}

var gParam LoadParam

// Load 模块载入
func Load(param *LoadParam) {
	gParam = *param

	// 模块载入
	gRxLoad()

	// 模块运行
	go gThreadBlockRxRun()
	go gThreadBlockTxRun()
}

// StructToBytes 结构体转字节流
func StructToBytes(s interface{}) ([]uint8, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, s)
	if err != nil {
		return nil, err
	} else {
		return buf.Bytes(), err
	}
}

// BytesToStruct 字节流转结构体 结构体中的元素首字母要求大写
// s是结构体指针,保存转换后的结构体
func BytesToStruct(data []uint8, s interface{}) error {
	err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, s)
	return err
}
