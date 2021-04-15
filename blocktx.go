// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 块传输发送模块
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
	"container/list"
	"github.com/jdhxyy/crc16"
	"sync"
	"time"
)

type tBlockTxItem struct {
	protocol int
	pipe     uint64
	dstIA    uint64
	code     int
	rid      int
	token    int

	// 第一帧需要重发控制
	isFirstFrame        bool
	firstFrameRetryTime int64
	firstFrameRetryNum  int

	lastRxAckTime int64

	crc16 uint16
	data  []uint8
}

var blockTxItems list.List
var blockTxItemsMutex sync.Mutex

// gThreadBlockTxRun 块传输发送模块运行线程
func gThreadBlockTxRun() {
	for {
		blockTxItemsMutex.Lock()

		node := blockTxItems.Front()
		var nextNode *list.Element
		for {
			if node == nil {
				break
			}
			nextNode = node.Next()
			checkTimeoutAndRetrySendFirstFrame(node)
			node = nextNode
		}

		blockTxItemsMutex.Unlock()

		time.Sleep(gInterval)
	}
}

// checkTimeoutAndRetrySendFirstFrame 检查超时节点和重发首帧
func checkTimeoutAndRetrySendFirstFrame(node *list.Element) {
	item := node.Value.(*tBlockTxItem)
	now := gGetTime()
	if item.isFirstFrame == false {
		// 非首帧
		if now-item.lastRxAckTime > int64(gParam.BlockRetryInterval*gParam.BlockRetryMaxNum*1000) {
			logWarn("block tx timeout!remove task.token:%d", item.token)
			blockTxItems.Remove(node)
		}
		return
	}

	// 首帧处理
	if now-item.firstFrameRetryTime < int64(gParam.BlockRetryInterval*1000) {
		return
	}

	if item.firstFrameRetryNum >= gParam.BlockRetryMaxNum {
		logWarn("block tx timeout!first frame send retry too many.token:%d", item.token)
		blockTxItems.Remove(node)
	} else {
		item.firstFrameRetryNum++
		item.firstFrameRetryTime = now
		logInfo("block tx send first frame.token:%d retry num:%d", item.token, item.firstFrameRetryNum)
		blockTxSendFrame(item, 0)
	}
}

func blockTxSendFrame(item *tBlockTxItem, offset int) {
	logInfo("block tx send.token:%d offset:%d", item.token, offset)
	delta := len(item.data) - offset
	payloadLen := gSingleFrameSizeMax - gBlockHeaderLen
	if payloadLen > delta {
		payloadLen = delta
	}

	var frame tBlockFrame
	frame.controlWord.code = item.code
	frame.controlWord.blockFlag = 1
	frame.controlWord.rid = item.rid
	frame.controlWord.token = item.token
	frame.controlWord.payloadLen = gBlockHeaderLen + payloadLen
	frame.blockHeader.crc16 = item.crc16
	frame.blockHeader.total = len(item.data)
	frame.blockHeader.offset = offset
	frame.payload = append(frame.payload, item.data[offset:offset+payloadLen]...)
	gBlockSend(item.protocol, item.pipe, item.dstIA, &frame)
}

// gBlockTx 块传输发送
func gBlockTx(protocol int, pipe uint64, dstIA uint64, code int, rid int, token int, data []uint8) {
	if len(data) <= gSingleFrameSizeMax {
		return
	}

	blockTxItemsMutex.Lock()
	defer blockTxItemsMutex.Unlock()

	if blockTxIsNodeExist(protocol, pipe, dstIA, code, rid, token) {
		return
	}

	logInfo("block tx new task.token:%d dst ia:0x%x code:%d rid:%d", token, dstIA, code, rid)
	item := blockTxCreateItem(protocol, pipe, dstIA, code, rid, token, data)
	blockTxSendFrame(item, 0)
	item.firstFrameRetryNum++
	item.firstFrameRetryTime = gGetTime()
	blockTxItems.PushBack(item)
}

func blockTxIsNodeExist(protocol int, pipe uint64, dstIA uint64, code int, rid int, token int) bool {
	node := blockTxItems.Front()
	var item *tBlockTxItem
	for {
		if node == nil {
			break
		}
		item = node.Value.(*tBlockTxItem)

		if item.protocol == protocol && item.pipe == pipe && item.dstIA == dstIA && item.code == code &&
			item.rid == rid && item.token == token {
			return true
		}
		node = node.Next()
	}
	return false
}

func blockTxCreateItem(protocol int, pipe uint64, dstIA uint64, code int, rid int, token int, data []uint8) *tBlockTxItem {
	var item tBlockTxItem
	item.protocol = protocol
	item.pipe = pipe
	item.dstIA = dstIA
	item.code = code
	item.rid = rid
	item.token = token
	item.data = append(item.data, data...)
	item.crc16 = crc16.Checksum(data)

	item.isFirstFrame = true
	item.firstFrameRetryNum = 0
	now := gGetTime()
	item.firstFrameRetryTime = now
	item.lastRxAckTime = now
	return &item
}

// gBlockRxBackFrame 接收到BACK帧时处理函数
func gBlockRxBackFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	if frame.controlWord.code != gCodeBack {
		return
	}

	blockTxItemsMutex.Lock()
	defer blockTxItemsMutex.Unlock()

	node := blockTxItems.Front()
	var nextNode *list.Element
	for {
		if node == nil {
			break
		}
		nextNode = node.Next()
		if checkNodeAndDealBackFrame(protocol, pipe, srcIA, frame, node) {
			break
		}
		node = nextNode
	}
}

// checkNodeAndDealBackFrame 检查节点是否符合条件,符合则处理BACK帧
// 返回true表示节点符合条件
func checkNodeAndDealBackFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame, node *list.Element) bool {
	item := node.Value.(*tBlockTxItem)

	if item.protocol != protocol || item.pipe != pipe || item.dstIA != srcIA || item.rid != frame.controlWord.rid ||
		item.token != frame.controlWord.token {
		return false
	}
	logInfo("block tx receive back.token:%d", item.token)
	if frame.controlWord.payloadLen != 2 {
		logWarn("block rx receive back deal failed!token:%d payload len is wrong:%d", item.token,
			frame.controlWord.payloadLen)
		return false
	}
	startOffset := (int(frame.payload[0]) << 8) + int(frame.payload[1])
	if startOffset >= len(item.data) {
		// 发送完成
		logInfo("block tx end.receive back token:%d start offset:%d >= data len:%d", item.token, startOffset,
			len(item.data))
		blockTxItems.Remove(node)
		return true
	}

	if item.isFirstFrame {
		item.isFirstFrame = false
	}
	item.lastRxAckTime = gGetTime()

	blockTxSendFrame(item, startOffset)
	return true
}

// gBlockTxDealRstFrame 块传输发送模块处理复位连接帧
func gBlockTxDealRstFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	blockTxItemsMutex.Lock()
	defer blockTxItemsMutex.Unlock()

	node := blockTxItems.Front()
	var item *tBlockTxItem
	for {
		if node == nil {
			break
		}

		item = node.Value.(*tBlockTxItem)
		if item.protocol == protocol && item.pipe == pipe && item.dstIA == srcIA && item.rid == frame.controlWord.rid &&
			item.token == frame.controlWord.token {
			logWarn("block tx receive rst.token:%d", item.token)
			blockTxItems.Remove(node)
			return
		}

		node = node.Next()
	}
}

// gBlockRemove 块传输发送移除任务
func gBlockRemove(protocol int, pipe uint64, dstIA uint64, code int, rid int, token int) {
	blockTxItemsMutex.Lock()
	defer blockTxItemsMutex.Unlock()

	node := blockTxItems.Front()
	var item *tBlockTxItem
	for {
		if node == nil {
			break
		}
		item = node.Value.(*tBlockTxItem)

		if item.protocol == protocol && item.pipe == pipe && item.dstIA == dstIA && item.code == code &&
			item.rid == rid && item.token == token {
			logWarn("block tx remove task.token:%d", item.token)
			blockTxItems.Remove(node)
			break
		}
		node = node.Next()
	}
}
