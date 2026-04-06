# Operating Rules

## Project-specific constraints

- Backend uses Go standard library only (no frameworks)
- File uploads must use streaming I/O — never load full file into memory
- All uploaded files are stored in `/app/uploads/` inside the container
- Server must clean up all uploaded files on shutdown (SIGTERM/SIGINT)
- Nginx must be configured for large file uploads (client_max_body_size 0)
- Frontend is static HTML/CSS/JS — no build tools required
- Docker Compose is the only supported deployment method

## Safety rails

- Never expose files outside the uploads directory (path traversal protection)
- Sanitize all filenames before storing
- Do not store or log sensitive metadata
