ARG CUDA_IMAGE="12.1.1-devel-ubuntu22.04"
FROM nvidia/cuda:${CUDA_IMAGE} as base

FROM base as fetch
WORKDIR /var/model

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    python3 python3-pip \
  && rm -rf /var/lib/apt/lists/*

RUN python3 -m pip install --upgrade pip requests

ARG MODEL_ACCOUNT
ARG MODEL_TAG
ARG MODEL_SEARCH
COPY services/llama-cpp-python/hug_model.py .
# RUN python3 ./hug_model.py -a TheBloke -t llama -s llama2_70b_chat_uncensored-GGML -f q4_K_S
RUN python3 ./hug_model.py -a "${MODEL_ACCOUNT}" -t "${MODEL_TAG}" -s "${MODEL_SEARCH}"

FROM base as dist

# We need to set the host to 0.0.0.0 to allow outside access
ENV HOST 0.0.0.0

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    python3 python3-pip gcc wget \
    ocl-icd-opencl-dev opencl-headers clinfo \
    libclblast-dev libopenblas-dev \
  && mkdir -p /etc/OpenCL/vendors && echo "libnvidia-opencl.so.1" > /etc/OpenCL/vendors/nvidia.icd \
  && rm -rf /var/lib/apt/lists/*

# COPY . .

# setting build related env vars
ENV CUDA_DOCKER_ARCH=all
ENV LLAMA_CUBLAS=1

# Install depencencies
RUN python3 -m pip install --upgrade pip pytest cmake scikit-build setuptools fastapi uvicorn sse-starlette pydantic-settings

# Install llama-cpp-python (build with cuda)
RUN CMAKE_ARGS="-DLLAMA_CUBLAS=on" FORCE_CMAKE=1 pip install llama-cpp-python

COPY --from=fetch /var/model/* /var/model/
ENV MODEL=/var/model/model.bin

# Run the server
ENTRYPOINT ["python3", "-m", "llama_cpp.server"]
