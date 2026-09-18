package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Model struct {
	ID          string         `yaml:"id"`
	Name        string         `yaml:"name"`
	APIKey      string         `yaml:"api_key"`
	BaseURL     string         `yaml:"base_url"`
	Temperature *float32       `yaml:"temperature"`
	ExtraBody   map[string]any `yaml:"extra_body"`
	// Thinking 控制是否在请求体里注入 {"thinking": {"type": "enabled"}}。
	// Python 版因为流程简单，两个角色都固定关闭思考。Go 版按角色给默认值：
	// 行为分析（pro）开启、命中验证（lite）关闭——思考会显著抬高 token 与
	// 延迟，而 agent 循环最多允许 50 次模型调用。显式配置可覆盖默认。
	Thinking *bool `yaml:"thinking"`
	// ResponseFormat 选择结构化输出的实现方式，取值为 "tool" 或 "json_schema"。
	// 默认 "tool"（forced tool call，与 Python 的 LangChain ToolStrategy 等价）；
	// 网关支持 json_schema 时可设为 "json_schema"，仅影响单文件快速路径。
	ResponseFormat string `yaml:"response_format"`
}

type LangSmith struct {
	Tracing  *bool  `yaml:"tracing"`
	Endpoint string `yaml:"endpoint"`
	APIKey   string `yaml:"api_key"`
	Project  string `yaml:"project"`
}

type Config struct {
	Models    []Model           `yaml:"models"`
	Roles     map[string]string `yaml:"roles"`
	LangSmith *LangSmith        `yaml:"langsmith"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Read config file error: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("Parse config file error: %w", err)
	}
	if len(c.Roles) == 0 {
		return nil, fmt.Errorf("No role mapping found in config: %s", path)
	}
	return &c, nil
}

func (c *Config) ModelForRole(role string) (*Model, error) {
	id, ok := c.Roles[role]
	if !ok || id == "" {
		return nil, fmt.Errorf("No role mapping found for: %s", role)
	}
	for i := range c.Models {
		if c.Models[i].ID != id {
			continue
		}
		m := c.Models[i]
		if m.APIKey == "" {
			return nil, fmt.Errorf("Model %q (role: %s) has empty api_key", id, role)
		}
		if m.BaseURL == "" {
			return nil, fmt.Errorf("Model %q (role: %s) has empty base_url", id, role)
		}
		return &m, nil
	}
	return nil, fmt.Errorf("No model info found for: %s (role: %s)", id, role)
}

// WithThinking 返回一份副本，按角色默认值决定是否开启思考。
func (m *Model) WithThinking(role string) *Model {
	enabled := role == "pro"
	if m.Thinking != nil {
		enabled = *m.Thinking
	}
	cp := *m
	cp.Thinking = &enabled
	return &cp
}
