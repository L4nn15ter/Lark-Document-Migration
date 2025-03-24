```
├── core                 // 核心代码
    ├── auth             // 鉴权相关结构体         
    ├── members          // 获取消息群组人员信息
    ├── message          // 群组发送消息卡片方法
    ├── process_file     // 文件处理方法       
├── model                // 用户、用户组相关的数据库表结构体暂存本地目录
    ├── card             // 消息卡片结构体
    ├── constant         // 常量
    ├── response         // 飞书API返回结构体
├── main                 // 程序入口  
```

## 飞书云文档迁移脚本
### 1. 前言
由于飞书不提供企业切换飞书账号/转移平台所需的云文档批量导出服务，所以自己写了一个脚本用来导出公司成员的所有云文档（由于接口限制，仅导出doc/docx/xlsx格式文件），脚本写的比较粗陋，欢迎各位一起完善。

### 2.迁移前准备
脚本依赖飞书后台应用来完成文档导出任务，所以需要用户在[飞书开放平台](https://open.feishu.cn/app/)自建应用 -> 为应用添加机器人能力 -> 开启应用群组权限、用户权限、云文档相关权限（具体权限请参照下文调用的API接口，在飞书的API调试台中对应开启）-> 创建用以文件迁移的群组 -> 将需要迁移的用户拉入群 -> 填写脚本 model/constant 下所需的常量 -> 执行脚本

### 3.程序执行流程

- a. 初始化阶段
加载配置：
加载 model/constant.go 中的常量配置（如 Redis Key 前缀、检查间隔、AppID、AppSecret 等）。
初始化 Redis 连接：
调用 core/auth.go 中的 InitRedis 方法，连接 Redis 数据库。
获取群组成员信息：
调用 core/members.go 中的 InitMembers 方法，通过飞书 API 获取群组成员列表。
如果获取失败，发送告警消息（调用 core/message.go 中的 sendMemberAlertMessage 方法）。
- b. 遍历成员列表
检查成员是否已完成任务：
调用 core/members.go 中的 IsMemberComplete 方法，检查 Redis 中是否存在对应成员的任务完成标记。
如果已存在，跳过该成员；否则继续执行后续逻辑。
- c. 等待授权 Key 存在
阻塞等待 Key 存在：
调用 core/auth.go 中的 WaitForKeyExistenceForUser 方法，循环检查 Redis 中是否存在指定用户的授权 Key。
如果不存在，发送请求授权消息（调用 core/message.go 中的 sendAuthorizeMessage 方法），并继续等待。
刷新用户 Token：
当 Key 存在时，调用 getConfigFromRedisForUser 方法从 Redis 获取用户配置。
设置定时器，在 Token 即将过期时调用 RefreshUserToken 方法刷新 Token。
- d. 执行业务逻辑
启动业务流程：
调用 core/process_file.go 中的 ExecuteBusinessLogicForUser 方法，针对单个用户执行文件迁移逻辑。
发送开始通知（调用 core/message.go 中的 SendCompletionMessage 方法）。
获取文件列表：
调用 GetFileListWithToken 方法，递归获取指定文件夹下的所有文件和子文件夹。
处理文件：
对每个文件调用 processFile 方法：
如果是文件夹，递归创建目录结构。
如果是文档类型文件（如 docx, sheet），调用 downloadFile 方法下载文件。
下载文件：
创建下载任务（调用 CreateDownLoadTask 方法）。
查询任务状态（调用 SelectTask 方法）。
下载文件（调用 DownloadFile 方法）。
如果下载失败，记录失败文件信息。
发送失败文件通知：
如果存在失败文件，调用 core/message.go 中的 sendAlertMessage 方法发送失败文件列表。
- e. 更新任务状态
设置任务完成标记：
在 Redis 中设置任务完成标记，有效期为 7 天。
发送完成通知：
调用 core/message.go 中的 SendCompletionMessage 方法发送任务完成通知。
 
#### 模块交互总结
- 主流程：
main.go 是程序入口，负责初始化 Redis 和成员列表，并遍历成员执行业务逻辑。
- 授权管理：
auth.go 提供了 Redis 操作、Token 刷新和授权等待功能。
- 文件处理：
process_file.go 实现了文件列表获取、文件夹递归处理和文件下载的核心逻辑。
- 消息通知：
message.go 提供了多种消息通知功能，包括错误告警、任务完成通知和失败文件列表通知。
- 成员管理：
members.go 负责获取群组成员信息，并检查成员任务状态。

### 调用接口汇总
- [刷新 user_access_token](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/authentication-management/access-token/refresh-user-access-token)
- [自建应用获取 app_access_token](https://open.feishu.cn/document/server-docs/authentication-management/access-token/app_access_token_internal)
- [获取群信息](https://open.feishu.cn/document/server-docs/group/chat/get-2)
- [获取用户文件清单](https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/list)
- [创建导出任务](https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/create)
- [查询导出任务结果](https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/get)
- [下载导出文件](https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/download)
