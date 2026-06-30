# Task: Checkout & Payment - Frontend

## Task ID

`TS-10`

## Goal

Implement the payment checkout user interface on the `/checkout` page and the post-purchase Confirmation Page. The Checkout Page must present a form, handle loading states, and send the request with an idempotency key. The Confirmation Page must display the purchase details and a unique reference code.

## Files Expected to Change

- [NEW] `frontend/src/pages/ConfirmationPage.tsx` (the post-purchase Confirmation Page)
- [MODIFY] `frontend/src/pages/CheckoutPage.tsx` (implement the checkout form, payment simulation inputs, and submit action)
- [MODIFY] `frontend/src/App.tsx` (add routing for `/confirmation`)

## Dependencies

- `TS-06` (Requires the Checkout Page shell and countdown timer)
- `TS-09` (Requires the backend checkout API and idempotency middleware)

## Acceptance Criteria

1. **Checkout Form**: The Checkout Page displays a form with fields:
   - Email Address (validated for format).
   - Cardholder Name (non-empty).
   - Card Number, Expiry, and CVV (mock validation).
   - A toggle or dropdown to select "Simulate Success" or "Simulate Failure" for testing.
2. **Submit Handling**: Clicking the "Pay Now" button:
   - Disables all form inputs and the button.
   - Displays a loading spinner.
   - Generates a unique UUIDv4 `Idempotency-Key` and sends it in the headers of the `POST /api/v1/payments/checkout` request.
3. **Success Scenario**: If the payment succeeds, the user is redirected to `/confirmation`, which displays:
   - A success message.
   - The ticket category and price paid.
   - The unique Ticket Reference Code.
   - A button to return to the Home Page.
4. **Failure Scenario**: If the payment fails (e.g., `402 Payment Required` from the server):
   - The loading state is cleared and the form inputs/button are re-enabled.
   - An error message is displayed (e.g., _"Payment failed. Please check your card details and try again."_).
   - The countdown timer continues ticking down without interruption.
5. **Expired Hold Scenario**: If the payment is submitted but fails due to hold expiration (`410 Gone`), the page displays the expiration modal and redirects the user back to the Home Page.

## Estimated Complexity

Medium (3 - 4 hours)
