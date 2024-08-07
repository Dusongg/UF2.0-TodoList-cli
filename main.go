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

//2024.7.30
//TODO: 1. 改用map存储，展示近五天  √
//TODO: 2. 新增一行：添加任务 + 下一页   √
//TODO: 3. 添加任务选项    √
//TODO: 4. 添加或者就该任务之后,对于本地以及其他用户的界面刷新数据问题  √
//TODO: 5. 导入数据按钮    √
//TODO: 6. 处理过期任务    √

//2024.7.31
//TODO: 1. 批量导入数据	√
//TODO: 2. 收件箱界面     √
//TODO: 3. 表单点击输入框将默认值写出来  √
//TODO: 4. 登录 & 自动登录   session-redis?
//TODO: 5. 删除操作   √

//2024.8.1
//TODO 1. 完成收件箱界面    √
//TODO 2. 完成“补丁”      √
//TODO 3. 预览模块选择展示自己或所有人    √
//TODO 4. 考虑任务持续时间     √
//TODO 5. 定时邮件
//TODO 6. 收件箱刷新     √

//2024.8.2
//TODO 1.补丁界面的任务单显示    √
//TODO 2.完成补丁界面的任务栏部分的功能实现     √

//2024.8.3
//TODO 1. 补丁树下的颜色渐变    √
//TODO 2. 需求和任务下添加任务功能
//TODO 3. 检查ModForm函数  √

const DAYSPERPAGE = 5

var colorTheme1 = color.RGBA{R: 57, G: 72, B: 94, A: 255}

var LoginUser string
var myapp = app.New()

func main() {
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
