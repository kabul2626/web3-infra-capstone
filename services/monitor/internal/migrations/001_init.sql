CREATE TABLE IF NOT EXISTS prices (
  id BIGSERIAL PRIMARY KEY,
  base TEXT NOT NULL,
  quote TEXT NOT NULL,
  price NUMERIC NOT NULL,
  ts TIMESTAMPTZ NOT NULL,
  tx_hash TEXT NOT NULL,
  block_num BIGINT NOT NULL,
  log_index INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_prices_base_quote_ts ON prices (base, quote, ts DESC);
CREATE INDEX IF NOT EXISTS idx_prices_block_num ON prices (block_num DESC);
