# Reasoning Model Example / 推理模型示例

演示如何在 agent-go 中使用推理模型。

Demonstrates how to use reasoning models in agent-go.

## 支持的推理模型 / Supported Reasoning Models

### OpenAI
- o1-preview
- o1-mini
- o3 系列
- o4 系列

### Google Gemini
- gemini-2.5-flash-thinking
- 所有包含 "thinking" 关键词的模型

### Anthropic Claude
- 需要显式配置 thinking 参数
- Requires explicit thinking configuration

## 运行示例 / Run Example

```bash
# 设置 API Key / Set API Key
export OPENAI_API_KEY=your-api-key

# 运行示例 / Run example
go run main.go
```

## 功能特性 / Features

- ✅ **自动检测**: 自动识别推理模型
- ✅ **推理提取**: 自动提取并存储推理内容
- ✅ **优雅降级**: 提取失败不影响 Agent 执行
- ✅ **零配置**: 无需额外配置,开箱即用
- ✅ **性能优化**: 仅对推理模型执行提取操作

- ✅ **Auto Detection**: Automatically identifies reasoning models
- ✅ **Reasoning Extraction**: Automatically extracts and stores reasoning content
- ✅ **Graceful Degradation**: Extraction failures don't interrupt Agent execution
- ✅ **Zero Configuration**: Works out of the box
- ✅ **Performance Optimized**: Extraction only runs for reasoning models

## 示例输出 / Example Output

```
🤖 ReasoningAgent is thinking...
📝 Input: Solve this complex problem: ...

💬 Response:
[Model's final answer]

🧠 Reasoning Process:
[Model's step-by-step thinking process]

📊 Reasoning Tokens: 1250
```

## 代码说明 / Code Explanation

```go
// 推理内容自动提取和存储
// Reasoning content is automatically extracted and stored
for _, msg := range output.Messages {
    if msg.ReasoningContent != nil {
        // 访问推理内容 / Access reasoning content
        fmt.Println(msg.ReasoningContent.Content)

        // 可选字段 / Optional fields
        if msg.ReasoningContent.TokenCount != nil {
            // Token 统计 / Token count
        }
        if msg.ReasoningContent.RedactedContent != nil {
            // 脱敏内容 / Redacted content
        }
    }
}
```

## 高级用法 / Advanced Usage

### 使用不同的推理模型 / Using Different Reasoning Models

```go
// OpenAI o1-mini (更快更便宜)
model, _ := openaiModel.New(openaiModel.Config{
    Model: "o1-mini",
})

// Gemini 2.5 Flash Thinking
// (需要安装 Gemini 支持 / Requires Gemini support)
// model, _ := geminiModel.New(geminiModel.Config{
//     Model: "gemini-2.5-flash-thinking",
// })
```

### 访问完整推理历史 / Accessing Full Reasoning History

```go
output, _ := agent.Run(ctx, input)

// 遍历所有消息,包括历史推理 / Iterate through all messages
for i, msg := range output.Messages {
    if msg.Role == types.RoleAssistant && msg.ReasoningContent != nil {
        fmt.Printf("Turn %d Reasoning:\n%s\n", i, msg.ReasoningContent.Content)
    }
}
```

## 注意事项 / Notes

1. **API 费用**: 推理模型通常比标准模型更昂贵
2. **响应时间**: 推理模型可能需要更长的处理时间
3. **Token 限制**: 注意推理内容会占用额外的 token

1. **API Costs**: Reasoning models are typically more expensive
2. **Response Time**: Reasoning models may take longer to process
3. **Token Limits**: Be aware that reasoning content uses additional tokens
