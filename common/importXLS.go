package common

import (
	"OrderManager-cli/pb"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/extrame/xls"
	"os/exec"
	"time"
)

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

type importXLS struct{}

var ImportXLS = &importXLS{}

func (*importXLS) ImportXLStoTaskList(xlsFile string, client pb.ServiceClient, resChan chan string) {
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
		req.Tasks = append(req.Tasks, &pb.Task{TaskId: t.taskId, Comment: t.comment, EmergencyLevel: t.emergencyLevel, Principal: t.principal, ReqNo: t.reqNo})
	}

	_, err = client.ImportToTaskListTable(context.Background(), &req)
	if err != nil {
		resChan <- fmt.Sprintf(err.Error())
		return
	}
	resChan <- fmt.Sprintf("%s import complete", xlsFile)
	return
}

func (*importXLS) ImportXLStoPatchTable(xlsFile string, client pb.ServiceClient, resChan chan string) {
	cmd := exec.Command("python", "D:\\Golang\\OrderManager-cli\\pytool\\read_xls.py", xlsFile)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		resChan <- fmt.Errorf("cmd.Run() failed with %s: %s", err, stderr.String()).Error()
	}

	var data [][]string
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		resChan <- err.Error()
		return
	}
	req := pb.ImportXLSToPatchRequest{}
	for _, row := range data {
		t, _ := time.Parse("20060102", row[5])
		patch := &pb.Patch{
			ReqNo:      row[0],
			PatchNo:    row[1],
			Describe:   row[2],
			ClientName: row[3],
			Reason:     row[4],
			Deadline:   t.Format("2006-01-02"),
			Sponsor:    row[6],
		}

		//TODO: 最后一行读取错误问题
		//fmt.Printf("Reading row %d: %+v\n", i, patch)

		req.Patchs = append(req.Patchs, patch)
	}
	_, err = client.ImportXLSToPatchTable(context.Background(), &req)
	if err != nil {
		resChan <- fmt.Sprintf(err.Error())
		return
	}
	resChan <- fmt.Sprintf("%s import complete", xlsFile)
	return

}
