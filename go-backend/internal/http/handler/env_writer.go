package handler

import (
	"bufio"
	"os"
	"strings"
)

func UpdateEnvFile(licenseKey, domain, hmacKey string) error {
	envPath := "/opt/flvx-svc/.env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}

	f, err := os.Open(envPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// 更新或追加的标记
	foundLicenseKey := false
	foundDomain := false
	foundHmacKey := false

	for i, line := range lines {
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			switch k {
			case "LICENSE_KEY":
				lines[i] = "LICENSE_KEY=" + licenseKey
				foundLicenseKey = true
			case "SERVER_DOMAIN":
				lines[i] = "SERVER_DOMAIN=" + domain
				foundDomain = true
			case "LICENSE_SERVER_URL":
				lines[i] = ""
			case "HMAC_SECRET_KEY":
				if hmacKey != "" {
					lines[i] = "HMAC_SECRET_KEY=" + hmacKey
					foundHmacKey = true
				}
			}
		}
	}

	// 追加不存在的
	if !foundLicenseKey && licenseKey != "" {
		lines = append(lines, "LICENSE_KEY="+licenseKey)
	}
	if !foundDomain && domain != "" {
		lines = append(lines, "SERVER_DOMAIN="+domain)
	}
	if !foundHmacKey && hmacKey != "" {
		lines = append(lines, "HMAC_SECRET_KEY="+hmacKey)
	}

	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	content := strings.Join(out, "\n") + "\n"
	return osWrite(envPath, []byte(content), 0644)
}

func osWrite(name string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
