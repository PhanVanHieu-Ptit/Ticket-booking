# Session Management Module

The **Session Management Module** is responsible for managing the lifecycle, authentication, and verification of both public user sessions and administrative sessions. It serves as the security and identity foundation for enforcing business rules such as ticket purchase limits, active hold restrictions, and admin dashboard security.

---

## 1. Purpose

To provide a secure, stateless, and lightweight mechanism to identify unique users (without requiring registration) and authenticate administrators. This identification is critical to preventing ticket scalping, double-booking, and unauthorized access to real-time sales and revenue metrics.

---

## 2. Responsibilities

- **User Session Generation**: Automatically generate and issue a cryptographically signed JSON Web Token (JWT) in a secure cookie (`session_token`) for anonymous visitors landing on the site.
- **User Session Verification**: Intercept public requests to validate the signature and expiration of the `session_token` cookie, making the session ID available to the application context.
- **Admin Authentication**: Verify the administrator passcode against the hashed passcode stored in the database, and issue an administrative JWT Bearer token upon success.
- **Admin Token Verification**: Validate administrative JWTs on protected `/api/v1/admin/*` endpoints.
- **Session Lifecycle Enforcment**: Enforce session durations:
  - User Session: 30 minutes (`Max-Age=1800` seconds).
  - Admin Session: 2 hours (`Max-Age=7200` seconds).

---

## 3. Dependencies

This is a foundational, cross-cutting module and has **no upward dependencies** on other business modules. It is depended upon by:

- [Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md) (to identify the holding session).
- [Checkout & Payment Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/checkout_payment.md) (to identify the paying session and enforce purchase limits).
- [Administration & Analytics Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/administration_analytics.md) (to authenticate administrators).

---

## 4. APIs

### Public APIs

- **`POST /api/v1/sessions`**: Explicitly initializes a new user session.
  - _Request_: None.
  - _Response_: Sets the `session_token` cookie and returns the session ID and expiration timestamp.

### Admin APIs

- **`POST /api/v1/admin/login`**: Authenticates an administrator.
  - _Request_: `{ "passcode": "string" }`
  - _Response_: Returns an administrative JWT bearer token.

---

## 5. Data Ownership

### PostgreSQL

- **`admin_configs`**: Owns the configuration key `admin_passcode` containing the Bcrypt hash of the admin passcode.

### Redis

- None directly. Session states are stateless and verified cryptographically using the JWT signing secret. (Session IDs are stored inside other modules' Redis keys, e.g., `hold:{session_id}`).
