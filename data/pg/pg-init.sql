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

CREATE UNIQUE INDEX ledger_bluescore_payee_idx ON ledger(bluescore int8_ops,payee text_ops);

-- Coinbase -------------------------------------------------------

CREATE TABLE coinbase_payments (
    tx text PRIMARY KEY UNIQUE,
    wallet text NOT NULL,
    amount bigint NOT NULL,
    daascore bigint NOT NULL
);

-- Indices -------------------------------------------------------

CREATE UNIQUE INDEX coinbase_payments_tx_key ON coinbase_payments(tx text_ops);

-- Blocks -------------------------------------------------------
CREATE TYPE block_status AS ENUM ('unconfirmed', 'confirmed', 'paid', 'error');

CREATE TABLE blocks (
    hash text PRIMARY KEY,
    timestamp timestamp without time zone NOT NULL,
    miner text NOT NULL,
    payee text NOT NULL,
    round_time interval NOT NULL,
    daascore bigint NOT NULL,
    bluescore bigint NOT NULL,
    luck double precision NOT NULL,
    block_json jsonb NOT NULL,
    status block_status DEFAULT 'unconfirmed'::block_status,
    coinbase_reward text REFERENCES coinbase_payments(tx) UNIQUE,
    pool text
);