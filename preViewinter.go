package main

import (
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
	"image/color"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	AccountView = 0
	TeamView    = 1
)

func CreatePreviewInterface(appTab *container.AppTabs, client pb.ServiceClient, mw fyne.Window) *fyne.Container {
	now := time.Now()
	var retInterface *fyne.Container
	curView := AccountView //0:个人， 1：团队
	// 创建一个网格布局，每行显示一天的任务
	viewPage := 0
	viewGrid := make([]*fyne.Container, 7)
	for i := range viewGrid {
		viewGrid[i] = container.NewGridWithRows(1)
	}

	data, expired := loadAccountGrid(client, mw)

	list := make(map[int]*widget.List, 31)
	expiredList := widget.NewList(
		// 获取列表项的数量
		func() int {
			return len(expired)
		},
		// 创建列表项的模板
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		// 更新列表项的内容
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("%s : %s", expired[i].TaskId, expired[i].Principal))
		},
	)

	expiredList.OnSelected = func(id widget.ListItemID) {
		succeed := ModForm(&expired[id], client)
		fmt.Println(succeed)
		if succeed == 0 { //update
			updateItem := expired[id]
			expired = append(expired[:id], expired[id+1:]...)
			newDateId := addData(updateItem, &expired, data)
			expiredList.Refresh()
			if newDateId != -1 {
				list[newDateId].Refresh()
				retInterface.Refresh()
			}
		} else if succeed == 1 { //delete
			expired = append(expired[:id], expired[id+1:]...)
			retInterface.Refresh()
		}

	}
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
				return widget.NewLabel("Template")
			},
			// 更新列表项的内容
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(fmt.Sprintf("%s : %s", data[d][i].TaskId, data[d][i].Principal))
			},
		)
		list[d].OnSelected = func(id widget.ListItemID) {
			succeed := ModForm(&data[d][id], client)
			if succeed == 0 { //update
				updateItem := data[d][id]
				data[d] = append(data[d][:id], data[d][id+1:]...)
				addData(updateItem, &expired, data)
				expiredList.Refresh()
				list[d].Refresh()
				retInterface.Refresh()
			} else if succeed == 1 { //delete
				data[d] = append(data[d][:id], data[d][id+1:]...)
				retInterface.Refresh()
			}
		}

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
			retInterface.Refresh()
		}
	})
	importBtn := widget.NewButtonWithIcon("", theme.UploadIcon(), func() {
		importWd := myapp.NewWindow("import")
		input := widget.NewMultiLineEntry()
		input.Resize(fyne.NewSize(600, 400))

		output := widget.NewMultiLineEntry()
		output.Resize(fyne.NewSize(600, 400))
		outputChan := make(chan string)
		input.Validator = func(s string) error {
			if s == "" {
				return errors.New("can not be empty")
			}
			return nil
		}
		form := &widget.Form{
			Items: []*widget.FormItem{
				{Text: "input:", Widget: input},
				{Text: "result", Widget: output},
			},
			OnSubmit: func() {
				paths := strings.Split(input.Text, "\n")
				for _, path := range paths {
					go ImportXLSForTaskList(path, client, outputChan)
				}
				go func() {
					for res := range outputChan {
						output.Append(res + "\n\r")
					}
				}()
			},
			OnCancel: func() {
				importWd.Close()
			},
		}
		importWd.SetContent(form)
		importWd.Resize(fyne.NewSize(600, 400))
		importWd.Show()
	})
	accountBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		curView = AccountView
		data, expired = loadAccountGrid(client, mw)
		retInterface.Refresh()
	})
	teamBtn := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		curView = TeamView
		data, expired = loadTeamGrid(client, mw)
		retInterface.Refresh()
	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		switch curView {
		case AccountView:
			data, expired = loadAccountGrid(client, mw)
			retInterface.Refresh()
		case TeamView:
			data, expired = loadTeamGrid(client, mw)
			retInterface.Refresh()
		}
	})
	prevPageBtn := widget.NewButtonWithIcon("", theme.MediaFastRewindIcon(), func() {
		viewPage = max(viewPage-1, 0)
		updateCurViewGrid()
		fmt.Println(viewPage)
	})
	nextPageBtn := widget.NewButtonWithIcon("", theme.MediaFastForwardIcon(), func() {
		viewPage = min(viewPage+1, 6)
		updateCurViewGrid()
		fmt.Println(viewPage)
	})
	btnBar := container.NewHBox(addBtn, importBtn, accountBtn, teamBtn, flushBtn, layout.NewSpacer(), prevPageBtn, nextPageBtn)
	bg := canvas.NewRectangle(color.RGBA{R: 217, G: 213, B: 213, A: 255})
	topBtn = container.NewStack(bg, btnBar)

	retInterface = container.NewBorder(topBtn, nil, nil, nil, viewGrid[0])
	return retInterface
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
		data[weekday] = append(data[weekday], t)
	}
	return weekday
}

func ModForm(task **pb.Task, client pb.ServiceClient) int {
	modTaskWindow := myapp.NewWindow("Update")

	Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	Id.SetText((*task).TaskId)
	ReqNo.SetText((*task).ReqNo)
	Deadline.SetText((*task).Deadline)
	Deadline.Validator = func(in string) error {
		_, err := time.Parse("2006-01-02", Deadline.Text)
		if err != nil {
			return errors.New("deadline format error  Usage: 2006-01-02")
		}
		return nil
	}

	Comment.SetText((*task).Comment)

	EmergencyLevel.SetText(fmt.Sprintf("%d", (*task).EmergencyLevel))
	emergencyLevel32 := (*task).EmergencyLevel
	EmergencyLevel.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		num, _ := strconv.Atoi(in)
		emergencyLevel32 = int32(num)
		return nil
	}

	EstimatedWorkHours.SetText(fmt.Sprintf("%d", (*task).EstimatedWorkHours))
	estimatedWorkHours64 := (*task).EstimatedWorkHours
	EstimatedWorkHours.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		tmp, _ := strconv.Atoi(in)
		estimatedWorkHours64 = int64(tmp)
		return nil
	}
	State.SetText((*task).State)
	State.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		return nil
	}
	Type.SetText(fmt.Sprintf("%d", (*task).TypeId))
	typeid32 := (*task).TypeId
	Type.Validator = func(in string) error {
		if in == "" {
			return errors.New("can not be empty")
		}
		tmp, _ := strconv.Atoi(in)
		typeid32 = int32(tmp)
		return nil
	}
	Principal.SetText((*task).Principal)
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
	isSucceed := make(chan int) //-1 : 失败， 0：update  1 ： delete

	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		dialog.NewConfirm("Please Confirm", "Are you sure to delete", func(confirm bool) {
			if confirm {
				_, err := client.DelTask(context.Background(), &pb.DelTaskRequest{TaskNo: (*task).TaskId})
				if err != nil {
					log.Printf("error deleting task: %v", err)
					dialog.ShowError(err, modTaskWindow)
					isSucceed <- -1
				} else {
					isSucceed <- 1
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
			{Text: "预计工时", Widget: EstimatedWorkHours},
			{Text: "紧急程度", Widget: EmergencyLevel},
			{Text: "任务描述", Widget: Comment},
			{Text: "任务类型", Widget: Type},
			{Text: "任务状态", Widget: State},
			{Widget: delBtn},
		},
		OnSubmit: func() {

			newTask := &pb.Task{
				TaskId:             (*task).TaskId,
				Comment:            Comment.Text,
				Deadline:           Deadline.Text,
				EmergencyLevel:     emergencyLevel32,
				EstimatedWorkHours: estimatedWorkHours64,
				State:              State.Text,
				TypeId:             typeid32,
				ReqNo:              (*task).ReqNo,
				Principal:          Principal.Text,
			}
			*task = newTask
			_, err := client.ModTask(context.Background(), &pb.ModTaskRequest{T: newTask})
			if err != nil {
				isSucceed <- -1
				log.Println(err)
				return
			} else {
				isSucceed <- 0
				log.Println("update succeed")
			}
			modTaskWindow.Close()
		},
		OnCancel: func() {
			isSucceed <- -1
			modTaskWindow.Close()
		},
	}

	modTaskWindow.SetContent(form)
	modTaskWindow.Resize(fyne.NewSize(300, 200))
	modTaskWindow.Show()

	modTaskWindow.SetOnClosed(func() {
		isSucceed <- -1
	})
	return <-isSucceed
}

func addForm(client pb.ServiceClient) *pb.Task {
	addTaskWindow := myapp.NewWindow("Update")

	idEty, deadlineEty, reqNoEty, commentEty, emergencyLevelEty, estimatedWorkHoursEty, stateEty, typeEty, principalEty := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()

	deadlineEty.SetText(time.Now().Format("2006-01-02"))
	commentEty.SetText("null")
	emergencyLevelEty.SetText("0")
	estimatedWorkHoursEty.SetText("72")
	stateEty.SetText("带启动")
	typeEty.SetText("0")
	principalEty.SetText(UserName) //TODO: 默认登录者自己的名字，最后考虑权限的问题

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

			_, err := client.AddTask(context.Background(), &pb.AddTaskRequest{T: newTask})
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
	}
	for _, task := range reply.Tasks {
		addData(task, &expired, data)
	}
	return data, expired
}
func loadAccountGrid(client pb.ServiceClient, mw fyne.Window) (data map[int][]*pb.Task, expired []*pb.Task) {
	data = make(map[int][]*pb.Task)
	expired = make([]*pb.Task, 0)
	reply, err := client.GetTaskListOne(context.Background(), &pb.GetTaskListOneRequest{Name: UserName})
	if err != nil {
		dialog.ShowError(err, mw)
	}
	for _, task := range reply.Tasks {
		addData(task, &expired, data)
	}
	return data, expired
}
