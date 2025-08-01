FROM golang:1.24.2

RUN apt update

RUN apt install gh nodejs npm -y

RUN npm install -g @anthropic-ai/claude-code
