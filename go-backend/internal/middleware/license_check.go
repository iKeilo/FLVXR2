package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// LicenseVerifier handles license verification with remote server
type LicenseVerifier struct {
	serverURL      string
	licenseKey     string
	domain         string
	accessDomain   string
	accessProtocol string
	httpClient     *http.Client
}

// VerifyRequest is the request body for license verification
type VerifyRequest struct {
	LicenseKey     string `json:"license_key"`
	Domain         string `json:"domain"`
	AccessDomain   string `json:"access_domain,omitempty"`
	AccessProtocol string `json:"access_protocol,omitempty"`
	ServerIP       string `json:"server_ip,omitempty"` // 新增：面板服务端上报的 IP
}

// VerifyResponse is the response body for license verification
type VerifyResponse struct {
	Valid          bool                   `json:"valid"`
	ExpireTime     int64                  `json:"expire_time,omitempty"`
	Username       string                 `json:"username,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	IsTrial        bool                   `json:"is_trial"`
	PlanName       string                 `json:"plan_name,omitempty"`
	LicenseProfile string                 `json:"license_profile,omitempty"`
	Entitlements   CommercialEntitlements `json:"entitlements"`
	Signature      string                 `json:"signature,omitempty"`
}

type CommercialEntitlements struct {
	PersonalUseOnly    bool   `json:"personal_use_only"`
	CommercialAllowed  bool   `json:"commercial_allowed"`
	ResaleAllowed      bool   `json:"resale_allowed"`
	MultiTenantAllowed bool   `json:"multi_tenant_allowed"`
	WhiteLabelAllowed  bool   `json:"white_label_allowed"`
	APIAccessAllowed   bool   `json:"api_access_allowed"`
	BillingAllowed     bool   `json:"billing_allowed"`
	DeploymentScope    string `json:"deployment_scope"`
	SupportLevel       string `json:"support_level"`
	TeamSizeLimit      int    `json:"team_size_limit"`
}

// licenseState stores the current license state
type licenseState struct {
	valid          bool
	expireTime     int64
	reason         string
	isTrial        bool
	planName       string
	licenseProfile string
	entitlements   CommercialEntitlements
	LastCheck      time.Time
	mu             sync.RWMutex
}

// ObscuredHMACKey returns the HMAC secret key used to verify license server signatures.
func ObscuredHMACKey() string {
	return os.Getenv("HMAC_SECRET_KEY")
}

// VerifyResponseSignature checks the HMAC signature of a license server response.
func VerifyResponseSignature(resp *VerifyResponse, secret string) bool {
	if resp.Signature == "" || secret == "" {
		return true
	}
	sigPayload := fmt.Sprintf("%v:%d:%s", resp.Valid, resp.ExpireTime, resp.Reason)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigPayload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(resp.Signature))
}

var globalLicenseState = &licenseState{}

func normalizeLicenseProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "", "trial", "evaluation", "eval":
		return "evaluation"
	case "personal", "community", "selfhost":
		return "personal"
	case "business", "commercial", "pro":
		return "business"
	case "enterprise", "corp":
		return "enterprise"
	case "channel", "reseller", "partner":
		return "channel"
	default:
		return "business"
	}
}

func defaultEntitlementsForProfile(profile string) CommercialEntitlements {
	switch normalizeLicenseProfile(profile) {
	case "personal", "evaluation":
		return CommercialEntitlements{
			PersonalUseOnly:    true,
			CommercialAllowed:  false,
			ResaleAllowed:      false,
			MultiTenantAllowed: false,
			WhiteLabelAllowed:  false,
			APIAccessAllowed:   false,
			BillingAllowed:     false,
			DeploymentScope:    "self_hosted",
			SupportLevel:       "community",
			TeamSizeLimit:      1,
		}
	case "enterprise":
		return CommercialEntitlements{
			CommercialAllowed:  true,
			ResaleAllowed:      false,
			MultiTenantAllowed: true,
			WhiteLabelAllowed:  true,
			APIAccessAllowed:   true,
			BillingAllowed:     true,
			DeploymentScope:    "single_company",
			SupportLevel:       "enterprise",
			TeamSizeLimit:      20,
		}
	case "channel":
		return CommercialEntitlements{
			CommercialAllowed:  true,
			ResaleAllowed:      true,
			MultiTenantAllowed: true,
			WhiteLabelAllowed:  true,
			APIAccessAllowed:   true,
			BillingAllowed:     true,
			DeploymentScope:    "multi_customer",
			SupportLevel:       "partner",
			TeamSizeLimit:      50,
		}
	default:
		return CommercialEntitlements{
			CommercialAllowed:  true,
			ResaleAllowed:      false,
			MultiTenantAllowed: false,
			WhiteLabelAllowed:  false,
			APIAccessAllowed:   true,
			BillingAllowed:     true,
			DeploymentScope:    "single_company",
			SupportLevel:       "standard",
			TeamSizeLimit:      5,
		}
	}
}

// NewLicenseVerifier creates a new LicenseVerifier instance
func NewLicenseVerifier(serverURL, licenseKey, domain, accessDomain, accessProtocol string) *LicenseVerifier {
	return &LicenseVerifier{
		serverURL:      serverURL,
		licenseKey:     licenseKey,
		domain:         domain,
		accessDomain:   accessDomain,
		accessProtocol: accessProtocol,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Verify performs license verification
func (v *LicenseVerifier) Verify(ctx context.Context) (*VerifyResponse, error) {
	if v.serverURL == "" || v.licenseKey == "" {
		return &VerifyResponse{Valid: false, Reason: "未配置授权服务"}, nil
	}

	reqBody := VerifyRequest{
		LicenseKey:     v.licenseKey,
		Domain:         v.domain,
		AccessDomain:   v.accessDomain,
		AccessProtocol: v.accessProtocol,
		ServerIP:       GetServerIP(), // 自动附加服务器 IP
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.serverURL+"/api/verify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify request: %w", err)
	}
	defer resp.Body.Close()

	var result VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !VerifyResponseSignature(&result, ObscuredHMACKey()) {
		return nil, fmt.Errorf("invalid response signature")
	}

	return &result, nil
}

// GetServerDomain extracts domain from environment or hostname
func GetServerDomain() string {
	checkParams.mu.RLock()
	domainFromConfig := checkParams.domainFromConfig
	checkParams.mu.RUnlock()
	if domainFromConfig != "" {
		return domainFromConfig
	}

	domain := os.Getenv("SERVER_DOMAIN")
	if domain != "" {
		return domain
	}

	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}

// GetServerIP returns the public IP of the server.
// Priority: 1. SERVER_IP env var 2. HTTP request to external API.
func GetServerIP() string {
	if ip := os.Getenv("SERVER_IP"); ip != "" {
		return ip
	}

	client := &http.Client{Timeout: 3 * time.Second}
	urls := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://ip.sb/ip",
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					ip := strings.TrimSpace(string(body))
					if net.ParseIP(ip) != nil {
						return ip
					}
				}
			}
		}
	}

	return ""
}

var checkParams struct {
	serverURL        string
	licenseKey       string
	domain           string
	accessDomain     string
	accessProtocol   string
	domainFromConfig string
	mu               sync.RWMutex
}

// StartLicenseVerification starts license verification and stores the result
func StartLicenseVerification(serverURL, licenseKey, domain, accessDomain, accessProtocol string) error {
	if serverURL == "" || licenseKey == "" {
		globalLicenseState.mu.Lock()
		globalLicenseState.valid = false
		globalLicenseState.reason = "未配置授权服务"
		globalLicenseState.licenseProfile = "personal"
		globalLicenseState.entitlements = defaultEntitlementsForProfile("personal")
		globalLicenseState.LastCheck = time.Now()
		globalLicenseState.mu.Unlock()
		return nil
	}

	checkParams.mu.Lock()
	checkParams.serverURL = serverURL
	checkParams.licenseKey = licenseKey
	checkParams.domain = domain
	checkParams.accessDomain = accessDomain
	checkParams.accessProtocol = accessProtocol
	checkParams.mu.Unlock()

	if err := doVerify(); err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			lockedReason := getLockedReason()
			if lockedReason != "" {
				ticker.Reset(3 * time.Minute)
			} else {
				ticker.Reset(10 * time.Minute)
			}

			if err := doVerify(); err != nil {
			}
		}
	}()

	return nil
}

// TriggerAsyncCheck triggers a background verification immediately
func TriggerAsyncCheck() {
	go func() {
		doVerify()
	}()
}

// ForceSyncCheck performs synchronous verification and updates global state
func ForceSyncCheck() {
	checkParams.mu.Lock()
	serverURL := checkParams.serverURL
	licenseKey := checkParams.licenseKey
	domain := checkParams.domain
	accessDomain := checkParams.accessDomain
	accessProtocol := checkParams.accessProtocol
	checkParams.mu.Unlock()

	if serverURL == "" || licenseKey == "" {
		globalLicenseState.mu.Lock()
		globalLicenseState.valid = false
		globalLicenseState.reason = "未配置授权服务"
		globalLicenseState.licenseProfile = "personal"
		globalLicenseState.entitlements = defaultEntitlementsForProfile("personal")
		globalLicenseState.LastCheck = time.Now()
		globalLicenseState.mu.Unlock()
		return
	}

	verifier := NewLicenseVerifier(serverURL, licenseKey, domain, accessDomain, accessProtocol)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := verifier.Verify(ctx)

	globalLicenseState.mu.Lock()
	if err != nil {
		globalLicenseState.valid = false
		globalLicenseState.reason = fmt.Sprintf("同步验证失败: %v", err)
		globalLicenseState.licenseProfile = "personal"
		globalLicenseState.entitlements = defaultEntitlementsForProfile("personal")
	} else {
		globalLicenseState.valid = resp.Valid
		globalLicenseState.expireTime = resp.ExpireTime
		globalLicenseState.reason = resp.Reason
		globalLicenseState.isTrial = resp.IsTrial
		globalLicenseState.planName = resp.PlanName
		globalLicenseState.licenseProfile = normalizeLicenseProfile(resp.LicenseProfile)
		globalLicenseState.entitlements = resp.Entitlements
		if globalLicenseState.entitlements.DeploymentScope == "" {
			globalLicenseState.entitlements = defaultEntitlementsForProfile(globalLicenseState.licenseProfile)
		}
	}
	globalLicenseState.LastCheck = time.Now()
	globalLicenseState.mu.Unlock()
}

func getLockedReason() string {
	globalLicenseState.mu.RLock()
	defer globalLicenseState.mu.RUnlock()
	return globalLicenseState.reason
}

func doVerify() error {
	checkParams.mu.Lock()
	serverURL := checkParams.serverURL
	licenseKey := checkParams.licenseKey
	domain := checkParams.domain
	accessDomain := checkParams.accessDomain
	accessProtocol := checkParams.accessProtocol
	checkParams.mu.Unlock()

	if serverURL == "" || licenseKey == "" {
		globalLicenseState.mu.Lock()
		globalLicenseState.valid = false
		globalLicenseState.reason = "未配置授权服务"
		globalLicenseState.licenseProfile = "personal"
		globalLicenseState.entitlements = defaultEntitlementsForProfile("personal")
		globalLicenseState.LastCheck = time.Now()
		globalLicenseState.mu.Unlock()
		return nil
	}

	verifier := NewLicenseVerifier(serverURL, licenseKey, domain, accessDomain, accessProtocol)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := verifier.Verify(ctx)
	if err != nil {
		globalLicenseState.mu.Lock()
		globalLicenseState.valid = false
		globalLicenseState.reason = fmt.Sprintf("验证服务不可达：%v", err)
		globalLicenseState.licenseProfile = "personal"
		globalLicenseState.entitlements = defaultEntitlementsForProfile("personal")
		globalLicenseState.LastCheck = time.Now()
		globalLicenseState.mu.Unlock()
		return err
	}

	globalLicenseState.mu.Lock()
	globalLicenseState.valid = resp.Valid
	globalLicenseState.expireTime = resp.ExpireTime
	globalLicenseState.reason = resp.Reason
	globalLicenseState.isTrial = resp.IsTrial
	globalLicenseState.planName = resp.PlanName
	globalLicenseState.licenseProfile = normalizeLicenseProfile(resp.LicenseProfile)
	globalLicenseState.entitlements = resp.Entitlements
	if globalLicenseState.entitlements.DeploymentScope == "" {
		globalLicenseState.entitlements = defaultEntitlementsForProfile(globalLicenseState.licenseProfile)
	}
	globalLicenseState.LastCheck = time.Now()
	globalLicenseState.mu.Unlock()

	return nil
}

// TierType 定义授权等级
type TierType string

const (
	TierFree    TierType = "free"
	TierPremium TierType = "premium"
	TierBlocked TierType = "blocked"
)

// GetLicenseTier 获取当前授权等级
func GetLicenseTier() (TierType, string) {
	globalLicenseState.mu.RLock()
	defer globalLicenseState.mu.RUnlock()

	checkParams.mu.RLock()
	hasKey := checkParams.licenseKey != ""
	checkParams.mu.RUnlock()

	if !hasKey {
		return TierFree, "未配置授权服务"
	}

	if !globalLicenseState.valid {
		switch globalLicenseState.reason {
		case "域名不匹配", "授权已过期", "授权已被禁用":
			return TierBlocked, globalLicenseState.reason
		default:
			return TierFree, "验证服务不可达，商业授权暂不可用"
		}
	}

	return TierPremium, ""
}

// CheckResourceLimit keeps backward compatibility for handlers that still call it.
// Infrastructure capacity is no longer constrained by the commercial license tier.
func CheckResourceLimit(resourceType string, currentCount int) error {
	tier, reason := GetLicenseTier()
	if tier == TierPremium || tier == TierFree {
		return nil
	}
	if tier == TierBlocked {
		return fmt.Errorf("license unavailable: %s", reason)
	}
	return nil
}

func GetCommercialEntitlements() CommercialEntitlements {
	globalLicenseState.mu.RLock()
	defer globalLicenseState.mu.RUnlock()
	entitlements := globalLicenseState.entitlements
	if entitlements.DeploymentScope == "" {
		entitlements = defaultEntitlementsForProfile(globalLicenseState.licenseProfile)
	}
	return entitlements
}

func GetCommercialProfile() string {
	globalLicenseState.mu.RLock()
	defer globalLicenseState.mu.RUnlock()
	return normalizeLicenseProfile(globalLicenseState.licenseProfile)
}

func CheckCommercialFeature(feature string) error {
	tier, reason := GetLicenseTier()
	if tier == TierBlocked {
		return fmt.Errorf("license unavailable: %s", reason)
	}
	entitlements := GetCommercialEntitlements()
	switch strings.ToLower(strings.TrimSpace(feature)) {
	case "billing", "commerce", "store":
		if !entitlements.BillingAllowed {
			return fmt.Errorf("commercial billing, store, or payment features are not enabled for this license")
		}
	case "api":
		if !entitlements.APIAccessAllowed {
			return fmt.Errorf("commercial API access is not enabled for this license")
		}
	case "white_label":
		if !entitlements.WhiteLabelAllowed {
			return fmt.Errorf("white-label capability is not enabled for this license")
		}
	case "multi_tenant":
		if !entitlements.MultiTenantAllowed {
			return fmt.Errorf("multi-tenant capability is not enabled for this license")
		}
	case "resale":
		if !entitlements.ResaleAllowed {
			return fmt.Errorf("resale or distribution capability is not enabled for this license")
		}
	}
	return nil
}

func GetLicenseState() (valid bool, expireTime int64, reason string, isTrial bool) {
	globalLicenseState.mu.RLock()
	defer globalLicenseState.mu.RUnlock()
	return globalLicenseState.valid, globalLicenseState.expireTime, globalLicenseState.reason, globalLicenseState.isTrial
}

// UpdateServerDomainFromConfig sets the domain recovered from DB config.
func UpdateServerDomainFromConfig(domain string) {
	checkParams.mu.Lock()
	checkParams.domainFromConfig = domain
	checkParams.mu.Unlock()
}

// UpdateCheckParams updates the stored check parameters for license verification
func UpdateCheckParams(serverURL, licenseKey, domain, accessDomain, accessProtocol string) {
	checkParams.mu.Lock()
	defer checkParams.mu.Unlock()
	checkParams.serverURL = serverURL
	checkParams.licenseKey = licenseKey
	checkParams.domain = domain
	checkParams.accessDomain = accessDomain
	checkParams.accessProtocol = accessProtocol
}
