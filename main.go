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
	// 记得关闭链接
	defer connect.Close()
	// 调用ASerer.pb.go里面的NewUserServiceClient
	client := pb.NewServiceClient(connect)

	result, err := client.GetTaskListOne(context.Background(), &pb.GetTaskListOneRequest{Name: "罗清杰"})

	if err != nil {
		fmt.Println(err)
		return
	}
	for _, res := range result.Tasks {
		fmt.Println(res)
	}

	//importXLSForTaskList("xx", client)

	r, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		log.Fatalf("could not get tasks: %v", err)
	}

	// 获取当前日期
	now := time.Now()

	// 创建一个网格布局，每行显示一天的任务
	grid := container.New(layout.NewGridLayout(7))
	appTab := container.NewAppTabs(
		container.NewTabItem("日程", grid),
		container.NewTabItem("库", widget.NewLabel("TODO")),
	)
	// 创建每天的任务列表
	data := make([][]*pb.Task, 7)
	list := make([]*widget.List, 7)

	for _, task := range r.Tasks {
		taskDate, err := time.Parse("2006-01-02", task.Deadline)
		if err != nil {
			log.Printf("could not parse date: %v", err)
			continue
		}
		weekday := int(taskDate.Sub(now).Hours() / 24) //任务时间距离现在有多少天
		if weekday >= 0 && weekday < 7 {
			data[weekday] = append(data[weekday], task)
		}
	}
	for d := 0; d < 7; d++ {
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
			pushTaskButton(myapp, data[d][id], client)
		}

		// 创建按钮，向列表中添加新元素
		addButton := widget.NewButton("Add Item", func() {
			//TODO:添加
			fmt.Println("TODO")
			list[d].Refresh() // 刷新列表以显示新元素
		})
		date := now.AddDate(0, 0, d)
		//dateLabel := widget.NewLabel(date.Format("2006-01-02"))

		dateLabel := canvas.NewText(date.Format("2006-01-02"), theme.Color(theme.ColorNameForeground))
		// 设置Label的字体颜色
		dateLabel.TextStyle = fyne.TextStyle{Bold: true}
		dateLabel.Color = color.White

		dayContainer := container.NewVBox(dateLabel)
		bg := canvas.NewRectangle(colorTheme1)
		grid.Add(container.NewBorder(container.NewStack(bg, dayContainer), addButton, nil, nil, list[d]))
	}

	//appTab.SetTabLocation(container.TabLocationLeading) //竖着的标签

	// 设置窗口内容
	mw.SetContent(appTab)

	// 设置窗口大小并显示
	mw.Resize(fyne.NewSize(1400, 600))
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

func pushTaskButton(myapp fyne.App, task *pb.Task, client pb.ServiceClient) {
	modTaskWindow := myapp.NewWindow("Small Window")

	Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	Id.SetPlaceHolder(task.TaskId)
	Deadline.SetPlaceHolder(task.Deadline)
	ReqNo.SetPlaceHolder(task.ReqNo)
	Comment.SetPlaceHolder(task.Comment)
	EmergencyLevel.SetPlaceHolder(fmt.Sprintf("%d", task.EmergencyLevel))
	EstimatedWorkHours.SetPlaceHolder(fmt.Sprintf("%d", task.EstimatedWorkHours))
	State.SetPlaceHolder(task.State)
	Type.SetPlaceHolder(fmt.Sprintf("%d", task.TypeId))
	Principal.SetPlaceHolder(task.Principal)

	Id.Disable() // 设置为只读
	//Deadline.Disable()           // 设置为只读
	ReqNo.Disable() // 设置为只读
	//Comment.Disable()            // 设置为只读
	//EmergencyLevel.Disable()     // 设置为只读
	//EstimatedWorkHours.Disable() // 设置为只读
	//State.Disable()              // 设置为只读
	//Type.Disable()               // 设置为只读
	//Principal.Disable()          // 设置为只读

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			//Id.Enable()
			//ReqNo.Enable()
			//Principal.Enable()
			Deadline.Enable()
			Comment.Enable()
			EmergencyLevel.Enable()
			EstimatedWorkHours.Enable()
			State.Enable()
			Type.Enable()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ContentCutIcon(), func() {}),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {}),
		widget.NewToolbarAction(theme.ContentPasteIcon(), func() {}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			log.Println("Display help")
		}),
	)

	// Id, Deadline, ReqNo, Comment, EmergencyLevel, EstimatedWorkHours, State, Type, Principal

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
			dialog.NewConfirm("Confirm", "Are you sure you want to commit?", func(confirm bool) {
				if confirm {
					//TODO:七个修改都为空，则不需要提交
					needCommit := 7
					txtDeadline, txtComment, txtState, txtPrincipal := Deadline.Text, Comment.Text, State.Text, Principal.Text
					txtEmergencyLevel, err := strconv.Atoi(EmergencyLevel.Text)
					txtEmergencyLevel32 := int32(txtEmergencyLevel)
					if err != nil {
						needCommit--
						txtEmergencyLevel32 = task.EmergencyLevel
					}
					txtEstimatedWorkHours, err := strconv.Atoi(EstimatedWorkHours.Text)
					txtEstimatedWorkHours64 := int64(txtEstimatedWorkHours)
					if err != nil {
						needCommit--
						txtEstimatedWorkHours64 = task.EstimatedWorkHours
					}
					txtType, err := strconv.Atoi(Type.Text)
					txtType32 := int32(txtType)
					if err != nil {
						needCommit--
						txtType32 = task.TypeId
					}

					if txtDeadline == "" {
						needCommit--
						txtDeadline = task.Deadline
					}
					if txtComment == "" {
						needCommit--
						txtComment = task.Comment
					}
					if txtState == "" {
						needCommit--
						txtState = task.State
					}
					if txtPrincipal == "" {
						needCommit--
						txtPrincipal = task.Principal
					}

					if needCommit != 0 {
						reply, err := client.ModTask(context.Background(), &pb.ModTaskRequest{
							T: &pb.Task{
								TaskId:             task.TaskId,
								Comment:            txtComment,
								Deadline:           txtDeadline,
								EmergencyLevel:     txtEmergencyLevel32,
								EstimatedWorkHours: txtEstimatedWorkHours64,
								State:              txtState,
								TypeId:             txtType32,
								ReqNo:              task.ReqNo,
								Principal:          txtPrincipal,
							},
						})
						if err != nil {
							log.Println(err)
						}
						if reply.Succeed {
							log.Println("update succeed")
						}
					}
					modTaskWindow.Close()
					fyne.LogError("User confirmed", nil)
				} else {
					fyne.LogError("User canceled", nil)
					return
				}
			}, modTaskWindow).Show()
		},
	}

	content := container.NewVBox(toolbar, form)
	modTaskWindow.SetContent(content)
	modTaskWindow.Resize(fyne.NewSize(300, 200))
	modTaskWindow.Show()
}
