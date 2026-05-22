package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/middleware"
)

func (h *Handler) licenseConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteJSON(w, response.ErrDefault("请求失败"))
		return
	}

	var req struct {
		LicenseKey string `json:"license_key"`
		Domain     string `json:"domain"`
		HmacKey    string `json:"hmac_key"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}

	if req.Domain == "" {
		response.WriteJSON(w, response.ErrDefault("面板域名不能为空"))
		return
	}

	const defaultLicenseServerURL = "https://sq.abai.eu.org"
	url := defaultLicenseServerURL
	if os.Getenv("LICENSE_SERVER_URL") != "" {
		url = os.Getenv("LICENSE_SERVER_URL")
	}

	now := time.Now().UnixMilli()

	// 授权码为空时自动生成7天体验授权
	actualLicenseKey := req.LicenseKey
	if actualLicenseKey == "" {
		trialKey, err := requestTrialLicense(url, req.Domain)
		if err != nil {
			response.WriteJSON(w, response.ErrDefault("获取体验授权失败: "+err.Error()))
			return
		}
		actualLicenseKey = trialKey
	}

	if err := h.repo.UpsertConfig("license_key", actualLicenseKey, now); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	if err := h.repo.UpsertConfig("server_domain", req.Domain, now); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}

	if err := h.repo.UpsertConfig("license_server_url", url, now); err != nil {
		log.Printf("⚠️ sync config license_server_url failed: %v", err)
	}

	if req.HmacKey != "" {
		if err := h.repo.UpsertConfig("hmac_key", req.HmacKey, now); err != nil {
			log.Printf("⚠️ sync config hmac_key failed: %v", err)
		}
	}

	middleware.UpdateCheckParams(url, actualLicenseKey, req.Domain)
	go middleware.TriggerAsyncCheck()

	go func() {
		if err := UpdateEnvFile(actualLicenseKey, req.Domain, url, req.HmacKey); err != nil {
			log.Printf("⚠️ failed to write .env persistence: %v", err)
		}
	}()

	response.WriteJSON(w, response.OK(map[string]interface{}{
		"triggered_check": true,
	}))
}

func requestTrialLicense(serverURL, domain string) (string, error) {
	body, _ := json.Marshal(map[string]string{"domain": domain})
	resp, err := http.Post(serverURL+"/api/trial", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		LicenseKey string `json:"license_key"`
		Error      string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf(result.Error)
	}
	return result.LicenseKey, nil
}
