package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
	"xixuanyunsign/utils"
)

type RequestPayload struct {
	Contents []Content `json:"contents"`
}
type Response struct {
	Candidates    []Candidate `json:"candidates"`
	UsageMetadata interface{} `json:"usageMetadata"` // 根据需要定义具体类型
	ModelVersion  string      `json:"modelVersion"`
}
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
	AvgLogprobs  float64 `json:"avgLogprobs"`
}

type Content struct {
	Parts []ContentPart `json:"parts"`
	Role  string        `json:"role"`
}
type ContentPart struct {
	Text string `json:"text"`
}

var (
	filePath string
	role     string
	month    int8
)

var ExperimentalCmd = &cobra.Command{
	Use:   "experimental",
	Short: "实验性命令(自动月报)",
	Run: func(cmd *cobra.Command, args []string) {
		UploadImages(filePath)
	},
}

func init() {
	ExperimentalCmd.Flags().StringVarP(&filePath, "filePath", "f", "", "文件地址")
	ExperimentalCmd.Flags().StringVarP(&role, "role", "r", "", "工作角色")
	ExperimentalCmd.Flags().Int8VarP(&month, "month", "M", 1, "第几月（默认为1）")
	ExperimentalCmd.MarkFlagRequired("filePath")
	ExperimentalCmd.MarkFlagRequired("role")
	ExperimentalCmd.MarkFlagRequired("week")
}

// MonthReportUploadSelectFile uploads a report file to the API and returns the URI from the server's response.
// Accepts the file path and user token as input parameters and processes the HTTP request for file upload.
// Returns an empty string if an error occurs during file upload or response parsing.
func MonthReportUploadSelectFile(filePath, UserToken string) string {
	// API URL
	url := fmt.Sprintf("https://api.xixunyun.com/file/form?token=%s", UserToken)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return ""
	}
	defer file.Close()

	// 创建 multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件字段
	part, err := writer.CreateFormFile("addFile", fmt.Sprintf("img_%d.jpg", time.Now().Unix())) // 根据需要调整字段名
	if err != nil {
		fmt.Printf("Error creating form file: %v\n", err)
		return ""
	}

	// 将文件内容写入 part
	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Printf("Error copying file content: %v\n", err)
		return ""
	}

	// 添加其他字段
	if err := writer.WriteField("input_name", "addFile"); err != nil {
		fmt.Printf("Error writing field input_name: %v\n", err)
		return ""
	}
	if err := writer.WriteField("business", "report"); err != nil {
		fmt.Printf("Error writing field business: %v\n", err)
		return ""
	}

	// 关闭 multipart writer
	err = writer.Close()
	if err != nil {
		fmt.Printf("Error closing writer: %v\n", err)
		return ""
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return ""
	}

	// 添加请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Authorization", UserToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "https://www.xixunyun.com")
	req.Header.Set("Referer", "https://www.xixunyun.com/")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return ""
	}

	fmt.Printf("Response Body: %s\n", string(respBody))

	// 解析 JSON 响应并提取 URI
	var responseBody struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			URI string `json:"uri"`
		} `json:"data"`
	}

	err = json.Unmarshal(respBody, &responseBody)
	if err != nil {
		fmt.Printf("Error parsing response body: %v\n", err)
		return ""
	}

	// 返回提取的 URI
	if responseBody.Data.URI != "" {
		return responseBody.Data.URI
	}

	return ""
}

func UploadImages(filePath string) {
	token, _, _, err := utils.GetUser(account)
	if err != nil || token == "" {
		if debug {
			fmt.Printf("获取用户信息失败: %v\n", err)
		}
		fmt.Println("未找到该账号的 token，请先登录。")
		return
	}
	MonthReportUploadSelectFile(filePath, token)
}

// GenerateContent generates internship monthly report content based on the provided role and API key.
// It sends a request to a generative language API, builds the payload dynamically, and returns the generated text.
// Returns an error if request creation, response handling, or JSON parsing fails.
func GenerateContent(role, apiKey string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)

	// Build request payload
	payload := RequestPayload{
		Contents: []Content{
			{
				Role: "user",
				Parts: []ContentPart{
					{Text: fmt.Sprintf("我是%s。请求生成第%d个月的实习月报内容，并以API形式返回，不带`json`或其他多余信息。", role, month)},
				},
			},
			{
				Role: "user",
				Parts: []ContentPart{
					{Text: `[{"title":"实习工作具体情况及实习任务完成情况","content":"","require":"1","sort":1},{"title":"主要收获及工作成绩","content":"","require":"0","sort":2},{"title":"工作中的问题及需要老师的指导帮助","content":"","require":"0","sort":3}]`},
				},
			},
		},
	}

	// Serialize payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("Failed to serialize JSON payload: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("Failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second, // Add timeout to prevent infinite wait
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP request failed, status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Read and parse response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response body: %v", err)
	}

	var responseData Response
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		return "", fmt.Errorf("Failed to parse response JSON: %v", err)
	}

	// Extract generated text
	if len(responseData.Candidates) == 0 || len(responseData.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Response did not contain any content")
	}

	generatedText := responseData.Candidates[0].Content.Parts[0].Text
	return generatedText, nil
}