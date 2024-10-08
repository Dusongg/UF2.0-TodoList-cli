package main

import (
	"OrderManager-cli/common"
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"
	"image/color"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	AccountView = 0
	TeamView    = 1
)

var taskColors = []color.Color{
	color.RGBA{R: 255, G: 90, B: 33, A: 255},
	color.RGBA{R: 42, G: 43, B: 61, A: 255},
	color.RGBA{R: 58, G: 178, B: 218, A: 255},
	color.RGBA{R: 111, G: 219, B: 161, A: 255},
	color.RGBA{R: 242, G: 15, B: 98, A: 255},
	color.RGBA{R: 240, G: 53, B: 9, A: 255},
	color.RGBA{R: 240, G: 53, B: 9, A: 255},
	color.RGBA{R: 247, G: 153, B: 174, A: 255},
	color.RGBA{R: 79, G: 179, B: 129, A: 255},
	color.RGBA{R: 180, G: 222, B: 180, A: 255},
	color.RGBA{G: 146, B: 199, A: 255},
	color.RGBA{R: 120, G: 94, A: 255},
}

var allAccountNames = make([]string, 0)

func CreatePreviewInterface(appTab *container.AppTabs, client pb.ServiceClient, mw fyne.Window, msgChan <-chan string) *fyne.Container {
	now := time.Now()
	flushActivity := widget.NewActivity()

	var previewInterface *fyne.Container
	curView := AccountView //0:个人， 1：团队
	// 创建一个网格布局，每行显示一天的任务
	accountViewPage := 0
	viewGrid := make([]*fyne.Container, 7)
	var teamGrid fyne.CanvasObject
	for i := range viewGrid {
		viewGrid[i] = container.NewGridWithRows(1)
	}

	data, expired := loadAccountGrid(client, mw, config.Cfg.Login.UserName)
	//var allData map[int][]*pb.Task
	//var allExpired []*pb.Task

	var allTidyData map[string]soredTV
	var allTidyExpired []*pb.Task

	var topBtn *fyne.Container
	var bottom *fyne.Container
	var teamBottom *fyne.Container
	flushInterface := func() {
		flushActivity.Start()
		switch curView {
		case AccountView:
			appTab.Items[0].Content = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[accountViewPage])

			data, expired = loadAccountGrid(client, mw, config.Cfg.Login.UserName)
			//previewInterface = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[0])
			previewInterface.Objects[2] = viewGrid[accountViewPage]
		case TeamView:
			allTidyData, allTidyExpired = loadTidyTeamGrid(client, mw)
			if len(allAccountNames) == 0 {
				rep, _ := client.GetAllUserName(context.Background(), &pb.GetAllUserNameRequest{})
				allAccountNames = rep.Names
			}
			//BUG:

			teamGrid = tidyTeamGrid(allTidyData, allTidyExpired, mw)
			appTab.Items[0].Content = container.NewBorder(topBtn, teamBottom, nil, nil, teamGrid)
		}
		appTab.Refresh()
		flushActivity.Stop()
	}
	list := make(map[int]*widget.List, 31)
	expiredList := widget.NewList(
		// 获取列表项的数量
		func() int {
			return len(expired)
		},
		// 创建列表项的模板
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Black)
			btn := widget.NewButton("Do Something", nil)

			return container.NewPadded(
				container.NewBorder(nil, nil, bg, nil, btn),
			)
		},
		// 更新列表项的内容
		func(id widget.ListItemID, item fyne.CanvasObject) {
			colid, _ := strconv.Atoi(expired[id].TaskId)
			item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Rectangle).FillColor = taskColors[colid%len(taskColors)]
			item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).SetText(fmt.Sprintf("%s : %s", expired[id].TaskId, expired[id].Principal))
			item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
				if err := ModForm(expired[id].TaskId, client); err != nil {
					dialog.ShowError(err, mw)
				} else {
					flushInterface()
				}
			}
		},
	)

	expiredLabel := canvas.NewText("Expired", theme.Color(theme.ColorNameForeground))
	// 设置Label的字体颜色
	expiredLabel.TextStyle = fyne.TextStyle{Bold: true}
	expiredLabel.Color = color.White
	bgTheme1 := canvas.NewRectangle(config.ColorTheme1)
	viewGrid[0].Add(container.NewBorder(container.NewStack(bgTheme1, expiredLabel), nil, nil, nil, expiredList))

	for d := 0; d < 31; d++ {
		list[d] = widget.NewList(
			// 获取列表项的数量
			func() int {
				return len(data[d])
			},
			// 创建列表项的模板
			func() fyne.CanvasObject {
				bg := canvas.NewRectangle(color.Black)
				btn := widget.NewButton("Do Something", nil)

				return container.NewPadded(
					container.NewBorder(nil, nil, bg, nil, btn),
				)
			},
			// 更新列表项的内容
			func(id widget.ListItemID, item fyne.CanvasObject) {
				colid, _ := strconv.Atoi(data[d][id].TaskId)
				item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Rectangle).FillColor = taskColors[colid%len(taskColors)]
				item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).SetText(fmt.Sprintf("%s : %s", data[d][id].TaskId, data[d][id].Principal))
				item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
					if err := ModForm(data[d][id].TaskId, client); err != nil {
						dialog.ShowError(err, mw)
					} else {
						flushInterface()
					}
				}
			},
		)

		date := now.AddDate(0, 0, d)
		//dateLabel := widget.NewLabel(date.Format("2006-01-02"))

		dateLabel := canvas.NewText(date.Format("2006-01-02"), theme.Color(theme.ColorNameForeground))
		// 设置Label的字体颜色
		dateLabel.TextStyle = fyne.TextStyle{Bold: true}
		dateLabel.Color = color.White

		viewGrid[d/DAYSPERPAGE].Add(container.NewBorder(container.NewStack(bgTheme1, dateLabel), nil, nil, nil, list[d]))
	}

	//TODO:team

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		newTask := addForm(client)
		if newTask != nil {
			addData(newTask, &expired, data)
			previewInterface.Refresh()
		}
	})
	importBtn := widget.NewButtonWithIcon("", theme.UploadIcon(), func() {
		flushChan := make(chan struct{})
		common.ImportController(myapp, client, common.ImportXLStoTaskListByPython, flushChan)
		for {
			_, ok := <-flushChan
			if ok {
				flushInterface()
			} else {
				break
			}
		}
	})

	accountEty := widget.NewEntry()
	accountEty.PlaceHolder = "请输入姓名进行查询"
	preSearch := ""
	accountBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		if accountEty.Text != "" {
			preSearch = accountEty.Text
			//TODO:没有考虑用户是否存在
			data, expired = loadAccountGrid(client, mw, accountEty.Text)
		} else if preSearch != "" { //防止重复刷新
			preSearch = ""
			data, expired = loadAccountGrid(client, mw, config.Cfg.Login.UserName)
		}
		curView = AccountView
		appTab.Items[0].Content = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[accountViewPage])
		appTab.Refresh()

	})
	accountBtnWithBg := container.NewStack(canvas.NewRectangle(config.ColorTheme1), accountBtn)

	//accountBg := canvas.NewRectangle(colorTheme1)
	//accountBox := container.NewStack(accountBg, container.NewHBox(accountBtn, accountEty, nil))

	teamBtn := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		curView = TeamView

		if teamGrid == nil {
			//allData, allExpired = loadTeamGrid(client, mw)
			allTidyData, allTidyExpired = loadTidyTeamGrid(client, mw)

			if len(allAccountNames) == 0 {
				rep, err := client.GetAllUserName(context.Background(), &pb.GetAllUserNameRequest{})
				if err != nil {
					log.Fatal(err)
				}
				allAccountNames = rep.Names
			}
			//teamGrid = createTeamGrid(allData, allExpired, mw)
			teamGrid = tidyTeamGrid(allTidyData, allTidyExpired, mw)
		}

		appTab.Items[0].Content = container.NewBorder(topBtn, teamBottom, nil, nil, teamGrid)
		appTab.Refresh()
	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		flushInterface()
	})
	updateCurViewGrid := func() {
		appTab.Items[0].Content = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[accountViewPage])
		appTab.Refresh()
	}
	prevPageBtn := widget.NewButtonWithIcon("", theme.MediaFastRewindIcon(), func() {
		accountViewPage = max(accountViewPage-1, 0)
		updateCurViewGrid()
	})
	nextPageBtn := widget.NewButtonWithIcon("", theme.MediaFastForwardIcon(), func() {
		accountViewPage = min(accountViewPage+1, 6)
		updateCurViewGrid()
	})

	btnBar := container.NewBorder(
		nil,
		nil,
		container.NewHBox(addBtn, importBtn, flushBtn, flushActivity, teamBtn, accountBtnWithBg),
		nil,
		container.NewStack(canvas.NewRectangle(config.ColorTheme1), accountEty))
	bg := canvas.NewRectangle(color.RGBA{241, 241, 240, 255})
	topBtn = container.NewStack(bg, btnBar)

	personalBtn := widget.NewButton(config.Cfg.Login.UserName, func() {
		if err := personalView(client); err != nil {
			dialog.ShowError(err, mw)
		}
	})
	personalBtn.Importance = widget.HighImportance

	hideWd := false
	newMsgWdFunc := func() fyne.Window {
		msgWd := myapp.NewWindow("msg")
		msgWd.SetContent(widget.NewLabel("xx"))
		msgWd.SetIcon(theme.MailComposeIcon())
		msgWd.Resize(fyne.NewSize(550, 300))
		msgWd.SetCloseIntercept(func() {
			msgWd.Hide()
			hideWd = false
			return
		})
		return msgWd
	}

	msgActivity := widget.NewActivity()
	msgCntLabel := widget.NewLabel(strconv.Itoa(msgCnt))
	msgWd := newMsgWdFunc()
	var msgData []string
	msgList := widget.NewList(
		func() int {
			return len(msgData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(msgData[id])
		},
	)

	msgWd.SetContent(msgList)
	msgBtn := widget.NewButtonWithIcon("", theme.MailComposeIcon(), func() {
		if hideWd {
			msgWd.Hide()
			hideWd = false
			return
		}
		msgCnt = 0
		msgCntLabel.SetText(fmt.Sprintf("%d", msgCnt))
		msgActivity.Stop()
		msgWd.Show()
		hideWd = !hideWd
		//清除消息
	})
	//后台管理消息窗口
	go func() {
		for msg := range msgChan {
			msgActivity.Start()
			msgCnt++
			if msgCnt >= 10 {
				msgCntLabel.SetText("9+")
			} else {
				msgCntLabel.SetText(fmt.Sprintf("%d", msgCnt))
			}
			msgData = append([]string{msg}, msgData...)
		}
	}()

	undoBtn := widget.NewButtonWithIcon("", theme.ContentUndoIcon(), func() {
		if err := logChain.undo(); err != nil {
			dialog.ShowError(err, mw)
		}
		flushInterface()
	})
	redoBtn := widget.NewButtonWithIcon("", theme.ContentRedoIcon(), func() {
		if err := logChain.redo(); err != nil {
			dialog.ShowError(err, mw)
		}
		flushInterface()
	})

	msgBox := container.NewHBox(undoBtn, msgActivity, msgBtn, msgCntLabel, redoBtn)

	bottom = container.NewStack(bg, container.NewHBox(personalBtn, layout.NewSpacer(), msgBox, layout.NewSpacer(), prevPageBtn, nextPageBtn))
	teamBottom = container.NewStack(bg, container.NewHBox(personalBtn, layout.NewSpacer(), msgBox, layout.NewSpacer()))
	previewInterface = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[0])
	return previewInterface
}

func tidyTeamGrid(data map[string]soredTV, expired []*pb.Task, mw fyne.Window) fyne.CanvasObject {
	// 0        expired d1 d2 d3 d4 d5 ...
	// dusong1
	// dusong2
	infoMap := make(map[int64][]string)
	nameMap := make(map[string]int)
	grid := make([][]string, len(allAccountNames)+1)
	for i := range grid {
		grid[i] = make([]string, 33)
	}
	for i, name := range allAccountNames {
		nameMap[name] = i
		grid[i+1][0] = name
	}

	grid[0][0] = "姓名\\日期"
	grid[0][1] = "expired"
	for i := 0; i < 31; i++ {
		grid[0][i+2] = time.Now().Add(time.Duration(i) * 24 * time.Hour).Format("2006-01-02")
	}
	for _, exp := range expired {
		infoMap[(int64(nameMap[exp.Principal]+1)<<32)|1] = append(infoMap[int64(nameMap[exp.Principal]+1)<<32|1], exp.TaskId)
	}
	for i := 0; i < len(allAccountNames); i++ { //姓名  +1为grid表格行
		val := data[allAccountNames[i]]
		for j := 0; len(val) != 0 && j < 31; j++ { //日期   +2为grid表格
			if val[0].deadPoint == j {
				for val[0].deadPoint == j {
					if val[0].time > 8 {
						infoMap[(int64(i+1)<<32)|int64(j+2)] = append(infoMap[(int64(i+1)<<32)|int64(j+2)], fmt.Sprintf("%s (%d)", val[0].taskId, val[0].time))
					} else {
						infoMap[(int64(i+1)<<32)|int64(j+2)] = append(infoMap[(int64(i+1)<<32)|int64(j+2)], val[0].taskId)
					}
					val = val[1:]
					if len(val) == 0 {
						break
					}
				}
			} else {
				infoMap[(int64(i+1)<<32)|int64(j+2)] = append(infoMap[(int64(i+1)<<32)|int64(j+2)], val[0].taskId)
				val[0].time -= 8
				if val[0].time <= 0 {
					val = val[1:]
				}
			}

		}
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(grid), len(grid[0])
		},
		func() fyne.CanvasObject {
			//BUG:
			return widget.NewButton("", func() {})
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			btn := obj.(*widget.Button)
			//TODO:dus重置按钮状态
			btn.Hidden = false
			btn.OnTapped = nil
			btn.Importance = widget.LowImportance
			//tmpGrid : (n + 1) * 6\
			if id.Row == 0 || id.Col == 0 {
				btn.SetText(grid[id.Row][id.Col])
			} else if value, exists := infoMap[(int64(id.Row)<<32)|int64(id.Col)]; exists {
				if len(value) > 1 {
					btn.SetText(value[0] + "+")
				} else {
					btn.SetText(value[0])
				}

				if len(value) > 1 || strings.HasSuffix(value[0], ")") {
					btn.Importance = widget.DangerImportance
				}
				btn.OnTapped = func() {
					//dialog.ShowInformation("TaskInfo", strings.Join(value, "\r\n"), mw)
					popupWindow := fyne.CurrentApp().NewWindow("tasklist window")
					label := widget.NewLabel(strings.Join(value, "\r\n"))
					copyBtn := widget.NewButton("Copy", func() {
						clipboard := fyne.CurrentApp().Driver().AllWindows()[0].Clipboard()
						clipboard.SetContent(label.Text)
					})
					popupContent := container.NewBorder(nil, copyBtn, nil, nil, label)
					popupWindow.SetContent(popupContent)
					popupWindow.Resize(fyne.NewSize(300, 150))
					popupWindow.Show()
				}
			} else {
				btn.Hidden = true
			}
			btn.Refresh() // 强制刷新按钮
		})
	for k := 0; k < len(grid[0]); k++ {
		table.SetColumnWidth(k, 150)
	}

	return table
}

func createTeamGrid(data map[int][]*pb.Task, expired []*pb.Task, mw fyne.Window) fyne.CanvasObject {
	// 0        expired d1 d2 d3 d4 d5 ...
	// dusong1
	// dusong2
	infoMap := make(map[int64][]string)
	nameMap := make(map[string]int)
	grid := make([][]string, len(allAccountNames)+1)
	for i := range grid {
		grid[i] = make([]string, 33)
	}
	for i, name := range allAccountNames {
		nameMap[name] = i
		grid[i+1][0] = name
	}

	grid[0][0] = "姓名\\日期"
	grid[0][1] = "expired"
	for i := 0; i < 31; i++ {
		grid[0][i+2] = time.Now().Add(time.Duration(i) * 24 * time.Hour).Format("2006-01-02")
	}
	for _, exp := range expired {
		infoMap[(int64(nameMap[exp.Principal]+1)<<32)|0] = append(infoMap[int64(nameMap[exp.Principal]+1)<<32|0], exp.TaskId)
	}
	for day, tasks := range data {
		for _, task := range tasks {
			infoMap[(int64(nameMap[task.Principal]+1)<<32)|int64(day+1)] = append(infoMap[(int64(nameMap[task.Principal]+1)<<32)|int64(day+1)], task.TaskId)
		}
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(grid), len(grid[0])
		},
		func() fyne.CanvasObject {
			//BUG:
			return widget.NewButton("", func() {})
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			btn := obj.(*widget.Button)
			//TODO:dus重置按钮状态
			btn.Hidden = false
			btn.OnTapped = nil
			btn.Importance = widget.LowImportance
			//tmpGrid : (n + 1) * 6\
			if id.Row == 0 || id.Col == 0 {
				btn.SetText(grid[id.Row][id.Col])
			} else if value, exists := infoMap[(int64(id.Row)<<32)|int64(id.Col-1)]; exists {
				if len(value) > 1 {
					btn.SetText(value[0] + "+")
				} else {
					btn.SetText(value[0])
				}

				if len(value) > 1 {
					btn.Importance = widget.DangerImportance
				}
				btn.OnTapped = func() {
					dialog.ShowInformation("TaskInfo", strings.Join(value, "\r\n"), mw)
				}
			} else {
				btn.Hidden = true
			}
			btn.Refresh() // 强制刷新按钮
		})
	for k := 0; k < len(grid[0]); k++ {
		table.SetColumnWidth(k, 150)
	}

	return table
}

func addData(t *pb.Task, expired *[]*pb.Task, data map[int][]*pb.Task) {
	now := time.Now()
	taskDate, err := time.Parse("2006-01-02", t.Deadline)
	if err != nil {
		logrus.Warningf("could not parse date: %v", err)
		return
	}
	gap := int(taskDate.Sub(now.Truncate(24*time.Hour)).Hours() / 24) //任务时间距离现在有多少天
	if gap < 0 {
		*expired = append(*expired, t)
	} else {
		workDay := int(math.Ceil(float64(t.EstimatedWorkHours) / 8.0)) //预计完成天数（上取整）
		//fmt.Println(gap, workDay, t.TaskId)
		if gap-workDay+1 < 0 {
			for d := 0; d < workDay; d++ { //gap:2, workday:3
				data[min(gap, d)] = append(data[min(gap, d)], t)
			}
		} else {
			for d := gap - workDay + 1; d <= gap; d++ {
				data[d] = append(data[d], t)
			}
		}

	}
}

func ModForm(taskId string, client pb.ServiceClient) error {
	modTaskWindow := myapp.NewWindow("Update")

	reply, err := client.GetTaskById(context.Background(), &pb.GetTaskByIdRequest{
		TaskId: taskId,
	})
	if err != nil {
		return err
	}
	task := reply.T

	Deadline, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal := widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	Deadline.SetText((*task).Deadline)
	Deadline.Validator = func(in string) error {
		_, err := time.Parse("2006-01-02", Deadline.Text)
		if err != nil {
			return errors.New("deadline format error  Usage: 2006-01-02")
		}
		return nil
	}

	Comment.SetText(task.Comment)

	EmergencyLevel.SetText(fmt.Sprintf("%d", task.EmergencyLevel))
	emergencyLevel32 := task.EmergencyLevel
	EmergencyLevel.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		num, _ := strconv.Atoi(in)
		emergencyLevel32 = int32(num)
		return nil
	}

	EstimatedWorkHours.SetText(fmt.Sprintf("%d", task.EstimatedWorkHours))
	estimatedWorkHours64 := task.EstimatedWorkHours
	EstimatedWorkHours.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		tmp, _ := strconv.Atoi(in)
		estimatedWorkHours64 = int32(tmp)
		return nil
	}
	State.SetText(task.State)
	State.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		return nil
	}
	Type.SetText(fmt.Sprintf("%d", task.TypeId))
	typeid32 := task.TypeId
	Type.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		tmp, _ := strconv.Atoi(in)
		typeid32 = int32(tmp)
		return nil
	}
	Principal.SetText(task.Principal)
	Principal.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		return nil
	}

	// Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal
	isSucceed := make(chan error)

	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		go func() {
			_, err := client.DelTask(context.Background(), &pb.DelTaskRequest{TaskNo: task.TaskId, User: config.Cfg.Login.UserName, Principal: task.Principal})
			if err == nil {
				logChain.append("del", task, nil)
			}
			isSucceed <- err
			modTaskWindow.Close()

		}()
	})
	delBtn.Importance = widget.HighImportance
	//
	//idLabel := widget.NewLabel(task.TaskId)
	//idLabel.TextStyle = fyne.TextStyle{Bold: true}
	//
	//reqNoLabel := widget.NewLabel(task.ReqNo)
	//reqNoLabel.TextStyle = fyne.TextStyle{Bold: true}
	idEty, reqNoEty := widget.NewEntry(), widget.NewEntry()
	idEty.SetText(task.TaskId)
	idEty.Disable()
	reqNoEty.SetText(task.ReqNo)
	reqNoEty.Disable()
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "任务单号", Widget: idEty},
			{Text: "需求号", Widget: reqNoEty},
			{Text: "截止日期", Widget: Deadline},
			{Text: "负责人", Widget: Principal},
			{Text: "预计工时(8h/d)", Widget: EstimatedWorkHours},
			{Text: "紧急程度", Widget: EmergencyLevel},
			{Text: "任务描述", Widget: Comment},
			{Text: "任务类型", Widget: Type},
			{Text: "任务状态", Widget: State},
			{Widget: delBtn},
		},
		OnSubmit: func() {

			newTask := &pb.Task{
				TaskId:             task.TaskId,
				Comment:            Comment.Text,
				Deadline:           Deadline.Text,
				EmergencyLevel:     emergencyLevel32,
				EstimatedWorkHours: estimatedWorkHours64,
				State:              State.Text,
				TypeId:             typeid32,
				ReqNo:              task.ReqNo,
				Principal:          Principal.Text,
			}
			_, err := client.ModTask(context.Background(), &pb.ModTaskRequest{T: newTask, User: config.Cfg.Login.UserName})
			if err == nil {
				logChain.append("update", task, newTask)
			}
			isSucceed <- err
		},
		OnCancel: func() {
			isSucceed <- nil
		},
	}

	modTaskWindow.SetContent(form)
	modTaskWindow.Resize(fyne.NewSize(300, 200))
	modTaskWindow.Show()

	modTaskWindow.SetOnClosed(func() {
		isSucceed <- nil
	})
	ret := <-isSucceed
	modTaskWindow.Close()

	return ret
}

func addForm(client pb.ServiceClient) *pb.Task {
	addTaskWindow := myapp.NewWindow("Update")

	idEty, deadlineEty, reqNoEty, commentEty, emergencyLevelEty, estimatedWorkHoursEty, stateEty, typeEty, principalEty := widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()

	deadlineEty.SetText(time.Now().Format("2006-01-02"))
	commentEty.SetText("null")
	emergencyLevelEty.SetText("0")
	estimatedWorkHoursEty.SetText("4")
	stateEty.SetText("带启动")
	typeEty.SetText("0")
	principalEty.SetText(config.Cfg.Login.UserName) //TODO: 默认登录者自己的名字，最后考虑权限的问题

	deadlineEty.Validator = func(in string) error {
		_, err := time.Parse("2006-01-02", in)
		if err != nil {
			return errors.New("deadline format error  Usage: 2006-01-02")
		}
		return nil
	}
	idEty.Validator = func(in string) error {
		if idEty.Text == "" {
			return errors.New("can not be empty")
		}
		return nil
	}
	reqNoEty.Validator = func(in string) error {
		if reqNoEty.Text == "" {
			return errors.New("can not be empty")
		}
		return nil
	}

	retChan := make(chan *pb.Task)
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "任务单号", Widget: idEty},
			{Text: "需求号", Widget: reqNoEty},
			{Text: "截止日期", Widget: deadlineEty},
			{Text: "负责人", Widget: principalEty},
			{Text: "预计工时", Widget: estimatedWorkHoursEty},
			{Text: "紧急程度", Widget: emergencyLevelEty},
			{Text: "任务描述", Widget: commentEty},
			{Text: "任务类型", Widget: typeEty},
			{Text: "任务状态", Widget: stateEty},
		},
		OnSubmit: func() { // optional, handle form submission
			numType, _ := strconv.Atoi(typeEty.Text)
			taskType := int32(numType)
			numHours, _ := strconv.Atoi(estimatedWorkHoursEty.Text)
			estimatedWorkHours := int32(numHours)
			numLevel, _ := strconv.Atoi(emergencyLevelEty.Text)
			emergencyLevel := int32(numLevel)

			newTask := &pb.Task{
				TaskId:             idEty.Text,
				Comment:            commentEty.Text,
				Deadline:           deadlineEty.Text,
				EmergencyLevel:     emergencyLevel,
				EstimatedWorkHours: estimatedWorkHours,
				State:              stateEty.Text,
				TypeId:             taskType,
				ReqNo:              reqNoEty.Text,
				Principal:          principalEty.Text,
			}

			request := &pb.AddTaskRequest{T: newTask, User: config.Cfg.Login.UserName}
			_, err := client.AddTask(context.Background(), request)
			if err != nil {
				dialog.NewInformation("error", err.Error(), addTaskWindow).Show()
			} else {
				logChain.append("add", nil, request)
				retChan <- newTask
				addTaskWindow.Close()
			}
		},
		OnCancel: func() {
			retChan <- nil
			addTaskWindow.Close()
		},
	}

	addTaskWindow.SetContent(form)
	addTaskWindow.Resize(fyne.NewSize(300, 200))
	addTaskWindow.Show()

	addTaskWindow.SetOnClosed(func() {
		retChan <- nil
	})
	return <-retChan

}

func loadTeamGrid(client pb.ServiceClient, mw fyne.Window) (data map[int][]*pb.Task, expired []*pb.Task) {
	data = make(map[int][]*pb.Task)
	expired = make([]*pb.Task, 0)
	reply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
		return
	}
	for _, task := range reply.Tasks {
		addData(task, &expired, data)
	}
	//log.Println("data dict size: " + strconv.Itoa(len(data)))
	return data, expired
}

type tidyValue struct {
	taskId    string
	time      int8
	deadPoint int
}

type soredTV []*tidyValue

func (stv soredTV) Len() int {
	return len(stv)
}
func (stv soredTV) Less(i, j int) bool {
	return stv[i].deadPoint < stv[j].deadPoint
}
func (stv soredTV) Swap(i, j int) {
	stv[i], stv[j] = stv[j], stv[i]
}

func loadTidyTeamGrid(client pb.ServiceClient, mw fyne.Window) (data map[string]soredTV, expired []*pb.Task) {
	data = make(map[string]soredTV)
	expired = make([]*pb.Task, 0)
	reply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
		return
	}
	for _, task := range reply.Tasks {
		addTidyData(task, &expired, data)
	}
	//log.Println("data dict size: " + strconv.Itoa(len(data)))
	for _, v := range data {
		sort.Sort(v)
	}
	return data, expired
}

func addTidyData(task *pb.Task, exp *[]*pb.Task, data map[string]soredTV) {
	now := time.Now()
	taskDate, err := time.Parse("2006-01-02", task.Deadline)
	if err != nil {
		logrus.Warningf("could not parse date: %v", err)
		return
	}
	gap := int(taskDate.Sub(now.Truncate(24*time.Hour)).Hours() / 24) //任务时间距离现在有多少天
	if gap < 0 {
		*exp = append(*exp, task)
	} else {
		data[task.Principal] = append(data[task.Principal], &tidyValue{task.TaskId, int8(task.EstimatedWorkHours), gap})
	}
}

func loadAccountGrid(client pb.ServiceClient, mw fyne.Window, name string) (data map[int][]*pb.Task, expired []*pb.Task) {
	data = make(map[int][]*pb.Task)
	expired = make([]*pb.Task, 0)
	reply, err := client.GetTaskListByName(context.Background(), &pb.GetTaskListOneRequest{Name: name})
	if err != nil {
		dialog.ShowError(err, mw)
		return
	}
	for _, task := range reply.Tasks {
		addData(task, &expired, data)
	}
	return data, expired
}
