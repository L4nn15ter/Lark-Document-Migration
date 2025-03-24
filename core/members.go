package core

import (
	"context"
	"fmt"

	"move_project/model"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

var (
	appClient *lark.Client
	members   = make([]model.Members, 0)
)

func IsMemberComplete(ctx context.Context, member model.Members) bool {
	exist, _ := rdb.Exists(ctx, model.Prefix+":"+fmt.Sprintf("%s:%s", member.UserName, member.OpenID)).Result()
	if exist == 1 {
		return true
	}

	return false
}

func InitMembers(ctx context.Context) []model.Members {
	appClient = lark.NewClient(model.AppID, model.AppSecret)
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(`cli_a3873a3889b9d013`).
			AppSecret(`3t9ev5VL5k8D4yEBTKxiIE3iLWApkVg4`).
			Build()).
		Build()

	// 发起请求
	resp, err := appClient.Auth.V3.TenantAccessToken.Internal(context.Background(), req)

	// 处理错误
	if err != nil {
		sendMemberAlertMessage(ctx, err)
		return nil
	}

	// 服务端错误处理
	if !resp.Success() {
		sendMemberAlertMessage(ctx, resp.CodeError)
	}

	GetGroupMember(ctx, "")

	return members
}

func GetGroupMember(ctx context.Context, pageToken string) {
	memberReq := larkim.NewGetChatMembersReqBuilder().
		ChatId(model.ChatID).
		MemberIdType(`open_id`)

	if pageToken != "" {
		memberReq.PageToken(pageToken)
	}
	memberResp, err := appClient.Im.V1.ChatMembers.Get(context.Background(), memberReq.Build())
	if err != nil {
		sendMemberAlertMessage(ctx, err)
		return
	}

	if !memberResp.Success() {
		sendMemberAlertMessage(ctx, memberResp.CodeError)
	}

	for _, item := range memberResp.Data.Items {
		members = append(members, model.Members{
			OpenID:   *item.MemberId,
			UserName: *item.Name,
		})
	}

	if *memberResp.Data.PageToken != "" {
		GetGroupMember(ctx, *memberResp.Data.PageToken)
	}

}
