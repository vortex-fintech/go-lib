CREATE TABLE IF NOT EXISTS idempotency_keys (
    principal TEXT NOT NULL,
    grpc_method TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('IN_PROGRESS', 'SUCCEEDED', 'FAILED_RETRYABLE', 'FAILED_FINAL')),
    response_code INTEGER NOT NULL DEFAULT 0,
    response_payload BYTEA,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT idempotency_keys_pkey PRIMARY KEY (principal, grpc_method, idempotency_key),
    CONSTRAINT idempotency_keys_expiry_chk CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_terminal
    ON idempotency_keys (expires_at)
    WHERE status IN ('SUCCEEDED', 'FAILED_RETRYABLE', 'FAILED_FINAL');
