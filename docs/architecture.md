# Architecture

## System Overview

```
┌─────────┐     ┌─────────┐     ┌──────────┐
│ Browser │────▶│  Nginx  │────▶│ Go Server│
│ (A / B) │◀────│ :80     │◀────│ :8080    │
└─────────┘     └─────────┘     └──────────┘
                  │                   │
                  │ static files      │ /app/uploads/
                  │ /frontend/        │
```

## Components

### Nginx (Port 80)
- Reverse proxy to Go backend (`/api/*`)
- Serves static frontend files (`/`)
- Configured for unlimited upload size (`client_max_body_size 0`)
- Proxy buffering disabled for streaming

### Go Backend (Port 8080)
- `POST /api/upload` — streaming file upload
- `GET /api/files` — list uploaded files
- `GET /api/download/{filename}` — streaming file download
- `DELETE /api/files/{filename}` — delete a file
- Auto-cleanup on SIGTERM/SIGINT

### Frontend
- Upload with progress bar (XHR for progress events)
- File listing with download links
- Delete button per file

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Go over Java | Lower memory, native streaming, tiny image |
| No framework | stdlib `net/http` is sufficient |
| Nginx proxy | Handles buffering, static files, future TLS |
| Docker Compose | Simple single-command deployment |
| Streaming I/O | Support 1GB+ files without memory issues |
