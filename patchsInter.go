package main

import (
	"OrderManager-cli/common"
	"OrderManager-cli/pb"
	"context"
	"errors"
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
			return container.NewPadded(widget.NewButton("Do something", nil))
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			o.(*fyne.Container).Objects[0].(*widget.Button).SetText(id)
			o.(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
				if strings.HasPrefix(id, "P") {
					retNo := ModPatchsForm(id, client)
					if retNo == 1 { //删除
						delAndFlushTree(id)
					}
				} else if strings.HasPrefix(id, "T") {
					if ModForm(id, client) {
						orderMap = loadAllPatchs(client, mw)
						tree.Refresh()
					}
				} else { //TODO：需求下添加任务

				}
			}

		})

	importBtn := widget.NewButtonWithIcon("", theme.UploadIcon(), func() {
		importController(client, common.ImportXLS.ImportXLStoPatchTable)
		orderMap = loadAllPatchs(client, mw)
		tree.Refresh()
	})
	searchEntry := widget.NewEntry()
	searchBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		if searchEntry.Text == "" {
			orderMap = loadAllPatchs(client, mw)
			tree.Refresh()
		} else {
			log.Println(searchEntry.Text)
			orderMap = loadQueryPatchs(searchEntry.Text, client, mw)
			tree.Refresh()
		}
	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		orderMap = loadAllPatchs(client, mw)
		tree.Refresh()
	})
	bg := canvas.NewRectangle(color.RGBA{R: 217, G: 213, B: 213, A: 255})
	searchBar := container.NewStack(bg, container.NewBorder(nil, nil, importBtn, container.NewHBox(searchBtn, flushBtn), searchEntry))

	return container.NewBorder(searchBar, nil, nil, nil, tree)
}

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
		keys = append(keys, patch.PatchNo)
		orderMap[patch.PatchNo] = append(orderMap[patch.PatchNo], patch.ReqNo)
	}
	for _, task := range tasksData {
		orderMap[task.ReqNo] = append(orderMap[task.ReqNo], task.TaskId)
	}
	orderMap[""] = keys
	return orderMap
}

// TODO: 该功能暂时只支持单个补丁查询
// TODO: 问： 一个补丁对应一个需求在xls是在一行内还是在多行？  （当前：1补丁 -> 1需求 -> n任务）
func loadQueryPatchs(patchNo string, client pb.ServiceClient, mw fyne.Window) map[string][]string {
	patchsReply, err := client.GetOnePatchs(context.Background(), &pb.GetOnePatchsRequest{PatchNo: patchNo})
	if err != nil {
		dialog.ShowError(err, mw)
		return nil
	}
	patchsData := patchsReply.P
	tasksReply, err := client.QueryTaskWithField(context.Background(), &pb.QueryTaskWithFieldRequest{Field: "req_no", FieldValue: patchsData.ReqNo})
	if err != nil {
		dialog.ShowError(err, mw)
		return nil
	}
	tasksData := tasksReply.Tasks

	orderMap := make(map[string][]string)
	keys := make([]string, 0)

	//当前考虑搜索结果只有一个
	keys = append(keys, patchNo)
	orderMap[patchsData.PatchNo] = append(orderMap[patchsData.PatchNo], patchsData.ReqNo)

	for _, task := range tasksData {
		orderMap[task.ReqNo] = append(orderMap[task.ReqNo], task.TaskId)
	}
	orderMap[""] = keys
	return orderMap
}

func ModPatchsForm(patchNo string, client pb.ServiceClient) int {
	modTaskWindow := myapp.NewWindow("Update")

	reply, err := client.GetOnePatchs(context.Background(), &pb.GetOnePatchsRequest{PatchNo: patchNo})
	if err != nil {
		dialog.ShowError(err, modTaskWindow)
	}
	patchs := reply.P

	patchNoEty, reqNoEty, describeEty, clientNameEty, deadlineEty, reasonEty, sponsorEty := widget.NewEntry(), widget.NewEntry(), widget.NewMultiLineEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	patchNoEty.SetText(patchs.PatchNo)
	patchNoEty.Disable()
	reqNoEty.SetText(patchs.ReqNo)
	reqNoEty.Disable()
	describeEty.SetText(patchs.Describe)
	clientNameEty.SetText(patchs.ClientName)
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
				_, err := client.DelPatch(context.Background(), &pb.DelPatchRequest{PatchNo: patchNoEty.Text})
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
			}
			_, err := client.ModPatch(context.Background(), &pb.ModPatchRequest{P: newPatch})
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
