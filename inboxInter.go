package main

import (
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func buildData(tasks []*pb.Task) [][]interface{} {
	tableData := make([][]interface{}, 0)
	//tableData = append(tableData, []interface{}{"TaskId", "Principal", "ReqNo", "Deadline", "WorkHours", "Comment", "State", "Level", "TypeId"})
	tableData = append(tableData, []interface{}{"任务号", "负责人", "需求号", "截止日期", "预计工时", "任务描述", "状态", "紧急程度", "任务类型"})
	for _, task := range tasks {
		tableData = append(tableData, []interface{}{task.TaskId, task.Principal, task.ReqNo, task.Deadline, task.EstimatedWorkHours, task.Comment, task.State, task.EmergencyLevel, task.TypeId})
	}
	return tableData
}

func flushData(client pb.ServiceClient, mw fyne.Window) [][]interface{} {
	reply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
		return nil
	}
	tableData := make([][]interface{}, 0)
	tableData = append(tableData, []interface{}{"任务号", "负责人", "需求号", "截止日期", "预计工时", "任务描述", "状态", "紧急程度", "任务类型"})
	for _, task := range reply.Tasks {
		tableData = append(tableData, []interface{}{task.TaskId, task.Principal, task.ReqNo, task.Deadline, task.EstimatedWorkHours, task.Comment, task.State, task.EmergencyLevel, task.TypeId})
	}
	return tableData
}

// TODO: 更新操作
func CreateInBoxInterface(client pb.ServiceClient, mw fyne.Window) fyne.CanvasObject {
	reply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
		errRet := widget.NewEntry()
		errRet.SetPlaceHolder(err.Error())
		return errRet
	}
	tableData := buildData(reply.Tasks)

	table := widget.NewTable(
		func() (int, int) {
			return len(tableData), len(tableData[0])
		},
		func() fyne.CanvasObject {
			return widget.NewButton("", nil)
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			if i.Row == 0 {
				o.(*widget.Button).SetText(tableData[i.Row][i.Col].(string))
			} else {
				if v, ok := tableData[i.Row][i.Col].(string); ok {
					o.(*widget.Button).SetText(v)
				} else {
					o.(*widget.Button).SetText(fmt.Sprintf("%d", tableData[i.Row][i.Col]))
				}
				o.(*widget.Button).OnTapped = func() {
					if err := ModForm(tableData[i.Row][0].(string), client); err != nil {
						dialog.ShowError(err, mw)
					} else {
						flushData(client, mw)
					}
				}
			}
		})
	for i := 0; i < 9; i++ {
		table.SetColumnWidth(i, 150)
	}

	searchEntry := widget.NewEntry()
	searchChoose := widget.NewSelect([]string{"TaskId", "Principal", "ReqNo", "Deadline", "SQL"}, func(s string) {
		switch s {
		case "TaskId":
			searchEntry.SetText("")
			searchEntry.Refresh()
		case "Principal":
			searchEntry.SetText("")
			searchEntry.Refresh()
		case "ReqNo":
			searchEntry.SetText("")
			searchEntry.Refresh()
		case "Deadline":
			searchEntry.SetText("Usage: 2006-01-02")
			searchEntry.Refresh()
		case "SQL":
			searchEntry.SetText("Usage: select * from tasklist_table where task_id = ? (or principal = ? or req_no = ? deadline = ?)")
			searchEntry.Refresh()
		}
	})

	//sql语句查询, 或者输入字段查询
	flushTableByField := func(fieldName string) {
		rep, err := client.QueryTaskByField(context.Background(), &pb.QueryTaskWithFieldRequest{Field: fieldName, FieldValue: searchEntry.Text})
		if err != nil {
			dialog.ShowError(err, mw)
			return
		}
		tableData = buildData(rep.Tasks)
		table.Refresh()
	}
	activity := widget.NewActivity()
	searchBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		activity.Start()
		if searchEntry.Text == "" {
			tableData = flushData(client, mw)
			table.Refresh()
			return
		}
		switch searchChoose.SelectedIndex() {
		case 0:
			flushTableByField("task_id")
		case 1:
			flushTableByField("principal")
		case 2:
			flushTableByField("req_no")
		case 3:
			flushTableByField("deadline")
		case 4:
			rep, err := client.QueryTaskBySQL(context.Background(), &pb.QueryTaskWithSQLRequest{Sql: searchEntry.Text})
			if err != nil {
				dialog.ShowError(err, mw)
				return
			}
			tableData = buildData(rep.Tasks)
			table.Refresh()
		}
		activity.Stop()
	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		searchBtn.SetText("")
		tableData = flushData(client, mw)
		table.Refresh()
	})
	searchBar := container.NewStack(config.BarBg, container.NewBorder(nil, nil, searchChoose, container.NewHBox(searchBtn, activity, flushBtn), searchEntry))

	return container.NewBorder(searchBar, nil, nil, nil, table)
}
