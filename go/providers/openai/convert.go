// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"

	"github.com/microsoft/agent-framework-go/chat"
	openailib "github.com/sashabaranov/go-openai"
)

// toOpenAIMessages converts chat.Message slice to OpenAI format.
func toOpenAIMessages(messages []chat.Message, instructionRole string) []openailib.ChatCompletionMessage {
	result := make([]openailib.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		result = append(result, toOpenAIMessage(msg, instructionRole))
	}
	return result
}

// toOpenAIMessage converts a single chat.Message to OpenAI format.
func toOpenAIMessage(msg chat.Message, instructionRole string) openailib.ChatCompletionMessage {
	role := string(msg.Role)
	if msg.Role == chat.RoleSystem && instructionRole == "developer" {
		role = "developer"
	}

	oaiMsg := openailib.ChatCompletionMessage{
		Role: role,
		Name: msg.Name,
	}

	// Handle tool result messages
	if msg.Role == chat.RoleTool && msg.ToolCallID != "" {
		oaiMsg.ToolCallID = msg.ToolCallID
		if len(msg.Contents) > 0 {
			oaiMsg.Content = extractTextContent(msg.Contents)
		}
		return oaiMsg
	}

	// Handle single text content as simple string
	if len(msg.Contents) == 1 {
		if tc, ok := msg.Contents[0].(*chat.TextContent); ok {
			oaiMsg.Content = tc.Text
			return oaiMsg
		}
	}

	// Handle multiple contents or non-text content
	if len(msg.Contents) > 0 {
		oaiMsg.MultiContent = toOpenAIContentParts(msg.Contents)
	}

	// Handle tool calls from assistant messages
	if len(msg.ToolCalls) > 0 {
		oaiMsg.ToolCalls = toOpenAIToolCalls(msg.ToolCalls)
	}

	return oaiMsg
}

// extractTextContent extracts concatenated text from content items.
func extractTextContent(contents []chat.Content) string {
	var text string
	for _, c := range contents {
		if tc, ok := c.(*chat.TextContent); ok {
			text += tc.Text
		}
	}
	return text
}

// toOpenAIContentParts converts chat.Content items to OpenAI message parts.
func toOpenAIContentParts(contents []chat.Content) []openailib.ChatMessagePart {
	parts := make([]openailib.ChatMessagePart, 0, len(contents))

	for _, c := range contents {
		switch content := c.(type) {
		case *chat.TextContent:
			parts = append(parts, openailib.ChatMessagePart{
				Type: openailib.ChatMessagePartTypeText,
				Text: content.Text,
			})

		case *chat.ImageContent:
			imgPart := openailib.ChatMessagePart{
				Type: openailib.ChatMessagePartTypeImageURL,
			}

			if content.URL != "" {
				imgPart.ImageURL = &openailib.ChatMessageImageURL{
					URL:    content.URL,
					Detail: openailib.ImageURLDetail(content.Detail),
				}
			} else if content.Base64Data != "" {
				// Construct data URL for base64 images
				dataURL := "data:" + content.MediaType + ";base64," + content.Base64Data
				imgPart.ImageURL = &openailib.ChatMessageImageURL{
					URL:    dataURL,
					Detail: openailib.ImageURLDetail(content.Detail),
				}
			}

			parts = append(parts, imgPart)
		}
	}

	return parts
}

// toOpenAIToolCalls converts chat.ToolCall slice to OpenAI format.
func toOpenAIToolCalls(calls []chat.ToolCall) []openailib.ToolCall {
	result := make([]openailib.ToolCall, len(calls))
	for i, call := range calls {
		result[i] = openailib.ToolCall{
			ID:   call.ID,
			Type: openailib.ToolTypeFunction,
			Function: openailib.FunctionCall{
				Name:      call.Name,
				Arguments: string(call.Arguments),
			},
		}
	}
	return result
}

// toOpenAITools converts chat.ToolDefinition slice to OpenAI tool format.
func toOpenAITools(tools []chat.ToolDefinition) []openailib.Tool {
	result := make([]openailib.Tool, len(tools))
	for i, t := range tools {
		var params json.RawMessage
		if t.Parameters != nil {
			params, _ = json.Marshal(t.Parameters)
		}

		result[i] = openailib.Tool{
			Type: openailib.ToolTypeFunction,
			Function: &openailib.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return result
}

// fromOpenAIMessage converts an OpenAI response choice to chat.Message.
func fromOpenAIMessage(choice openailib.ChatCompletionChoice) chat.Message {
	msg := chat.Message{
		Role: chat.Role(choice.Message.Role),
	}

	// Handle text content
	if choice.Message.Content != "" {
		msg.Contents = []chat.Content{chat.NewTextContent(choice.Message.Content)}
	}

	// Handle tool calls
	if len(choice.Message.ToolCalls) > 0 {
		msg.ToolCalls = fromOpenAIToolCalls(choice.Message.ToolCalls)
	}

	return msg
}

// fromOpenAIToolCalls converts OpenAI tool calls to chat.ToolCall slice.
func fromOpenAIToolCalls(calls []openailib.ToolCall) []chat.ToolCall {
	result := make([]chat.ToolCall, len(calls))
	for i, call := range calls {
		result[i] = chat.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: json.RawMessage(call.Function.Arguments),
		}
	}
	return result
}

// fromOpenAIFinishReason converts OpenAI finish reason string to chat.FinishReason.
func fromOpenAIFinishReason(reason string) chat.FinishReason {
	switch reason {
	case "stop":
		return chat.FinishReasonStop
	case "length":
		return chat.FinishReasonLength
	case "tool_calls", "function_call":
		return chat.FinishReasonToolCalls
	case "content_filter":
		return chat.FinishReasonContentFilter
	default:
		return chat.FinishReasonStop
	}
}
