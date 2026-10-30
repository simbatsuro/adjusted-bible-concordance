# Dictionary Translator

A web-based interface for processing text using the Go dictionary and
concordance API. The frontend is built with Next.js and the backend is
written in Go. The application supports text input and `.txt` file
uploads, letter or phrase concordance processing, on-page previews and
downloadable output.

## Project structure

The project is organised so that the frontend application can grow into
a multi-client workspace.

``` text
project-root/
├── compose.yaml                 # or docker-compose.yml
├── backend/
│   ├── Dockerfile
│   └── ...                      # Go API and supporting data
└── frontend/
    └── apps/
        └── web/
            ├── app/
            │   ├── api/
            │   │   └── process/
            │   │       └── route.ts
            │   ├── globals.css
            │   ├── layout.tsx
            │   └── page.tsx
            ├── Dockerfile.dev
            ├── package.json
            ├── package-lock.json
            ├── next.config.ts
            └── tsconfig.json
```

The paths above show the relevant application files. Your existing
backend may contain additional packages, data files and configuration.

## Requirements

-   Docker Desktop (or Docker Engine with Docker Compose)
-   A modern web browser
-   `curl` for the API checks below (available by default on macOS and
    most Linux distributions)

For local development without Docker, you will also need:

-   Go, at the version required by the backend's `go.mod`
-   Node.js 22 or later and npm

## Configuration

### Frontend Dockerfile

The Next.js Dockerfile is located at:

`frontend/apps/web/Dockerfile.dev`

The Docker Compose frontend build configuration must use
`frontend/apps/web` as its build context:

``` yaml
frontend:
  build:
    context: ./frontend/apps/web
    dockerfile: Dockerfile.dev
  container_name: dictionary-translator-frontend
  ports:
    - "3000:3000"
  environment:
    NODE_ENV: production
    BACKEND_API_URL: http://backend:8080
  depends_on:
    - backend
  restart: unless-stopped
```

This assumes the Go backend Compose service is named `backend` and
listens on port `8080` inside the Compose network. Keep your existing
backend service configuration.

### Next.js configuration

The file `frontend/apps/web/next.config.ts` should enable standalone
output:

``` ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  poweredByHeader: false,
  output: "standalone",
};

export default nextConfig;
```

The frontend's server-side API route forwards requests to the Go backend
using `BACKEND_API_URL`. Inside Docker Compose, use
`http://backend:8080`, not `http://localhost:8081`. The latter is the
host-side address used by tools running on your Mac.

## Start the application with Docker

Run these commands from the project root, where your Compose file is
located:

``` bash
docker compose up --build -d
```

Check that the services are running:

``` bash
docker compose ps
```

View the frontend logs:

``` bash
docker compose logs -f frontend
```

View the backend logs:

``` bash
docker compose logs -f backend
```

To stop the application:

``` bash
docker compose down
```

To rebuild after changing application code or Docker configuration:

``` bash
docker compose up --build -d
```

## Application URLs

When running locally with the port mappings described above:

  ---------------------------------------------------------------------------------------------
  Component               URL                                           Purpose
  ----------------------- --------------------------------------------- -----------------------
  Web application         <http://localhost:3000>                       Open the Dictionary
                                                                        Translator interface

  Backend health          <http://localhost:8081/api/v1/health>         Check whether the Go
                                                                        API is responding

  Letter concordance      <http://localhost:8081/api/v1/concordance/>   Access the letter
                                                                        concordance endpoint

  Phrase concordance      <http://localhost:8081/api/v1/phrase/>        Access the phrase
                                                                        concordance endpoint

  Frontend processing     <http://localhost:3000/api/process>           Process text through
  proxy                                                                 the Next.js API route

  Backend processing API  `http://localhost:8081/api/v1/process`        Process text directly
                                                                        through the Go API
  ---------------------------------------------------------------------------------------------

The backend URLs assume that the Compose backend service publishes
container port `8080` on host port `8081`. Adjust the URLs if your port
mapping is different.

## Test the application

### 1. Check the frontend

Open <http://localhost:3000> in your browser.

Confirm that the Dictionary Translator page loads and that the input,
file upload, concordance mode and output options are visible.

### 2. Check the backend health endpoint

Run from your Mac:

``` bash
curl -i http://localhost:8081/api/v1/health
```

A successful response should return an HTTP success status. The response
body depends on the health handler implemented in the backend.

### 3. Test letter concordance processing directly

Replace `YOUR TEST TEXT` with text that is supported by your concordance
data:

``` bash
curl -i -X POST http://localhost:8081/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{"mode":"letter","text":"YOUR TEST TEXT"}'
```

### 4. Test phrase concordance processing directly

``` bash
curl -i -X POST http://localhost:8081/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{"mode":"phrase","text":"YOUR TEST TEXT"}'
```

These requests send JSON to the Go API. Successful processing should
return the processed text as plain text. The actual output depends on
the input and the concordance data loaded by the backend.

### 5. Test through the Next.js API proxy

This verifies that the frontend server can reach the Go backend over the
Docker Compose network:

``` bash
curl -i -X POST http://localhost:3000/api/process \
  -H "Content-Type: application/json" \
  -d '{"mode":"phrase","text":"YOUR TEST TEXT"}'
```

The proxy should return the backend's processing response. If the direct
backend request works but this request fails, check the frontend's
`BACKEND_API_URL` setting and the frontend logs.

### 6. Test the complete workflow in the browser

Use this checklist to test the user-facing application:

-   [ ] Open `http://localhost:3000`.
-   [ ] Enter a short text sample.
-   [ ] Select **Letter Concordance** and process the text.
-   [ ] Select **Phrase Concordance** and process the text.
-   [ ] Upload a `.txt` file and process it.
-   [ ] Select the preview option and confirm that the result appears on
    the page.
-   [ ] Select the download option and confirm that an output file
    downloads.
-   [ ] Test a larger input and confirm that the application uses the
    download behaviour rather than rendering an excessively large result
    on the page.
-   [ ] Submit empty input and confirm that the application displays a
    useful validation message.

### 7. Test from another device on your local network

On your Mac, find its local IP address:

``` bash
ipconfig getifaddr en0
```

Then open the following address on another device connected to the same
network:

``` text
http://YOUR-MAC-IP:3000
```

If this does not work, check your Mac firewall and confirm that Docker
publishes port `3000`.

## Troubleshooting

### Frontend does not load

Check the service status and logs:

``` bash
docker compose ps
docker compose logs --tail=100 frontend
```

Confirm that the frontend publishes port `3000`:

``` yaml
ports:
  - "3000:3000"
```

### Backend health check fails

Check the backend logs:

``` bash
docker compose logs --tail=100 backend
```

Confirm that the backend listens on port `8080` inside its container and
that Compose publishes it to host port `8081`, for example:

``` yaml
ports:
  - "8081:8080"
```

### Frontend cannot reach the backend

Check that the frontend service has:

``` yaml
environment:
  BACKEND_API_URL: http://backend:8080
```

The hostname `backend` must match the actual backend service name in
your Compose file. Do not use `localhost:8081` for communication between
containers: inside the frontend container, `localhost` refers to the
frontend container itself.

### Docker build fails

Rebuild and inspect the logs:

``` bash
docker compose build --no-cache frontend
docker compose up -d
docker compose logs --tail=100 frontend
```

Check that the frontend build context points to `frontend/apps/web` and
that `Dockerfile.dev` exists in that directory.

## Notes and current limits

-   The frontend proxy and Go processing endpoint currently use a JSON
    request containing `mode` and `text`.
-   The backend request size limit is currently 10 MiB. The frontend
    upload limit should be kept consistent with the backend limit.
-   The interface can download large results instead of displaying the
    full result in the browser. This does not remove the request-size
    limit or make the backend a streaming processor.
-   For substantially larger files or multiple concurrent clients,
    consider asynchronous jobs, streaming or chunked processing,
    per-client job tracking and suitable storage.
-   The Compose service names, port mappings and endpoint paths in this
    README must match the actual Compose and Go API configuration in
    your project.

## Useful commands

``` bash
# Start or rebuild all services
docker compose up --build -d

# Show running services
docker compose ps

# Follow frontend logs
docker compose logs -f frontend

# Follow backend logs
docker compose logs -f backend

# Stop services
docker compose down

# Test backend health
curl -i http://localhost:8081/api/v1/health
```
