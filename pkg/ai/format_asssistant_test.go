package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChat(t *testing.T) {
	question := `水恨重要\🤔`
	answer, err := Chat(question)
	if err != nil {
		t.Errorf("Chat failed: %v", err)
	}
	assert.Equal(t, `水很重要🤔`, strings.TrimSpace(answer))
}
