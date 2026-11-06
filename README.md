# Dictionary Translator

## Project overview

(Yet to read and edit this file)


Adjusted Bible Concordance is a web-based tool for exploring and processing Hebrew text through a structured letter and phrase concordance.

The project aims to make the relationship between Hebrew words, their component meanings and English translations easier to examine. It combines Hebrew letter analysis, lexical lookup and morphological interpretation to produce readable, word-aligned English output while preserving the original Hebrew text structure.

The longer-term goal is to develop a flexible research and study tool that helps users explore how Hebrew words are constructed and how their meanings relate to one another in a passage. It is intended to support textual study and analysis rather than function as a conventional Bible translation.

The application has a Next.js frontend and a Go backend. It supports text input and `.txt` file uploads, letter or phrase concordance processing, on-page previews and downloadable output.

## Current development focus

The project is currently focused on improving the Hebrew phrase concordance and its treatment of word structure.

The main areas of development are:

- **HK-based lexical lookup:** Use the adjusted HK concordance as the primary lexical reference, retaining its associated H entries as supporting data.
- **Preferred translations:** Use `preferred_trans` as the source of English translations. Do not substitute `preferred_def` or Strong's definitions when a preferred translation is unavailable.
- **Hebrew normalisation:** Recognise final Hebrew letter forms during matching without changing the original source text.
- **Morphological analysis:** Identify possible prefixes, lexical stems and suffixes, including alternative valid interpretations.
- **Word-aligned output:** Keep the English components belonging to one Hebrew word together, while preserving spaces between separate Hebrew words.
- **Ambiguity handling:** Display multiple valid interpretations in square brackets, separated by `|`, rather than silently selecting one interpretation.

### English output formatting

The output uses two distinct separators:

- An underscore (`_`) joins English words or components derived from the same Hebrew word.
- A space separates translations belonging to different Hebrew words.

For example:

**Hebrew input**

```text
חזון עבדיה כה אמר אדני יהוה לאדום שמועה
```

**Illustrative English output**

```text
a_revelation Obadiah like_this said my_basis LORD onto_I_will_their_him_ report
```

This makes the original Hebrew word boundaries identifiable in the English output.

Where a Hebrew word has multiple valid interpretations, the alternatives are grouped:

```text
[interpretation_one | interpretation_two]
```

The aim is to make the output useful for studying the structure and possible meanings of the original Hebrew, rather than flattening every component into an ordinary English sentence.

---

## Project structure

The project is organised so that the frontend application can grow into a multi-client workspace.

```text
project-root/
├── compose.yaml                 # or docker-compose.yml
├── backend/
│   ├── Dockerfile
│   └── ...                      # Go API, concordance logic and supporting data
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

The paths above show the main application files. The backend may contain additional packages, data files and configuration.

## Requirements

- Docker Engine with Docker Compose (or Docker Desktop)
- A modern web browser
- `curl` for API checks

For local development without Docker:

- Go, at the version required by the backend's `go.mod`
- Node.js 22 or later
- npm

## Configuration

### Frontend Dockerfile

The Next.js Dockerfile is located at:

`frontend/apps/web/Dockerfile.dev`

The Docker Compose frontend build configuration should use `frontend/apps/web` as its build context:

```yaml
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

This assumes the Go backend Compose service is named `backend` and listens on port `8080` inside the Compose network. Keep the existing backend service configuration.

### Next.js configuration

The file `frontend/apps/web/next.config.ts` should enable standalone output:

```ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  poweredByHeader: false,
  output: "standalone",
};

export default nextConfig;
```

The frontend's server-side API route forwards requests to the Go backend using `BACKEND_API_URL`.

Inside Docker Compose, use:

`http://backend:8080`

Do not use `localhost:8081` for communication between containers. That is the host-side address.

## Start the application with Docker

Run these commands from the project root, where your Compose file is located:

```bash
docker compose up --build -d
```

Check that the services are running:

```bash
docker compose ps
```

View the frontend logs:

```bash
docker compose logs -f frontend
```

View the backend logs:

```bash
docker compose logs -f backend
```

To stop the application:

```bash
docker compose down
```

To rebuild after changing application code or Docker configuration:

```bash
docker compose up --build -d
```

## Application URLs

When running locally with the port mappings described above:

| Component | URL | Purpose |
|---|---|---|
| Web application | http://localhost:3000 | Open the Dictionary Translator interface |
| Backend health | http://localhost:8081/api/v1/health | Check whether the Go API is responding |
| Letter concordance | http://localhost:8081/api/v1/concordance/ | Access the letter concordance endpoint |
| Phrase concordance | http://localhost:8081/api/v1/phrase/ | Access the phrase concordance endpoint |
| Frontend processing proxy | http://localhost:3000/api/process | Process text through the Next.js API route |
| Backend processing API | http://localhost:8081/api/v1/process | Process text directly through the Go API |

The backend URLs assume that the Compose backend service publishes container port `8080` on host port `8081`. Adjust the URLs if your port mapping is different.

## Test the application

### 1. Check the frontend

Open:

http://localhost:3000

Confirm that the Dictionary Translator page loads and that the input, file upload, concordance mode and output options are visible.

### 2. Check the backend health endpoint

```bash
curl -i http://localhost:8081/api/v1/health
```

A successful response should return an HTTP success status.

### 3. Test letter concordance processing directly

Replace `YOUR TEST TEXT` with text supported by your concordance data:

```bash
curl -i -X POST http://localhost:8081/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{"mode":"letter","text":"YOUR TEST TEXT"}'
```

### 4. Test phrase concordance processing directly

```bash
curl -i -X POST http://localhost:8081/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{"mode":"phrase","text":"YOUR TEST TEXT"}'
```

These requests send JSON to the Go API. Successful processing returns plain text. The output depends on the input and the concordance data loaded by the backend.

### 5. Test through the Next.js API proxy

```bash
curl -i -X POST http://localhost:3000/api/process \
  -H "Content-Type: application/json" \
  -d '{"mode":"phrase","text":"YOUR TEST TEXT"}'
```

The proxy should return the backend's processing response. If the direct backend request works but this request fails, check the frontend's `BACKEND_API_URL` setting and the frontend logs.

### 6. Test the complete workflow in the browser

- [ ] Open `http://localhost:3000`.
- [ ] Enter a short text sample.
- [ ] Select **Letter Concordance** and process the text.
- [ ] Select **Phrase Concordance** and process the text.
- [ ] Upload a `.txt` file and process it.
- [ ] Select the preview option and confirm that the result appears on the page.
- [ ] Select the download option and confirm that an output file downloads.
- [ ] Test a larger input and confirm that the application uses the download behaviour rather than rendering an excessively large result on the page.
- [ ] Submit empty input and confirm that the application displays a useful validation message.

### 7. Test from another device on your local network

On Linux, find your local IP address with:

```bash
hostname -I
```

Then open the following address on another device connected to the same network:

```text
http://YOUR-IP-ADDRESS:3000
```

If this does not work, check your firewall and confirm that Docker publishes port `3000`.

## Hebrew phrase lookup and translation rules

### HK concordance

The phrase translator uses:

`data/hebrew-phrase-concordance-HK-mapped.json`

HK groups are the primary lexical lookup records. Their associated H Strong's entries are retained as supporting references.

The HK identifier is the reference used for lexical matching and display. Individual H identifiers should not replace the associated HK value.

### Preferred translations

The application uses the `preferred_trans` field for English translation.

The following fields must not be used as automatic translation fallbacks:

- `preferred_def`
- `strongs_def`
- Individual KJV definitions or usage counts

If an HK entry does not have a `preferred_trans` value, it should remain unresolved rather than displaying a definition as though it were the preferred translation.

### Hebrew normalisation

The five Hebrew final forms are normalised for matching:

| Final form | Matching form |
|---|---|
| ך | כ |
| ם | מ |
| ן | נ |
| ף | פ |
| ץ | צ |

Normalisation is for lookup only. The original Hebrew source text must remain unchanged in the output.

### Prefixes, suffixes and alternative readings

When an exact whole-word match is unavailable, the analyser checks possible prefixes, stems and suffixes against the HK concordance.

It should retain multiple valid interpretations where the available lexical evidence supports them.

For example, `בגוים` may have alternative analyses involving:

- HK1077, associated with H1458–H1460
- HK1085, associated with H1471

The application should not discard one valid interpretation simply because another suffix can be identified.

Unmatched words should be returned unchanged rather than being split into unverified meanings.

### Phrase output and word boundaries

The phrase processor preserves the original order of Hebrew words.

English output follows these conventions:

| Separator | Meaning |
|---|---|
| Space | Separates different Hebrew words |
| Underscore (`_`) | Joins English components belonging to one Hebrew word |
| Square brackets (`[ ]`) | Groups alternative interpretations |
| Vertical bar (`\|`) | Separates alternative interpretations |

For example:

```text
Hebrew:
חזון עבדיה כה אמר

English:
a_revelation Obadiah like_this said
```

The formatting is intended to make it possible to trace each English translation block back to its corresponding Hebrew word.

## Notes and current limits

- The frontend proxy and Go processing endpoint currently use a JSON request containing `mode` and `text`.
- The backend request size limit is currently 10 MiB. The frontend upload limit should be kept consistent with the backend limit.
- The interface can download large results instead of displaying the full result in the browser. This does not remove the request-size limit or make the backend a streaming processor.
- Phrase analysis is dependent on the coverage and quality of the supplied HK concordance and affix data. Not every Hebrew form will necessarily have a confirmed analysis.
- Alternative morphological interpretations should be treated as possibilities supported by the available data, not as definitive grammatical conclusions.
- The current output is a structured lexical and morphological rendering, not a fully contextual English translation of a passage.
- For substantially larger files or multiple concurrent clients, consider asynchronous jobs, streaming or chunked processing, per-client job tracking and suitable storage.
- The Compose service names, port mappings and endpoint paths in this README must match the actual Compose and Go API configuration in the project.

## Useful commands

```bash
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

## Future development direction

The project can develop beyond basic concordance lookup into a more comprehensive Hebrew text analysis workspace.

Potential areas for further development include:

1. **Word-level analysis:** Present the original Hebrew word alongside its HK reference, stem, prefixes, suffixes and preferred translation.
2. **Phrase-level presentation:** Display Hebrew words and their corresponding English blocks in clearly identifiable aligned sections.
3. **Morphological refinement:** Improve the validation and ranking of alternative word segmentations using additional Hebrew grammatical evidence.
4. **Concordance exploration:** Allow users to open an HK entry and explore its associated H entries, lexical information and usage.
5. **Passage analysis:** Support longer passages while retaining verse, sentence and word boundaries.
6. **Multi-client workspace:** Develop the frontend into a workspace where users can manage texts, analysis sessions and exported results.

The guiding principle is to preserve the relationship between the original Hebrew text, its lexical references and the English meanings derived from the concordance.
