# File Share Server

Read these files before starting work:

1. `docs/operating-rules.md` — safety, scope, validation, and project-specific constraints
2. `docs/architecture.md` — system architecture and design decisions

## Architecture

### Frontend
- Static HTML/CSS/JS served by Nginx
- Streaming upload via `fetch` + `ReadableStream`
- Download via direct file links

### Backend
- Go HTTP server with streaming I/O (`io.Copy`)
- Auto-cleanup of uploads on shutdown (signal handler)
- REST API for upload, download, list, delete

### Infrastructure
- Docker Compose orchestration
- Nginx reverse proxy (handles large file uploads)

## Workflow

```
Upload: Browser → Nginx → Go Backend → Disk (streaming)
Download: Browser → Nginx → Go Backend → Disk (streaming)
Cleanup: SIGTERM/SIGINT → Go signal handler → rm uploads/
```
