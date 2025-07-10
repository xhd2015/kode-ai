package anthropic

import (
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// caching: see https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching#caching-tool-definitions
// cache breakpoint:
//
//	A cache breakpoint in Anthropic's prompt caching refers to a marker in the prompt, defined using the cache_control parameter, that indicates where a reusable section of the prompt ends for caching purposes. It allows developers to split a prompt into distinct, cacheable segments, enabling the system to store and reuse these segments across multiple API calls. This reduces processing time and costs by avoiding repeated processing of identical content.
//	a cache breakpoint in Anthropic's prompt caching includes the message or content block that contains the cache_control parameter itself. The breakpoint marks the end of a cacheable segment, and everything up to and including that block is cached for reuse.

func MarkTextBlocksEphemeralCache(textBlocks []anthropic.TextBlockParam) []anthropic.TextBlockParam {
	if len(textBlocks) == 0 {
		return nil
	}
	clone := make([]anthropic.TextBlockParam, len(textBlocks))
	copy(clone, textBlocks)
	last := len(clone) - 1

	cloneLast := clone[last]
	cloneLast.CacheControl = anthropic.NewCacheControlEphemeralParam()

	clone[last] = cloneLast

	return clone
}

func MarkToolsEphemeralCache(tools []anthropic.ToolUnionParam) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	clone := make([]anthropic.ToolUnionParam, len(tools))
	copy(clone, tools)
	last := len(clone) - 1

	cloneLast := clone[last]
	if cloneLast.OfTool != nil {
		cloneTool := *cloneLast.OfTool
		cloneTool.CacheControl = anthropic.NewCacheControlEphemeralParam()

		cloneLast.OfTool = &cloneTool
	} else {
		panic(fmt.Errorf("unhandled tool type"))
	}

	clone[last] = cloneLast
	return clone
}

func MarkMsgsEphemeralCache(msgs []anthropic.MessageParam) []anthropic.MessageParam {
	if len(msgs) == 0 {
		return nil
	}
	clone := make([]anthropic.MessageParam, len(msgs))
	copy(clone, msgs)
	last := len(clone) - 1

	cloneLast := clone[last]

	if len(cloneLast.Content) == 0 {
		panic(fmt.Errorf("empty content blocks"))
	}
	cloneLast.Content = MarkContentBlocksEphemeralCache(cloneLast.Content)

	clone[last] = cloneLast
	return clone
}

func MarkContentBlocksEphemeralCache(contentBlocks []anthropic.ContentBlockParamUnion) []anthropic.ContentBlockParamUnion {
	if len(contentBlocks) == 0 {
		return nil
	}
	cloneList := make([]anthropic.ContentBlockParamUnion, len(contentBlocks))
	copy(cloneList, contentBlocks)
	last := len(cloneList) - 1

	cloneLast := cloneList[last]
	if cloneLast.OfText != nil {
		clone := *cloneLast.OfText
		clone.CacheControl = anthropic.NewCacheControlEphemeralParam()
		cloneLast.OfText = &clone
	} else if cloneLast.OfToolResult != nil {
		clone := *cloneLast.OfToolResult
		clone.CacheControl = anthropic.NewCacheControlEphemeralParam()
		cloneLast.OfToolResult = &clone
	} else if cloneLast.OfToolUse != nil {
		clone := *cloneLast.OfToolUse
		clone.CacheControl = anthropic.NewCacheControlEphemeralParam()
		cloneLast.OfToolUse = &clone
	} else {
		panic(fmt.Errorf("unhandled content block type"))
	}

	cloneList[last] = cloneLast
	return cloneList
}
