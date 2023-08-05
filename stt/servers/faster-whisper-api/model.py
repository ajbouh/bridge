import os
from faster_whisper import WhisperModel

model = WhisperModel(
    os.environ.get("MODEL_SIZE", "small"),
    device=os.environ.get("MODEL_DEVICE", "cpu"),
    compute_type=os.environ.get("MODEL_COMPUTE_TYPE", "int8"),
)
