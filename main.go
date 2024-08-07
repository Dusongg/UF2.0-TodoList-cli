package main

// go build -ldflags="-H windowsgui"
import (
	"OrderManager-cli/pb"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"image/color"
	"log"
)

const DAYSPERPAGE = 5

var colorTheme1 = color.RGBA{R: 57, G: 72, B: 94, A: 255}

var LoginUser string
var myapp = app.New()

func main() {

	//defer func() {
	//	if r := recover(); r != nil {
	//		fmt.Println("Recovered in f", r)
	//	}
	//}()

	mw := myapp.NewWindow("Task List for the Week")
	//月光石主题:深-》浅

	// 建立一个链接，请求A服务
	// 真实项目里肯定是通过配置中心拿服务名称，发给注册中心请求真实的A服务地址，这里都是模拟
	// 第二个参数是配置了一个证书，因为没有证书会报错，但是我们目前没有配置证书，所以需要insecure.NewCredentials()返回一个禁用传输安全的凭据
	connect, err := grpc.NewClient(":8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("connect success")
	}
	defer connect.Close()
	client := pb.NewServiceClient(connect)

	loginChan := make(chan bool)
	loginWd := myapp.NewWindow("Login/Register")
	loginWd.Resize(fyne.NewSize(500, 300))
	go showLoginScreen(client, loginWd, loginChan)
	loginWd.Show()
	go func() {
		if isSuccess := <-loginChan; isSuccess {
			loginWd.Hide()
			showMainInterface(client, mw)
			mw.Resize(fyne.NewSize(1000, 600))
			mw.Show()
		} else {
			myapp.Quit()
		}
	}()
	mw.SetOnClosed(func() {
		myapp.Quit()
	})
	myapp.Run()

}

// 规范化导出文件的导入
// 修改单信息： task_id,  principal,s tate,   升级说明： task_id, req_no,comment
