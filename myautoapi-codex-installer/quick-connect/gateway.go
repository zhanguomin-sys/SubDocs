package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

func configureWithKey(ctx context.Context, client *http.Client, origin, input string, save func(string) (string, error)) (string, error) {
	key, err := validateKey(input)
	if err == nil {
		err = verifyGatewayKey(ctx, client, origin, key)
	}
	if err != nil {
		return "", errors.New(err.Error() + "\n客户端配置没有改动。")
	}
	return save(key)
}

// Check the key using the gateway's read-only usage endpoint. Do not pass
// response text or transport error details to the UI.
func verifyGatewayKey(ctx context.Context, client *http.Client, origin, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/v1/usage", nil)
	if err != nil {
		return errors.New("无法准备 MyAutoAPI 密钥校验")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("暂时无法连接 MyAutoAPI，请检查网络后重试")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return errors.New("MyAutoAPI 没有识别这个密钥。请从本站的 API 密钥页重新复制完整密钥。")
	case http.StatusForbidden:
		return errors.New("MyAutoAPI 拒绝使用此密钥。请检查启用状态及 IP 访问限制。")
	case http.StatusTooManyRequests:
		return errors.New("MyAutoAPI 请求过多，请稍后重试。")
	default:
		return fmt.Errorf("MyAutoAPI 暂时无法验证密钥（HTTP %d）", resp.StatusCode)
	}

	var usage struct {
		Valid     *bool      `json:"isValid"`
		Status    string     `json:"status"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	const maxUsageBytes = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxUsageBytes+1))
	if err != nil || len(data) > maxUsageBytes || json.Unmarshal(data, &usage) != nil || usage.Valid == nil {
		return errors.New("MyAutoAPI 返回的密钥校验结果无法识别，请稍后重试。")
	}
	if usage.Status == "expired" || (usage.ExpiresAt != nil && !usage.ExpiresAt.After(time.Now())) {
		return errors.New("这个 MyAutoAPI 密钥已过期，请换一个有效密钥。")
	}
	if usage.Status == "quota_exhausted" {
		return errors.New("这个 MyAutoAPI 密钥已用完设定额度，请在密钥管理页检查。")
	}
	if !*usage.Valid {
		return errors.New("这个 MyAutoAPI 密钥当前不可用，请在密钥管理页检查。")
	}
	return nil
}

type keyGateway struct {
	client *http.Client
	origin string
}

// Fetch the key's model catalog without sending an inference request.
func listGatewayModels(ctx context.Context, client *http.Client, origin, key string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/v1/models", nil)
	if err != nil {
		return nil, errors.New("无法准备 MyAutoAPI 模型查询。")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("暂时无法查询 MyAutoAPI 模型，请检查网络后重试。")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, errors.New("此密钥无法查询模型。请在 MyAutoAPI 的「API 密钥」页确认密钥有效且分组正确。")
	case http.StatusTooManyRequests:
		return nil, errors.New("MyAutoAPI 请求过多，请稍后重试。")
	default:
		return nil, fmt.Errorf("MyAutoAPI 暂时无法查询模型（HTTP %d）。", resp.StatusCode)
	}

	var listing struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	const maxModelsBytes = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxModelsBytes+1))
	if err != nil || len(data) > maxModelsBytes || json.Unmarshal(data, &listing) != nil || listing.Data == nil {
		return nil, errors.New("无法识别 MyAutoAPI 返回的模型列表，请稍后重试。")
	}
	models := make([]string, 0, len(listing.Data))
	for _, model := range listing.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func (g keyGateway) requireModels(ctx context.Context, key, keyGroup string, required ...string) error {
	models, err := listGatewayModels(ctx, g.client, g.origin, key)
	if err != nil {
		return err
	}
	var missing []string
	for _, model := range required {
		if !slices.Contains(models, model) {
			missing = append(missing, model)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("当前密钥不支持 %s。请到 MyAutoAPI 的「API 密钥」页选择「%s」的密钥。", strings.Join(missing, "、"), keyGroup)
	}
	return nil
}
