package run

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	anth_opt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	openai_opt "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/internal/ioread"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/internal/terminal"
	"github.com/xhd2015/kode-ai/providers"
	anthropic_helper "github.com/xhd2015/kode-ai/providers/anthropic"
	"github.com/xhd2015/kode-ai/tools"
	"google.golang.org/genai"
)

type ChatOptions struct {
	maxRound int

	systemPrompt string
	toolBuiltins []string
	toolFiles    []string
	toolJSONs    []string
	recordFile   string

	toolDefaultCwd string

	ignoreDuplicateMsg bool
	noCache            bool

	logRequest bool
	verbose    bool
	logChat    bool

	// MCP server configuration
	mcpServers []string
}

type ChatHandler struct {
	APIShape providers.APIShape

	// openAIKeepMultipleSystemPrompts_DO_NOT_SET only for testing purpose
	// please do not set it.
	//
	// to test open ai behavior when multiple system prompts exists, which will win?
	// the result shows the last one wins:
	//  > system: Your name is Lubby, your age is 18
	//  > user: who are you?
	//  > assistant: Hi! I'm Lubby, an 18-year-old here to help you out. What can I do for you today?
	//  > system: Your name is Chuk, your age is 12
	//  > user: how old are you?
	//  > assistant: I'm 12 years old! How about you?
	openAIKeepMultipleSystemPrompts_DO_NOT_SET bool
}
type ClientUnion struct {
	OpenAI    *openai.Client
	Anthropic *anthropic.Client
	Gemini    *genai.Client
}

type MessageHistoryUnion struct {
	FullHistory   Messages
	SystemPrompts []string

	OpenAI    []openai.ChatCompletionMessageParamUnion
	Anthropic []anthropic.MessageParam
	Gemini    []*genai.Content
}

func (c *ChatHandler) Handle(model string, baseUrl string, token string, msg string, opts ChatOptions) error {
	toolBuiltins := opts.toolBuiltins
	recordFile := opts.recordFile
	ignoreDuplicateMsg := opts.ignoreDuplicateMsg
	maxRound := opts.maxRound
	absDefaultToolCwd := opts.toolDefaultCwd
	openAIKeepMultipleSystemPrompts := c.openAIKeepMultipleSystemPrompts_DO_NOT_SET

	noCache := opts.noCache

	logRequest := opts.logRequest
	verbose := opts.verbose
	logChat := opts.logChat

	_ = verbose

	// Load historical messages from record file if it exists
	historyMsgs, err := c.readHistoryMessages(recordFile, openAIKeepMultipleSystemPrompts)
	if err != nil {
		return err
	}

	historicalMessagesOpenAI := historyMsgs.OpenAI
	historicalMessagesAnthropic := historyMsgs.Anthropic
	historicalMessagesGemini := historyMsgs.Gemini
	historicalSystemPrompts := historyMsgs.SystemPrompts

	msg, stop, err := c.checkUserMsgDuplicate(msg, historyMsgs.FullHistory, ignoreDuplicateMsg)
	if err != nil {
		return err
	}
	if stop {
		return nil
	}
	ctx := context.TODO()
	clients, err := c.createClient(ctx, baseUrl, token, logRequest)
	if err != nil {
		return err
	}
	clientOpenAI := clients.OpenAI
	clientAnthropic := clients.Anthropic
	clientGemini := clients.Gemini

	toolInfoMapping := make(ToolInfoMapping)
	toolSchemas, err := tools.ParseSchemas(opts.toolFiles, opts.toolJSONs)
	if err != nil {
		return err
	}
	for _, tool := range toolSchemas {
		if err := toolInfoMapping.AddTool(tool.Name, &ToolInfo{
			Name:           tool.Name,
			ToolDefinition: tool,
		}); err != nil {
			return err
		}
	}

	builtinTools, err := tools.GetBuiltinTools(toolBuiltins)
	if err != nil {
		return err
	}
	for _, tool := range builtinTools {
		if err := toolInfoMapping.AddTool(tool.Name, &ToolInfo{
			Name:           tool.Name,
			Builtin:        true,
			ToolDefinition: tool,
		}); err != nil {
			return err
		}
	}
	toolSchemas = append(toolSchemas, builtinTools...)

	// Setup MCP client if configured
	for _, mcpServer := range opts.mcpServers {
		mcpClient, err := connectToMCPServer(mcpServer)
		if err != nil {
			return fmt.Errorf("connect to MCP server: %v", err)
		}
		res, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{})
		if err != nil {
			return fmt.Errorf("initialize MCP client: %v", err)
		}
		_ = res

		// Get MCP tools and add them to the tool schemas
		mcpTools, err := getMCPTools(ctx, mcpClient)
		if err != nil {
			return fmt.Errorf("list mcp tools: %s %w", opts.mcpServers, err)
		}
		// validate
		for _, tool := range mcpTools {
			if err := toolInfoMapping.AddTool(tool.Name, &ToolInfo{
				Name:           tool.Name,
				MCPServer:      mcpServer,
				MCPClient:      mcpClient,
				ToolDefinition: tool,
			}); err != nil {
				return err
			}
		}
		toolSchemas = append(toolSchemas, mcpTools...)
	}

	var toolsOpenAI []openai.ChatCompletionToolParam
	var toolsAnthropic []anthropic.ToolUnionParam
	var toolsGemini []*genai.Tool
	switch c.APIShape {
	case providers.APIShapeOpenAI:
		toolsOpenAI, err = toolSchemas.ToOpenAI()
		if err != nil {
			return err
		}
	case providers.APIShapeAnthropic:
		toolsAnthropic, err = toolSchemas.ToAnthropic()
		if err != nil {
			return err
		}
	case providers.APIShapeGemini:
		toolsGemini, err = toolSchemas.ToGemini()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported provider: %s", c.APIShape)
	}

	var systemMessageOpenAI *openai.ChatCompletionMessageParamUnion
	var systemAnthropic []anthropic.TextBlockParam
	var systemMessageGemini *genai.Content

	var userSystemPrompt string
	if opts.systemPrompt != "" {
		content, err := ioread.ReadOrContent(opts.systemPrompt)
		if err != nil {
			return err
		}
		userSystemPrompt = content

		switch c.APIShape {
		case providers.APIShapeOpenAI:
			systemMessageOpenAI = &openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.NewOpt(content),
					},
				},
			}
		case providers.APIShapeAnthropic:
			systemMsg := anthropic.TextBlockParam{
				Text: content,
			}
			systemAnthropic = append(systemAnthropic, systemMsg)
		case providers.APIShapeGemini:
			systemMessageGemini = &genai.Content{
				Parts: []*genai.Part{
					{
						Text: content,
					},
				},
			}
		default:
			return fmt.Errorf("unsupported provider: %s", c.APIShape)
		}
	} else if len(historicalSystemPrompts) > 0 {
		switch c.APIShape {
		case providers.APIShapeOpenAI:
			// do nothing
			if !openAIKeepMultipleSystemPrompts {
				lastSystemPrompt := historicalSystemPrompts[len(historicalSystemPrompts)-1]
				systemMessageOpenAI = &openai.ChatCompletionMessageParamUnion{
					OfSystem: &openai.ChatCompletionSystemMessageParam{
						Content: openai.ChatCompletionSystemMessageParamContentUnion{
							OfString: param.NewOpt(lastSystemPrompt),
						},
					},
				}
			}
		case providers.APIShapeAnthropic:
			// If user doesn't provide a system prompt, use the last system prompt in historical messages
			lastSystemPrompt := historicalSystemPrompts[len(historicalSystemPrompts)-1]

			systemMsg := anthropic.TextBlockParam{
				Text: lastSystemPrompt,
			}
			systemAnthropic = append(systemAnthropic, systemMsg)
		case providers.APIShapeGemini:
			lastSystemPrompt := historicalSystemPrompts[len(historicalSystemPrompts)-1]
			systemMessageGemini = &genai.Content{
				Parts: []*genai.Part{
					{
						Text: lastSystemPrompt,
					},
				},
			}
		default:
			return fmt.Errorf("system prompt: unsupported provider: %s", c.APIShape)
		}
	}

	// build messages
	msgsUnion, err := c.buildMessages(msg, openAIKeepMultipleSystemPrompts, systemMessageOpenAI, historicalMessagesOpenAI, historicalMessagesAnthropic, historicalMessagesGemini)
	if err != nil {
		return err
	}
	messagesOpenAI := msgsUnion.OpenAI
	messagesAnthropic := msgsUnion.Anthropic
	messagesGemini := msgsUnion.Gemini

	LOOP := maxRound
	if maxRound <= 0 {
		LOOP = 1
	}

	hasMaxRound := maxRound > 1

	var totalTokenUsage TokenUsage
	var totalCost TokenCost

	var needCache bool
	if !noCache {
		needCache = true
		if recordFile == "" && !hasMaxRound {
			needCache = false
		}
	}

	if logChat {
		cacheLog := "enabled"
		if !needCache {
			cacheLog = "disabled"
		}
		fmt.Printf("Prompt cache %s with %s\n", cacheLog, model)
	}

	if needCache && c.APIShape == providers.APIShapeAnthropic {
		systemAnthropic = anthropic_helper.MarkTextBlocksEphemeralCache(systemAnthropic)
		toolsAnthropic = anthropic_helper.MarkToolsEphemeralCache(toolsAnthropic)
	}

	var toolUseNum int
	for _, msg := range historyMsgs.FullHistory {
		if msg.Type == MsgType_ToolCall {
			toolUseNum++
		}
	}

	i := 0
	for ; i < LOOP; i++ {
		var resultOpenAI *openai.ChatCompletion
		var resultAnthropic *anthropic.Message
		var resultGemini *genai.GenerateContentResponse

		startTime := time.Now()
		if logChat {
			// log making request
			fmt.Println("Request...")
		}

		switch c.APIShape {
		case providers.APIShapeOpenAI:
			resultOpenAI, err = clientOpenAI.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Model:    model,
				Messages: messagesOpenAI,
				Tools:    toolsOpenAI,
				N:        param.NewOpt(int64(1)),
			})
			if err != nil {
				return err
			}
		case providers.APIShapeAnthropic:
			sendMessage := messagesAnthropic
			if needCache {
				sendMessage = anthropic_helper.MarkMsgsEphemeralCache(messagesAnthropic)
			}
			resultAnthropic, err = anthropic_helper.Chat(ctx, clientAnthropic, anthropic.MessageNewParams{
				MaxTokens: 2048,
				Model:     anthropic.Model(model),
				Messages:  sendMessage,
				System:    systemAnthropic,
				Tools:     toolsAnthropic,
			})
			if err != nil {
				return err
			}
		case providers.APIShapeGemini:
			resultGemini, err = clientGemini.Models.GenerateContent(ctx, model, messagesGemini, &genai.GenerateContentConfig{
				HTTPOptions: &genai.HTTPOptions{
					APIVersion: "v1",
					Headers: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %s", token)},
					},
				},
				SystemInstruction: systemMessageGemini,
				Tools:             toolsGemini,
				CandidateCount:    1,
			})
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("call api: unsupported provider: %s", c.APIShape)
		}

		if logChat {
			// log making request
			fmt.Printf("Response: %fs\n", (float64(int(time.Since(startTime).Seconds()*1000)) / 1000.0))
		}

		if i == 0 && recordFile != "" {
			// Record the user message if recordFile is specified
			if err := c.recordInitialMsg(msg, userSystemPrompt, historicalSystemPrompts, model, recordFile); err != nil {
				return err
			}
		}
		var tokenUsage TokenUsage

		var newToolUseNum int
		switch c.APIShape {
		case providers.APIShapeOpenAI:
			res, err := c.processOpenAIResponse(ctx, resultOpenAI, hasMaxRound, model, recordFile, absDefaultToolCwd, toolInfoMapping)
			if err != nil {
				return err
			}
			tokenUsage = res.TokenUsage
			respMessages := res.Messages
			toolResults := res.ToolResults

			messagesOpenAI = append(messagesOpenAI, respMessages...)
			messagesOpenAI = append(messagesOpenAI, toolResults...)

			newToolUseNum = res.ToolUseNum
		case providers.APIShapeAnthropic:
			res, err := c.processAnthropicResponse(ctx, resultAnthropic, hasMaxRound, model, recordFile, absDefaultToolCwd, toolInfoMapping)
			if err != nil {
				return err
			}
			tokenUsage = res.TokenUsage
			respMessages := res.Messages
			toolResults := res.ToolResults
			if len(respMessages) > 0 {
				messagesAnthropic = append(messagesAnthropic, anthropic.MessageParam{
					Role:    anthropic.MessageParamRole(resultAnthropic.Role),
					Content: respMessages,
				})
			}
			if len(toolResults) > 0 {
				messagesAnthropic = append(messagesAnthropic, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleUser,
					Content: toolResults,
				})
			}
			newToolUseNum = res.ToolUseNum
		case providers.APIShapeGemini:
			res, err := c.processGeminiResponse(ctx, resultGemini, toolUseNum, hasMaxRound, model, recordFile, absDefaultToolCwd, toolInfoMapping)
			if err != nil {
				return err
			}
			tokenUsage = res.TokenUsage
			respMessages := res.Messages
			toolResults := res.ToolResults
			if len(respMessages) > 0 {
				messagesGemini = append(messagesGemini, respMessages...)
			}
			if len(toolResults) > 0 {
				messagesGemini = append(messagesGemini, toolResults...)
			}
			newToolUseNum = res.ToolUseNum
		default:
			return fmt.Errorf("unsupported provider: %s", c.APIShape)
		}

		totalTokenUsage = totalTokenUsage.Add(tokenUsage)

		if recordFile != "" {
			tokenUsageMsg := Message{
				Type:       MsgType_TokenUsage,
				Role:       Role_Assistant,
				Model:      model,
				TokenUsage: &tokenUsage,
			}
			if err := appendToRecordFile(recordFile, &tokenUsageMsg); err != nil {
				return fmt.Errorf("failed to record token usage message: %v", err)
			}
		}

		stopped, stopMsg, err := c.checkRecordStopReason(recordFile, model, resultOpenAI, resultAnthropic, resultGemini)
		if err != nil {
			return err
		}
		if recordFile != "" && stopMsg != nil {
			if err := appendToRecordFile(recordFile, stopMsg); err != nil {
				return fmt.Errorf("record stop reason message: %v", err)
			}
		}
		cost, costOK := computeCost(c.APIShape, model, tokenUsage)
		var costUSD string
		if costOK {
			costUSD = "$" + cost.TotalUSD
			totalCost = totalCost.Add(cost)
		}
		if logChat {
			printTokenUsage(os.Stderr, "Token Usage", tokenUsage, costUSD)
		}
		if stopped {
			break
		}
		toolUseNum += newToolUseNum
		if newToolUseNum == 0 {
			break
		}
	}
	if logChat && i > 1 {
		var totalCostUSD string
		if totalCost.TotalUSD != "" {
			totalCostUSD = "$" + totalCost.TotalUSD
		}

		fmt.Fprintf(os.Stderr, "----\n")
		printTokenUsage(os.Stderr, "Total Usage", totalTokenUsage, totalCostUSD)
	}

	if hasMaxRound && i >= LOOP {
		fmt.Fprintf(os.Stderr, "max round: %d\n", i)
	}

	return nil
}

func (c *ChatHandler) readHistoryMessages(recordFile string, openAIKeepMultipleSystemPrompts bool) (*MessageHistoryUnion, error) {
	var msgHistory Messages

	var historicalSystemPrompts []string

	var historicalMessagesOpenAI []openai.ChatCompletionMessageParamUnion
	var historicalMessagesAnthropic []anthropic.MessageParam
	var historicalMessagesGemini []*genai.Content
	if recordFile != "" {

		var err error
		msgHistory, err = loadHistoricalMessages(recordFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load historical messages: %v", err)
		}

		switch c.APIShape {
		case providers.APIShapeOpenAI:
			historicalMessagesOpenAI, historicalSystemPrompts, err = msgHistory.ToOpenAI(openAIKeepMultipleSystemPrompts)
			if err != nil {
				return nil, fmt.Errorf("convert anthropic messages: %w", err)
			}
		case providers.APIShapeAnthropic:
			historicalMessagesAnthropic, historicalSystemPrompts, err = msgHistory.ToAnthropic()
			if err != nil {
				return nil, fmt.Errorf("convert anthropic messages: %w", err)
			}
		case providers.APIShapeGemini:
			historicalMessagesGemini, historicalSystemPrompts, err = msgHistory.ToGemini()
			if err != nil {
				return nil, fmt.Errorf("convert gemini messages: %w", err)
			}
		default:
			return nil, fmt.Errorf("read recording: unsupported provider: %s", c.APIShape)
		}

	}
	return &MessageHistoryUnion{
		FullHistory:   msgHistory,
		SystemPrompts: historicalSystemPrompts,

		OpenAI:    historicalMessagesOpenAI,
		Anthropic: historicalMessagesAnthropic,
		Gemini:    historicalMessagesGemini,
	}, nil
}

func (c *ChatHandler) checkUserMsgDuplicate(msg string, history Messages, ignoreDuplicateMsg bool) (string, bool, error) {
	var lastUserMsg *Message
	n := len(history)
	for i := n - 1; i >= 0; i-- {
		historyMsg := history[i]
		if historyMsg.Type == MsgType_Msg && historyMsg.Role == Role_User {
			lastUserMsg = &historyMsg
			break
		}
	}

	if msg != "" && lastUserMsg != nil && msg == lastUserMsg.Content {
		if ignoreDuplicateMsg {
			msg = ""
		} else {
			if !terminal.IsStdinTTY() {
				return "", true, fmt.Errorf("duplicate user msg, either clear the msg or run with --ignore-duplicate-msg")
			}
			// prompt user: duplicate msg, continune?
			prompt := fmt.Sprintf("Duplicate input with last msg created at %s, proceed?\n  c:proceed with duplicate, x:proceed without duplicate, q:quit", lastUserMsg.Time)
			for {
				reader := bufio.NewReader(os.Stdin)
				fmt.Println(prompt)
				fmt.Print("user> ")
				response, err := reader.ReadString('\n')
				if err != nil {
					return "", false, fmt.Errorf("failed to read response: %v", err)
				}
				decision := strings.TrimSuffix(response, "\n")
				if decision == "c" {
					break
				}
				if decision == "q" {
					return "", true, nil
				}
				if decision == "x" {
					msg = ""
					break
				}
			}
		}
	}
	return msg, false, nil
}

func (c *ChatHandler) createClient(ctx context.Context, baseURL string, token string, logRequest bool) (*ClientUnion, error) {
	var clientOpenAI *openai.Client
	var clientAnthropic *anthropic.Client
	var clientGemini *genai.Client
	switch c.APIShape {
	case providers.APIShapeOpenAI:
		var clientOptions []openai_opt.RequestOption
		if baseURL != "" {
			clientOptions = append(clientOptions, openai_opt.WithBaseURL(baseURL))
		}
		clientOptions = append(clientOptions, openai_opt.WithAPIKey(token))
		if logRequest {
			logger := log.New(os.Stderr, "", log.LstdFlags)
			clientOptions = append(clientOptions, openai_opt.WithDebugLog(logger))
		}
		client := openai.NewClient(
			clientOptions...,
		)
		clientOpenAI = &client
	case providers.APIShapeAnthropic:
		var clientOpts []anth_opt.RequestOption
		if baseURL != "" {
			clientOpts = append(clientOpts, anth_opt.WithBaseURL(baseURL))
		}
		clientOpts = append(clientOpts, anth_opt.WithAPIKey(token))
		if logRequest {
			logger := log.New(os.Stderr, "", log.LstdFlags)
			clientOpts = append(clientOpts, anth_opt.WithDebugLog(logger))
		}
		clientAnthropic = anthropic_helper.NewClient(
			clientOpts...,
		)
	case providers.APIShapeGemini:
		var err error
		// see https://ai.google.dev/gemini-api/docs#rest
		clientGemini, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  token,
			Backend: genai.BackendGeminiAPI,
			HTTPOptions: genai.HTTPOptions{
				BaseURL: baseURL,
			},
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.APIShape)
	}
	return &ClientUnion{
		OpenAI:    clientOpenAI,
		Anthropic: clientAnthropic,
		Gemini:    clientGemini,
	}, nil
}

const maxLimit = 256

func limitPrintLength(s string) string {
	if len(s) < maxLimit+3 {
		return s
	}
	return s[:maxLimit] + "..."
}

type MessagesUnion struct {
	OpenAI    []openai.ChatCompletionMessageParamUnion
	Anthropic []anthropic.MessageParam
	Gemini    []*genai.Content
}

// Add historical messages first
func (c *ChatHandler) buildMessages(msg string, openAIKeepMultipleSystemPrompts bool, systemMessageOpenAI *openai.ChatCompletionMessageParamUnion, historicalMessagesOpenAI []openai.ChatCompletionMessageParamUnion, historicalMessagesAnthropic []anthropic.MessageParam, historicalMessagesGemini []*genai.Content) (*MessagesUnion, error) {
	var messagesOpenAI []openai.ChatCompletionMessageParamUnion
	var messagesAnthropic []anthropic.MessageParam
	var messagesGemini []*genai.Content
	switch c.APIShape {
	case providers.APIShapeOpenAI:
		if openAIKeepMultipleSystemPrompts {
			messagesOpenAI = append(messagesOpenAI, historicalMessagesOpenAI...)
			if systemMessageOpenAI != nil {
				messagesOpenAI = append(messagesOpenAI, *systemMessageOpenAI)
			}
		} else {
			if systemMessageOpenAI != nil {
				messagesOpenAI = append(messagesOpenAI, *systemMessageOpenAI)
			}
			messagesOpenAI = append(messagesOpenAI, historicalMessagesOpenAI...)
		}
		if len(historicalMessagesOpenAI) == 0 {
			if msg == "" {
				return nil, fmt.Errorf("requires msg")
			}
		}
		if msg != "" {
			messagesOpenAI = append(messagesOpenAI, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.NewOpt(msg),
					},
				},
			})
		}
		if len(messagesOpenAI) == 0 {
			return nil, fmt.Errorf("requires msg")
		}
	case providers.APIShapeAnthropic:
		messagesAnthropic = append(messagesAnthropic, historicalMessagesAnthropic...)
		// Add current user message
		if len(historicalMessagesAnthropic) == 0 {
			if msg == "" {
				return nil, fmt.Errorf("requires msg")
			}
		}
		if msg != "" {
			// no cache user message
			messagesAnthropic = append(messagesAnthropic, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg)),
			)
		}

		if len(messagesAnthropic) == 0 {
			return nil, fmt.Errorf("requires msg")
		}
	case providers.APIShapeGemini:
		messagesGemini = append(messagesGemini, historicalMessagesGemini...)
		if len(historicalMessagesGemini) == 0 {
			if msg == "" {
				return nil, fmt.Errorf("requires msg")
			}
		}
		if msg != "" {
			// no cache user message
			messagesGemini = append(messagesGemini, &genai.Content{
				Parts: []*genai.Part{
					{
						Text: msg,
					},
				},
				Role: genai.RoleUser,
			})
		}

		if len(messagesGemini) == 0 {
			return nil, fmt.Errorf("requires msg")
		}
	default:
		return nil, fmt.Errorf("msg: unsupported provider: %s", c.APIShape)
	}
	return &MessagesUnion{
		OpenAI:    messagesOpenAI,
		Anthropic: messagesAnthropic,
		Gemini:    messagesGemini,
	}, nil
}

func (c *ChatHandler) recordInitialMsg(msg string, userSystemPrompt string, historicalSystemPrompts []string, model string, recordFile string) error {
	if userSystemPrompt != "" {
		var sameWithLast bool
		if len(historicalSystemPrompts) > 0 && historicalSystemPrompts[len(historicalSystemPrompts)-1] == userSystemPrompt {
			sameWithLast = true
		}
		if !sameWithLast {
			userMsg := Message{
				Type:    MsgType_Msg,
				Role:    Role_System,
				Model:   model,
				Content: userSystemPrompt,
			}
			if err := appendToRecordFile(recordFile, &userMsg); err != nil {
				return fmt.Errorf("failed to record user system prompt: %v", err)
			}
		}
	}
	if msg != "" {
		userMsg := Message{
			Type:    MsgType_Msg,
			Role:    Role_User,
			Model:   model,
			Content: msg,
		}
		if err := appendToRecordFile(recordFile, &userMsg); err != nil {
			return fmt.Errorf("failed to record user message: %v", err)
		}
	}
	return nil
}

type ResponseResultOpenAI struct {
	ToolUseNum  int
	Messages    []openai.ChatCompletionMessageParamUnion
	ToolResults []openai.ChatCompletionMessageParamUnion
	TokenUsage  TokenUsage
}

func (c *ChatHandler) processOpenAIResponse(ctx context.Context, resultOpenAI *openai.ChatCompletion, hasMaxRound bool, model string, recordFile string, defaultToolCwd string, mcpInfoMapping map[string]*ToolInfo) (*ResponseResultOpenAI, error) {
	if len(resultOpenAI.Choices) == 0 {
		return nil, fmt.Errorf("response no choices")
	}
	firstChoice := resultOpenAI.Choices[0]
	var toolUseNum int
	var messages []openai.ChatCompletionMessageParamUnion

	// Handle main content
	if firstChoice.Message.Content != "" {
		fmt.Println(firstChoice.Message.Content)

		messages = append(messages, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: param.NewOpt(firstChoice.Message.Content),
				},
			},
		})

		// Record the text response individually if recordFile is specified
		if recordFile != "" {
			assistantMsg := Message{
				Type:    MsgType_Msg,
				Role:    Role_Assistant,
				Model:   model,
				Content: firstChoice.Message.Content,
			}
			if err := appendToRecordFile(recordFile, &assistantMsg); err != nil {
				return nil, fmt.Errorf("failed to record assistant text message: %v", err)
			}
		}
	}

	// Handle tool calls
	var recordToolCalls []openai.ChatCompletionMessageToolCallParam
	var recordToolResults []openai.ChatCompletionMessageParamUnion
	for _, toolCall := range firstChoice.Message.ToolCalls {
		toolUseNum++
		toolCallStr := fmt.Sprintf("<tool_call>%s(%s)</tool_call>", toolCall.Function.Name, toolCall.Function.Arguments)
		fmt.Println(toolCallStr)

		recordToolCalls = append(recordToolCalls, openai.ChatCompletionMessageToolCallParam{
			ID: toolCall.ID,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		})

		// Record the tool call individually if recordFile is specified
		if recordFile != "" {
			toolCallMsg := Message{
				Type:      MsgType_ToolCall,
				Role:      Role_Assistant,
				Model:     model,
				Content:   toolCall.Function.Arguments,
				ToolUseID: toolCall.ID,
				ToolName:  toolCall.Function.Name,
			}
			if err := appendToRecordFile(recordFile, &toolCallMsg); err != nil {
				return nil, fmt.Errorf("failed to record tool call message: %v", err)
			}
		}

		// Check if tool name matches a builtin and execute it
		result, ok := executeTool(ctx, toolCall.Function.Name, toolCall.Function.Arguments, defaultToolCwd, mcpInfoMapping)
		if ok {
			toolResultStr := fmt.Sprintf("<tool_result>%s</tool_result>", limitPrintLength(result))
			fmt.Println(toolResultStr)

			recordToolResults = append(recordToolResults, openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					ToolCallID: toolCall.ID,
					Content: openai.ChatCompletionToolMessageParamContentUnion{
						OfString: param.NewOpt(result),
					},
				},
			})

			// Record the tool result individually if recordFile is specified
			if recordFile != "" {
				toolResultMsg := Message{
					Type:      MsgType_ToolResult,
					Role:      Role_User,
					Model:     model,
					Content:   result,
					ToolUseID: toolCall.ID,
					ToolName:  toolCall.Function.Name,
				}
				if err := appendToRecordFile(recordFile, &toolResultMsg); err != nil {
					return nil, fmt.Errorf("failed to record tool result message: %v", err)
				}
			}
		} else if hasMaxRound {
			return nil, fmt.Errorf("max round > 1 requires tool to be executed: %s", toolCall.Function.Name)
		}
	}

	if len(recordToolCalls) > 0 {
		messages = append(messages, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				ToolCalls: recordToolCalls,
			},
		})
	}

	return &ResponseResultOpenAI{
		ToolUseNum:  toolUseNum,
		Messages:    messages,
		ToolResults: recordToolResults,
		TokenUsage: TokenUsage{
			Input:  resultOpenAI.Usage.PromptTokens,
			Output: resultOpenAI.Usage.CompletionTokens,
			Total:  resultOpenAI.Usage.TotalTokens,

			InputBreakdown: TokenUsageInputBreakdown{
				CacheRead:    resultOpenAI.Usage.PromptTokensDetails.CachedTokens,
				NonCacheRead: resultOpenAI.Usage.PromptTokens - resultOpenAI.Usage.PromptTokensDetails.CachedTokens,
			},
			OutputBreakdown: TokenUsageOutputBreakdown{},
		},
	}, nil
}

type ResponseResultAnthropic struct {
	ToolUseNum int

	Messages    []anthropic.ContentBlockParamUnion
	ToolResults []anthropic.ContentBlockParamUnion
	TokenUsage  TokenUsage
}

func (c *ChatHandler) processAnthropicResponse(ctx context.Context, resultAnthropic *anthropic.Message, hasMaxRound bool, model string, recordFile string, defaultToolCwd string, mcpInfoMapping map[string]*ToolInfo) (*ResponseResultAnthropic, error) {
	var toolUseNum int
	var respContents []anthropic.ContentBlockParamUnion
	var toolResults []anthropic.ContentBlockParamUnion
	for _, msg := range resultAnthropic.Content {
		switch msg.Type {
		case "text":
			txt := msg.AsText()
			fmt.Println(txt.Text)

			respContents = append(respContents, anthropic.ContentBlockParamUnion{
				OfText: &anthropic.TextBlockParam{
					Text: txt.Text,
				},
			})

			// Record the text response individually if recordFile is specified
			if recordFile != "" {
				assistantMsg := Message{
					Type:    MsgType_Msg,
					Role:    Role_Assistant,
					Model:   model,
					Content: txt.Text,
				}
				if err := appendToRecordFile(recordFile, &assistantMsg); err != nil {
					return nil, fmt.Errorf("failed to record assistant text message: %v", err)
				}
			}
		case "tool_use":
			toolUseNum++
			toolUse := msg.AsToolUse()
			toolCallStr := fmt.Sprintf("<tool_call>%s(%s)</tool_call>", toolUse.Name, string(toolUse.Input))
			fmt.Println(toolCallStr)

			input := json.RawMessage(toolUse.Input)
			respContents = append(respContents, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    toolUse.ID,
					Input: input,
					Name:  toolUse.Name,
				},
			})

			// Record the tool call individually if recordFile is specified
			if recordFile != "" {
				toolCallMsg := Message{
					Type:      MsgType_ToolCall,
					Role:      Role_Assistant,
					Model:     model,
					Content:   string(toolUse.Input),
					ToolUseID: toolUse.ID,
					ToolName:  toolUse.Name,
				}
				if err := appendToRecordFile(recordFile, &toolCallMsg); err != nil {
					return nil, fmt.Errorf("failed to record tool call message: %v", err)
				}
			}

			// Check if tool name matches a builtin and execute it
			result, ok := executeTool(ctx, toolUse.Name, string(toolUse.Input), defaultToolCwd, mcpInfoMapping)
			if ok {
				toolResultStr := fmt.Sprintf("<tool_result>%s</tool_result>", limitPrintLength(result))
				fmt.Println(toolResultStr)

				toolResults = append(toolResults, anthropic.ContentBlockParamUnion{
					OfToolResult: &anthropic.ToolResultBlockParam{
						ToolUseID: toolUse.ID,
						Content: []anthropic.ToolResultBlockParamContentUnion{
							{
								OfText: &anthropic.TextBlockParam{
									Text: result,
								},
							},
						},
					},
				})

				// Record the tool result individually if recordFile is specified
				if recordFile != "" {
					toolResultMsg := Message{
						Type:      MsgType_ToolResult,
						Role:      Role_User,
						Model:     model,
						Content:   result,
						ToolUseID: toolUse.ID,
						ToolName:  toolUse.Name,
					}
					if err := appendToRecordFile(recordFile, &toolResultMsg); err != nil {
						return nil, fmt.Errorf("failed to record tool result message: %v", err)
					}
				}
			} else if hasMaxRound {
				return nil, fmt.Errorf("max round > 1 requires tool to be executed: %s", toolUse.Name)
			}
		default:
			return nil, fmt.Errorf("unrecognized message type: %s", msg.Type)
		}
	}

	totalInput := resultAnthropic.Usage.InputTokens + resultAnthropic.Usage.CacheCreationInputTokens + resultAnthropic.Usage.CacheReadInputTokens
	return &ResponseResultAnthropic{
		Messages:    respContents,
		ToolResults: toolResults,
		ToolUseNum:  toolUseNum,
		TokenUsage: TokenUsage{
			Input:  totalInput,
			Output: resultAnthropic.Usage.OutputTokens,
			Total:  totalInput + resultAnthropic.Usage.OutputTokens,

			InputBreakdown: TokenUsageInputBreakdown{
				CacheWrite:   resultAnthropic.Usage.CacheCreationInputTokens,
				CacheRead:    resultAnthropic.Usage.CacheReadInputTokens,
				NonCacheRead: resultAnthropic.Usage.InputTokens,
			},
			OutputBreakdown: TokenUsageOutputBreakdown{},
		},
	}, nil
}

type ResponseResultGemini struct {
	ToolUseNum  int
	Messages    []*genai.Content
	ToolResults []*genai.Content
	TokenUsage  TokenUsage
}

func (c *ChatHandler) processGeminiResponse(ctx context.Context, resultGemini *genai.GenerateContentResponse, toolUsedNum int, hasMaxRound bool, model string, recordFile string, defaultToolCwd string, mcpInfoMapping map[string]*ToolInfo) (*ResponseResultGemini, error) {
	var toolUseNum int
	var respContents []*genai.Content
	var toolResults []*genai.Content

	if len(resultGemini.Candidates) == 0 {
		return nil, fmt.Errorf("empty result candidates")
	}
	choice := resultGemini.Candidates[0]

	for _, part := range choice.Content.Parts {
		if part.FunctionCall != nil {
			toolUseNum++
			toolUse := part.FunctionCall

			toolRecordID := toolUse.ID
			if toolUse.ID == "" {
				toolRecordID = strconv.FormatInt(int64(toolUsedNum+1), 10)
				// does not support ID?
				// NOTE: Gemini no need to put an ID
				// though other providers needs this ID
				// cloneToolUse := *toolUse
				// cloneToolUse.ID = uuid.New().String()
				// toolUse = &cloneToolUse
			}
			argsJSON, err := json.Marshal(toolUse.Args)
			if err != nil {
				return nil, fmt.Errorf("marshal args: %w", err)
			}
			argsJSONStr := string(argsJSON)
			toolCallStr := fmt.Sprintf("<tool_call>%s(%s)</tool_call>", toolUse.Name, argsJSONStr)
			fmt.Println(toolCallStr)

			cloneToolUse := *toolUse
			respContents = append(respContents, &genai.Content{
				Parts: []*genai.Part{
					{
						FunctionCall: &cloneToolUse,
					},
				},
				Role: choice.Content.Role,
			})

			// Record the tool call individually if recordFile is specified
			if recordFile != "" {
				toolCallMsg := Message{
					Type:      MsgType_ToolCall,
					Role:      Role_Assistant,
					Model:     model,
					Content:   argsJSONStr,
					ToolUseID: toolRecordID,
					ToolName:  toolUse.Name,
				}
				if err := appendToRecordFile(recordFile, &toolCallMsg); err != nil {
					return nil, fmt.Errorf("failed to record tool call message: %v", err)
				}
			}

			result, ok := executeTool(ctx, toolUse.Name, argsJSONStr, defaultToolCwd, mcpInfoMapping)
			if ok {
				toolResultStr := fmt.Sprintf("<tool_result>%s</tool_result>", limitPrintLength(result))
				fmt.Println(toolResultStr)

				var response map[string]any
				err := jsondecode.UnmarshalSafe([]byte(result), &response)
				if err != nil {
					return nil, fmt.Errorf("unmarshal tool result: %w", err)
				}

				toolResults = append(toolResults, &genai.Content{
					Role: genai.RoleUser,
					Parts: []*genai.Part{
						{
							FunctionResponse: &genai.FunctionResponse{
								ID:       toolUse.ID,
								Name:     toolUse.Name,
								Response: response,
							},
						},
					},
				})

				// Record the tool result individually if recordFile is specified
				if recordFile != "" {
					toolResultMsg := Message{
						Type:      MsgType_ToolResult,
						Role:      Role_User,
						Model:     model,
						Content:   result,
						ToolUseID: toolRecordID,
						ToolName:  toolUse.Name,
					}
					if err := appendToRecordFile(recordFile, &toolResultMsg); err != nil {
						return nil, fmt.Errorf("failed to record tool result message: %v", err)
					}
				}
			} else if hasMaxRound {
				return nil, fmt.Errorf("max round > 1 requires tool to be executed: %s", toolUse.Name)
			}
		} else if part.Text != "" {
			txt := part.Text
			fmt.Println(txt)
			respContents = append(respContents, &genai.Content{
				Role: choice.Content.Role,
			})
			// Record the text response individually if recordFile is specified
			if recordFile != "" {
				assistantMsg := Message{
					Type:    MsgType_Msg,
					Role:    Role_Assistant,
					Model:   model,
					Content: txt,
				}
				if err := appendToRecordFile(recordFile, &assistantMsg); err != nil {
					return nil, fmt.Errorf("failed to record assistant text message: %v", err)
				}
			}
		}
	}

	var tokenUsage TokenUsage
	if resultGemini.UsageMetadata != nil {
		usage := resultGemini.UsageMetadata

		inputToken := usage.PromptTokenCount + usage.ToolUsePromptTokenCount
		outputToken := usage.CandidatesTokenCount
		cacheRead := usage.CachedContentTokenCount

		tokenUsage = TokenUsage{
			Input:  int64(inputToken),
			Output: int64(outputToken),
			Total:  int64(usage.TotalTokenCount),
			InputBreakdown: TokenUsageInputBreakdown{
				CacheRead:    int64(cacheRead),
				NonCacheRead: int64(inputToken - cacheRead),
			},
		}
	}

	return &ResponseResultGemini{
		Messages:    respContents,
		ToolResults: toolResults,
		ToolUseNum:  toolUseNum,
		TokenUsage:  tokenUsage,
	}, nil
}

func (c *ChatHandler) checkRecordStopReason(recordFile string, model string, resultOpenAI *openai.ChatCompletion, resultAnthropic *anthropic.Message, resultGemini *genai.GenerateContentResponse) (bool, *Message, error) {
	switch c.APIShape {
	case providers.APIShapeOpenAI:
		if len(resultOpenAI.Choices) == 0 {
			return false, nil, fmt.Errorf("response no choices")
		}
		if recordFile != "" {
			firstChoice := resultOpenAI.Choices[0]
			if firstChoice.FinishReason != "" {
				return false, &Message{
					Type:  MsgType_StopReason,
					Role:  Role_Assistant,
					Model: model,
					Content: mustMarshal(map[string]string{
						"finish_reason": firstChoice.FinishReason,
					}),
				}, nil
			}
		}
	case providers.APIShapeAnthropic:
		if resultAnthropic.StopReason != "" {
			return resultAnthropic.StopReason == "end_turn", &Message{
				Type:  MsgType_StopReason,
				Role:  Role_Assistant,
				Model: model,
				Content: mustMarshal(map[string]string{
					"stop_reason":   string(resultAnthropic.StopReason),
					"stop_sequence": resultAnthropic.StopSequence,
				}),
			}, nil
		}
	case providers.APIShapeGemini:
		if len(resultGemini.Candidates) == 0 {
			return false, nil, fmt.Errorf("response no candidates")
		}
		first := resultGemini.Candidates[0]
		if first.FinishReason == "" {
			return false, nil, nil
		}
		return false, &Message{
			Type:  MsgType_StopReason,
			Role:  Role_Assistant,
			Model: model,
			Content: mustMarshal(map[string]string{
				"finish_reason":  string(first.FinishReason),
				"finish_message": first.FinishMessage,
			}),
		}, nil
	default:
		return false, nil, fmt.Errorf("stop reason: unsupported provider: %s", c.APIShape)
	}
	return false, nil, nil
}
