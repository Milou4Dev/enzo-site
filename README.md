```markdown
# Simple Website with Gin

A website made for a friend using Go (Gin). The project focuses on simplicity.

## Technologies

- Go 1.23.4
- Gin Web Framework
- HTML/CSS/JavaScript
- Docker

## Installation

```

# Run with Go
go mod tidy
go run main.go

# Or with Docker
docker build -t personal-site .
docker run -p 8080:8080 personal-site
```

## Structure

```

.
├── main.go
├── static/
├── templates/
└── Dockerfile
```
