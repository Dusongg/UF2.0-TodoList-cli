package main

import (
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"context"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"log"
	"strconv"
)

func showLoginScreen(client pb.ServiceClient, loginWd fyne.Window, loginChan chan bool) {
	username := widget.NewEntry()
	password := widget.NewPasswordEntry()

	//
	username.SetText("dusong")
	password.SetText("123123")
	//
	loginForm := widget.NewForm(
		widget.NewFormItem("Username/Id", username),
		widget.NewFormItem("Password", password),
	)

	loginButton := widget.NewButton("Login", func() {
		user := username.Text
		pass := password.Text
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
