# File Server Example

Static file serving with wildcard routes and SPA fallback.

## Run

```bash
mkdir -p public uploads
echo "<h1>Hello</h1>" > public/index.html
go run .
```

## Endpoints

- `GET /static/*filepath` - Serve from ./public
- `GET /uploads/*filepath` - Serve from ./uploads
- `GET /download/:filename` - Download with attachment header
- `GET /api/files` - List files in ./public
- `GET /*` - SPA fallback to index.html

## Features

- Wildcard route parameters
- Directory traversal protection
- Content-Type detection
- Custom 404 handler for SPA
