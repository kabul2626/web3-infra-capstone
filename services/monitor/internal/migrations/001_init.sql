-- Prices table stores all ingested price updates from blockchain events
CREATE TABLE IF NOT EXISTS prices (
  id BIGSERIAL PRIMARY KEY,
  base TEXT NOT NULL,              -- Base asset symbol (e.g., ETH)
  quote TEXT NOT NULL,             -- Quote asset symbol (e.g., USD)
  price NUMERIC NOT NULL,          -- Price value (stored as string for precision)
  ts TIMESTAMPTZ NOT NULL,         -- Event timestamp from smart contract
  tx_hash TEXT NOT NULL,           -- Transaction hash for traceability
  block_num BIGINT NOT NULL,       -- Block number for ordering
  log_index INTEGER NOT NULL,      -- Log index within block for uniqueness
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- Ingestion time
  UNIQUE (tx_hash, log_index)      -- Prevent duplicate ingestion of same event
);

-- Index for querying latest prices by symbol pair
CREATE INDEX IF NOT EXISTS idx_prices_base_quote_ts ON prices (base, quote, ts DESC);
-- Index for block-based queries and reorg detection
CREATE INDEX IF NOT EXISTS idx_prices_block_num ON prices (block_num DESC);
