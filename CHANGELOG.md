# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-04-06
### Added
- Initial file sharing service with Go backend and Nginx reverse proxy.
- Streaming upload/download pipeline for large files (1GB+).
- Frontend for drag-and-drop upload, file listing, download, and delete.
- Docker Compose orchestration for backend and Nginx services.
- Upload-state lock to block download/delete while upload is in progress.
- Structured backend logs for upload, download, and delete events.
- Download counter and uploader IP metadata exposed in file list API.
- Automatic cleanup behavior on service shutdown and volume removal.
