# Simple Website with Gin

A website made for a friend using Go (Gin). The project focuses on simplicity.

## Technologies

- Go 1.23.4
- Gin Web Framework
- HTML/CSS/JavaScript
- Docker

## Installation

### Running with Go

```bash
# Install dependencies
go mod tidy

# Run the application
go run main.go
```

### Running with Docker

```bash
# Build the Docker image
docker build -t personal-site .

# Run the Docker container
docker run -p 8080:8080 personal-site
```
