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
	"strings"
	"time"
)

var (
	COLOR_PATCHS = color.RGBA{52, 70, 94, 255}
	COLOR_REQ    = color.RGBA{84, 117, 161, 255}
	COLOR_TASK   = color.RGBA{125, 169, 227, 255}
)

func CreatePatchsInterface(client pb.ServiceClient, mw fyne.Window) fyne.CanvasObject {
	orderMap := loadAllPatchs(client, mw)

	var tree *widget.Tree
	delAndFlushTree := func(id string) {
		delete(orderMap, id)
		var newroot []string
		for _, root := range orderMap[""] {
			if root != id {
				newroot = append(newroot, root)
			}
		}
		orderMap[""] = newroot
		tree.Refresh()
	}
	tree = widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			return orderMap[id]
		},
		func(id widget.TreeNodeID) bool {
			return len(orderMap[id]) > 0
		},
		func(branch bool) fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Black)
			btn := widget.NewButton("Do something", nil)
			return container.NewPadded(
				container.NewBorder(nil, nil, bg, nil, btn),
			)
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			if strings.HasPrefix(id, "P") { //补丁
				o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Rectangle).FillColor = COLOR_PATCHS

			} else if strings.HasPrefix(id, "T") {
				o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Rectangle).FillColor = COLOR_TASK
			} else {
				o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Rectangle).FillColor = COLOR_REQ
			}

			//TODO
			if toshow, ok := patchsInfoMap[id]; ok {
				info := fmt.Sprintf("%s : <客户: %s -- 预计发布时间: %s -- 发布状态: %s>", id, toshow.clientName, toshow.deadline, toshow.state)
				o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).SetText(info)
			} else {
				o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).SetText(id)
			}
			o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
				if strings.HasPrefix(id, "P") { //补丁

					retNo := ModPatchsForm(id, client)
					if retNo == 1 { //删除
						delAndFlushTree(id)
					}
				} else if strings.HasPrefix(id, "T") { //任务/修改单

					if ModForm(id, client) {
						orderMap = loadAllPatchs(client, mw)
						tree.Refresh()
					}
				}
			}
		})

	importBtn := widget.NewButtonWithIcon("", theme.UploadIcon(), func() {
		flushChan := make(chan struct{})
		common.ImportController(myapp, client, common.ImportXLStoPatchTableByPython, flushChan)
		for {
			_, ok := <-flushChan
			if ok {
				orderMap = loadAllPatchs(client, mw)
				tree.Refresh()
			} else {
				break
			}
		}
	})
	searchEntry := widget.NewEntry()
	searchBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		if searchEntry.Text == "" {
			orderMap = loadAllPatchs(client, mw)
			tree.Refresh()
		} else {
			orderMap = loadQueryPatchs(searchEntry.Text, client, mw)
			tree.Refresh()
		}
	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		orderMap = loadAllPatchs(client, mw)
		tree.Refresh()
		tree.CloseAllBranches()
	})
	bg := canvas.NewRectangle(color.RGBA{R: 217, G: 213, B: 213, A: 255})
	searchBar := container.NewStack(bg, container.NewBorder(nil, nil, importBtn, container.NewHBox(searchBtn, flushBtn), searchEntry))

	return container.NewBorder(searchBar, nil, nil, nil, tree)
}

type treeInfo struct {
	clientName string
	deadline   string
	state      string
}

var patchsInfoMap = make(map[string]treeInfo)

func loadAllPatchs(client pb.ServiceClient, mw fyne.Window) map[string][]string {
	patchsReply, err := client.GetPatchsAll(context.Background(), &pb.GetPatchsAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
		return nil
	}
	patchsData := patchsReply.Patchs
	tasksReply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
		return nil
	}
	tasksData := tasksReply.Tasks

	orderMap := make(map[string][]string)
	keys := make([]string, 0)
	for _, patch := range patchsData {

		patchsInfoMap[patch.PatchNo] = treeInfo{
			clientName: patch.ClientName,
			deadline:   patch.Deadline,
			state:      patch.State,
		}

		keys = append(keys, patch.PatchNo)
		orderMap[patch.PatchNo] = append(orderMap[patch.PatchNo], strings.Split(patch.ReqNo, ",")...)
	}
	for _, task := range tasksData {
		orderMap[task.ReqNo] = append(orderMap[task.ReqNo], task.TaskId)
	}
	orderMap[""] = keys
	return orderMap
}

// TODO: 该功能暂时只支持单个补丁查询
// TODO:一个补丁和多个需求对应关系在一行中，不同需求由“ , ”隔开，不同补丁可能对应相同需求
func loadQueryPatchs(patchNo string, client pb.ServiceClient, mw fyne.Window) map[string][]string {
	patchsReply, err := client.GetOnePatchs(context.Background(), &pb.GetOnePatchsRequest{PatchNo: patchNo})
	if err != nil {
		dialog.ShowError(err, mw)
		return nil
	}
	orderMap := make(map[string][]string)
	keys := make([]string, 0)
	keys = append(keys, patchNo)

	patchsData := patchsReply.P
	reqNos := strings.Split(patchsData.ReqNo, ",")
	for _, reqNo := range reqNos {
		tasksReply, err := client.QueryTaskWithField(context.Background(), &pb.QueryTaskWithFieldRequest{Field: "req_no", FieldValue: reqNo})
		if err != nil {
			dialog.ShowError(err, mw)
			return nil
		}
		tasksData := tasksReply.Tasks

		orderMap[patchsData.PatchNo] = append(orderMap[patchsData.PatchNo], reqNo)

		for _, task := range tasksData {
			orderMap[patchsData.ReqNo] = append(orderMap[task.ReqNo], task.TaskId)
		}
	}

	orderMap[""] = keys
	return orderMap
}

// TODO:考虑查询到的第一个
func ModPatchsForm(patchNo string, client pb.ServiceClient) int {
	modTaskWindow := myapp.NewWindow("Update")

	reply, err := client.GetOnePatchs(context.Background(), &pb.GetOnePatchsRequest{PatchNo: patchNo})
	if err != nil {
		dialog.ShowError(err, modTaskWindow)
	}
	patchs := reply.P
	log.Println(reply.P.State)

	patchNoEty, reqNoEty, describeEty, clientNameEty, deadlineEty, reasonEty, sponsorEty, stateEty := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	patchNoEty.SetText(patchs.PatchNo)
	patchNoEty.Disable()
	reqNoEty.SetText(patchs.ReqNo)
	reqNoEty.Disable()
	describeEty.SetText(patchs.Describe)
	clientNameEty.SetText(patchs.ClientName)
	stateEty.SetText(patchs.State)

	clientNameEty.Validator = func(in string) error {
		if in == "" {
			return errors.New("client name is empty")
		}
		return nil
	}
	deadlineEty.SetText(patchs.Deadline)
	deadlineEty.Validator = func(in string) error {
		_, err := time.Parse("2006-01-02", deadlineEty.Text)
		if err != nil {
			return errors.New("deadline format error  Usage: 2006-01-02")
		}
		return nil
	}
	reasonEty.SetText(patchs.Reason)
	sponsorEty.SetText(patchs.Sponsor)
	sponsorEty.Validator = func(in string) error {
		if in == "" {
			return errors.New("sponsor is empty")
		}
		return nil
	}

	isSucceed := make(chan int) //-1 : 失败， 0：update  1 ： delete

	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		dialog.NewConfirm("Please Confirm", "Are you sure to delete", func(confirm bool) {
			if confirm {
				_, err := client.DelPatch(context.Background(), &pb.DelPatchRequest{PatchNo: patchNoEty.Text, User: config.LoginUser})
				if err != nil {
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
			{Text: "补丁号", Widget: patchNoEty},
			{Text: "需求号", Widget: reqNoEty},
			{Text: "截止日期", Widget: deadlineEty},
			{Text: "发起人", Widget: sponsorEty},
			{Text: "客户名称", Widget: clientNameEty},
			{Text: "问题描述", Widget: describeEty},
			{Text: "补丁原因", Widget: reasonEty},
			{Text: "发布状态", Widget: stateEty},
			{Widget: delBtn},
		},
		OnSubmit: func() {
			newPatch := &pb.Patch{
				PatchNo:    patchNoEty.Text,
				ReqNo:      reqNoEty.Text,
				Describe:   describeEty.Text,
				Deadline:   deadlineEty.Text,
				ClientName: clientNameEty.Text,
				Reason:     reasonEty.Text,
				Sponsor:    sponsorEty.Text,
				State:      stateEty.Text,
			}
			log.Println("cur user:", config.LoginUser)
			_, err := client.ModPatch(context.Background(), &pb.ModPatchRequest{P: newPatch, User: config.LoginUser})
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
