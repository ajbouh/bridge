FROM huggingface/transformers-pytorch-gpu

# Set the working directory in the container
WORKDIR /app

# Copy the requirements file to the container
COPY ./requirements.txt .

# Install the dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the source code to the container
COPY ./. .

# Expose the port on which the application will run (default is 8000)
EXPOSE 8000
ENV PYTHONUNBUFFERED=1
RUN python3 model.py
# Start the application with Uvicorn
CMD ["uvicorn", "FastapiServer:app", "--host", "0.0.0.0", "--port", "8000"]