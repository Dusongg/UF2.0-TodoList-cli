package main

import (
	"OrderManager-cli/pb"
	"context"
	"fmt"
	"github.com/extrame/xls"
	"log"
)

func ImportXLSForTaskList(xlsFile string, client pb.ServiceClient, resChan chan string) {
	// 打开.xls文件
	workbook, err := xls.Open(xlsFile, "utf-8")
	if err != nil {
		resChan <- fmt.Sprintf("err: %v", err)
		return
	}

	allInsert := make(map[string]*task)

	// 读取“修改单信息”工作表中的数据
	sheet := workbook.GetSheet(2)
	if sheet == nil {
		resChan <- fmt.Sprintf("err: 没有找到工作表：修改单信息 in %s", xlsFile)
		return
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
		resChan <- fmt.Sprintf("err: 没有找到工作表：升级说明 in %s", xlsFile)
		return
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
	resChan <- fmt.Sprintf("%s import complete, insert count: %d", xlsFile, reply.InsertCnt)
	return
}

// TODO:front
func ImportXLSForPatchTable(xlsFile string, client pb.ServiceClient) {
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
