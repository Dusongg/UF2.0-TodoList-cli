package main

import (
	"OrderManager-cli/pb"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"strings"
)

func importController(client pb.ServiceClient, importFunc func(string, pb.ServiceClient, chan string)) {
	importWd := myapp.NewWindow("import")
	input := widget.NewMultiLineEntry()
	input.Resize(fyne.NewSize(600, 400))

	output := widget.NewMultiLineEntry()
	output.Resize(fyne.NewSize(600, 400))
	outputChan := make(chan string)
	input.Validator = func(s string) error {
		if s == "" {
			return errors.New("can not be empty")
		}
		return nil
	}
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "input:", Widget: input},
			{Text: "result", Widget: output},
		},
		OnSubmit: func() {
			output.SetText("")
			paths := strings.Split(input.Text, "\n")
			for _, path := range paths {
				//去除粘贴过来时的引号
				go importFunc(path[1:len(path)-1], client, outputChan)
			}
			go func() {
				for res := range outputChan {
					output.Append(res + "\n\r")
				}
			}()
		},
		OnCancel: func() {
			importWd.Close()
		},
	}
	importWd.SetContent(form)
	importWd.Resize(fyne.NewSize(600, 400))
	importWd.Show()
}
