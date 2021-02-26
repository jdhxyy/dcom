// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 公共模块
// Authors: jdh99 <jdh821@163.com>

package dcom

import "time"

const Tag = "dcom"

var tokenValue = 0

// gGetToken 获取token
// token范围:0-1023
func gGetToken() int {
	tokenValue++
	if tokenValue > 1023 {
		tokenValue = 0
	}
	return tokenValue
}

// gControlWordToBytes 控制字转换为字节流.字节流是大端顺序
func gControlWordToBytes(word *tControlWord) []uint8 {
	var value uint32
	value = (uint32(word.code) << 29) + (uint32(word.blockFlag) << 28) + (uint32(word.rid) << 18) + (uint32(word.token) << 8) +
		uint32(word.payloadLen)
	bytes := make([]uint8, 4)
	bytes[0] = uint8(value >> 24)
	bytes[1] = uint8(value >> 16)
	bytes[2] = uint8(value >> 8)
	bytes[3] = uint8(value)
	return bytes
}

// gBytesToControlWord 字节流转控制字.字节流是大端顺序
func gBytesToControlWord(bytes []uint8) *tControlWord {
	if len(bytes) < gControlWordLen {
		return nil
	}
	var word tControlWord
	word.code = int((bytes[0] >> 5) & 0x7)
	word.blockFlag = int((bytes[0] >> 4) & 0x1)
	word.rid = int((bytes[0]&0xf)<<6) + int((bytes[1]>>2)&0x3f)
	word.token = (int(bytes[1]&0x3) << 8) + int(bytes[2])
	word.payloadLen = int(bytes[3])
	return &word
}

// gFrameToBytes 将帧转换为字节流.字节流是大端顺序
func gFrameToBytes(frame *tFrame) []uint8 {
	var bytes []uint8
	bytes = append(bytes, gControlWordToBytes(&frame.controlWord)...)
	bytes = append(bytes, frame.payload...)
	return bytes
}

// gByetsToFrame 字节流转换为帧.字节流是大端顺序
// 转换失败返回nil
func gByetsToFrame(bytes []uint8) *tFrame {
	var word = gBytesToControlWord(bytes)
	if word == nil {
		return nil
	}
	if len(bytes) < gControlWordLen+word.payloadLen {
		return nil
	}

	var frame tFrame
	frame.controlWord = *word
	frame.payload = append(frame.payload, bytes[gControlWordLen:gControlWordLen+word.payloadLen]...)
	return &frame
}

// gBlockHeaderToBytes 块传输头部转换为字节流
func gBlockHeaderToBytes(header *tBlocHeader) []uint8 {
	bytes := make([]uint8, 6)
	j := 0
	bytes[j] = uint8(header.crc16 >> 8)
	j++
	bytes[j] = uint8(header.crc16)
	j++
	bytes[j] = uint8(header.total >> 8)
	j++
	bytes[j] = uint8(header.total)
	j++
	bytes[j] = uint8(header.offset >> 8)
	j++
	bytes[j] = uint8(header.offset)
	j++
	return bytes
}

// gBytesToBlockHeader 字节流转块传输头部
func gBytesToBlockHeader(bytes []uint8) *tBlocHeader {
	if len(bytes) < gBlockHeaderLen {
		return nil
	}
	var header tBlocHeader
	j := 0
	header.crc16 = (uint16(bytes[j]) << 8) + uint16(bytes[j+1])
	j += 2
	header.total = (int(bytes[j]) << 8) + int(bytes[j+1])
	j += 2
	header.offset = (int(bytes[j]) << 8) + int(bytes[j+1])
	j += 2
	return &header
}

// gBlockFrameToBytes 块传输帧转字节流
func gBlockFrameToBytes(frame *tBlockFrame) []uint8 {
	var bytes []uint8
	bytes = append(bytes, gControlWordToBytes(&frame.controlWord)...)
	bytes = append(bytes, gBlockHeaderToBytes(&frame.blockHeader)...)
	bytes = append(bytes, frame.payload...)
	return bytes
}

// gByetsToBlockFrame 字节流转换为帧.字节流是大端顺序
// 转换失败返回nil
func gByetsToBlockFrame(bytes []uint8) *tBlockFrame {
	var word = gBytesToControlWord(bytes)
	if word == nil {
		return nil
	}
	if len(bytes) < gControlWordLen+word.payloadLen || word.payloadLen < gBlockHeaderLen {
		return nil
	}

	var blockHeader = gBytesToBlockHeader(bytes[gControlWordLen:])
	if blockHeader == nil {
		return nil
	}

	var frame tBlockFrame
	frame.controlWord = *word
	frame.blockHeader = *blockHeader
	frame.payload = append(frame.payload, bytes[gControlWordLen+gBlockHeaderLen:gControlWordLen+word.payloadLen]...)
	return &frame
}

// gGetTime 获取当前时间.单位:us
func gGetTime() int64 {
	return time.Now().UnixNano() / 1000
}
