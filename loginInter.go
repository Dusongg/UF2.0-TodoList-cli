package main

import (
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"bufio"
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	userName string
	passwd   string
)

func saveUserNameAndPass(user, pass string) {
	file, err := os.OpenFile(config.SaveUserInfoPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Println("文件打开错误：", err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(user + "\n" + pass)
	if err != nil {
		log.Println("无法写入文件:", err)
		return
	}

}

func init() {
	if _, err := os.Stat(config.SaveUserInfoPath); os.IsNotExist(err) {
		// 文件不存在，创建文件
		file, err := os.Create(config.SaveUserInfoPath)
		if err != nil {
			log.Println("无法创建文件:", err)
			return
		}
		defer file.Close()
	}

	file, err := os.Open(config.SaveUserInfoPath) //read only
	if err != nil {
		log.Println("无法打开文件:", err)
		return
	}
	defer file.Close()

	// 读取文件内容
	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		log.Println("读取文件时出错:", err)
		return
	}

	if len(lines) >= 2 {
		userName = lines[0]
		passwd = lines[1]
	} else {
		fmt.Println("文件内容格式不正确 / 未读取到保存的密码与用户")
	}
}

func showLoginScreen(client pb.ServiceClient, loginWd fyne.Window, loginChan chan bool) {
	usernameEty := widget.NewEntry()
	passwordEty := widget.NewPasswordEntry()

	//
	usernameEty.SetText(userName)
	passwordEty.SetText(passwd)
	//
	loginForm := widget.NewForm(
		widget.NewFormItem("Username", usernameEty),
		widget.NewFormItem("Password", passwordEty),
	)
	rememberCheck := widget.NewCheck("Remember", func(checked bool) {})
	rememberCheck.Checked = true

	loginButton := widget.NewButton("Login", func() {
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
			config.LoginUser = user
			if rememberCheck.Checked {
				// 在文件中写入默认的用户名和密码
				go saveUserNameAndPass(user, pass)
			}
			log.Printf("login success, user: %s\n", config.LoginUser)
			loginChan <- true
			return
		}
	})

	registerButton := widget.NewButton("Register", func() {
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
