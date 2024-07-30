package main

import (
	"OrderManager-cli/pb"
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/extrame/xls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"image/color"
	"log"
	"strconv"
	"time"
)

//2024.7.30
//TODO: 1. 改用map存储，展示近五天  √
//TODO: 2. 新增一行：添加任务 + 下一页   √
//TODO: 3. 添加任务选项
//TODO: 4. 添加或者就该任务之后,对于本地以及其他用户的界面刷新数据问题
//TODO: 5. 导入数据按钮    √
//TODO: 6. 处理过期任务    √
//TODO: 7. 给任务添加颜色

//2024.7.31
//TODO: 1. 批量导入数据
//TODO: 2. 收件箱界面

const DAYSPERPAGE = 5

func main() {

	myapp := app.New()
	mw := myapp.NewWindow("Task List for the Week")
	//月光石主题:深-》浅
	colorTheme1 := color.RGBA{57, 72, 94, 255}

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
		fmt.Println(expired[id].Deadline)
		succeed := updateForm(myapp, &expired[id], client)
		fmt.Println(expired[id].Deadline)
		fmt.Println(succeed)
		if succeed {
			updateItem := expired[id]
			expired = append(expired[:id], expired[id+1:]...)
			newdateid := addData(updateItem, &expired, data)
			expiredList.Refresh()
			if newdateid != -1 {
				list[newdateid].Refresh()
				appTab.Refresh()
			}
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
			succeed := updateForm(myapp, &data[d][id], client)
			if succeed {
				updateItem := data[d][id]
				data[d] = append(data[d][:id], data[d][id+1:]...)
				addData(updateItem, &expired, data)
				expiredList.Refresh()
				list[d].Refresh()
				appTab.Refresh()
			}
		}

		// TODO:添加
		//addButton := widget.NewButton("Add Item", func() {
		//	//TODO:添加
		//	fmt.Println("TODO")
		//	list[d].Refresh() // 刷新列表以显示新元素
		//})

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

	addBtn := widget.NewButtonWithIcon("添加任务", theme.ContentAddIcon(), func() {
		addForm(myapp, client)
	})
	importBtn := widget.NewButtonWithIcon("批量导入", theme.UploadIcon(), func() {
		fileDialog := dialog.NewFileOpen(
			func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, mw)
					return
				}
				if reader == nil {
					return
				}

				// Do something with the selected file
				fmt.Println("Selected file:", reader.URI().Path())
				defer reader.Close()
			}, mw)

		// Set file dialog properties
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".txt", ".md"})) // Filter by file extension
		fileDialog.Show()
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

func addData(t *pb.Task, expired *[]*pb.Task, data map[int][]*pb.Task) int {
	now := time.Now()
	taskDate, err := time.Parse("2006-01-02", t.Deadline)
	if err != nil {
		log.Printf("could not parse date: %v", err)
		return -1
	}
	weekday := int(taskDate.Sub(now).Hours() / 24) //任务时间距离现在有多少天
	if weekday < 0 {
		*expired = append(*expired, t)
	} else {
		data[weekday] = append(data[weekday], t)
	}
	return weekday
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
func importXLSForTaskList(xlsFile string, client pb.ServiceClient) {
	// 打开.xls文件
	workbook, err := xls.Open("./xlsFile/规范化导出文件.xls", "utf-8")
	if err != nil {
		log.Fatalf("无法打开文件: %v", err)
	}

	allInsert := make(map[string]*task)

	// 读取“修改单信息”工作表中的数据
	sheet := workbook.GetSheet(2)
	if sheet == nil {
		log.Fatalf("没有找到工作表：修改单信息")
	}

	// 读取B,C,D列的数据
	for i := 2; i <= int(sheet.MaxRow); i++ {
		row := sheet.Row(i)
		colTaskID := row.Col(1)
		colState := row.Col(2)
		colPrincipal := row.Col(3)

		allInsert[colTaskID] = &task{taskId: colTaskID, state: colState, principal: colPrincipal}
	}

	sheet = workbook.GetSheet(3)
	if sheet == nil {
		log.Fatalf("没有找到工作表：升级说明")
	}

	// 读取C,D,I列的数据
	for i := 2; i <= int(sheet.MaxRow); i++ {
		row := sheet.Row(i)
		colTaskID2 := row.Col(2)
		colComment := row.Col(3)
		colReqNo := row.Col(8)

		if task, ok := allInsert[colTaskID2]; ok {
			task.comment = colComment
			task.reqNo = colReqNo
		}
	}

	req := pb.ImportToTaskListRequest{}
	for _, t := range allInsert {
		req.Tasks = append(req.Tasks, &pb.Task{TaskId: t.taskId, Comment: t.comment, EmergencyLevel: t.emergencyLevel,
			Deadline: t.deadline, Principal: t.principal, ReqNo: t.reqNo,
			EstimatedWorkHours: t.estimatedWorkHours, State: t.state, TypeId: t.typeId})
	}

	reply, err := client.ImportToTaskListTable(context.Background(), &req)
	if err != nil {
		fmt.Println(err)
	}
	log.Println("insert count: ", reply.InsertCnt)

}

// TODO:front
func importXLSForPatchTable(xlsFile string, client pb.ServiceClient) {
	workbook, err := xls.Open(xlsFile, "utf-8")
	if err != nil {
		log.Fatalf("无法打开文件: %v", err)
	}

	sheet := workbook.GetSheet(0)
	if sheet == nil {
		log.Fatalf("没有找到工作表：修改单信息")
	}
	req := pb.ImportXLSToPatchRequest{}
	for i := 2; i <= int(sheet.MaxRow); i++ {
		row := sheet.Row(i)
		//t, err := time.Parse("20060102", row.Col(14))
		//if err != nil {
		//	log.Println("err to parse time", err)
		//}

		req.Patchs = append(req.Patchs, &pb.Patch{ReqNo: row.Col(0), PatchNo: row.Col(1), Describe: row.Col(2),
			ClientName: row.Col(3), Reason: row.Col(12),
			Deadline: row.Col(14), Sponsor: row.Col(19)})
	}

	//TODO:调用rpc
	_, err = client.ImportXLSToPatchTable(context.Background(), &req)
	if err != nil {
		log.Println(err)
	}
}

func updateForm(myapp fyne.App, task **pb.Task, client pb.ServiceClient) bool {
	modTaskWindow := myapp.NewWindow("Update")

	txtDeadline, txtComment, txtState, txtPrincipal, txtEmergencyLevel32, txtEstimatedWorkHours64, txtType32 := (*task).Deadline, (*task).Comment, (*task).State, (*task).Principal, (*task).EmergencyLevel, (*task).EstimatedWorkHours, (*task).TypeId

	isChanged := false
	Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	Id.SetPlaceHolder((*task).TaskId)
	ReqNo.SetPlaceHolder((*task).ReqNo)
	Deadline.SetPlaceHolder(txtDeadline)
	Deadline.OnSubmitted = func(in string) {
		_, err := time.Parse("2006-01-02", Deadline.Text)
		if err != nil {
			dialog.NewInformation("提示", "deadline格式错误\nUsage: 2006-01-02", modTaskWindow).Show()
			return
		}
		txtDeadline = in
		log.Printf("pre: %s, after : %s\n", txtDeadline, in)
		isChanged = true
	}

	Comment.SetPlaceHolder(txtComment)
	Comment.OnSubmitted = func(in string) {
		txtComment = in
		isChanged = true
	}
	EmergencyLevel.SetPlaceHolder(fmt.Sprintf("%d", txtEmergencyLevel32))
	EmergencyLevel.OnSubmitted = func(in string) {
		tmp, _ := strconv.Atoi(in)
		txtEmergencyLevel32 = int32(tmp)
		isChanged = true
	}
	EstimatedWorkHours.SetPlaceHolder(fmt.Sprintf("%d", txtEstimatedWorkHours64))
	EstimatedWorkHours.OnSubmitted = func(in string) {
		tmp, _ := strconv.Atoi(in)
		txtEstimatedWorkHours64 = int64(tmp)
		isChanged = true
	}
	State.SetPlaceHolder(txtState)
	State.OnSubmitted = func(in string) {
		txtState = in
		isChanged = true
	}
	Type.SetPlaceHolder(fmt.Sprintf("%d", txtType32))
	Type.OnSubmitted = func(in string) {
		tmp, _ := strconv.Atoi(in)
		txtType32 = int32(tmp)
		isChanged = true
	}
	Principal.SetPlaceHolder(txtPrincipal)
	Principal.OnSubmitted = func(in string) {
		txtPrincipal = in
		isChanged = true
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
		},
		OnSubmit: func() { // optional, handle form submission
			//特判:deadline格式
			//TODO:七个修改都为空，则不需要提交

			if isChanged {
				newTask := &pb.Task{
					TaskId:             (*task).TaskId,
					Comment:            txtComment,
					Deadline:           txtDeadline,
					EmergencyLevel:     txtEmergencyLevel32,
					EstimatedWorkHours: txtEstimatedWorkHours64,
					State:              txtState,
					TypeId:             txtType32,
					ReqNo:              (*task).ReqNo,
					Principal:          txtPrincipal,
				}
				*task = newTask
				reply, err := client.ModTask(context.Background(), &pb.ModTaskRequest{T: newTask})
				if err != nil {
					isSucceed <- false
					log.Println(err)
					return
				}
				if reply.Succeed {
					isSucceed <- true
					log.Println("update succeed")
				} else {
					isSucceed <- false
				}
			} else { //没有变化，省去不必要提交
				isSucceed <- false
				log.Println("nothing need to update")
			}
			modTaskWindow.Close()
			fyne.LogError("User confirmed", nil)
		},
		OnCancel: func() {
			isSucceed <- false
			return
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

func addForm(myapp fyne.App, client pb.ServiceClient) bool {
	modTaskWindow := myapp.NewWindow("Update")

	idEty, deadlineEty, reqNoEty, commentEty, emergencyLevelEty, estimatedWorkHoursEty, stateEty, typeEty, principalEty := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()

	deadlineEty.OnSubmitted = func(in string) {
		_, err := time.Parse("2006-01-02", in)
		if err != nil {
			dialog.NewInformation("提示", "deadline格式错误\nUsage: 2006-01-02", modTaskWindow).Show()
			return
		}
		log.Printf("pre: %s, after : %s\n", deadlineEty.Text, in)
	}

	isSucceed := make(chan bool)
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

			if isChanged {
				newTask := &pb.Task{
					TaskId:             (*task).TaskId,
					Comment:            txtComment,
					Deadline:           txtDeadline,
					EmergencyLevel:     txtEmergencyLevel32,
					EstimatedWorkHours: txtEstimatedWorkHours64,
					State:              txtState,
					TypeId:             txtType32,
					ReqNo:              (*task).ReqNo,
					Principal:          txtPrincipal,
				}
				*task = newTask
				reply, err := client.ModTask(context.Background(), &pb.ModTaskRequest{T: newTask})
				if err != nil {
					isSucceed <- false
					log.Println(err)
					return
				}
				if reply.Succeed {
					isSucceed <- true
					log.Println("update succeed")
				} else {
					isSucceed <- false
				}
			} else { //没有变化，省去不必要提交
				isSucceed <- false
				log.Println("nothing need to update")
			}
			modTaskWindow.Close()
			fyne.LogError("User confirmed", nil)
		},
		OnCancel: func() {
			isSucceed <- false
			return
		},
	}

	modTaskWindow.SetContent(form)
	modTaskWindow.Resize(fyne.NewSize(300, 200))
	modTaskWindow.Show()

	modTaskWindow.SetOnClosed(func() {
		isSucceed <- false
	})
	return <-isSucceed
	return false
}
