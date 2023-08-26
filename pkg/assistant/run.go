package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ajbouh/bridge/pkg/chat"
	"github.com/ajbouh/bridge/pkg/stt"
)

type Assistant struct {
	Name            string
	Client          *chat.Client
	MaxPromptLength int

	systemMessage string
}

func New(name string, client *chat.Client) *Assistant {
	return &Assistant{
		Name:            name,
		Client:          client,
		MaxPromptLength: 1024,
		systemMessage: strings.ReplaceAll(`
Your name is {}.

{} is a conversational, vocal, artificial intelligence assistant.

{}'s job is to converse with humans to help them accomplish goals.

{} is able to help with a wide variety of tasks from answering questions to assisting the human with creative writing.

Overall {} is a powerful system that can help humans with a wide range of tasks and provide valuable insights as well as taking actions for the human.
`, "{}", name),
	}
}

func (o *Assistant) generate(system, prompt string) (string, error) {
	var (
		chunk string
		err   error
	)
	resp, err := o.Client.CreateChatCompletion(
		context.Background(),
		chat.ChatCompletionRequest{
			MaxTokens: 4096,
			Messages: []chat.ChatCompletionMessage{
				{
					Role:    chat.ChatMessageRoleSystem,
					Content: system,
				},
				{
					Role:    chat.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return chunk, err
	}

	if len(resp.Choices) == 0 {
		return chunk, errors.New("chat returned empty choices")
	}

	fmt.Printf("assistant prompt=%q response=%#v\n", prompt, resp)
	return resp.Choices[0].Message.Content, nil

}

// TODO look into basarn?

func transcriptionAsString(t *stt.Transcription, filter func(s *stt.TranscriptionSegment) bool) string {
	// For now only say something if the word "bridge" occurs in the text.
	texts := []string{}
	lastEnd := float32(0.0)
	for _, segment := range t.Segments {
		if filter != nil && !filter(&segment) {
			continue
		}

		if lastEnd > 0 {
			if segment.End-lastEnd > 0.5 {
				texts = append(texts, "\n", segment.Speaker, ":", " ")
			} else {
				texts = append(texts, " ")
			}
		}
		lastEnd = segment.End

		if segment.Text != "" {
			texts = append(texts, segment.Text)
			continue
		}

		for _, word := range segment.Words {
			texts = append(texts, word.Word)
		}
	}

	return strings.Join(texts, "")
}

func (s *Assistant) Run(transcriptionStream chan<- *stt.Transcription, listener <-chan stt.Document) {
	name := strings.ToLower(s.Name)

	for doc := range listener {
		t := doc.Transcriptions[len(doc.Transcriptions)-1]
		// Only respond to things that aren't based on other parts of the transcript. This avoids loops.
		if len(t.TranscriptSources) > 0 {
			continue
		}

		// only consider text said by a person. TODO don't consider speakerLabel
		text := transcriptionAsString(t, func(s *stt.TranscriptionSegment) bool { return !s.IsAssistant })
		if !strings.Contains(strings.ToLower(text), name) {
			continue
		}

		prompt := ""
		for i := len(doc.Transcriptions) - 1; i >= 0; i-- {
			next := transcriptionAsString(doc.Transcriptions[i], nil)

			if len(prompt)+len(next) >= s.MaxPromptLength {
				break
			}

			prompt = next + "\n" + prompt
		}

		start := t.StartTimestamp
		gen, err := s.generate(s.systemMessage, prompt)

		if err != nil {
			fmt.Printf("error generating: %s", err)
			continue
		}

		transcriptionStream <- &stt.Transcription{
			TranscriptSources: []*stt.Transcription{t},
			StartTimestamp:    start,
			EndTimestamp:      start,
			Segments: []stt.TranscriptionSegment{
				{
					Speaker:     s.Name,
					IsAssistant: true,
					Text:        gen,
				},
			},
		}
	}
}
