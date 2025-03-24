package core

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"move_project/model"

	"github.com/go-redis/redis/v8"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkauthen "github.com/larksuite/oapi-sdk-go/v3/service/authen/v1"
	larkext "github.com/larksuite/oapi-sdk-go/v3/service/ext"
	"github.com/pkg/errors"
)

var (
	client *model.FeishuClient
	config *larkext.AuthenAccessTokenRespBody
	rdb    *redis.Client
)

func NewFeishuClient(AccessToken string) *model.FeishuClient {
	return &model.FeishuClient{
		AccessToken: AccessToken,
		Client:      lark.NewClient(model.AppID, model.AppSecret),
	}
}

func InitRedis() *redis.Client {
	rdb = redis.NewClient(&redis.Options{
		Addr:     model.RedisAddress,
		Password: model.RedisPassword,
	})

	return rdb
}

func RefreshUserToken(ctx context.Context) error {
	fmt.Printf("====================刷新用户token====================")
	req := larkauthen.NewCreateOidcRefreshAccessTokenReqBuilder().
		Body(larkauthen.NewCreateOidcRefreshAccessTokenReqBodyBuilder().
			GrantType(`refresh_token`).
			RefreshToken(config.RefreshToken).
			Build()).
		Build()

	resp, err := client.Client.Authen.V1.OidcRefreshAccessToken.Create(ctx, req)
	if err != nil {
		sendMemberAlertMessage(ctx, err)
		return err
	}

	if !resp.Success() {
		sendMemberAlertMessage(ctx, resp.CodeError)
		return errors.New("刷新用户Token失败")
	}

	config.RefreshToken = *resp.Data.RefreshToken
	config.AccessToken = *resp.Data.AccessToken
	config.ExpiresIn = int64(*resp.Data.ExpiresIn)
	cacheMap, _ := larkcore.StructToMap(config)
	rdb.HMSet(ctx, model.Prefix+":"+fmt.Sprintf("%s:%s", config.Name, config.OpenID), cacheMap)
	client = NewFeishuClient(config.AccessToken)
	fmt.Printf("====================刷新用户token成功====================")
	return nil
}

// 阻塞等待 Key 存在
func WaitForKeyExistenceForUser(ctx context.Context, openID string, userName string) error {
	key := fmt.Sprintf("%s:%s", model.Prefix, openID)
	for {
		exists, err := rdb.Exists(ctx, key).Result()
		if err != nil {
			SendErrorMessage(ctx, userName, fmt.Errorf("%s用户检查 Key 存在性失败: %w", config.Name, err))
			return fmt.Errorf("检查 Key 存在性失败: %w", err)
		}

		if exists == 1 {
			// 获取配置以获取最新的 token
			err = getConfigFromRedisForUser(ctx, openID)
			if err != nil {
				SendErrorMessage(ctx, userName, fmt.Errorf("%s用户获取配置失败: %v\n", config.Name, err))
				return fmt.Errorf("获取配置失败: %v\n", err)
			}

			client = NewFeishuClient(config.AccessToken)

			// 设置定时器，当剩余时间小于等于 30 second时，调用 RefreshUserToken 方法
			go func() {
				wait := time.Duration(config.ExpiresIn)*time.Second - 30*time.Second
				time.Sleep(wait)
				if err = RefreshUserToken(ctx); err != nil {
					SendErrorMessage(ctx, userName, errors.New("刷新授权失败，请用户手动授权"))
					sendAuthorizeMessage(ctx, userName, openID)
				}
			}()

			break
		} else {
			// 调用 sendAuthorizeMessage 方法请求重新授权
			sendAuthorizeMessage(ctx, userName, openID)
		}

		time.Sleep(model.CheckInterval)

	}

	return nil
}

// 获取并解析配置，针对单个用户
func getConfigFromRedisForUser(ctx context.Context, openID string) error {
	key := fmt.Sprintf("%s:%s", model.Prefix, openID)
	value, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("获取配置失败: %w", err)
	}

	expires, _ := strconv.ParseInt(value["expires_in"], 10, 64)

	config = &larkext.AuthenAccessTokenRespBody{
		OpenID:       value["open_id"],
		Name:         value["name"],
		AccessToken:  value["access_token"],
		RefreshToken: value["refresh_token"],
		ExpiresIn:    expires,
	}

	return nil
}
