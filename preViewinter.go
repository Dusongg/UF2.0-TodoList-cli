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
	"math"
	"strconv"
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

func CreatePreviewInterface(appTab *container.AppTabs, client pb.ServiceClient, mw fyne.Window, msgChan <-chan string) *fyne.Container {
	now := time.Now()
	var previewInterface *fyne.Container
	curView := AccountView //0:个人， 1：团队
	// 创建一个网格布局，每行显示一天的任务
	viewPage := 0
	viewGrid := make([]*fyne.Container, 7)
	for i := range viewGrid {
		viewGrid[i] = container.NewGridWithRows(1)
	}

	data, expired := loadAccountGrid(client, mw, config.Cfg.Login.UserName)

	flushInterface := func() {
		switch curView {
		case AccountView:
			data, expired = loadAccountGrid(client, mw, config.Cfg.Login.UserName)
			previewInterface.Refresh()
		case TeamView:
			data, expired = loadTeamGrid(client, mw)
			previewInterface.Refresh()
		}
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

	var topBtn *fyne.Container
	var bottom *fyne.Container
	updateCurViewGrid := func() {
		appTab.Items[0].Content = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[viewPage])
		appTab.Refresh()
	}

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
	accountBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		curView = AccountView
		if accountEty.Text != "" {
			//TODO:没有考虑用户是否存在
			data, expired = loadAccountGrid(client, mw, accountEty.Text)
		} else {
			data, expired = loadAccountGrid(client, mw, config.Cfg.Login.UserName)
		}
		previewInterface.Refresh()
	})
	accountBtnWithBg := container.NewStack(canvas.NewRectangle(config.ColorTheme1), accountBtn)

	//accountBg := canvas.NewRectangle(colorTheme1)
	//accountBox := container.NewStack(accountBg, container.NewHBox(accountBtn, accountEty, nil))

	teamBtn := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		curView = TeamView
		data, expired = loadTeamGrid(client, mw)
		previewInterface.Refresh()
	})
	flushActivity := widget.NewActivity()
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		flushActivity.Start()
		flushInterface()
		flushActivity.Stop()
	})
	prevPageBtn := widget.NewButtonWithIcon("", theme.MediaFastRewindIcon(), func() {
		viewPage = max(viewPage-1, 0)
		updateCurViewGrid()
	})
	nextPageBtn := widget.NewButtonWithIcon("", theme.MediaFastForwardIcon(), func() {
		viewPage = min(viewPage+1, 6)
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
	previewInterface = container.NewBorder(topBtn, bottom, nil, nil, viewGrid[0])
	return previewInterface
}

func addData(t *pb.Task, expired *[]*pb.Task, data map[int][]*pb.Task) int {
	now := time.Now()
	taskDate, err := time.Parse("2006-01-02", t.Deadline)
	if err != nil {
		logrus.Warningf("could not parse date: %v", err)
		return -1
	}
	weekday := int(taskDate.Sub(now.Truncate(24*time.Hour)).Hours() / 24) //任务时间距离现在有多少天
	if weekday < 0 {
		*expired = append(*expired, t)
	} else {
		low := max(0, weekday-(int(math.Ceil(float64(t.EstimatedWorkHours)/8.0))-1))
		for d := low; d <= weekday; d++ {
			data[d] = append(data[d], t)
		}
	}
	return weekday
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
		estimatedWorkHours64 = int64(tmp)
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
	estimatedWorkHoursEty.SetText("24")
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
			estimatedWorkHours := int64(numHours)
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
