package model

import "github.com/larksuite/oapi-sdk-go/v3"

type Response struct {
	Code int    `json:"code"`
	Data Data   `json:"data"`
	Msg  string `json:"msg"`
}

type Data struct {
	Files         []File `json:"files"`
	HasMore       bool   `json:"has_more"`
	NextPageToken string `json:"next_page_token"`
}

type File struct {
	CreatedTime  string `json:"created_time"`
	ModifiedTime string `json:"modified_time"`
	Name         string `json:"name"`
	OwnerId      string `json:"owner_id"`
	ParentToken  string `json:"parent_token"`
	Token        string `json:"token"`
	Type         string `json:"type"`
	Url          string `json:"url"`
}

type FeishuClient struct {
	AppId       string
	AppSecret   string
	AccessToken string
	Client      *lark.Client
}

type DownLoadTaskInfo struct {
	FileExtension string
	Token         string
	Type          string
	SubId         string
}
type Members struct {
	OpenID   string `json:"open_id"`
	UserName string `json:"user_name"`
}
