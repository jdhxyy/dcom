// Copyright 2021-2021 The TZIOT Authors. All rights reserved.
// 等待队列
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
    "container/list"
    "sync"
    "time"
)

// Resp 异步调用应答
type Resp struct {
    Error ErrorCode
    Bytes []uint8
    Done  chan *Resp
}

// done 结果返回.框架内调用
func (resp *Resp) done() {
    select {
    case resp.Done <- resp:
    default:
    }
}

type tWaitItem struct {
    resp *Resp
    end  chan bool

    port      int
    timeoutUs int64
    req       []uint8
    timeStart int64
    // 上次发送时间.用于重传
    lastTxTime int64
    retryNum   int
    code       int

    dstIA uint64
    rid   int
    token int
}

var waitItems list.List
var waitItemsMutex sync.Mutex

// Call RPC同步调用
// timeout是超时时间,单位:ms.为0表示不需要应答
// 返回值是应答字节流和错误码.错误码非SystemOK表示调用失败
func Call(port int, dstIA uint64, rid int, timeout int, req []uint8) ([]uint8, ErrorCode) {
    resp := CallAsync(port, dstIA, rid, timeout, req)
    <-resp.Done
    return resp.Bytes, resp.Error
}

// CallAsync RPC异步调用
// timeout是超时时间,单位:ms.为0表示不需要应答
// 返回值中错误码非SystemOK表示调用失败
func CallAsync(port int, dstIA uint64, rid int, timeout int, req []uint8) *Resp {
    var resp Resp
    resp.Done = make(chan *Resp, 10)

    code := gCodeCon
    if timeout == 0 {
        code = gCodeNon
    }

    token := gGetToken()
    waitlistSendFrame(port, dstIA, code, rid, token, req)

    if code == gCodeNon {
        resp.Error = SystemOK
        go func() {
            select {
            case <-time.After(time.Millisecond):
                resp.done()
            }
        }()
        return &resp
    }

    var item tWaitItem
    item.resp = &resp
    item.end = make(chan bool)
    item.port = port
    item.timeoutUs = int64(timeout) * 1000
    item.req = req
    item.timeStart = time.Now().Unix()

    item.dstIA = dstIA
    item.rid = rid
    item.token = token
    waitItems.PushBack(&item)

    // 等待数据
    go func() {
        select {
        case <-item.end:
            item.resp.done()
        case <-time.After(time.Duration(timeout) * time.Millisecond):
            item.resp.Error = SystemErrorRxTimeout
            item.resp.done()
        }
    }()
    return &resp
}

func waitlistSendFrame(port int, dstIA uint64, code int, rid int, token int, data []uint8) {
    if len(data) >= gSingleFrameSizeMax {
        gBlockTx(port, dstIA, code, rid, token, data)
        return
    }

    var frame tFrame
    frame.controlWord.code = code
    frame.controlWord.blockFlag = 0
    frame.controlWord.rid = rid
    frame.controlWord.token = token
    frame.controlWord.payloadLen = len(data)
    frame.payload = append(frame.payload, data...)
    gSend(port, dstIA, &frame)
}

// gRxAckFrame 接收到ACK帧时处理函数
func gRxAckFrame(port int, srcIA uint64, frame *tFrame) {
    waitItemsMutex.Lock()
    defer waitItemsMutex.Unlock()

    node := waitItems.Front()
    var nodeNext *list.Element
    for {
        if node == nil {
            break
        }
        nodeNext = node.Next()
        if checkNodeAndDealAckFrame(port, srcIA, frame, node) {
            break
        }
        node = nodeNext
    }
}

func checkNodeAndDealAckFrame(port int, srcIA uint64, frame *tFrame, node *list.Element) bool {
    item := node.Value.(*tWaitItem)
    if item.port != port || item.dstIA != srcIA || item.rid != frame.controlWord.rid ||
        item.token != frame.controlWord.token {
        return false
    }

    item.resp.Bytes = append(item.resp.Bytes, frame.payload...)
    item.resp.Error = SystemOK
    item.end<-true
    waitItems.Remove(node)
    return true
}

// gRxRstFrame 接收到RST帧时处理函数
func gRxRstFrame(port int, srcIA uint64, frame *tFrame) {
    waitItemsMutex.Lock()
    defer waitItemsMutex.Unlock()

    node := waitItems.Front()
    var nodeNext *list.Element
    for {
        if node == nil {
            break
        }
        nodeNext = node.Next()
        if dealRstFrame(port, srcIA, frame, node) {
            break
        }
        node = nodeNext
    }
}

// dealRstFrame 处理复位连接帧
// 返回true表示节点符合条件
func dealRstFrame(port int, srcIA uint64, frame *tFrame, node *list.Element) bool {
    item := node.Value.(*tWaitItem)
    if item.port != port || item.dstIA != srcIA || item.rid != frame.controlWord.rid ||
        item.token != frame.controlWord.token {
        return false
    }
    err := ErrorCode(frame.payload[0])
    item.resp.Error = err
    item.end<-true
    waitItems.Remove(node)
    return true
}
