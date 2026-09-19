package plugin

import (
	"fmt"
	"regexp"
	"strings"
)

var ipPattern = regexp.MustCompile(`\b(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

// IsPublicIP 只支持 IPv4 点分四段。刻意不过滤 100.64/10、198.18/15、
// 203.0.113/24 这类特殊用途网段：判定口径改动会连带改变插件命中集合。
func IsPublicIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	octets := make([]int, 4)
	for i, p := range parts {
		n := 0
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
			n = n*10 + int(c-'0')
		}
		if n > 255 {
			return false
		}
		octets[i] = n
	}
	a, b := octets[0], octets[1]
	switch {
	case a == 10:
		return false
	case a == 172 && b >= 16 && b <= 31:
		return false
	case a == 192 && b == 168:
		return false
	case a == 127:
		return false
	case a == 0:
		return false
	case a == 169 && b == 254:
		return false
	case a >= 224 && a <= 239:
		return false
	case a >= 240:
		return false
	}
	return true
}

type PublicIPCheckPlugin struct{}

func (PublicIPCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "public_ip",
		Name:        "Public IP Address",
		Description: "Detects hard-coded public IPv4 addresses, commonly used as C2 endpoints or exfiltration targets.",
	}
}

func (PublicIPCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	matches := ipPattern.FindAllString(content, -1)
	seen := map[string]bool{}
	var publics []string
	for _, ip := range matches {
		if !IsPublicIP(ip) {
			continue
		}
		if seen[ip] {
			continue
		}
		seen[ip] = true
		publics = append(publics, ip)
	}
	if len(publics) == 0 {
		return nil
	}
	// 固定为首次出现顺序：集合的哈希序每次运行都不同，会让这段描述文本漂移。
	return []Issue{{
		Name:     "Discover public IP address",
		Severity: SeverityMedium,
		Category: CatNetworkAbuse,
		Description: fmt.Sprintf(
			"Public IP addresses are often used as C2 addresses or as recipients of data breaches. %d IP addresses were found: %s",
			len(publics), pySet(publics)),
		FilePath:      relativePath(filePath, skillDir),
		AuditRequired: true,
	}}
}

// pySet 渲染成集合字面量，保持提示词里看到的文本形态。
func pySet(items []string) string {
	var sb strings.Builder
	sb.WriteString("{")
	for i, s := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("'")
		sb.WriteString(s)
		sb.WriteString("'")
	}
	sb.WriteString("}")
	return sb.String()
}
