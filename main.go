//

//2024.8.8
//TODO: 1. 导入后直接刷新    √
//TODO: 2. 导入补丁时补丁树的需求层没有分割逗号	√
//TODO: 3. 优化登录界面: 记住密码与当前用户    √
//TODO: 5. redis消息队列实现订阅发布模式   √
//TODO: 6. 避免重复登录相同用户     √
//TODO: 7. 考虑要不要做补丁过期自动删除     （cancel）

//2024.8.9
//TODO 1. 补丁树增加信息  		√
//TODO 2. 导入补丁后，将补丁下的任务deadline修改了    √
//TODO 3. BUG:补丁重复导入补丁树的需求行会有空行   √

// 2024.8.12
// TODO: 1. 导入任务是否要将其关联的补丁同步deadline  √
// TODO: 2. 日志库（客户端和服务端）
// TODO: 3. 权限设置  √
// TODO: 4. 收件箱点击功能？   √
// TODO: 5. 当前登录用户预览（邮箱，密码，用户身份）   √
// TODO: 6. 补丁里搜索客户和发布状态
// TODO: 7. 测试删除用例   √

// 2024.8.13 & 2024.8.14  & 2024.8.15
// TODO 1.将广播到客户端的消息改为收件箱模式   √
// TODO 2.撰写帮助文档     40%
// BUG: 删除任务时panic     √
// TODO 3.界面显示用户名   √
// TODO 4. 修改服务端的checklogin（重复了）    √

// 2024.8.16
// TODO 1. 考虑是否加redis缓存
// TODO 2. 撤销操作
package main

// go build -ldflags="-H windowsgui"
import (
	"OrderManager-cli/common"
	"OrderManager-cli/config"
	"OrderManager-cli/pb"
	"context"
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"path/filepath"
)

//go:embed pytool/read_xls_task.exe
var EmbeddedExeTask []byte

//go:embed pytool/read_xls.exe
var EmbeddedExePatchs []byte

const DAYSPERPAGE = 5

var myapp = app.New()

// 计算订阅到的消息数量，即msgChan里面的消息数
var msgCnt = 0

func main() {
	//log.SetFlags(log.LstdFlags | log.Lshortfile)

	//logrus.SetOutput(&lumberjack.Logger{
	//	Filename:   "./logs/app.log",
	//	MaxSize:    100, // MB
	//	MaxBackups: 30,
	//	MaxAge:     0, // Disable age-based rotation
	//	Compress:   true,
	//})

	//测试
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	tempDir, err := os.MkdirTemp("", "embedded_exe")
	if err != nil {
		fmt.Println("Failed to create temp dir:", err)
		return
	}
	defer os.RemoveAll(tempDir)

	common.ExePathTask = filepath.Join(tempDir, "read_xls_task.exe")
	err = os.WriteFile(common.ExePathTask, EmbeddedExeTask, 0755)
	if err != nil {
		fmt.Println("Failed to write embedded exe:", err)
		return
	}

	common.ExePathPatchs = filepath.Join(tempDir, "read_xls.exe")
	err = os.WriteFile(common.ExePathPatchs, EmbeddedExePatchs, 0755)
	if err != nil {
		fmt.Println("Failed to write embedded exe:", err)
		return
	}

	//defer func() {
	//	if r := recover(); r != nil {
	//		fmt.Println("Recovered in f", r)
	//	}
	//}()

	mw := myapp.NewWindow("Task List for the Week")
	mw.SetMaster()

	// 建立一个链接，请求A服务
	// 真实项目里肯定是通过配置中心拿服务名称，发给注册中心请求真实的A服务地址，这里都是模拟
	// 第二个参数是配置了一个证书，因为没有证书会报错，但是我们目前没有配置证书，所以需要insecure.NewCredentials()返回一个禁用传输安全的凭据
	connect, err := grpc.NewClient(fmt.Sprintf("%s:%s", config.Cfg.Conn.Host, config.Cfg.Conn.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Fatal(err)
	} else {
		logrus.Info("connect success")
	}
	defer connect.Close()
	client := pb.NewServiceClient(connect)

	logChain = NewLogChain(client)

	loginChan := make(chan bool)
	loginWd := myapp.NewWindow("Login/Register")
	loginWd.Resize(fyne.NewSize(500, 300))
	go showLoginScreen(client, loginWd, loginChan)
	loginWd.Show()
	go func() {
		if isSuccess := <-loginChan; isSuccess {
			loginWd.Hide()
			notifyClient := pb.NewNotificationServiceClient(connect)
			stream, err := notifyClient.Subscribe(context.Background(), &pb.SubscriptionRequest{ClientId: config.Cfg.Login.UserName})
			if err != nil {
				logrus.Fatalf("Failed to subscribe: %v", err)
			}

			msgChan := make(chan string, 10)
			go func() {
				for {
					notification, err := stream.Recv()
					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to receive notification: %v", err), mw)
						return
					}
					msgChan <- notification.Message
				}
			}()

			showMainInterface(client, mw, msgChan)
			mw.Resize(fyne.NewSize(1000, 600))
			mw.Show()
		} else {
			myapp.Quit()
		}
	}()

	myapp.Run()

}

// 规范化导出文件的导入
// 修改单信息： task_id,  principal,s tate,   升级说明： task_id, req_no,comment
