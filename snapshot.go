package vt

import (
	"maps"
	"strings"
)

// CommandSnapshot 是当前光标所在逻辑行的只读快照，不提交行或修改屏幕。
// CommandKnown 表示 OSC 133 标记的命令区域仍有效；空命令也可为已知。
// Line 包括提示符，仅供没有 shell integration 的调用方自行识别。
type CommandSnapshot struct {
	Line         string
	Command      string
	CommandKnown bool
	Modes        ModeSnapshot
}

// SnapshotCommand 在同一把读锁下取得行、命令边界和终端模式。
func (vt *virtualTerminal) SnapshotCommand() CommandSnapshot {
	vt.RLock()
	defer vt.RUnlock()
	snapshot := CommandSnapshot{Modes: maps.Clone(vt.privateModes)}
	if vt.rows < 0 || vt.rows >= len(vt.rowList) || snapshot.Modes.IsAltScreen() {
		return snapshot
	}
	start, end := vt.rows, vt.rows
	for start > 0 && vt.rowList[start].wrappedFromPrev {
		start--
	}
	// 光标可能在命令中间，必须包含后方的软折行。
	for end+1 < len(vt.rowList) && vt.rowList[end+1].wrappedFromPrev {
		end++
	}
	var line strings.Builder
	for i := start; i <= end; i++ {
		line.WriteString(vt.rowList[i].String())
	}
	snapshot.Line = line.String()
	if vt.cmdZoneActive && start == vt.cmdZoneRow {
		var command strings.Builder
		command.WriteString(vt.rowList[start].textFromCol(vt.cmdZoneCol))
		for i := start + 1; i <= end; i++ {
			command.WriteString(vt.rowList[i].String())
		}
		snapshot.Command, snapshot.CommandKnown = command.String(), true
	}
	return snapshot
}

// ResizeCols 更新后续屏幕重建的列宽，不尝试重排历史输出。
func (vt *virtualTerminal) ResizeCols(cols int) {
	vt.Lock()
	defer vt.Unlock()
	if cols > 0 && cols <= maxScreenDim {
		vt.cols = cols
	}
}
