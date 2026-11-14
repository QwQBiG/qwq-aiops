package server

import (
	"context"
	"fmt"
	"net/http"
	"qwq/internal/agent"
	"qwq/internal/logger"
	"qwq/internal/utils"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func handleWSChat(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Info("WebSocket upgrade failed: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer connection.Close()
	inputs := make(chan string, 1)
	// Read independently so a disconnect cancels an in-flight model or command.
	go func() {
		defer cancel()
		for {
			_, message, err := connection.ReadMessage()
			if err != nil {
				return
			}
			select {
			case inputs <- string(message):
			case <-ctx.Done():
				return
			default:
				// A client must wait for completion instead of filling an unbounded
				// queue. Keep the reader free to observe disconnect/control frames.
				return
			}
		}
	}()
	go func() {
		<-ctx.Done()
		connection.Close()
	}()
	write := func(kind, content string) bool {
		if err := connection.WriteJSON(map[string]string{"type": kind, "content": content}); err != nil {
			cancel()
			return false
		}
		return true
	}
	messages := agent.GetBaseMessages()
	for {
		var input string
		select {
		case <-ctx.Done():
			return
		case input = <-inputs:
		}
		if response := agent.CheckStaticResponse(input); response != "" {
			if !write("answer", response) {
				return
			}
		} else if command := agent.GetQuickCommand(input); command != "" {
			if !write("status", "executing") {
				return
			}
			output := utils.ExecuteShellContext(ctx, command)
			if strings.TrimSpace(output) == "" {
				output = "(No output)"
			}
			if !write("answer", fmt.Sprintf("```\n%s\n```", output)) {
				return
			}
		} else {
			messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: input + " (Context: Current Linux Server)"})
			for step := 0; step < 5 && ctx.Err() == nil; step++ {
				if !write("status", "thinking") {
					return
				}
				response, more := agent.ProcessAgentStepForWebContext(ctx, &messages, func(message string) { write("log", message) })
				if response.Content != "" && !write("answer", response.Content) {
					return
				}
				if !more {
					break
				}
			}
		}
		if ctx.Err() != nil || !write("status", "done") {
			return
		}
	}
}
