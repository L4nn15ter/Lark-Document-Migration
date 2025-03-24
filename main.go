package main

import (
	"context"
	"fmt"

	"move_project/core"

	"github.com/pkg/errors"
)

var (
	ctx = context.Background()
)

// 初始化 Redis 连接

func main() {
	rdb := core.InitRedis()
	defer rdb.Close()

	// 获取群组成员信息
	members := core.InitMembers(ctx)

	// 遍历 members 列表，对每个用户执行业务逻辑
	for _, member := range members {
		complete := core.IsMemberComplete(ctx, member)
		if complete {
			continue
		}
		// 等待 Key 存在
		if err := core.WaitForKeyExistenceForUser(ctx, member.OpenID, member.UserName); err != nil {
			fmt.Printf("等待 Key 出错: %v\n", err)
			core.SendErrorMessage(ctx, member.UserName, errors.Wrap(err, fmt.Sprintf("用户 %s 授权key出错，执行跳过", member.UserName)))
			continue
		}

		if err := core.ExecuteBusinessLogicForUser(ctx, member.UserName, member.OpenID); err != nil {
			core.SendErrorMessage(ctx, member.UserName, err)
		}
	}
}
