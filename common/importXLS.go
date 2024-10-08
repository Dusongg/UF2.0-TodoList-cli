package common

import (
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/extrame/xls"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type importTask struct {
	comment   string
	taskId    string
	principal string
	reqNo     string
	state     string
}

var ExePathTask string
var ExePathPatchs string

func ImportController(myapp fyne.App, client pb.ServiceClient, importFunc func(string, pb.ServiceClient, chan string), flushChan chan struct{}) {
	importWd := myapp.NewWindow("import")
	input := widget.NewMultiLineEntry()
	input.Resize(fyne.NewSize(600, 400))

	output := widget.NewMultiLineEntry()
	output.Resize(fyne.NewSize(600, 400))
	outputChan := make(chan string, 100)
	var once sync.Once

	input.Validator = func(s string) error {
		if s == "" {
			return errors.New("can not be empty")
		}
		return nil
	}
	importActivity := widget.NewActivity()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "input:", Widget: input},
			{Text: "result", Widget: output},
			{Text: "", Widget: importActivity},
		},
		OnSubmit: func() {
			importActivity.Start()
			output.SetText("")
			paths := strings.Split(input.Text, "\n")
			wg := sync.WaitGroup{}
			processsXLS := func(path string) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					importFunc(path, client, outputChan)
				}()
			}
			go func() {
				for res := range outputChan {
					output.Append(res + "\n\r")
				}
			}()
			for _, path := range paths {
				//TODO:windows下直接ctrl + shift + C 复制文件/文件夹路径带有“”
				//TODO: 考虑一个文件夹里有规范化导出和补丁导出文件两种`
				if strings.HasPrefix(path, "\"") && strings.HasSuffix(path, "\"") {
					path = path[1 : len(path)-1]
				}
				info, err := os.Stat(path)
				if err != nil {
					outputChan <- fmt.Sprintf("<err>-><%s>:%s", path, err.Error())
					continue
				}

				if info.IsDir() {
					if err := filepath.Walk(info.Name(), func(path string, info fs.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if !info.IsDir() {
							processsXLS(path)
						}
						return nil
					}); err != nil {
						outputChan <- fmt.Sprintf("<err>-><%s>: %s", path, err.Error())
					}
				} else {
					processsXLS(path)
				}

			}
			wg.Wait()
			importActivity.Stop()
			flushChan <- struct{}{}
		},
		OnCancel: func() {
			once.Do(func() {
				importWd.Close()
				close(outputChan)
				close(flushChan)
			})
		},
	}
	importWd.SetOnClosed(func() {
		once.Do(func() {
			importWd.Close()
			close(outputChan)
			close(flushChan)
		})
	})
	importWd.SetContent(form)
	importWd.Resize(fyne.NewSize(600, 400))
	importWd.Show()
}

func ImportXLStoTaskListByPython(xlsFile string, client pb.ServiceClient, resChan chan string) {
	cmd := exec.Command(ExePathTask, xlsFile)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil || stderr.Len() > 0 {
		resChan <- fmt.Sprintf("<err> -> <%s>:cmd.Run() failed with %s: %s", xlsFile, err, stderr.String())
		return
	}

	//
	//output, err := cmd.CombinedOutput()
	//if err != nil {
	//	resChan <- fmt.Sprintf("Failed to execute command: %v", err)
	//	return
	//}
	//
	//outputStr := string(output)
	//if cmd.ProcessState.ExitCode() != 0 {
	//	resChan <- fmt.Sprintf("Python script error: %s\n", outputStr)
	//	return
	//}

	var result map[string][][]string
	err = json.Unmarshal(out.Bytes(), &result)
	if err != nil {
		resChan <- fmt.Sprintf("<err> -> <%s>:%s", xlsFile, err.Error())
		return
	}

	//修改单信息
	taskMap := make(map[string]*importTask)
	for _, row := range result["sheet3"] {
		taskMap[row[0]] = &importTask{
			taskId:    row[0],
			state:     row[1],
			principal: row[2],
		}
	}
	for _, row := range result["sheet4"] {
		if task, ok := taskMap[row[0]]; ok {
			task.comment = row[1]

			parts := strings.Split(row[2], ".")
			if len(parts) != 0 {
				task.reqNo = parts[0]
			} else {
				task.reqNo = row[2]
			}
		}
	}
	req := &pb.ImportToTaskListRequest{}
	for _, task := range taskMap {
		req.Tasks = append(req.Tasks, &pb.Task{
			Comment:   task.comment,
			TaskId:    task.taskId,
			ReqNo:     task.reqNo,
			Principal: task.principal,
			State:     task.state,
		})
		//fmt.Printf("id: %s, reqNo: %s, comment: %s, state: %s, principal: %s\n", task.taskId, task.reqNo, task.comment, task.state, task.principal)
	}
	req.User = config.Cfg.Login.UserName
	_, err = client.ImportXLSToTaskTable(context.Background(), req)
	if err != nil {
		resChan <- fmt.Sprintf("<err> -> <%s>:%s", xlsFile, stderr.String())
		return
	}
	resChan <- fmt.Sprintf("<success> -> <%s>", xlsFile)
	return
}

func ImportXLStoTaskList(xlsFile string, client pb.ServiceClient, resChan chan string) {
	// 打开.xls文件
	workbook, err := xls.Open(xlsFile, "utf-8")
	if err != nil {
		resChan <- fmt.Sprintf("<err>-><%s>:%s", xlsFile, err.Error())
		return
	}

	allInsert := make(map[string]*importTask)

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

		allInsert[colTaskID] = &importTask{taskId: colTaskID, state: colState, principal: colPrincipal}
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
		req.Tasks = append(req.Tasks, &pb.Task{TaskId: t.taskId, Comment: t.comment, Principal: t.principal, ReqNo: t.reqNo, State: t.state})
	}

	_, err = client.ImportXLSToTaskTable(context.Background(), &req)
	if err != nil {
		resChan <- fmt.Sprintf(err.Error())
		return
	}
	resChan <- fmt.Sprintf("%s import complete", xlsFile)
	return
}

func ImportXLStoPatchTableByPython(xlsFile string, client pb.ServiceClient, resChan chan string) {
	cmd := exec.Command(ExePathPatchs, xlsFile)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil || stderr.Len() > 0 {
		resChan <- fmt.Sprintf("<err> -> <%s>:cmd.Run() failed with %s: %s", xlsFile, err, stderr.String())
		return
	}

	var data [][]string
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		resChan <- fmt.Sprintf("<err> -> <%s>:%s", xlsFile, err.Error())
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
			State:      row[7],
		}

		//TODO: 最后一行读取错误问题
		//fmt.Printf("Reading row %d: %+v\n", i, patch)

		req.Patchs = append(req.Patchs, patch)
		req.User = config.Cfg.Login.UserName
	}
	_, err = client.ImportXLSToPatchTable(context.Background(), &req)
	if err != nil {
		resChan <- fmt.Sprintf("<err> -> <%s>:%s", xlsFile, err.Error())
		return
	}
	resChan <- fmt.Sprintf("<success> -> <%s>", xlsFile)
	return

}
