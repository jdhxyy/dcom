// Copyright 2021-2021 The TZIOT Authors. All rights reserved.
// 参数管理模块
// Authors: jdh99 <jdh821@163.com>

package dcom

var gBlockRetryInterval = 0
var gBlockRetryMaxNum = 0

// SetBlockRetryInterval 设置块传输帧重试间隔.单位:ms
func SetBlockRetryInterval(interval int) {
    gBlockRetryInterval = interval
}

// GetBlockRetryInterval 读取块传输帧重试间隔.单位:ms
func GetBlockRetryInterval() int {
    return gBlockRetryInterval
}

// SetBlockRetryMaxNum 设置块传输帧重试最大次数
func SetBlockRetryMaxNum(interval int) {
    gBlockRetryMaxNum = interval
}

// GetBlockRetryMaxNum 读取块传输帧重试最大次数
func GetBlockRetryMaxNum() int {
    return gBlockRetryMaxNum
}

