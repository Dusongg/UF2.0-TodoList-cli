# 安装&运行

1. `git clone git@github.com:Dusongg/UF2.0-TodoList-cli.git` 

   或 `git clone https://github.com/Dusongg/UF2.0-TodoList-cli.git`

2. 打开`./config/config.json`,  修改服务器的ip和端口， Login部分用于保存上一次的登录信息

   ```json
   {
       "Conn": {
           "host": "localhost",
           "port": "8001"
       },
       "Login": {
           "username": "dusong",
           "password": "123123"
       },
       "undo_log_task_size": "10"
   }
   ```

3. 运行（以Windows为例）：双击击`OrderManager-cli.exe`运行

   

# 界面介绍

## 1.1 登录界面

![image-20240816162854616](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816162854616.png)

## 1.2 预览界面

![image-20240816164355646](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816164355646.png)

- 个人信息界面

![12](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816165837086.png)

## 1.3 收件箱界面

![image-20240816164919869](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816164919869.png)

## 1.4 补丁界面

![image-20240816165547718](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816165547718.png)



## 1.5 发布订阅模式下的消息广播

### 1.5.1 管理员修改他人任务

1. ![image-20240816170232610](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816170232610.png)
2. ![image-20240816170345052](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816170345052.png)
3. ![image-20240816170632632](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816170632632.png)

### 1.5.2 通过redis发布

![image-20240816171059708](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240816171059708.png)



