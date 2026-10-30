
import { NextRequest } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const MAX_REQUEST_BYTES = 10 * 1024 * 1024;

export async function POST(request: NextRequest) {
  const contentLength = Number(request.headers.get("content-length") ?? 0);

  if (contentLength > MAX_REQUEST_BYTES) {
    return new Response("Input is too large. Maximum size is 10 MiB.", {
      status: 413,
    });
  }

  const body = await request.text();

  if (new TextEncoder().encode(body).byteLength > MAX_REQUEST_BYTES) {
    return new Response("Input is too large. Maximum size is 10 MiB.", {
      status: 413,
    });
  }

  const backendUrl = (
    process.env.BACKEND_API_URL ?? "http://backend:8080"
  ).replace(/\/+$/, "");

  let backendResponse: Response;

  try {
    backendResponse = await fetch(`${backendUrl}/api/v1/process`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
      cache: "no-store",
      signal: AbortSignal.timeout(120_000),
    });
  } catch {
    return new Response(
      "Could not reach the translation service. Check that the Go backend is running.",
      { status: 502 },
    );
  }

  return new Response(backendResponse.body, {
    status: backendResponse.status,
    headers: {
      "Content-Type":
        backendResponse.headers.get("content-type") ??
        "text/plain; charset=utf-8",
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
    },
  });
}
