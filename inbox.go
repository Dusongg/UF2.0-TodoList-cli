package main

import (
	"OrderManager-cli/pb"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TODO: 更新操作
func CreateInBox(tasks []*pb.Task) fyne.CanvasObject {
	tableData := make([][]interface{}, 0)
	tableData = append(tableData, []interface{}{"TaskId", "ReqNo", "Deadline", "Principal", "EstimatedWorkHours", "TaskId", "State", "EmergencyLevel", "TypeId"})
	for _, task := range tasks {
		tableData = append(tableData, []interface{}{task.TaskId, task.ReqNo, task.Deadline, task.Principal, task.EstimatedWorkHours, task.TaskId, task.State, task.EmergencyLevel, task.TypeId})
	}
	table := widget.NewTable(
		func() (int, int) {
			return len(tableData), len(tableData[0])
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("wide content")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			if v, ok := tableData[i.Row][i.Col].(string); ok {
				o.(*widget.Label).SetText(v)
			} else {
				o.(*widget.Label).SetText(fmt.Sprintf("%d", tableData[i.Row][i.Col]))
			}
		})

	table.CreateHeader = func() fyne.CanvasObject {
		header := container.NewGridWithColumns(len(tableData[0]))
		for _, txt := range tableData[0] {
			header.Add(widget.NewLabelWithStyle(txt.(string),
				fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		}
		return header
	}

	return table
}
