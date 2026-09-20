package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"vetix/internal/buildinfo"
)

// Fingerprint 是本次扫描引擎的指纹：buildinfo.Version 加上全部已注册插件的 ID 集合
// 一起做 sha256，取前 16 位 hex。报告把它存进 metadata.scan_key，读缓存时指纹不匹配
// 就判定缓存失效——升级 Vetix、增删插件之后，旧报告不会再被当成新结果直接渲染。
//
// 注意：这只覆盖"插件集合变了"和"版本号变了"两类失效。插件 ID 不变但检测逻辑改动
// 的情况靠版本号发布兜底；开发期规则频繁改动时用 -force 绕过缓存。
func Fingerprint() string {
	ids := make([]string, 0, len(registry))
	for _, p := range List() {
		ids = append(ids, p.ID)
	}
	// 注册表本身按 ID 排序，但排序在这里再保证一次：指纹只取决于 ID 集合本身，
	// 与注册顺序解耦。
	sort.Strings(ids)
	h := sha256.Sum256([]byte(buildinfo.Version + "\n" + strings.Join(ids, "\n")))
	return hex.EncodeToString(h[:])[:16]
}
