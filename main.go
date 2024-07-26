package main

import (
	"OrderManager-cli/pb"
	"context"
	"fmt"
	"github.com/extrame/xls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func main() {
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

	importXLSForTaskList("xx", client)

}

type task struct {
	comment            string
	taskId             string
	emergencyLevel     int32
	deadline           string
	principal          string
	reqNo              string
	estimatedWorkHours float32
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
