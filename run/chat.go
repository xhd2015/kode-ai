package run

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	anth_opt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openai_opt "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/internal/ioread"
	"github.com/xhd2015/kode-ai/internal/terminal"
	"github.com/xhd2015/kode-ai/providers"
	anthropic_helper "github.com/xhd2015/kode-ai/providers/anthropic"
	"github.com/xhd2015/kode-ai/tools"
)

type ChatOptions struct {
	maxRound int

	systemPrompt string
	toolPresets  []string
	toolFiles    []string
	toolJSONs    []string
	recordFile   string

	toolDefaultCwd string

	ignoreDuplicateMsg bool
	noCache            bool

	logRequest bool
	verbose    bool
	logChat    bool
}

type ChatHandler struct {
	Provider             providers.Provider
	TokenEnvKey          string
	BaseUrlEnvKey        string
	DefaultBaseUrlEnvKey string

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
}

type MessageHistoryUnion struct {
	FullHistory Messages

	OpenAI        []openai.ChatCompletionMessageParamUnion
	Anthropic     []anthropic.MessageParam
	SystemPrompts []string
}

type ToolsUnion struct {
}

func (c *ChatHandler) Handle(model string, baseUrl string, token string, msg string, opts ChatOptions) error {
	toolPresets := opts.toolPresets
	recordFile := opts.recordFile
	ignoreDuplicateMsg := opts.ignoreDuplicateMsg
	maxRound := opts.maxRound
	defaultToolCwd := opts.toolDefaultCwd
	openAIKeepMultipleSystemPrompts := c.openAIKeepMultipleSystemPrompts_DO_NOT_SET

	noCache := opts.noCache

	logRequest := opts.logRequest
	verbose := opts.verbose
	logChat := opts.logChat

	_ = verbose

	var absDefaultToolCwd string
	if defaultToolCwd != "" {
		var err error
		absDefaultToolCwd, err = filepath.Abs(defaultToolCwd)
		if err != nil {
			return err
		}
	}

	if token == "" {
		var envOption string
		if c.TokenEnvKey != "" {
			token = os.Getenv(c.TokenEnvKey)
			envOption = " or " + c.TokenEnvKey
		}
		if token == "" {
			return errors.New("requires --token" + envOption)
		}
	}
	if baseUrl == "" {
		var envBaseURL string
		if c.BaseUrlEnvKey != "" {
			envBaseURL = os.Getenv(c.BaseUrlEnvKey)
		}
		if envBaseURL == "" && c.DefaultBaseUrlEnvKey != "" {
			envBaseURL = os.Getenv(c.DefaultBaseUrlEnvKey)
		}
		baseUrl = envBaseURL
	}

	OPEN_AI := c.Provider == providers.ProviderOpenAI
	ANTHROPIC := c.Provider == providers.ProviderAnthropic
	isGemini := c.Provider == providers.ProviderGemini

	_ = isGemini
	// Load historical messages from record file if it exists

	historyMsgs, err := c.readHistoryMessages(recordFile, openAIKeepMultipleSystemPrompts)
	if err != nil {
		return err
	}
	historicalMessagesOpenAI := historyMsgs.OpenAI
	historicalMessagesAnthropic := historyMsgs.Anthropic
	historicalSystemPrompts := historyMsgs.SystemPrompts

	msg, stop, err := c.checkUserMsgDuplicate(msg, historyMsgs.FullHistory, ignoreDuplicateMsg)
	if err != nil {
		return err
	}
	if stop {
		return nil
	}

	clients, err := c.createClient(baseUrl, token, logRequest)
	if err != nil {
		return err
	}
	clientOpenAI := clients.OpenAI
	clientAnthropic := clients.Anthropic

	toolSchemas, err := tools.ParseSchemas(opts.toolFiles, opts.toolJSONs)
	if err != nil {
		return err
	}
	presetTools, err := tools.GetPresetTools(toolPresets)
	if err != nil {
		return err
	}
	toolSchemas = append(toolSchemas, presetTools...)

	var toolsOpenAI []openai.ChatCompletionToolParam
	var toolsAnthropic []anthropic.ToolUnionParam
	switch {
	case OPEN_AI:
		toolsOpenAI, err = toolSchemas.ToOpenAI()
		if err != nil {
			return err
		}
	case ANTHROPIC:
		toolsAnthropic, err = toolSchemas.ToAnthropic()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported provider: %s", c.Provider)
	}

	var systemMessageOpenAI *openai.ChatCompletionMessageParamUnion

	var systemAnthropic []anthropic.TextBlockParam
	var userSystemPrompt string
	if opts.systemPrompt != "" {
		content, err := ioread.ReadOrContent(opts.systemPrompt)
		if err != nil {
			return err
		}
		userSystemPrompt = content

		switch {
		case OPEN_AI:
			systemMessageOpenAI = &openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.NewOpt(content),
					},
				},
			}
		case ANTHROPIC:
			systemMsg := anthropic.TextBlockParam{
				Text: content,
			}
			systemAnthropic = append(systemAnthropic, systemMsg)
		default:
			return fmt.Errorf("unsupported provider: %s", c.Provider)
		}
	} else if len(historicalSystemPrompts) > 0 {
		switch {
		case OPEN_AI:
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
		case ANTHROPIC:
			// If user doesn't provide a system prompt, use the last system prompt in historical messages
			lastSystemPrompt := historicalSystemPrompts[len(historicalSystemPrompts)-1]

			systemMsg := anthropic.TextBlockParam{
				Text: lastSystemPrompt,
			}
			systemAnthropic = append(systemAnthropic, systemMsg)
		default:
			return fmt.Errorf("system prompt: unsupported provider: %s", c.Provider)
		}
	}

	// build messages
	msgsUnion, err := c.buildMessages(msg, openAIKeepMultipleSystemPrompts, systemMessageOpenAI, historicalMessagesOpenAI, historicalMessagesAnthropic)
	if err != nil {
		return err
	}
	messagesOpenAI := msgsUnion.OpenAI
	messagesAnthropic := msgsUnion.Anthropic

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

	if needCache && c.Provider == providers.ProviderAnthropic {
		systemAnthropic = anthropic_helper.MarkTextBlocksEphemeralCache(systemAnthropic)
		toolsAnthropic = anthropic_helper.MarkToolsEphemeralCache(toolsAnthropic)
	}

	ctx := context.Background()
	i := 0
	for ; i < LOOP; i++ {
		var resultOpenAI *openai.ChatCompletion
		var resultAnthropic *anthropic.Message

		startTime := time.Now()
		if logChat {
			// log making request
			fmt.Println("Request...")
		}

		switch {
		case OPEN_AI:
			resultOpenAI, err = clientOpenAI.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Model:    model,
				Messages: messagesOpenAI,
				Tools:    toolsOpenAI,
				N:        param.NewOpt(int64(1)),
			})
			if err != nil {
				return err
			}
		case ANTHROPIC:
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
		default:
			return fmt.Errorf("call api: unsupported provider: %s", c.Provider)
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

		var hasToolCalls bool
		switch {
		case OPEN_AI:
			res, err := c.processOpenAIResponse(resultOpenAI, hasMaxRound, model, recordFile, absDefaultToolCwd, opts.toolPresets)
			if err != nil {
				return err
			}
			tokenUsage = res.TokenUsage
			respMessages := res.Messages
			toolResults := res.ToolResults

			messagesOpenAI = append(messagesOpenAI, respMessages...)
			messagesOpenAI = append(messagesOpenAI, toolResults...)
			if res.HasToolCalls {
				hasToolCalls = true
			}
		case ANTHROPIC:
			res, err := c.processAnthropicResponse(resultAnthropic, hasMaxRound, model, recordFile, absDefaultToolCwd, opts.toolPresets)
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
			if res.HasToolCalls {
				hasToolCalls = true
			}
		default:
			return fmt.Errorf("unsupported provider: %s", c.Provider)
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

		stopped, err := c.checkRecordStopReason(recordFile, model, resultOpenAI, resultAnthropic)
		if err != nil {
			return err
		}
		cost, costOK := computeCost(c.Provider, model, tokenUsage)
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
		if !hasToolCalls {
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

func printTokenUsage(w io.Writer, title string, tokenUsage TokenUsage, cost string) {
	fmt.Fprintf(w, "%s - Input: %d, Cache/R: %d, Cache/W: %d, NonCache/R: %d, Output: %d, Total: %d, Cost: %s\n",
		title,
		tokenUsage.Input,
		tokenUsage.InputBreakdown.CacheRead,
		tokenUsage.InputBreakdown.CacheWrite,
		tokenUsage.InputBreakdown.NonCacheRead,
		tokenUsage.Output,
		tokenUsage.Total,
		cost,
	)
}

func (c *ChatHandler) readHistoryMessages(recordFile string, openAIKeepMultipleSystemPrompts bool) (*MessageHistoryUnion, error) {
	var msgHistory Messages

	var historicalMessagesOpenAI []openai.ChatCompletionMessageParamUnion
	var historicalMessagesAnthropic []anthropic.MessageParam
	var historicalSystemPrompts []string
	if recordFile != "" {

		var err error
		msgHistory, err = loadHistoricalMessages(recordFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load historical messages: %v", err)
		}

		switch c.Provider {
		case providers.ProviderOpenAI:
			historicalMessagesOpenAI, historicalSystemPrompts, err = msgHistory.ToOpenAI(openAIKeepMultipleSystemPrompts)
			if err != nil {
				return nil, fmt.Errorf("convert anthropic messages: %w", err)
			}
		case providers.ProviderAnthropic:
			historicalMessagesAnthropic, historicalSystemPrompts, err = msgHistory.ToAnthropic()
			if err != nil {
				return nil, fmt.Errorf("convert anthropic messages: %w", err)
			}
		default:
			return nil, fmt.Errorf("read recording: unsupported provider: %s", c.Provider)
		}

	}
	return &MessageHistoryUnion{
		FullHistory:   msgHistory,
		OpenAI:        historicalMessagesOpenAI,
		Anthropic:     historicalMessagesAnthropic,
		SystemPrompts: historicalSystemPrompts,
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

func (c *ChatHandler) createClient(baseURL string, token string, logRequest bool) (*ClientUnion, error) {
	var clientOpenAI *openai.Client
	var clientAnthropic *anthropic.Client
	switch c.Provider {
	case providers.ProviderOpenAI:
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
	case providers.ProviderAnthropic:
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
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.Provider)
	}
	return &ClientUnion{
		OpenAI:    clientOpenAI,
		Anthropic: clientAnthropic,
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
}

// Add historical messages first
func (c *ChatHandler) buildMessages(msg string, openAIKeepMultipleSystemPrompts bool, systemMessageOpenAI *openai.ChatCompletionMessageParamUnion, historicalMessagesOpenAI []openai.ChatCompletionMessageParamUnion, historicalMessagesAnthropic []anthropic.MessageParam) (*MessagesUnion, error) {
	var messagesOpenAI []openai.ChatCompletionMessageParamUnion
	var messagesAnthropic []anthropic.MessageParam
	switch c.Provider {
	case providers.ProviderOpenAI:
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
	case providers.ProviderAnthropic:
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
	default:
		return nil, fmt.Errorf("msg: unsupported provider: %s", c.Provider)
	}
	return &MessagesUnion{
		OpenAI:    messagesOpenAI,
		Anthropic: messagesAnthropic,
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
	HasToolCalls bool
	Messages     []openai.ChatCompletionMessageParamUnion
	ToolResults  []openai.ChatCompletionMessageParamUnion
	TokenUsage   TokenUsage
}

func (c *ChatHandler) processOpenAIResponse(resultOpenAI *openai.ChatCompletion, hasMaxRound bool, model string, recordFile string, defaultToolCwd string, toolPresets []string) (*ResponseResultOpenAI, error) {
	if len(resultOpenAI.Choices) == 0 {
		return nil, fmt.Errorf("response no choices")
	}
	firstChoice := resultOpenAI.Choices[0]
	var hasToolCalls bool
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
		hasToolCalls = true
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

		// Check if tool name matches a preset and execute it
		result, ok := executePresetTool(toolCall.Function.Name, toolCall.Function.Arguments, defaultToolCwd, toolPresets)
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
		HasToolCalls: hasToolCalls,
		Messages:     messages,
		ToolResults:  recordToolResults,
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
	HasToolCalls bool
	Messages     []anthropic.ContentBlockParamUnion
	ToolResults  []anthropic.ContentBlockParamUnion
	TokenUsage   TokenUsage
}

func (c *ChatHandler) processAnthropicResponse(resultAnthropic *anthropic.Message, hasMaxRound bool, model string, recordFile string, defaultToolCwd string, toolPresets []string) (*ResponseResultAnthropic, error) {
	var hasToolCalls bool
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
			hasToolCalls = true
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

			// Check if tool name matches a preset and execute it

			result, ok := executePresetTool(toolUse.Name, string(toolUse.Input), defaultToolCwd, toolPresets)
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
		Messages:     respContents,
		ToolResults:  toolResults,
		HasToolCalls: hasToolCalls,
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

func (c *ChatHandler) checkRecordStopReason(recordFile string, model string, resultOpenAI *openai.ChatCompletion, resultAnthropic *anthropic.Message) (bool, error) {
	switch c.Provider {
	case providers.ProviderOpenAI:
		if len(resultOpenAI.Choices) == 0 {
			return false, fmt.Errorf("response no choices")
		}
		if recordFile != "" {
			firstChoice := resultOpenAI.Choices[0]
			if firstChoice.FinishReason != "" {
				stopMsg := Message{
					Type:  MsgType_StopReason,
					Role:  Role_Assistant,
					Model: model,
					Content: mustMarshal(map[string]string{
						"finish_reason": firstChoice.FinishReason,
					}),
				}
				if err := appendToRecordFile(recordFile, &stopMsg); err != nil {
					return false, fmt.Errorf("failed to record stop reason message: %v", err)
				}
			}
		}
	case providers.ProviderAnthropic:
		if resultAnthropic.StopReason != "" {
			if recordFile != "" {
				stopMsg := Message{
					Type:  MsgType_StopReason,
					Role:  Role_Assistant,
					Model: model,
					Content: mustMarshal(map[string]string{
						"stop_reason":   string(resultAnthropic.StopReason),
						"stop_sequence": resultAnthropic.StopSequence,
					}),
				}
				if err := appendToRecordFile(recordFile, &stopMsg); err != nil {
					return false, fmt.Errorf("failed to record stop reason message: %v", err)
				}
			}
			if resultAnthropic.StopReason == "end_turn" {
				return true, nil
			}
		}
	default:
		return false, fmt.Errorf("stop reason: unsupported provider: %s", c.Provider)
	}
	return false, nil
}
