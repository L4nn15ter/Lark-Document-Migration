package core

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"move_project/model"

	jsoniter "github.com/json-iterator/go"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func NewNotify(ctx context.Context, template, msg string) (string, string) {
	card := larkcard.NewMessageCard().
		Header(&larkcard.MessageCardHeader{
			Title_:    larkcard.NewMessageCardPlainText().Content("文件迁移通知"),
			Template_: &template,
		}).Elements([]larkcard.MessageCardElement{
		larkcard.NewMessageCardDiv().Text(larkcard.NewMessageCardPlainText().Content(msg)),
	}).Build()

	cardJson, _ := card.JSON()
	return "", cardJson
}

func SendToGroup(ctx context.Context, content string) error {
	req := larkim.NewCreateMessageReqBuilder().ReceiveIdType("chat_id").
		Body(
			larkim.NewCreateMessageReqBodyBuilder().MsgType("interactive").
				ReceiveId(model.ChatID).
				Content(content).
				Build(),
		).Build()

	rsp, err := appClient.Im.Message.Create(ctx, req)
	if err != nil {
		return err
	}
	if !rsp.Success() {
		return rsp.CodeError
	}

	return nil
}

// 发送错误消息
func SendErrorMessage(ctx context.Context, userName string, err error) {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("用户： %s\n", userName))
	b.WriteString(fmt.Sprintf("错误： %+v\n", err))
	_, msg := NewNotify(ctx, larkcard.TemplateRed, b.String())
	if err = SendToGroup(ctx, msg); err != nil {
		fmt.Printf("发送告警消息给用户 %s，失败", userName)
	}
}

// 发送迁移失败文件消息
func sendAlertMessage(ctx context.Context, user string, failedFiles []model.FailedFiles) {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("项目： %s\n", "导出任务"))
	b.WriteString(fmt.Sprintf("用户： %s\n", user))

	columns := make([]model.Column, 0)
	columns = append(columns, model.Column{
		DataType:        "text",
		Name:            "customer_name",
		DisplayName:     "文件名",
		HorizontalAlign: "left",
		Width:           "auto",
	}, model.Column{
		DataType:        "markdown",
		Name:            "customer_scale",
		DisplayName:     "地址",
		HorizontalAlign: "left",
		Width:           "auto",
	})
	rows := make([]model.Row, 0)
	for _, file := range failedFiles {
		str, _ := strings.CutPrefix(file.Path, model.RootPath)
		rows = append(rows, model.Row{
			CustomerName:  fmt.Sprintf("%s/%s", str, file.Name),
			CustomerScale: file.Address,
		})
	}

	element := model.Element{
		Tag:       "table",
		Columns:   columns,
		Rows:      rows,
		RowHeight: "low",
		HeaderStyle: model.HeaderStyle{
			BackgroundStyle: "none",
			Bold:            true,
			Lines:           1,
		},
		PageSize: 20,
		Margin:   "0px 0px 0px 0px",
	}

	card := model.Card{
		Schema: "2.0",
		Config: model.Config{
			UpdateMulti: true,
		},
		Body: model.Body{
			Direction: "vertical",
			Padding:   "12px 12px 12px 12px",
			Elements:  []model.Element{element},
		},
		Header: model.Header{
			Title: model.Title{
				Tag:     "plain_text",
				Content: "文件迁移失败列表",
			},
			Subtitle: model.Title{
				Tag:     "plain_text",
				Content: "",
			},
			Template: larkcard.TemplateRed,
			Padding:  "12px 12px 12px 12px",
		},
	}

	json, _ := jsoniter.MarshalToString(card)
	if err := SendToGroup(ctx, json); err != nil {
		fmt.Printf("发送告警消息给用户 %s，失败", user)
	}
	return
}

// 发送完成消息
func SendCompletionMessage(ctx context.Context, user string, status string) {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("项目： %s\n", "导出任务"))
	b.WriteString(fmt.Sprintf("用户： %s\n", user))
	b.WriteString(fmt.Sprintf("状态： %s\n", status))
	_, msg := NewNotify(ctx, larkcard.TemplateGreen, b.String())
	if err := SendToGroup(ctx, msg); err != nil {
		fmt.Printf("发送成功消息给用户 %s，失败", user)
	}
	return
}

// 发送请求授权消息消息
func sendAuthorizeMessage(ctx context.Context, user string, openID string) {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("需要用户授权： %s\n", user))
	template := larkcard.TemplateYellow
	card := larkcard.NewMessageCard().
		Header(&larkcard.MessageCardHeader{
			Title_:    larkcard.NewMessageCardPlainText().Content("文件迁移通知"),
			Template_: &template,
		}).Elements([]larkcard.MessageCardElement{
		larkcard.NewMessageCardDiv().Text(larkcard.NewMessageCardPlainText().Content(b.String())),
		larkcard.NewMessageCardDiv().Text(larkcard.NewMessageCardLarkMd().Content(fmt.Sprintf("<at id=%s>Name</at> \n", openID))),
		larkcard.NewMessageCardMarkdown().Content(fmt.Sprintf("[请求授权地址](%s)", model.AuthAddress)),
	}).Build()

	msg, _ := card.JSON()
	req := larkim.NewCreateMessageReqBuilder().ReceiveIdType("chat_id").
		Body(
			larkim.NewCreateMessageReqBodyBuilder().MsgType("interactive").
				ReceiveId(model.ChatID).
				Content(msg).
				Build(),
		).Build()

	rsp, err := appClient.Im.Message.Create(ctx, req)
	if err != nil {
		fmt.Printf("发送告警消息给用户 %s，失败", user)
		return
	}
	if !rsp.Success() {
		fmt.Printf("发送告警消息给用户 %s，失败", user)
		return
	}

	return
}

// 发送请求成员列表错误消息
func sendMemberAlertMessage(ctx context.Context, err error) {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("获取群组成员信息失败\n"))
	b.WriteString(fmt.Sprintf("错误：%+v", err))
	_, msg := NewNotify(ctx, larkcard.TemplateRed, b.String())
	if err = SendToGroup(ctx, msg); err != nil {
		fmt.Printf("发送请求成员列表告警消息给用户 %s，失败", config.Name)
	}
}
