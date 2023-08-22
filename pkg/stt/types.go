package stt

type Transcriber interface {
	Transcribe(audioData []float32) (Transcription, error)
}

type Translator interface {
	Translate(audioData []float32, language string) (Transcription, error)
}

type Word struct {
	Start       float32 `json:"start"`
	End         float32 `json:"end"`
	Word        string  `json:"word"`
	Probability float32 `json:"prob"`
}

type TranscriptionSegment struct {
	ID               uint32  `json:"id"`
	Seek             uint32  `json:"seek"`
	Start            float32 `json:"start"`
	End              float32 `json:"end"`
	Text             string  `json:"text"`
	Temperature      float32 `json:"temperature"`
	AvgLogprob       float32 `json:"avg_logprob"`
	CompressionRatio float32 `json:"compression_ratio"`
	NoSpeechProb     float32 `json:"no_speech_prob"`
	Words            []Word  `json:"words"`

	Speaker     string `json:"speaker"`
	IsAssistant bool   `json:"is_assistant"`
}

type Transcription struct {
	EndTimestamp        uint64              `json:"endTimestamp"`
	Language            string              `json:"language"`
	LanguageProbability float32             `json:"language_prob"`
	Duration            float32             `json:"duration"`
	AllLanguageProbs    *map[string]float32 `json:"all_language_probs,omitempty"`

	Segments []TranscriptionSegment `json:"segments"`
}