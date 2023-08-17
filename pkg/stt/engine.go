package stt

import (
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	logr "github.com/ajbouh/bridge/pkg/log"
)

// FIXME make these configurable
const (
	// This is determined by the hyperparameter configuration that whisper was trained on.
	// See more here: https://github.com/ggerganov/whisper.cpp/issues/909
	SampleRate   = 16000 // 16kHz
	sampleRateMs = SampleRate / 1000
	// This determines how much audio we will be passing to whisper inference.
	// We will buffer up to (whisperSampleWindowMs - pcmSampleRateMs) of old audio and then add
	// audioSampleRateMs of new audio onto the end of the buffer for inference
	sampleWindowMs = 24000 // 24 second sample window
	windowSize     = sampleWindowMs * sampleRateMs
	// This is the minimum ammount of audio we want to buffer before running inference
	// 2 seconds of audio samples
	windowMinSize = 2000 * sampleRateMs
	// This determines how often we will try to run inference.
	// We will buffer (pcmSampleRateMs * whisperSampleRate / 1000) samples and then run inference
	pcmSampleRateMs = 500 // FIXME PLEASE MAKE ME AN CONFIG PARAM
	pcmWindowSize   = pcmSampleRateMs * sampleRateMs

	// this is an arbitrary number I picked after testing a bit
	// feel free to play around
	energyThresh  = 0.0005
	silenceThresh = 0.015
)

var Logger = logr.New()

type EngineParams struct {
	OnDocumentUpdate func(Document)
	Transcriber      Transcriber
}

type Document struct {
	Transcriptions       []Transcription `json:"transcriptions"`
	TranscribedText      string          `json:"transcribedText"`
	CurrentTranscription string          `json:"currentTranscription"`
	NewText              string          `json:"newText"`
	StartedAt            int64           `json:"startedAt"`
}

func (e *Engine) ComposeSimple(script Transcription) (Document, uint32) {
	doc := Document{
		NewText:   "",
		StartedAt: e.startedAt,
	}
	end := uint32(0)

	for i, segment := range script.Segments {
		if doc.NewText != "" && !strings.HasSuffix(doc.NewText, " ") {
			doc.NewText += " "
		}

		doc.NewText += segment.Text

		if i == len(script.Segments)-1 {
			end = uint32(segment.End * 1000)
		}
	}

	if e.finishedText != "" && !strings.HasSuffix(e.finishedText, " ") {
		e.finishedText += " "
	}
	e.finishedText += doc.NewText
	script.AllLanguageProbs = nil
	e.transcriptions = append(e.transcriptions, script)

	doc.TranscribedText = e.finishedText
	doc.Transcriptions = append([]Transcription{}, e.transcriptions...)

	return doc, end
}

type Engine struct {
	sync.Mutex
	// Buffer to store new audio. When this fills up we will try to run inference
	pcmWindow []float32
	// Buffer to store old and new audio to run inference on.
	// By inferring on old and new audio we can help smooth out cross word boundaries
	window               []float32
	lastHandledTimestamp uint32

	// document composer to handle incoming transcriptions
	finishedText   string
	transcriptions []Transcription
	startedAt      int64

	// callback when we have a document update
	onDocumentUpdate func(Document)

	transcriber Transcriber

	isSpeaking bool
}

func New(params EngineParams) (*Engine, error) {
	if params.Transcriber == nil {
		return nil, errors.New("you must supply a Transciber to create an engine")
	}

	return &Engine{
		window:               make([]float32, 0, windowSize),
		pcmWindow:            make([]float32, 0, pcmWindowSize),
		lastHandledTimestamp: 0,
		onDocumentUpdate:     params.OnDocumentUpdate,
		transcriber:          params.Transcriber,
		isSpeaking:           false,
		startedAt:            time.Now().Unix(),
	}, nil
}

func (e *Engine) OnDocumentUpdate(fn func(Document)) {
	e.onDocumentUpdate = fn
}

func (e *Engine) Write(pcm []float32, endTimestamp uint32) {
	e.writeVAD(pcm, endTimestamp)
}

// XXX DANGER XXX
// This is highly experiemential and will probably crash in very interesting ways. I have deadlines
// and am hacking towards what I want to demo. Use at your own risk :D
// XXX DANGER XXX
//
// writeVAD only buffers audio if somone is speaking. It will run inference after the audio transitions from
// speaking to not speaking
func (e *Engine) writeVAD(pcm []float32, endTimestamp uint32) {
	// TODO normalize PCM and see if we can make it better
	// endTimestamp is the latest packet timestamp + len of the audio in the packet
	// FIXME make these timestamps make sense
	e.Lock()
	defer e.Unlock()
	if len(e.pcmWindow)+len(pcm) > pcmWindowSize {
		// This shouldn't happen hopefully...
		Logger.Infof("GOING TO OVERFLOW PCM WINDOW BY %d", len(e.pcmWindow)+len(pcm)-pcmWindowSize)
	}
	e.pcmWindow = append(e.pcmWindow, pcm...)
	if len(e.pcmWindow) >= pcmWindowSize {
		// reset window
		defer func() {
			e.pcmWindow = e.pcmWindow[:0]
		}()

		isSpeaking, energy, silence := VAD(e.pcmWindow)

		defer func() {
			e.isSpeaking = isSpeaking
		}()

		if isSpeaking && e.isSpeaking {
			Logger.Debug("STILL SPEAKING")
			// add to buffer and wait
			// FIXME make sure we have space
			e.window = append(e.window, e.pcmWindow...)
			return
		} else if isSpeaking && !e.isSpeaking {
			Logger.Debug("JUST STARTED SPEAKING")
			e.isSpeaking = isSpeaking
			// we just started speaking, add to buffer and wait
			// FIXME make sure we have space
			e.window = append(e.window, e.pcmWindow...)
			return
		} else if !isSpeaking && e.isSpeaking {
			Logger.Debug("JUST STOPPED SPEAKING")
			// TODO consider waiting for a few more samples?
			e.window = append(e.window, e.pcmWindow...)

		} else if !isSpeaking && !e.isSpeaking {
			// by having this here it gives us a bit of an opportunity to pause in our speech
			if len(e.window) != 0 {
				// we have not been speaking for at least 500ms now so lets run inference
				Logger.Infof("running whisper inference with %d window length", len(e.window))

				transcript, err := e.transcriber.Transcribe(e.window)
				if err != nil {
					Logger.Error(err, "error running inference")
					return
				}
				transcript.EndTimestamp = uint64(endTimestamp)
				Logger.Debugf("GOT TRANSCRIPTION %+v", transcript)

				doc, _ := e.ComposeSimple(transcript)

				if e.onDocumentUpdate != nil {
					e.onDocumentUpdate(doc)
				}

				e.window = e.window[:0]
			}
			// not speaking do nothing
			Logger.Debugf("NOT SPEAKING energy=%#v (energyThreshold=%#v) silence=%#v (silenceThreshold=%#v) ", energy, energyThresh, silence, silenceThresh)
			return
		}
	}
}

// NOTE This is a very rough implemntation. We should improve it :D
// VAD performs voice activity detection on a frame of audio data.
func VAD(frame []float32) (bool, float32, float32) {
	// Compute frame energy
	energy := float32(0)
	for i := 0; i < len(frame); i++ {
		energy += frame[i] * frame[i]
	}
	energy /= float32(len(frame))

	// Compute frame silence
	silence := float32(0)
	for i := 0; i < len(frame); i++ {
		silence += float32(math.Abs(float64(frame[i])))
	}
	silence /= float32(len(frame))

	// Apply energy threshold
	if energy < energyThresh {
		return false, energy, silence
	}

	// Apply silence threshold
	if silence < silenceThresh {
		return false, energy, silence
	}

	return true, energy, silence
}
