CREATE TABLE wallets (
    id                   UUID PRIMARY KEY,
    player_id            UUID NOT NULL,
    currency             CHAR(3) NOT NULL,
    balance_minor_units  BIGINT NOT NULL CHECK (balance_minor_units >= 0),
    version              BIGINT NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wallets_player_id_currency_key UNIQUE (player_id, currency)
);

CREATE TABLE wager_transactions (
    id                                 UUID PRIMARY KEY,
    kind                               TEXT NOT NULL
        CHECK (kind IN ('OPENING','BET','WIN','LOSS','REFUND','ROLLBACK')),
    status                             TEXT NOT NULL
        CHECK (status IN ('PENDING','PENDING_REFERENCE','PROCESSED','REJECTED','FAILED')),
    wallet_id                          UUID NOT NULL REFERENCES wallets(id),
    player_id                          UUID NOT NULL,
    amount_minor_units                 BIGINT NOT NULL,
    currency                           CHAR(3) NOT NULL,
    provider_id                        TEXT,
    external_transaction_id            TEXT,
    idempotency_key                    TEXT,
    payload_hash                       TEXT,
    round_id                           TEXT,
    game_id                            TEXT,
    reference_external_transaction_id  TEXT,
    resolved_reference_id              UUID REFERENCES wager_transactions(id),
    failure_code                       TEXT,
    created_at                         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT wager_transactions_origin_check CHECK (
        (kind = 'OPENING'
            AND provider_id IS NULL AND external_transaction_id IS NULL
            AND idempotency_key IS NULL AND payload_hash IS NULL
            AND round_id IS NULL AND game_id IS NULL)
        OR
        (kind <> 'OPENING'
            AND provider_id IS NOT NULL AND external_transaction_id IS NOT NULL
            AND idempotency_key IS NOT NULL AND payload_hash IS NOT NULL
            AND round_id IS NOT NULL AND game_id IS NOT NULL)
    ),

    CONSTRAINT wager_transactions_reference_check CHECK (
        (kind IN ('REFUND','ROLLBACK') AND reference_external_transaction_id IS NOT NULL)
        OR
        (kind NOT IN ('REFUND','ROLLBACK') AND reference_external_transaction_id IS NULL)
    )
);

CREATE UNIQUE INDEX wager_transactions_provider_external_id_key
    ON wager_transactions (provider_id, external_transaction_id)
    WHERE provider_id IS NOT NULL;

CREATE UNIQUE INDEX wager_transactions_one_opening_per_wallet
    ON wager_transactions (wallet_id)
    WHERE kind = 'OPENING';

CREATE TABLE wallet_ledger_entries (
    id                          UUID PRIMARY KEY,
    wallet_id                   UUID NOT NULL REFERENCES wallets(id),
    transaction_id              UUID NOT NULL REFERENCES wager_transactions(id),
    direction                   TEXT NOT NULL CHECK (direction IN ('DEBIT','CREDIT')),
    amount_minor_units          BIGINT NOT NULL CHECK (amount_minor_units > 0),
    currency                    CHAR(3) NOT NULL,
    balance_before_minor_units  BIGINT NOT NULL,
    balance_after_minor_units   BIGINT NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wallet_ledger_entries_wallet_tx_key UNIQUE (wallet_id, transaction_id)
);

CREATE FUNCTION forbid_ledger_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'wallet_ledger_entries is append-only: % not allowed', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER wallet_ledger_entries_no_update
    BEFORE UPDATE ON wallet_ledger_entries
    FOR EACH ROW EXECUTE FUNCTION forbid_ledger_mutation();

CREATE TRIGGER wallet_ledger_entries_no_delete
    BEFORE DELETE ON wallet_ledger_entries
    FOR EACH ROW EXECUTE FUNCTION forbid_ledger_mutation();

CREATE TABLE outbox_entries (
    event_id         UUID PRIMARY KEY,
    aggregate_id     TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    payload          JSONB NOT NULL,
    occurred_at      TIMESTAMPTZ NOT NULL,
    attempts         INT NOT NULL DEFAULT 0,
    next_attempt_at  TIMESTAMPTZ NOT NULL,
    published_at     TIMESTAMPTZ
);

CREATE INDEX outbox_entries_pending_idx
    ON outbox_entries (next_attempt_at)
    WHERE published_at IS NULL;