package sink

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"time"

	"github.com/meiking/tidb-metrics-crawler/pkg/common"
	"github.com/meiking/tidb-metrics-crawler/pkg/config"
)

// FeishuSink sends processed data as CSV attachments via Feishu
type FeishuSink struct {
	appID         string
	appSecret     string
	receiveID     string
	receiveIDType string
	messageTitle  string
	accessToken   string
	tokenExpiry   time.Time
	httpClient    *http.Client
}

// Feishu token response structure
type feishuTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	AccessToken       string `json:"access_token"`
	ExpireIn          int    `json:"expire_in"`
	TenantAccessToken string `json:"tenant_access_token"`
}

// Feishu upload response structure
type feishuUploadResponse struct {
	Code int `json:"code"`
	Data struct {
		FileKey string `json:"file_key"`
	} `json:"data"`
}

// NewFeishuSink creates a new Feishu sink
func NewFeishuSink(cfg config.FeishuConfig) (*FeishuSink, error) {
	return &FeishuSink{
		appID:         cfg.AppID,
		appSecret:     cfg.AppSecret,
		receiveID:     cfg.ReceiveID,
		receiveIDType: cfg.ReceiveIDType,
		messageTitle:  cfg.MessageTitle,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Write sends processed data as CSV attachment via Feishu
func (s *FeishuSink) Write(metricName string, data []common.ProcessedData) error {
	if len(data) == 0 {
		return nil // Nothing to send
	}

	// Ensure we have a valid access token
	if err := s.ensureAccessToken(); err != nil {
		return fmt.Errorf("failed to get access token: %v", err)
	}

	// Create CSV content in memory
	csvContent, err := s.createCSVContent(metricName, data)
	if err != nil {
		return fmt.Errorf("failed to create CSV content: %v", err)
	}

	// Upload file to Feishu
	fileKey, err := s.uploadFile(metricName, csvContent)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	// Send message with attachment
	if err := s.sendMessage(metricName, fileKey); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// Close cleans up resources
func (s *FeishuSink) Close() error {
	// No resources to clean up for Feishu sink
	return nil
}

// ensureAccessToken gets a new token if current one is expired
func (s *FeishuSink) ensureAccessToken() error {
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return nil // Token is still valid
	}

	// Request new token
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	payload, err := json.Marshal(map[string]string{
		"app_id":     s.appID,
		"app_secret": s.appSecret,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tokenResp feishuTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	if tokenResp.Code != 0 {
		return fmt.Errorf("failed to get token: %s (code: %d)", tokenResp.Msg, tokenResp.Code)
	}

	// Store token with expiry (subtract 1 minute to be safe)
	s.accessToken = tokenResp.TenantAccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpireIn-60) * time.Second)

	return nil
}

// createCSVContent generates CSV content in memory
func (s *FeishuSink) createCSVContent(metricName string, data []common.ProcessedData) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	// Write header
	header, err := createHeaderRow(data[0])
	if err != nil {
		return nil, err
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data rows
	for _, item := range data {
		row, err := createDataRow(item)
		if err != nil {
			return nil, err
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// createHeaderRow generates CSV header from ProcessedData structure
func createHeaderRow(data common.ProcessedData) ([]string, error) {
	// Start with fixed columns
	header := []string{
		"prometheus_instance",
		"metric_name",
		"timestamp",
		"value",
	}

	// Add label keys sorted alphabetically for consistent ordering
	labelKeys := make([]string, 0, len(data.Labels))
	for k := range data.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)

	header = append(header, labelKeys...)
	return header, nil
}

// createDataRow converts ProcessedData to a CSV row
func createDataRow(data common.ProcessedData) ([]string, error) {
	// Start with fixed fields
	row := []string{
		data.PrometheusInstance,
		data.MetricName,
		data.Timestamp.Format(time.RFC3339),
		fmt.Sprintf("%v", data.Value),
	}

	// Get sorted label keys to match header order
	labelKeys := make([]string, 0, len(data.Labels))
	for k := range data.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)

	// Add label values in the same order as header
	for _, k := range labelKeys {
		row = append(row, data.Labels[k])
	}

	return row, nil
}

// uploadFile uploads CSV content to Feishu
func (s *FeishuSink) uploadFile(metricName string, content []byte) (string, error) {
	url := "https://open.feishu.cn/open-apis/drive/v1/files/upload_all"

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file content
	part, err := writer.CreateFormFile("file", fmt.Sprintf("%s_%s.csv", metricName, time.Now().Format("20060102150405")))
	if err != nil {
		return "", err
	}
	if _, err := part.Write(content); err != nil {
		return "", err
	}

	// Add other fields
	if err := writer.WriteField("file_type", "csv"); err != nil {
		return "", err
	}
	if err := writer.WriteField("folder_token", ""); err != nil {
		return "", err
	}

	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var uploadResp feishuUploadResponse
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return "", err
	}

	if uploadResp.Code != 0 {
		return "", fmt.Errorf("upload failed: %s", string(respBody))
	}

	return uploadResp.Data.FileKey, nil
}

// sendMessage sends a message with file attachment
func (s *FeishuSink) sendMessage(metricName, fileKey string) error {
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=%s", s.receiveIDType)

	payload, err := json.Marshal(map[string]interface{}{
		"receive_id": s.receiveID,
		"msg_type":   "file",
		"content": map[string]string{
			"file_key": fileKey,
			"title":    fmt.Sprintf("%s - %s", s.messageTitle, metricName),
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("message send failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}
