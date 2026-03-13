-- Add unique constraint on provider_reference to ensure webhook idempotency.
-- Stripe (and other payment providers) may deliver webhook events multiple times;
-- this constraint prevents duplicate payment_transaction rows for the same event.
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_transactions_provider_reference
    ON payment_transactions (provider_reference)
    WHERE provider_reference IS NOT NULL;
