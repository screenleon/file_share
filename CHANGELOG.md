# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0] - 2026-04-06
### Added
- All-in-one Docker image (`screenleon/file-share`) published to Docker Hub.
- Root-level multi-stage `Dockerfile`: Go binary + Nginx + frontend in a single image.
- `nginx/nginx-standalone.conf`: Nginx config for intra-container proxy (`127.0.0.1:8080`).
- `supervisord.conf`: manages `nginx` and `file-server` together as PID 1.
- GitHub Actions workflow (`.github/workflows/docker-publish.yml`):
  - PR → fast single-arch build-check only (no push).
  - Push to `main` → multi-arch (`linux/amd64`, `linux/arm64`) build + push `latest`.
  - Push semver tag → build + push versioned tags.
  - Blocks publish if the tag already exists on Docker Hub.
- `scripts/release.sh`: one-command version bump, commit, and tag.
- MIT `LICENSE` file.

### Changed
- `README.md`: added Docker Hub quick-start (`docker run` one-liner), volume mount example, and badges.

### Security
- `file-server` runs as `nobody` inside the all-in-one container.
- `supervisord` `stopwaitsecs=60` ensures graceful file cleanup before SIGKILL.
- Multi-arch build uses `--platform=$BUILDPLATFORM` with `TARGETOS`/`TARGETARCH` for correct cross-compilation.

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
