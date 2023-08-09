import os

if __name__ == "__main__":
    import faster_whisper.utils
    faster_whisper.utils.download_model(
        os.environ.get("MODEL_SIZE", "small"),
    )
else:
    from faster_whisper import WhisperModel
    model = WhisperModel(
        os.environ.get("MODEL_SIZE", "small"),
        device=os.environ.get("MODEL_DEVICE", "cpu"),
        compute_type=os.environ.get("MODEL_COMPUTE_TYPE", "int8"),
        local_files_only=True,
    )
