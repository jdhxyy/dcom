// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 日志模块
// Authors: jdh99 <jdh821@163.com>

package dcom

import "github.com/jdhxyy/lagan"

const Tag = "dcom"

var filterLevel = lagan.LevelWarn

// SetFilterLevel 设置日志过滤级别
func SetFilterLevel(level lagan.FilterLevel) {
	filterLevel = level
}

// logDebug 打印debug信息
func logDebug(format string, a ...interface{}) {
	if filterLevel == lagan.LevelOff || lagan.LevelDebug < filterLevel {
		return
	}
	lagan.Debug(Tag, format, a...)
}

// logInfo 打印info信息
func logInfo(format string, a ...interface{}) {
	if filterLevel == lagan.LevelOff || lagan.LevelInfo < filterLevel {
		return
	}
	lagan.Info(Tag, format, a...)
}

// logWarn 打印warn信息
func logWarn(format string, a ...interface{}) {
	if filterLevel == lagan.LevelOff || lagan.LevelWarn < filterLevel {
		return
	}
	lagan.Warn(Tag, format, a...)
}

// logError 打印error信息
func logError(format string, a ...interface{}) {
	if filterLevel == lagan.LevelOff || lagan.LevelWarn < filterLevel {
		return
	}
	lagan.Error(Tag, format, a...)
}
