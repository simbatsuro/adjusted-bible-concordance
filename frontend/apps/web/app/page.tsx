
"use client";

import { ChangeEvent, useEffect, useRef, useState } from "react";

type Mode = "phrase" | "letter";
type OutputMode = "preview" | "download";

const MAX_FILE_BYTES = 10 * 1024 * 1024;
const PREVIEW_LIMIT = 100_000;
const PREVIEW_LENGTH = 20_000;

function downloadText(text: string, filename: string) {
  const blob = new Blob([text], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");

  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();

  URL.revokeObjectURL(url);
}

export default function HomePage() {
  const [mode, setMode] = useState<Mode>("phrase");
  const [outputMode, setOutputMode] = useState<OutputMode>("preview");
  const [inputText, setInputText] = useState("");
  const [fileName, setFileName] = useState("");
  const [result, setResult] = useState("");
  const [resultSize, setResultSize] = useState(0);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (new TextEncoder().encode(inputText).byteLength > PREVIEW_LIMIT) {
      setOutputMode("download");
    }
  }, [inputText]);

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    setError("");
    setResult("");
    setResultSize(0);

    if (!file) {
      setFileName("");
      return;
    }

    if (file.size > MAX_FILE_BYTES) {
      setError("This file exceeds the 10 MiB limit. Please use a smaller text file.");
      event.target.value = "";
      return;
    }

    try {
      const text = await file.text();
      setInputText(text);
      setFileName(file.name);
      setStatus(`Loaded ${file.name} (${formatBytes(file.size)}).`);
      if (file.size > PREVIEW_LIMIT) setOutputMode("download");
    } catch {
      setError("The file could not be read. Please try another .txt file.");
    }
  }

  async function processText() {
    if (!inputText.trim()) {
      setError("Enter some text or upload a text file first.");
      return;
    }

    setBusy(true);
    setError("");
    setStatus("Processing your text…");
    setResult("");
    setResultSize(0);

    try {
      const response = await fetch("/api/process", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode, text: inputText }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Processing failed (${response.status}).`);
      }

      const blob = await response.blob();
      const bytes = blob.size;
      setResultSize(bytes);

      const originalName = fileName.replace(/\.[^.]+$/, "") || "translation";
      const outputName = `${originalName}.${mode}.txt`;

      if (outputMode === "download" || bytes > PREVIEW_LIMIT) {
        downloadText(await blob.text(), outputName);
        setStatus(
          bytes > PREVIEW_LIMIT
            ? `Output downloaded as ${outputName}. It was too large to display safely.`
            : `Output downloaded as ${outputName}.`,
        );
      } else {
        setResult(await blob.text());
        setStatus("Processing complete.");
      }
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : "An unexpected error occurred while processing the text.",
      );
      setStatus("");
    } finally {
      setBusy(false);
    }
  }

  function clearAll() {
    setInputText("");
    setFileName("");
    setResult("");
    setResultSize(0);
    setError("");
    setStatus("");
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  return (
    <main className="page-shell">
      <header className="topbar">
        <a className="brand" href="/" aria-label="Dictionary Translator home">
          <span className="brand-mark">א</span>
          <span>Dictionary Translator</span>
        </a>
        <span className="api-indicator">
          <span className="status-dot" />
          Translation workspace
        </span>
      </header>

      <section className="hero">
        <p className="eyebrow">HEBREW CONCORDANCE WORKSPACE</p>
        <h1>Words into meaning.</h1>
        <p className="hero-copy">
          Enter Hebrew text or upload a text file. Choose your translation
          method and view or download the result.
        </p>
      </section>

      <section className="workspace">
        <div className="panel input-panel">
          <div className="panel-heading">
            <div>
              <p className="step-label">STEP 1</p>
              <h2>Your text</h2>
            </div>
            <button className="text-button" type="button" onClick={clearAll}>
              Clear
            </button>
          </div>

          <label className="upload-zone">
            <input
              ref={fileInputRef}
              type="file"
              accept=".txt,text/plain"
              onChange={handleFileChange}
            />
            <span className="upload-icon" aria-hidden="true">↑</span>
            <span className="upload-title">Choose a text file</span>
            <span className="upload-help">
              .txt files up to 10 MiB
            </span>
            <span className="upload-action">Browse files</span>
          </label>

          {fileName && (
            <div className="file-chip">
              <span aria-hidden="true">▤</span>
              <span className="file-chip-name">{fileName}</span>
              <button
                type="button"
                aria-label="Remove uploaded file"
                onClick={() => {
                  setFileName("");
                  if (fileInputRef.current) fileInputRef.current.value = "";
                }}
              >
                ×
              </button>
            </div>
          )}

          <div className="or-divider"><span>OR ENTER TEXT</span></div>

          <label className="field-label" htmlFor="source-text">
            Input text
          </label>
          <textarea
            id="source-text"
            value={inputText}
            onChange={(event) => {
              setInputText(event.target.value);
              setResult("");
              setResultSize(0);
            }}
            placeholder="Paste or type Hebrew text here…"
            dir="auto"
            spellCheck={false}
            rows={10}
          />
          <div className="field-footer">
            <span>{inputText.length.toLocaleString()} characters</span>
            <span>{formatBytes(new TextEncoder().encode(inputText).byteLength)}</span>
          </div>

          <div className="section-divider" />

          <div className="panel-heading compact-heading">
            <div>
              <p className="step-label">STEP 2</p>
              <h2>Translation method</h2>
            </div>
          </div>

          <div className="mode-options">
            <label className={`choice-card ${mode === "phrase" ? "selected" : ""}`}>
              <input
                type="radio"
                name="mode"
                value="phrase"
                checked={mode === "phrase"}
                onChange={() => setMode("phrase")}
              />
              <span className="choice-symbol">מִלָּה</span>
              <span className="choice-title">Phrase concordance</span>
              <span className="choice-description">
                Look up phrases and apply prefix and suffix matching.
              </span>
            </label>

            <label className={`choice-card ${mode === "letter" ? "selected" : ""}`}>
              <input
                type="radio"
                name="mode"
                value="letter"
                checked={mode === "letter"}
                onChange={() => setMode("letter")}
              />
              <span className="choice-symbol">א ב ג</span>
              <span className="choice-title">Letter concordance</span>
              <span className="choice-description">
                Translate character sequences using preferred letter meanings.
              </span>
            </label>
          </div>

          <div className="section-divider" />

          <div className="panel-heading compact-heading">
            <div>
              <p className="step-label">STEP 3</p>
              <h2>Output format</h2>
            </div>
          </div>

          <div className="output-options">
            <label className="output-choice">
              <input
                type="radio"
                name="output"
                checked={outputMode === "preview"}
                onChange={() => setOutputMode("preview")}
              />
              <span>
                <strong>View on page</strong>
                <small>Show manageable results in the workspace.</small>
              </span>
            </label>
            <label className="output-choice">
              <input
                type="radio"
                name="output"
                checked={outputMode === "download"}
                onChange={() => setOutputMode("download")}
              />
              <span>
                <strong>Download as a file</strong>
                <small>Save the translation as a plain text file.</small>
              </span>
            </label>
          </div>

          <p className="size-note">
            Large results are automatically downloaded instead of being
            rendered in full on the page.
          </p>

          {error && <div className="message error-message" role="alert">{error}</div>}
          {status && !error && (
            <div className="message status-message" role="status">{status}</div>
          )}

          <button
            className="primary-button"
            type="button"
            onClick={processText}
            disabled={busy || !inputText.trim()}
          >
            {busy ? (
              <><span className="spinner" /> Processing…</>
            ) : (
              <>Process text <span aria-hidden="true">→</span></>
            )}
          </button>
          <p className="privacy-note">
            Your text is sent to your configured Go translation service.
          </p>
        </div>

        <aside className="panel result-panel">
          <div className="panel-heading">
            <div>
              <p className="step-label">RESULT</p>
              <h2>Translation output</h2>
            </div>
            {result && <span className="result-badge">Ready</span>}
          </div>

          {!result ? (
            <div className="empty-result">
              <div className="empty-icon" aria-hidden="true">文</div>
              <h3>Your translation will appear here</h3>
              <p>
                Add text, choose a translation method and select Process text.
                You can then view or download your result.
              </p>
              <div className="empty-features">
                <span>✓ Plain text output</span>
                <span>✓ Downloadable files</span>
                <span>✓ Large-output protection</span>
              </div>
            </div>
          ) : (
            <div className="result-content">
              <div className="result-meta">
                <span>{formatBytes(resultSize)}</span>
                <span>{result.length.toLocaleString()} characters</span>
              </div>
              <pre className="result-text" dir="auto">
                {result.length > PREVIEW_LENGTH
                  ? `${result.slice(0, PREVIEW_LENGTH)}\n\n… Preview shortened. Download the file to view the complete result.`
                  : result}
              </pre>
              <button
                className="secondary-button"
                type="button"
                onClick={() => {
                  const base = fileName.replace(/\.[^.]+$/, "") || "translation";
                  downloadText(result, `${base}.${mode}.txt`);
                }}
              >
                Download result (.txt)
              </button>
            </div>
          )}

          <div className="result-footer">
            <span className="footer-mark">א</span>
            <span>Dictionary Translator</span>
            <span className="footer-version">Text workspace</span>
          </div>
        </aside>
      </section>

      <footer className="page-footer">
        <span>Dictionary Translator</span>
        <span>Hebrew concordance and text processing</span>
      </footer>
    </main>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}
