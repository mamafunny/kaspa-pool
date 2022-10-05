-- Shares -------------------------------------------------------

CREATE TABLE IF NOT EXISTS shares (
    id SERIAL PRIMARY KEY,
    wallet text NOT NULL,
    bluescore bigint NOT NULL,
    timestamp timestamp without time zone NOT NULL,
    nonce bigint NOT NULL,
    diff integer NOT NULL
);

CREATE UNIQUE INDEX shares_bluescore_nonce_idx ON shares(bluescore int8_ops,nonce int8_ops);
CREATE INDEX shares_wallet_idx ON shares(wallet text_ops);

-- Blocks -------------------------------------------------------

CREATE TABLE blocks (
    hash text PRIMARY KEY,
    timestamp timestamp without time zone NOT NULL,
    miner text NOT NULL,
    payee text NOT NULL,
    block_json jsonb NOT NULL,
    bluescore bigint NOT NULL
);

-- Ledger -------------------------------------------------------

CREATE TYPE ledger_entry_status AS ENUM ('owed', 'pending', 'submitted', 'confirmed', 'error');

CREATE TABLE ledger (
    id SERIAL PRIMARY KEY,
    payee text NOT NULL,
    amount bigint NOT NULL,
    bluescore bigint NOT NULL,
    status ledger_entry_status DEFAULT 'owed'::ledger_entry_status,
    updated timestamp without time zone NOT NULL,
    tx_id text UNIQUE
);

CREATE UNIQUE INDEX ledger_tx_id_key ON ledger(tx_id text_ops);
CREATE UNIQUE INDEX ledger_bluescore_payee_idx ON ledger(bluescore int8_ops,payee text_ops);
