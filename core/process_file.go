package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"move_project/model"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/pkg/errors"
)

// 新增函数：针对单个用户执行业务逻辑
func ExecuteBusinessLogicForUser(ctx context.Context, userName string, openID string) error {
	fmt.Printf(">>> 启动业务流程 [任务: %s] <<<\n", openID)
	defer fmt.Println(">>> 业务流程结束 <<<")
	SendCompletionMessage(ctx, config.Name, "开始")

	// 根目录
	files := make([]larkdrive.File, 0)
	err := GetFileListWithToken(ctx, userName, "", "", &files)
	if err != nil {
		return err
	}

	var (
		failedFiles []model.FailedFiles
		filePath    string
	)
	filePath = fmt.Sprintf("%s%s", model.RootPath, config.Name)
	os.MkdirAll(filePath, os.ModePerm)
	for _, file := range files {

		*file.Name = strings.ReplaceAll(*file.Name, "/", "")
		if err = processFile(ctx, config.Name, &file, filePath, &failedFiles); err != nil {
			fmt.Printf("处理文件失败: %v\n", err)
			continue
		}
	}

	if len(failedFiles) > 0 {
		sendAlertMessage(ctx, config.Name, failedFiles)
	} else {
		SendCompletionMessage(ctx, config.Name, "结束")
	}

	rdb.Set(ctx, model.Prefix+":"+fmt.Sprintf("%s:%s", config.Name, openID), 1, 7*time.Hour*24)

	return nil
}

func processFile(ctx context.Context, user string, file *larkdrive.File, filePath string, failedFiles *[]model.FailedFiles) error {
	if *file.Type == "folder" {
		// 检查并创建文件夹路径
		filePath += fmt.Sprintf("/%s", strings.ReplaceAll(*file.Name, " ", ""))
		os.MkdirAll(filePath, os.ModePerm)
		curFiles := make([]larkdrive.File, 0)
		err := GetFileListWithToken(ctx, user, *file.Token, "", &curFiles)
		if err != nil {
			SendErrorMessage(ctx, user, errors.Wrap(err, "获取文件列表失败"))
			return err
		}
		for _, subFile := range curFiles {
			if err = processFile(ctx, user, &subFile, filePath, failedFiles); err != nil {
				continue
			}
		}

	} else if *file.Type == "docx" || *file.Type == "sheet" || *file.Type == "doc" || *file.Type == "docs" {
		if err := downloadFile(ctx, user, filePath, file, failedFiles); err != nil {
			return err
		}
	}
	return nil
}

func GetFileListWithToken(ctx context.Context, userName string, folderToken, pageToken string, files *[]larkdrive.File) error {
	req := larkdrive.NewListFileReqBuilder().
		OrderBy(`CreatedTime`).
		Direction(`DESC`)

	if folderToken != "" {
		req.FolderToken(folderToken) // 使用文件夹 token
	}

	if pageToken != "" {
		req.PageToken(pageToken) // 使用 page token
	}

	resp, err := client.Client.Drive.V1.File.List(context.Background(), req.Build(), larkcore.WithUserAccessToken(config.AccessToken))
	if err != nil {
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("获取文件列表失败")))
		return err
	}

	if !resp.Success() {
		err = errors.New("获取文件列表失败")
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("获取文件列表失败 返回： %s", larkcore.Prettify(resp.CodeError))))
		return fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	if resp == nil {
		return errors.New("获取文件列表 请求发生错误")
	}

	if len(resp.Data.Files) != 0 {
		for _, file := range resp.Data.Files {
			*files = append(*files, *file)
		}
	}
	if resp.Data.NextPageToken != nil {
		err = GetFileListWithToken(ctx, userName, folderToken, *resp.Data.NextPageToken, files) // 使用文件夹 token 和 page token
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(ctx context.Context, user, filePath string, file *larkdrive.File, failedFiles *[]model.FailedFiles) error {
	fileExtension := *file.Type
	if *file.Type == "sheet" || *file.Type == "bitable" {
		fileExtension = "xlsx"
	}
	if *file.Type == "docs" || *file.Type == "doc" {
		fileExtension = "docx"
	}
	ticket := CreateDownLoadTask(ctx, user, &model.DownLoadTaskInfo{
		FileExtension: fileExtension,
		Token:         *file.Token,
		Type:          *file.Type,
	})
	if ticket == nil {
		*failedFiles = append(*failedFiles, model.FailedFiles{
			Name:    *file.Name,
			Address: *file.Url,
			Path:    filePath,
		})
		return errors.New("创建任务失败")
	}
	time.Sleep(3 * time.Second)
	taskInfo := SelectTask(ctx, user, *ticket, *file.Token)
	waitIndex := 0
	if taskInfo == nil {
		*failedFiles = append(*failedFiles, model.FailedFiles{
			Name:    *file.Name,
			Address: *file.Url,
			Path:    filePath,
		})
		return errors.New("文件获取失败")
	}

	for *taskInfo.FileToken == "" {
		fmt.Printf("%s 文件重试中", *file.Name)
		if waitIndex > 1 {
			SendErrorMessage(ctx, user, errors.New(fmt.Sprintf("获取任务失败: %s", *file.Name)))
			break
		}
		time.Sleep(3 * time.Second)
		taskInfo = SelectTask(ctx, user, *ticket, *file.Token)
		if taskInfo == nil {
			SendErrorMessage(ctx, user, errors.New(fmt.Sprintf("获取任务失败: %s", *file.Name)))
			*failedFiles = append(*failedFiles, model.FailedFiles{
				Name:    *file.Name,
				Address: *file.Url,
				Path:    filePath,
			})
			return errors.New("文件获取失败")
		}
		waitIndex++
	}

	if err := DownloadFile(ctx, user, taskInfo, fileExtension, filePath); err != nil {
		*failedFiles = append(*failedFiles, model.FailedFiles{
			Name:    *file.Name,
			Address: *file.Url,
			Path:    filePath,
		})
		return err
	}
	return nil
}

func CreateDownLoadTask(ctx context.Context, userName string, info *model.DownLoadTaskInfo) *string {
	req := larkdrive.NewCreateExportTaskReqBuilder().
		ExportTask(larkdrive.NewExportTaskBuilder().
			FileExtension(info.FileExtension).
			Token(info.Token).
			Type(info.Type).
			SubId(info.SubId).
			Build()).
		Build()

	resp, err := client.Client.Drive.V1.ExportTask.Create(ctx, req, larkcore.WithUserAccessToken(config.AccessToken))
	if err != nil {
		SendErrorMessage(ctx, userName, errors.Wrap(err, "请求创建任务失败"))
		return nil
	}

	if !resp.Success() {
		err = errors.New("创建任务失败")
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("创建任务失败返回： %s", larkcore.Prettify(resp.CodeError))))
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return nil
	}

	fmt.Println(larkcore.Prettify(resp))
	return resp.Data.Ticket
}

func SelectTask(ctx context.Context, userName string, Ticket, Token string) *larkdrive.ExportTask {
	req := larkdrive.NewGetExportTaskReqBuilder().
		Ticket(Ticket).
		Token(Token).
		Build()

	resp, err := client.Client.Drive.V1.ExportTask.Get(ctx, req, larkcore.WithUserAccessToken(config.AccessToken))
	if err != nil {
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("查询任务失败")))
		return nil
	}

	if !resp.Success() {
		err = errors.New("查询任务失败")
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("查询任务失败 返回： %s", larkcore.Prettify(resp.CodeError))))
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return nil
	}

	fmt.Println("查询任务结果:", larkcore.Prettify(resp))
	return resp.Data.Result
}

func DownloadFile(ctx context.Context, userName string, taskInfo *larkdrive.ExportTask, fileType, folderName string) error {
	fmt.Println("===========下载任务============")
	req := larkdrive.NewDownloadExportTaskReqBuilder().
		FileToken(*taskInfo.FileToken).
		Build()

	if *taskInfo.FileToken == "" {
		return errors.New("fileToken is empty")
	}

	resp, err := client.Client.Drive.V1.ExportTask.Download(context.Background(), req, larkcore.WithUserAccessToken(config.AccessToken))
	if err != nil {
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("下载失败 %s", *taskInfo.FileName)))
		return err
	}

	if !resp.Success() {
		err = errors.New("下载失败")
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("下载失败 文件名: %s 返回： %s", *taskInfo.FileName, larkcore.Prettify(resp.CodeError))))
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return err
	}

	filePath := fmt.Sprintf("%s/%s.%s", folderName, *taskInfo.FileName, fileType)
	filePath = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filePath, "|", ""), " ", ""), "\n", "")
	filePath = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filePath, "%", ""), "^", ""), "*", "")
	filePath = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filePath, "?", ""), "#", ""), "@", "")

	if err = resp.WriteFile(filePath); err != nil {
		SendErrorMessage(ctx, userName, errors.Wrap(err, fmt.Sprintf("写入文件失败 %s", *taskInfo.FileName)))
		return err
	}

	return nil

}
