import { apiUrl } from "./apiBase";

interface SessionResponse {
  success: boolean;
  data: { session_token: string };
}

let cachedToken: string | null = null;
let pendingFetch: Promise<string> | null = null;

// Fetches (and caches for the lifetime of the tab) the raw signed session
// token so it can travel explicitly as the X-Session-Token header on the
// reserve -> hold -> checkout flow. The session_token cookie alone isn't
// reliable: private/incognito tabs block it outright since the frontend
// (Vercel) and backend (Render) are different origins, which otherwise
// causes each request to land on a different auto-generated session and
// makes /tickets/hold 404 right after a successful reserve.
export async function ensureSessionToken(): Promise<string> {
  if (cachedToken) return cachedToken;
  if (!pendingFetch) {
    pendingFetch = fetch(apiUrl("/api/v1/sessions"), {
      method: "POST",
      credentials: "include",
    })
      .then((response) => {
        if (!response.ok) throw new Error("Failed to initialize session");
        return response.json();
      })
      .then((result: SessionResponse) => {
        cachedToken = result.data.session_token;
        return cachedToken;
      })
      .finally(() => {
        pendingFetch = null;
      });
  }
  return pendingFetch;
}
