# Task: Session Management

## Task ID

`TS-02`

## Goal

Implement the security and identity foundation of the system. This includes the user session middleware that automatically generates and validates a cryptographically signed JWT `session_token` cookie for anonymous visitors, and the admin authentication flow that verifies the passcode and issues an admin JWT bearer token.

## Files Expected to Change

- [NEW] `internal/session/jwt.go` (helper functions to sign and verify user and admin JWTs)
- [NEW] `internal/middleware/session.go` (middleware to intercept public requests, issue/verify the `session_token` cookie, and inject the session ID into the request context)
- [NEW] `internal/middleware/admin_auth.go` (middleware to verify the `Authorization: Bearer <admin_token>` header on admin routes)
- [NEW] `internal/handlers/session_handler.go` (explicit session initialization endpoint `POST /api/v1/sessions`)
- [NEW] `internal/handlers/admin_handler.go` (admin login endpoint `POST /api/v1/admin/login` that verifies the passcode against `admin_configs` and returns a JWT)
- [MODIFY] `main.go` (wire up the middlewares and register the session and admin login routes)

## Dependencies

- `TS-01` (Requires database setup to query the hashed admin passcode from `admin_configs`)

## Acceptance Criteria

1. **User Session Auto-Generation**: Any HTTP request to the public API (`/api/v1/*`) that does not contain a valid `session_token` cookie must automatically receive a `Set-Cookie` header in the response containing a signed JWT.
2. **User Session Attributes**: The `session_token` cookie must be configured with `HttpOnly; Secure; SameSite=Strict; Max-Age=1800` (30 minutes).
3. **User Session Context**: The middleware must successfully parse valid `session_token` cookies, extract the unique `session_id`, and make it available in the request context for downstream handlers.
4. **Admin Login**: `POST /api/v1/admin/login` with the correct passcode in the request body must return a signed admin JWT with a 2-hour lifetime (`Max-Age=7200`). An incorrect passcode must return `401 Unauthorized` with the error code `ADMIN_UNAUTHORIZED`.
5. **Admin Route Protection**: Any request to `/api/v1/admin/*` (except `/login`) without a valid admin JWT in the `Authorization` header must be rejected with `401 Unauthorized`.

## Estimated Complexity

Medium (3 - 4 hours)
