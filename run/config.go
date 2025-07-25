package run

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	var_template "github.com/xhd2015/go-var-template"
	"github.com/xhd2015/kode-ai/internal/ioread"
)

//go:embed example-config.json
var ExampleConfig string

//go:embed config_def.go
var ConfigDef string

type FullConfig struct {
	Config
	RecordFile         string `json:"record_file,omitempty"`
	NoCache            bool   `json:"no_cache,omitempty"`
	ShowUsage          bool   `json:"show_usage,omitempty"`
	IgnoreDuplicateMsg bool   `json:"ignore_duplicate_msg,omitempty"`
	LogRequest         bool   `json:"log_request,omitempty"`
	LogChat            *bool  `json:"log_chat,omitempty"`
	Verbose            bool   `json:"verbose,omitempty"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(configFile string) (*FullConfig, error) {
	if configFile == "" {
		return &FullConfig{}, nil
	}

	// Handle relative paths
	if !filepath.IsAbs(configFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get current directory: %v", err)
		}
		configFile = filepath.Join(cwd, configFile)
	}

	// Read file content
	content, err := ioread.ReadOrContent(configFile)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %v", configFile, err)
	}

	var config FullConfig
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("parse config file %s: %v", configFile, err)
	}

	return &config, nil
}

// ApplyConfig applies configuration values to the provided variables, giving precedence to command line arguments
func ApplyConfig(config *FullConfig, token *string, maxRound *int, baseUrl *string, model *string, systemPrompt *string, tools *[]string, toolCustomFiles *[]string, toolCustomJSONs *[]string, toolDefaultCwd *string, recordFile *string, noCache *bool, showUsage *bool, ignoreDuplicateMsg *bool, logRequest *bool, logChatFlag **bool, verbose *bool, mcpServers *[]string) error {
	if config == nil {
		return nil
	}

	// Apply config values only if command line arguments are not set
	if *token == "" && config.Token != "" {
		*token = config.Token
	}
	if *maxRound == 0 && config.MaxRound != 0 {
		*maxRound = config.MaxRound
	}
	if *baseUrl == "" && config.BaseURL != "" {
		*baseUrl = config.BaseURL
	}
	if *model == "" && config.Model != "" {
		*model = config.Model
	}
	if *systemPrompt == "" {
		configSystempPrompt, err := getStrOrStrLines(config.SystemPrompt)
		if err != nil {
			return fmt.Errorf("config system: %w", err)
		}
		tpl := var_template.Compile(configSystempPrompt)
		systemPromtInterploted, err := tpl.Execute(map[string]string{
			"model": *model,
		})
		if err != nil {
			return fmt.Errorf("interpolate system prompt: %w", err)
		}
		// && config.SystemPrompt != "" {
		*systemPrompt = systemPromtInterploted
	}

	// Convert json.RawMessage to strings
	configToolJSONStrings := make([]string, len(config.ToolCustomJSONs))
	for i, rawMsg := range config.ToolCustomJSONs {
		jsonBytes, err := json.Marshal(rawMsg)
		if err != nil {
			return fmt.Errorf("config tool custom json: %w\n%v", err, rawMsg)
		}
		configToolJSONStrings[i] = string(jsonBytes)
	}
	*tools = append(*tools, config.Tools...)
	*toolCustomFiles = append(*toolCustomFiles, config.ToolCustomFiles...)
	*toolCustomJSONs = append(*toolCustomJSONs, configToolJSONStrings...)
	if *toolDefaultCwd == "" && config.ToolDefaultCwd != "" {
		*toolDefaultCwd = config.ToolDefaultCwd
	}
	if *recordFile == "" && config.RecordFile != "" {
		*recordFile = config.RecordFile
	}
	if !*noCache && config.NoCache {
		*noCache = config.NoCache
	}
	if !*showUsage && config.ShowUsage {
		*showUsage = config.ShowUsage
	}
	if !*ignoreDuplicateMsg && config.IgnoreDuplicateMsg {
		*ignoreDuplicateMsg = config.IgnoreDuplicateMsg
	}
	if !*logRequest && config.LogRequest {
		*logRequest = config.LogRequest
	}
	if *logChatFlag == nil && config.LogChat != nil {
		*logChatFlag = config.LogChat
	}
	if !*verbose && config.Verbose {
		*verbose = config.Verbose
	}
	if len(*mcpServers) == 0 && len(config.MCPServers) > 0 {
		*mcpServers = config.MCPServers
	}

	return nil
}

func getStrOrStrLines(v interface{}) (string, error) {
	if v == nil {
		return "", nil
	}
	switch v := v.(type) {
	case string:
		return v, nil
	case []interface{}:
		lines := make([]string, len(v))
		for i, line := range v {
			e, ok := line.(string)
			if !ok {
				return "", fmt.Errorf("must be a string or a list of strings, found %T at %d", line, i)
			}
			lines[i] = e
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", nil
}
