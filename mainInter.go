package main

import (
	"OrderManager-cli/pb"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func showMainInterface(client pb.ServiceClient, mw fyne.Window) {

	previewInterface := container.NewBorder(nil, nil, nil, nil, nil)
	appTab := container.NewAppTabs(
		container.NewTabItemWithIcon("预览", theme.ListIcon(), previewInterface),
		container.NewTabItemWithIcon("收件箱", theme.StorageIcon(), container.NewVScroll(widget.NewLabel("TODO"))),
		container.NewTabItemWithIcon("补丁", theme.VisibilityIcon(), widget.NewLabel("TODO")),
		//container.NewTabItem("库", widget.NewLabel("TODO")),
	)
	//default
	previewInterface = CreatePreviewInterface(appTab, client, mw)
	appTab.Items[0].Content = previewInterface

	appTab.SetTabLocation(container.TabLocationLeading) //竖着的标签

	var inboxInterface fyne.CanvasObject
	var patchsInterface fyne.CanvasObject
	appTab.OnSelected = func(item *container.TabItem) {
		if item == appTab.Items[1] && inboxInterface == nil {
			inboxInterface = CreateInBoxInterface(client, mw)
			appTab.Items[1].Content = inboxInterface
		}
		if item == appTab.Items[2] && patchsInterface == nil {
			patchsInterface = CreatePatchsInterface(client, mw)
			appTab.Items[2].Content = patchsInterface
		}
	}
	mw.SetContent(appTab)

}
