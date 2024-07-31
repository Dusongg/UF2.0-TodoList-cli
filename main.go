package main

import (
	"OrderManager-cli/pb"
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"image/color"
	"log"
	"strconv"
	"strings"
	"time"
)

//2024.7.30
//TODO: 1. 改用map存储，展示近五天  √
//TODO: 2. 新增一行：添加任务 + 下一页   √
//TODO: 3. 添加任务选项    √
//TODO: 4. 添加或者就该任务之后,对于本地以及其他用户的界面刷新数据问题
//TODO: 5. 导入数据按钮    √
//TODO: 6. 处理过期任务    √

//2024.7.31
//TODO: 1. 批量导入数据	√
//TODO: 2. 收件箱界面     %50
//TODO: 3. 表单点击输入框将默认值写出来  √
//TODO: 4. 登录 & 自动登录
//TODO: 5. 删除操作   √

//2024.8.1
//TODO 1. 完成收件箱界面
//TODO 2. 完成“today”姐买你

const DAYSPERPAGE = 5

var colorTheme1 = color.RGBA{R: 57, G: 72, B: 94, A: 255}

func main() {

	myapp := app.New()
	mw := myapp.NewWindow("Task List for the Week")
	//月光石主题:深-》浅

	// 建立一个链接，请求A服务
	// 真实项目里肯定是通过配置中心拿服务名称，发给注册中心请求真实的A服务地址，这里都是模拟
	// 第二个参数是配置了一个证书，因为没有证书会报错，但是我们目前没有配置证书，所以需要insecure.NewCredentials()返回一个禁用传输安全的凭据
	connect, err := grpc.NewClient(":8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer connect.Close()
	client := pb.NewServiceClient(connect)

	reply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})

	if err != nil {
		log.Fatalf("could not get tasks: %v", err)
	}

	now := time.Now()
	// 创建一个网格布局，每行显示一天的任务
	viewPage := 0
	viewGrid := make([]*fyne.Container, 7)
	for i := range viewGrid {
		viewGrid[i] = container.NewGridWithRows(1)
	}
	appTab := container.NewAppTabs(
		container.NewTabItemWithIcon("预览", theme.ListIcon(), viewGrid[0]),
		container.NewTabItemWithIcon("收件箱", theme.StorageIcon(), container.NewVScroll(widget.NewLabel("TODO"))),
		container.NewTabItemWithIcon("今天", theme.VisibilityIcon(), widget.NewLabel("TODO")),
		//container.NewTabItem("库", widget.NewLabel("TODO")),
	)
	appTab.SetTabLocation(container.TabLocationLeading) //竖着的标签

	var inboxGrid fyne.CanvasObject
	appTab.OnSelected = func(item *container.TabItem) {
		if item == appTab.Items[1] && inboxGrid == nil {
			inboxGrid = CreateInBox(reply.Tasks)
			appTab.Items[1].Content = inboxGrid
		}
	}

	// 创建每天的任务列表
	data := make(map[int][]*pb.Task)
	list := make(map[int]*widget.List, 31)
	expired := make([]*pb.Task, 0)
	var expiredList *widget.List

	for _, task := range reply.Tasks {
		addData(task, &expired, data)
	}

	expiredList = widget.NewList(
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
		succeed := ModForm(myapp, &expired[id], client)
		fmt.Println(succeed)
		if succeed == 0 { //update
			updateItem := expired[id]
			expired = append(expired[:id], expired[id+1:]...)
			newDateId := addData(updateItem, &expired, data)
			expiredList.Refresh()
			if newDateId != -1 {
				list[newDateId].Refresh()
				appTab.Refresh()
			}
		} else if succeed == 1 { //delete
			expired = append(expired[:id], expired[id+1:]...)
			appTab.Refresh()
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
			succeed := ModForm(myapp, &data[d][id], client)
			if succeed == 0 { //update
				updateItem := data[d][id]
				data[d] = append(data[d][:id], data[d][id+1:]...)
				addData(updateItem, &expired, data)
				expiredList.Refresh()
				list[d].Refresh()
				appTab.Refresh()
			} else if succeed == 1 { //delete
				data[d] = append(data[d][:id], data[d][id+1:]...)
				appTab.Refresh()
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
		newTask := addForm(myapp, client)
		if newTask != nil {
			addData(newTask, &expired, data)
			appTab.Refresh()
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
	btnBar := container.NewHBox(addBtn, importBtn, layout.NewSpacer(), prevPageBtn, nextPageBtn)
	bg := canvas.NewRectangle(color.RGBA{217, 213, 213, 255})
	topBtn = container.NewStack(bg, btnBar)

	appTab.Items[0].Content = container.NewBorder(topBtn, nil, nil, nil, viewGrid[0])

	// 设置窗口内容
	mw.SetContent(appTab)
	// 设置窗口大小并显示
	mw.Resize(fyne.NewSize(1000, 600))
	mw.ShowAndRun()

}

type task struct {
	comment            string
	taskId             string
	emergencyLevel     int32
	deadline           string
	principal          string
	reqNo              string
	estimatedWorkHours int64
	state              string
	typeId             int32
}

type patch struct {
	patchNo    string
	reqNo      string
	describe   string
	clientName string
	deadline   string
	reason     string
	sponsor    string
}

// 规范化导出文件的导入
// 修改单信息： task_id,  principal,s tate,   升级说明： task_id, req_no,comment

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

func ModForm(myapp fyne.App, task **pb.Task, client pb.ServiceClient) int {
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

func addForm(myapp fyne.App, client pb.ServiceClient) *pb.Task {
	addTaskWindow := myapp.NewWindow("Update")

	idEty, deadlineEty, reqNoEty, commentEty, emergencyLevelEty, estimatedWorkHoursEty, stateEty, typeEty, principalEty := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()

	deadlineEty.SetText(time.Now().Format("2006-01-02"))
	commentEty.SetText("null")
	emergencyLevelEty.SetText("0")
	estimatedWorkHoursEty.SetText("72")
	stateEty.SetText("带启动")
	typeEty.SetText("0")
	principalEty.SetText("myself") //TODO: 默认登录者自己的名字，最后考虑权限的问题

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
