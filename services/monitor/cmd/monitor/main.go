// Package main implements the price monitor service.
// It subscribes to PriceUpdated events from the smart contract,
// stores them in PostgreSQL, and exposes historical price queries via REST API.
// Features include reorg handling, confirmation counting, and real-time ingestion metrics.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const priceOracleABI = `[{"anonymous":false,"inputs":[{"indexed":false,"internalType":"uint256","name":"price","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"ts","type":"uint256"}],"name":"PriceUpdated","type":"event"}]`

var (
	metricIngested = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "monitor_prices_ingested_total",
		Help: "Total number of prices ingested",
	})
	metricLagSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "monitor_ingest_lag_seconds",
		Help: "Lag in seconds between latest block time and last processed event",
	})
	metricLastEventUnix = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "monitor_last_event_unix",
		Help: "Unix time of last ingested event",
	})
	metricRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "monitor_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method", "status"})
)

func main() {
	prometheus.MustRegister(metricIngested, metricLagSeconds, metricLastEventUnix, metricRequestDuration)

	rpcURL := mustEnv("RPC_URL")
	oracleAddress := common.HexToAddress(mustEnv("ORACLE_ADDRESS"))
	dbURL := mustEnv("DB_URL")
	base := getEnv("BASE_SYMBOL", "ETH")
	quote := getEnv("QUOTE_SYMBOL", "USD")
	confirmations := getEnvInt("CONFIRMATIONS", 3)
	reorgDepth := getEnvInt64("REORG_DEPTH", 5)
	pollInterval := getEnvDuration("POLL_INTERVAL", 10*time.Second)
	port := getEnv("PORT", "8082")
	startBlock := getEnvInt64("START_BLOCK", 0)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := ensureSchema(ctx, pool); err != nil {
		log.Fatal(err)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(priceOracleABI))
	if err != nil {
		log.Fatal(err)
	}
	eventSig := parsedABI.Events["PriceUpdated"].ID

	var lastProcessedBlock atomic.Int64
	if startBlock > 0 {
		lastProcessedBlock.Store(startBlock)
	} else {
		last, err := lastBlockFromDB(ctx, pool)
		if err != nil {
			log.Fatal(err)
		}
		lastProcessedBlock.Store(last)
	}

	go func() {
		for {
			err := indexOnce(ctx, client, pool, oracleAddress, eventSig, parsedABI, base, quote, confirmations, reorgDepth, &lastProcessedBlock)
			if err != nil {
				log.Printf("index error: %v", err)
			}
			time.Sleep(pollInterval)
		}
	}()

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	r.Get("/prices/latest", withMetrics(func(w http.ResponseWriter, r *http.Request) {
		price, err := latestPrice(ctx, pool, r.URL.Query().Get("base"), r.URL.Query().Get("quote"))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, price)
	}))

	r.Get("/prices", withMetrics(func(w http.ResponseWriter, r *http.Request) {
		limit := getQueryInt(r, "limit", 100)
		baseQ := r.URL.Query().Get("base")
		quoteQ := r.URL.Query().Get("quote")

		prices, err := listPrices(ctx, pool, baseQ, quoteQ, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, prices)
	}))

	log.Printf("monitor listening on :%s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, r))
}

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS prices (
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
CREATE INDEX IF NOT EXISTS idx_prices_block_num ON prices (block_num DESC);`)
	return err
}

func lastBlockFromDB(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var last sql.NullInt64
	err := pool.QueryRow(ctx, "SELECT COALESCE(MAX(block_num), 0) FROM prices").Scan(&last)
	if err != nil {
		return 0, err
	}
	if last.Valid {
		return last.Int64, nil
	}
	return 0, nil
}

func indexOnce(ctx context.Context, client *ethclient.Client, pool *pgxpool.Pool, oracle common.Address, eventSig common.Hash, parsedABI abi.ABI, base, quote string, confirmations int, reorgDepth int64, lastProcessed *atomic.Int64) error {
	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return err
	}
	var toBlock int64 = int64(latestBlock) - int64(confirmations)
	if toBlock < 0 {
		return nil
	}
	fromBlock := lastProcessed.Load()
	if fromBlock == 0 {
		fromBlock = toBlock
	}
	if reorgDepth > 0 && fromBlock > reorgDepth {
		fromBlock = fromBlock - reorgDepth
		_, _ = pool.Exec(ctx, "DELETE FROM prices WHERE block_num > $1", fromBlock)
	}
	if toBlock <= fromBlock {
		return nil
	}

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock + 1),
		ToBlock:   big.NewInt(toBlock),
		Addresses: []common.Address{oracle},
		Topics:    [][]common.Hash{{eventSig}},
	}

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	for _, vLog := range logs {
		var event struct {
			Price *big.Int
			Ts    *big.Int
		}
		err := parsedABI.UnpackIntoInterface(&event, "PriceUpdated", vLog.Data)
		if err != nil {
			log.Printf("unpack error: %v", err)
			continue
		}

		_, err = pool.Exec(ctx, `INSERT INTO prices (base, quote, price, ts, tx_hash, block_num, log_index)
			VALUES ($1, $2, $3, to_timestamp($4), $5, $6, $7)
			ON CONFLICT (tx_hash, log_index) DO NOTHING`,
			base, quote, event.Price.String(), event.Ts.Int64(), vLog.TxHash.Hex(), vLog.BlockNumber, vLog.Index)
		if err != nil {
			log.Printf("db insert error: %v", err)
			continue
		}
		metricIngested.Inc()
		metricLastEventUnix.Set(float64(event.Ts.Int64()))
	}

	lastProcessed.Store(toBlock)
	block, err := client.BlockByNumber(ctx, big.NewInt(toBlock))
	if err == nil {
		metricLagSeconds.Set(float64(time.Now().Unix() - int64(block.Time())))
	}

	return nil
}

func latestPrice(ctx context.Context, pool *pgxpool.Pool, base, quote string) (map[string]any, error) {
	query := `SELECT base, quote, price, ts, tx_hash, block_num FROM prices`
	args := []any{}
	if base != "" && quote != "" {
		query += " WHERE base=$1 AND quote=$2"
		args = append(args, base, quote)
	}
	query += " ORDER BY ts DESC LIMIT 1"

	row := pool.QueryRow(ctx, query, args...)
	var b, q, price, tx string
	var ts time.Time
	var block int64
	if err := row.Scan(&b, &q, &price, &ts, &tx, &block); err != nil {
		return nil, err
	}
	return map[string]any{
		"base":      b,
		"quote":     q,
		"price":     price,
		"timestamp": ts,
		"tx_hash":   tx,
		"block":     block,
	}, nil
}

func listPrices(ctx context.Context, pool *pgxpool.Pool, base, quote string, limit int) ([]map[string]any, error) {
	query := `SELECT base, quote, price, ts, tx_hash, block_num FROM prices`
	args := []any{}
	if base != "" && quote != "" {
		query += " WHERE base=$1 AND quote=$2"
		args = append(args, base, quote)
	}
	query += " ORDER BY ts DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var b, q, price, tx string
		var ts time.Time
		var block int64
		if err := rows.Scan(&b, &q, &price, &ts, &tx, &block); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"base":      b,
			"quote":     q,
			"price":     price,
			"timestamp": ts,
			"tx_hash":   tx,
			"block":     block,
		})
	}
	return result, nil
}

func withMetrics(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next(rw, r)
		dur := time.Since(start).Seconds()
		metricRequestDuration.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(rw.status)).Observe(dur)
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing %s", key)
	}
	return value
}

func getEnv(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}

func getEnvInt(key string, def int) int {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return v
}

func getEnvInt64(key string, def int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	v, err := time.ParseDuration(value)
	if err != nil {
		return def
	}
	return v
}

func getQueryInt(r *http.Request, key string, def int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return def
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return v
}
