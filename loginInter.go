package main

import (
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"context"
	"encoding/json"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

func saveUserNameAndPass() {
	file, err := os.OpenFile("./config/config.json", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Println("文件打开错误：", err)
		return
	}
	defer file.Close()
	data, _ := json.Marshal(config.Cfg)
	_, err = file.Write(data)
	if err != nil {
		logrus.Println("无法写入文件:", err)
		return
	}

}

func showLoginScreen(client pb.ServiceClient, loginWd fyne.Window, loginChan chan bool) {
	usernameEty := widget.NewEntry()
	passwordEty := widget.NewPasswordEntry()

	//
	usernameEty.SetText(config.Cfg.Login.UserName)
	passwordEty.SetText(config.Cfg.Login.Password)
	//
	loginForm := widget.NewForm(
		widget.NewFormItem("Username", usernameEty),
		widget.NewFormItem("Password", passwordEty),
	)
	rememberCheck := widget.NewCheck("Remember", func(checked bool) {})
	rememberCheck.Checked = true

	loginButton := widget.NewButtonWithIcon("Login", theme.LoginIcon(), func() {
		user := usernameEty.Text
		pass := passwordEty.Text
		if user == "" || pass == "" {
			dialog.ShowError(errors.New("请输入"), loginWd)
			return
		}

		_, err := client.Login(context.Background(), &pb.LoginRequest{
			Name:     user,
			Password: pass,
		})
		if err != nil {
			dialog.ShowError(err, loginWd)
			return
		} else {
			config.Cfg.Login.UserName = user
			config.Cfg.Login.Password = pass
			if rememberCheck.Checked {
				// 在文件中写入默认的用户名和密码
				go saveUserNameAndPass()
			}
			logrus.Printf("login success, user: %s\n", config.Cfg.Login.UserName)
			loginChan <- true
			return
		}
	})

	registerButton := widget.NewButtonWithIcon("Register", theme.FolderNewIcon(), func() {
		showRegisterScreen(client, loginWd, loginChan)
	})

	loginWd.SetContent(container.NewVBox(
		loginForm,
		container.NewBorder(nil, nil, layout.NewSpacer(), rememberCheck),
		loginButton,
		registerButton,
	))
}

func showRegisterScreen(client pb.ServiceClient, loginWd fyne.Window, loginChan chan bool) {
	username := widget.NewEntry()
	jobNum := widget.NewEntry()
	email := widget.NewEntry()
	password := widget.NewPasswordEntry()
	confirmPassword := widget.NewPasswordEntry()

	registerForm := widget.NewForm(
		widget.NewFormItem("Username", username),
		widget.NewFormItem("Job Number", jobNum),
		widget.NewFormItem("Password", password),
		widget.NewFormItem("Confirm Password", confirmPassword),
		widget.NewFormItem("Email", email),
	)

	registerButton := widget.NewButton("Register", func() {
		user := username.Text
		pass := password.Text
		confirmPass := confirmPassword.Text
		email_ := email.Text
		jobNum_, err := strconv.Atoi(jobNum.Text)
		if user == "" || pass == "" || confirmPass == "" || email_ == "" || err != nil {
			dialog.ShowError(errors.New("any item cannot be empty"), loginWd)
			return
		}
		if pass != confirmPass {
			dialog.ShowError(errors.New("confirm password error"), loginWd)
		}

		_, err = client.Register(context.Background(), &pb.RegisterRequest{
			User: &pb.User{
				Name:     user,
				Password: pass,
				Email:    email_,
				JobNum:   int64(jobNum_),
			}})
		if err != nil {
			dialog.ShowError(err, loginWd)
		} else {
			dialog.ShowInformation("Registration Successful", "User: "+user+" registered successfully!", loginWd)
			showLoginScreen(client, loginWd, loginChan)
		}
	})

	backButton := widget.NewButton("Back to Login", func() {
		showLoginScreen(client, loginWd, loginChan)
	})

	loginWd.SetContent(container.NewVBox(
		registerForm,
		registerButton,
		backButton,
	))

}

func personalView(client pb.ServiceClient) error {
	personalWd := myapp.NewWindow("personal setting")
	//
	done := make(chan error, 1)
	defer close(done)
	reply, err := client.GetUserInfo(context.Background(), &pb.GetUserInfoRequest{UserName: config.Cfg.Login.UserName})
	if err != nil {
		return err
	}

	emailEty, groupEty, roleNoEty := widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	emailEty.SetText(reply.Email)
	groupEty.SetText(strconv.Itoa(int(reply.Group)))
	roleNoEty.SetText(strconv.Itoa(int(reply.RoleNo)))
	passEty, confirmPassEty := widget.NewPasswordEntry(), widget.NewPasswordEntry()
	confirmPassEty.Validator = func(s string) error {
		if passEty.Text != "" && passEty.Text != confirmPassEty.Text {
			return errors.New("confirm password error")
		} else {
			return nil
		}
	}
	modPass := widget.NewCheck("修改密码", func(b bool) {
	})
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "姓名", Widget: widget.NewLabel(config.Cfg.Login.UserName)},
			{Text: "工号", Widget: widget.NewLabel(strconv.Itoa(int(reply.JobNO)))},
			{Text: "邮箱", Widget: emailEty},
			{Text: "小组", Widget: groupEty},
			{Text: "角色", Widget: roleNoEty},
			{Text: "", Widget: modPass},
			{Text: "输入密码", Widget: passEty},
			{Text: "确认密码", Widget: confirmPassEty},
		},
		OnSubmit: func() {
			groupI, _ := strconv.Atoi(groupEty.Text)
			roleI, _ := strconv.Atoi(roleNoEty.Text)
			if modPass.Checked {
				_, err := client.ModUserInfo(context.Background(), &pb.ModUserInfoRequest{
					ModPass: true,
					Pass:    passEty.Text,
					Email:   emailEty.Text,
					Group:   int32(groupI),
					RoleNo:  int32(roleI),
					Name:    config.Cfg.Login.UserName,
				})
				done <- err
			} else {
				_, err := client.ModUserInfo(context.Background(), &pb.ModUserInfoRequest{
					ModPass: false,
					Email:   emailEty.Text,
					Group:   int32(groupI),
					RoleNo:  int32(roleI),
					Name:    config.Cfg.Login.UserName,
				})
				done <- err
			}
		},
		OnCancel: func() {
			done <- nil
		},
	}
	personalWd.SetOnClosed(func() {
		done <- nil
	})

	personalWd.SetContent(form)
	personalWd.Resize(fyne.NewSize(300, 200))
	personalWd.Show()

	retErr := <-done
	personalWd.Close()
	return retErr

}
