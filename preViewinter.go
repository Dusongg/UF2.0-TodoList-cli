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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"log"
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
	color.RGBA{R: 247, G: 153, B: 174, A: 255},
	color.RGBA{R: 79, G: 179, B: 129, A: 255},
	color.RGBA{R: 180, G: 222, B: 180, A: 255},
	color.RGBA{G: 146, B: 199, A: 255},
	color.RGBA{120, 94, 0, 255},
}

func CreatePreviewInterface(appTab *container.AppTabs, client pb.ServiceClient, mw fyne.Window) *fyne.Container {

	now := time.Now()
	var previewInterface *fyne.Container
	curView := AccountView //0:个人， 1：团队
	// 创建一个网格布局，每行显示一天的任务
	viewPage := 0
	viewGrid := make([]*fyne.Container, 7)
	for i := range viewGrid {
		viewGrid[i] = container.NewGridWithRows(1)
	}

	data, expired := loadAccountGrid(client, mw, config.LoginUser)

	flushInterface := func() {
		switch curView {
		case AccountView:
			data, expired = loadAccountGrid(client, mw, config.LoginUser)
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
				succeed := ModForm(expired[id].TaskId, client)
				if succeed {
					flushInterface()
				}
			}
		},
	)

	expiredLabel := canvas.NewText("Expired", theme.Color(theme.ColorNameForeground))
	// 设置Label的字体颜色
	expiredLabel.TextStyle = fyne.TextStyle{Bold: true}
	expiredLabel.Color = color.White
	bgTheme1 := canvas.NewRectangle(colorTheme1)
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
					succeed := ModForm(data[d][id].TaskId, client)
					if succeed {
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

		viewGrid[d/5].Add(container.NewBorder(container.NewStack(bgTheme1, dateLabel), nil, nil, nil, list[d]))
	}

	var topBtn *fyne.Container
	updateCurViewGrid := func() {
		appTab.Items[0].Content = container.NewBorder(topBtn, nil, nil, nil, viewGrid[viewPage])
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
			data, expired = loadAccountGrid(client, mw, config.LoginUser)
		}
		previewInterface.Refresh()
	})
	accountBtnWithBg := container.NewStack(canvas.NewRectangle(colorTheme1), accountBtn)

	//accountBg := canvas.NewRectangle(colorTheme1)
	//accountBox := container.NewStack(accountBg, container.NewHBox(accountBtn, accountEty, nil))

	teamBtn := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		curView = TeamView
		data, expired = loadTeamGrid(client, mw)
		previewInterface.Refresh()
	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		flushInterface()
	})
	prevPageBtn := widget.NewButtonWithIcon("", theme.MediaFastRewindIcon(), func() {
		viewPage = max(viewPage-1, 0)
		updateCurViewGrid()
	})
	nextPageBtn := widget.NewButtonWithIcon("", theme.MediaFastForwardIcon(), func() {
		viewPage = min(viewPage+1, 6)
		updateCurViewGrid()
	})

	btnBar := container.NewBorder(nil, nil, container.NewHBox(addBtn, importBtn, flushBtn, teamBtn, accountBtnWithBg), container.NewHBox(prevPageBtn, nextPageBtn), container.NewStack(canvas.NewRectangle(colorTheme1), accountEty))
	bg := canvas.NewRectangle(color.RGBA{R: 217, G: 213, B: 213, A: 255})
	topBtn = container.NewStack(bg, btnBar)

	previewInterface = container.NewBorder(topBtn, nil, nil, nil, viewGrid[0])
	return previewInterface
}

func addData(t *pb.Task, expired *[]*pb.Task, data map[int][]*pb.Task) int {
	now := time.Now()
	taskDate, err := time.Parse("2006-01-02", t.Deadline)
	if err != nil {
		log.Printf("could not parse date: %v", err)
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

func ModForm(taskId string, client pb.ServiceClient) bool {
	modTaskWindow := myapp.NewWindow("Update")

	reply, err := client.QueryTaskWithField(context.Background(), &pb.QueryTaskWithFieldRequest{
		Field:      "task_id",
		FieldValue: taskId,
	})
	if err != nil {
		dialog.ShowError(err, modTaskWindow)
		return false
	}
	tasks := reply.Tasks
	if tasks == nil || len(tasks) == 0 {
		dialog.ShowError(errors.New("数据库中查询不到该id的任务 ："+taskId), modTaskWindow)
		return false
	}
	task := tasks[0]

	Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal := widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	Id.SetText(task.TaskId)
	ReqNo.SetText(task.ReqNo)
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

	Id.Disable()    // 设置为只读
	ReqNo.Disable() // 设置为只读
	//Deadline.Disable()           // 设置为只读
	//Comment.Disable()            // 设置为只读
	//EmergencyLevel.Disable()     // 设置为只读
	//EstimatedWorkHours.Disable() // 设置为只读
	//State.Disable()              // 设置为只读
	//Type.Disable()               // 设置为只读
	//Principal.Disable()          // 设置为只读

	// Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal
	isSucceed := make(chan bool)

	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		dialog.NewConfirm("Please Confirm", "Are you sure to delete", func(confirm bool) {
			if confirm {
				_, err := client.DelTask(context.Background(), &pb.DelTaskRequest{TaskNo: task.TaskId, User: config.LoginUser, Principal: task.Principal})
				if err != nil {
					log.Printf("error deleting task: %v", err)
					dialog.ShowError(err, modTaskWindow)
					isSucceed <- false
					return
				} else {
					isSucceed <- true
					modTaskWindow.Close()
					return
				}

			} else {
				return
			}
		}, modTaskWindow).Show()

	})
	delBtn.Importance = widget.HighImportance

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "任务单号", Widget: Id},
			{Text: "需求号", Widget: ReqNo},
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
			_, err := client.ModTask(context.Background(), &pb.ModTaskRequest{T: newTask, User: config.LoginUser})
			if err != nil {
				isSucceed <- false
				log.Println(err)
				return
			} else {
				isSucceed <- true
				log.Println("update succeed")
			}
			modTaskWindow.Close()
		},
		OnCancel: func() {
			isSucceed <- false
			modTaskWindow.Close()
		},
	}

	modTaskWindow.SetContent(form)
	modTaskWindow.Resize(fyne.NewSize(300, 200))
	modTaskWindow.Show()

	modTaskWindow.SetOnClosed(func() {
		isSucceed <- false
	})
	return <-isSucceed
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
	principalEty.SetText(config.LoginUser) //TODO: 默认登录者自己的名字，最后考虑权限的问题

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

			_, err := client.AddTask(context.Background(), &pb.AddTaskRequest{T: newTask, User: config.LoginUser})
			if err != nil {
				dialog.NewInformation("error", err.Error(), addTaskWindow).Show()
			} else {
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
	return data, expired
}

func loadAccountGrid(client pb.ServiceClient, mw fyne.Window, name string) (data map[int][]*pb.Task, expired []*pb.Task) {
	data = make(map[int][]*pb.Task)
	expired = make([]*pb.Task, 0)
	reply, err := client.GetTaskListOne(context.Background(), &pb.GetTaskListOneRequest{Name: name})
	if err != nil {
		dialog.ShowError(err, mw)
		return
	}
	for _, task := range reply.Tasks {
		addData(task, &expired, data)
	}
	return data, expired
}
