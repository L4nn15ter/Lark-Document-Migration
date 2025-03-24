package model

import "time"

const (
	Prefix        = "lark:userinfo:" // 监控的 Redis Key
	CheckInterval = 20 * time.Second // 检查间隔
	AppID         = ""
	AppSecret     = ""
	RootPath      = ""
	ChatID        = ""
	AuthAddress   = "https://applink.feishu.cn/client/web_app/open?appId="
	RedisAddress = ""
	RedisPassword = ""
)
