package main

import (
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
	orderMap := loadPatchs(client, mw)
	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			return orderMap[id]
		},
		func(id widget.TreeNodeID) bool {
			return len(orderMap[id]) > 0
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Node")
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(id)
		})
	tree.OnSelected = func(id widget.TreeNodeID) {
		if strings.HasPrefix(id, "P") {
			retNo := ModPatchsForm(id, client)
			if retNo == 1 { //删除
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
		} else if strings.HasPrefix(id, "T") {

		}
	}

	importBtn := widget.NewButtonWithIcon("", theme.UploadIcon(), func() {

	})
	searchEntry := widget.NewEntry()
	searchBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {

	})
	flushBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {

	})
	bg := canvas.NewRectangle(color.RGBA{R: 217, G: 213, B: 213, A: 255})
	searchBar := container.NewStack(bg, container.NewBorder(nil, nil, importBtn, container.NewHBox(searchBtn, flushBtn), searchEntry))

	return container.NewBorder(searchBar, nil, nil, nil, tree)
}

func loadPatchs(client pb.ServiceClient, mw fyne.Window) map[string][]string {
	patchsReply, err := client.GetPatchsAll(context.Background(), &pb.GetPatchsAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
	}
	patchsData := patchsReply.Patchs
	tasksReply, err := client.GetTaskListAll(context.Background(), &pb.GetTaskListAllRequest{})
	if err != nil {
		dialog.ShowError(err, mw)
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
