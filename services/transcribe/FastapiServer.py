import time
import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel
from typing import Dict, List, Optional
from model import model

app = FastAPI()

class Word(BaseModel):
    start: float
    end: float
    word: str
    probability: float

class TranscriptionSegment(BaseModel):
    # startTimestamp: int
    # endTimestamp: int

    id: int
    seek: int
    start: float
    end: float
    text: str
    # tokens: List[int]
    temperature: float
    avg_logprob: float
    compression_ratio: float
    no_speech_prob: float
    words: Optional[List[Word]]


class Transcription(BaseModel):
    language: str
    language_probability: float
    duration: float
    all_language_probs: Optional[Dict[str, float]]

    segments: List[TranscriptionSegment]


@app.post('/transcribe')
def transcribe(transcription_request: List[float]) -> Transcription:
    # Perform transcription on the audio data

    start = time.time()
    transcription = perform_transcription(transcription_request)
    end = time.time()

    print("Took:", end - start)
    print(transcription)
    return transcription


def perform_transcription(transcription_request):
    segments, info = model.transcribe(
        np.array(transcription_request, dtype=np.float32),
        vad_filter=True,
        beam_size=5,
        word_timestamps=True,
    )

    return Transcription(
        language=info.language,
        language_probability=info.language_probability,
        duration=info.duration,
        all_language_probs={
            language: prob
            for language, prob in info.all_language_probs
        } if info.all_language_probs else None,
        segments=[
            TranscriptionSegment(
                id=segment.id,
                seek=segment.seek,
                start=segment.start,
                end=segment.end,
                text=segment.text,
                temperature=segment.temperature,
                avg_logprob=segment.avg_logprob,
                compression_ratio=segment.compression_ratio,
                no_speech_prob=segment.no_speech_prob,
                words=[
                    Word(
                        start=word.start,
                        end=word.end,
                        word=word.word,
                        probability=word.probability,
                    )
                    for word in segment.words
                ] if segment.words else None,
            )
            for segment in segments
        ],
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="localhost", port=8000)
