package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const priceOracleABI = `[{"inputs":[{"internalType":"uint256","name":"_price","type":"uint256"}],"name":"updatePrice","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"price","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`

var (
	metricUpdatesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_price_updates_total",
		Help: "Total number of price update attempts",
	})
	metricUpdateFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_price_update_failures_total",
		Help: "Total number of price update failures",
	})
	metricExternalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_external_fetch_errors_total",
		Help: "Total number of external price fetch errors",
	})
	metricQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_queue_depth",
		Help: "Number of queued price updates pending retry",
	})
	metricLastPrice = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_last_price",
		Help: "Last fetched price (scaled integer)",
	})
	metricRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "agent_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method", "status"})
)

func main() {
	prometheus.MustRegister(metricUpdatesTotal, metricUpdateFailures, metricExternalErrors, metricQueueDepth, metricLastPrice, metricRequestDuration)

	rpcURL := os.Getenv("RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	oracleAddressHex := os.Getenv("ORACLE_ADDRESS")
	chainIDStr := os.Getenv("CHAIN_ID")
	priceAPIURL := os.Getenv("PRICE_API_URL")
	priceScale := getEnvInt64("PRICE_SCALE", 100)
	minInterval := getEnvDuration("MIN_UPDATE_INTERVAL", 15*time.Second)
	changeThreshold := getEnvFloat("PRICE_CHANGE_THRESHOLD", 0.01)
	rateLimitRPS := getEnvFloat("RATE_LIMIT_RPS", 0.2)
	queueFile := getEnv("QUEUE_FILE", "/tmp/agent-queue.jsonl")
	retryMax := getEnvInt("RETRY_MAX", 5)
	retryBase := getEnvDuration("RETRY_BASE_DELAY", 2*time.Second)
	port := getEnv("PORT", "8081")

	if rpcURL == "" || privateKeyHex == "" || oracleAddressHex == "" {
		log.Fatal("missing required environment variables")
	}

	chainID, _ := strconv.ParseInt(chainIDStr, 10, 64)

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:])
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	log.Printf("agent from address: %s", fromAddress.Hex())

	oracleAddress := common.HexToAddress(oracleAddressHex)

	parsedABI, err := abi.JSON(strings.NewReader(priceOracleABI))
	if err != nil {
		log.Fatal(err)
	}

	contract := bind.NewBoundContract(oracleAddress, parsedABI, client, client, client)
	state := &agentState{queueFile: queueFile}

	if err := state.loadQueue(); err != nil {
		log.Printf("queue load error: %v", err)
	}

	go func() {
		if priceAPIURL == "" {
			return
		}
		interval := time.Duration(float64(time.Second) / rateLimitRPS)
		if rateLimitRPS <= 0 {
			interval = 10 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			price, err := fetchExternalPrice(context.Background(), priceAPIURL)
			if err != nil {
				metricExternalErrors.Inc()
				log.Printf("price fetch error: %v", err)
				continue
			}
			scaled := int64(price * float64(priceScale))
			metricLastPrice.Set(float64(scaled))

			if !state.shouldSubmit(scaled, minInterval, changeThreshold) {
				continue
			}

			err = submitWithRetry(contract, privateKey, chainID, scaled, retryMax, retryBase)
			if err != nil {
				metricUpdateFailures.Inc()
				log.Printf("submit error: %v", err)
				_ = state.enqueue(scaled)
				continue
			}
		}
	}()

	go func() {
		for {
			if err := state.drainQueue(func(price int64) error {
				return submitWithRetry(contract, privateKey, chainID, price, retryMax, retryBase)
			}); err != nil {
				log.Printf("queue drain error: %v", err)
			}
			time.Sleep(30 * time.Second)
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)

	http.HandleFunc("/admin/trigger", withMetrics(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		priceStr := r.URL.Query().Get("price")
		priceInt, err := strconv.Atoi(priceStr)
		if err != nil || priceInt <= 0 {
			http.Error(w, "invalid price", http.StatusBadRequest)
			return
		}

		metricUpdatesTotal.Inc()
		err = submitWithRetry(contract, privateKey, chainID, int64(priceInt), retryMax, retryBase)
		if err != nil {
			metricUpdateFailures.Inc()
			_ = state.enqueue(int64(priceInt))
			http.Error(w, err.Error(), 500)
			return
		}

		resp := map[string]interface{}{
			"status":    "submitted",
			"new_price": priceInt,
		}
		json.NewEncoder(w).Encode(resp)
	}))

	log.Printf("agent listening on :%s", port)

	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
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

func getEnvFloat(key string, def float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	v, err := strconv.ParseFloat(value, 64)
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

type agentState struct {
	mu            sync.Mutex
	lastPrice     int64
	lastUpdate    time.Time
	lastSubmitKey string
	queueFile     string
	queued        []queuedPrice
}

type queuedPrice struct {
	Price int64 `json:"price"`
	TS    int64 `json:"ts"`
}

func (s *agentState) shouldSubmit(price int64, minInterval time.Duration, changeThreshold float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastUpdate.IsZero() {
		s.lastPrice = price
		s.lastUpdate = time.Now()
		s.lastSubmitKey = fmtKey(price)
		return true
	}

	if time.Since(s.lastUpdate) < minInterval {
		return false
	}

	change := 0.0
	if s.lastPrice > 0 {
		change = float64(absInt64(price-s.lastPrice)) / float64(s.lastPrice)
	}
	if change < changeThreshold {
		return false
	}

	key := fmtKey(price)
	if key == s.lastSubmitKey {
		return false
	}

	s.lastPrice = price
	s.lastUpdate = time.Now()
	s.lastSubmitKey = key
	return true
}

func (s *agentState) loadQueue() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.queueFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		var q queuedPrice
		if err := json.Unmarshal(line, &q); err == nil {
			s.queued = append(s.queued, q)
		}
	}
	metricQueueDepth.Set(float64(len(s.queued)))
	return scanner.Err()
}

func (s *agentState) enqueue(price int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	q := queuedPrice{Price: price, TS: time.Now().Unix()}
	s.queued = append(s.queued, q)
	metricQueueDepth.Set(float64(len(s.queued)))

	if err := os.MkdirAll(filepath.Dir(s.queueFile), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(s.queueFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	return enc.Encode(q)
}

func (s *agentState) drainQueue(submit func(price int64) error) error {
	s.mu.Lock()
	queued := append([]queuedPrice{}, s.queued...)
	s.mu.Unlock()

	if len(queued) == 0 {
		return nil
	}

	var remaining []queuedPrice
	for _, item := range queued {
		if err := submit(item.Price); err != nil {
			remaining = append(remaining, item)
		}
	}

	s.mu.Lock()
	s.queued = remaining
	metricQueueDepth.Set(float64(len(s.queued)))
	s.mu.Unlock()

	return s.writeQueueFile(remaining)
}

func (s *agentState) writeQueueFile(items []queuedPrice) error {
	if err := os.MkdirAll(filepath.Dir(s.queueFile), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(s.queueFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	enc := json.NewEncoder(writer)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func submitWithRetry(contract *bind.BoundContract, privateKey *ecdsa.PrivateKey, chainID int64, price int64, maxRetries int, baseDelay time.Duration) error {
	priceBI := big.NewInt(price)
	for attempt := 0; attempt <= maxRetries; attempt++ {
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(chainID))
		if err != nil {
			return err
		}
		auth.Context = context.Background()

		_, err = contract.Transact(auth, "updatePrice", priceBI)
		if err == nil {
			return nil
		}
		if attempt == maxRetries {
			return err
		}
		backoff := baseDelay * time.Duration(1<<attempt)
		time.Sleep(backoff)
	}
	return nil
}

func fetchExternalPrice(ctx context.Context, url string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, errors.New("non-200 response")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, err
	}
	if v, ok := payload["price"]; ok {
		return toFloat(v)
	}
	if data, ok := payload["data"].(map[string]any); ok {
		if v, ok := data["price"]; ok {
			return toFloat(v)
		}
		if v, ok := data["priceUsd"]; ok {
			return toFloat(v)
		}
		if v, ok := data["amount"]; ok {
			return toFloat(v)
		}
	}
	if eth, ok := payload["ethereum"].(map[string]any); ok {
		if v, ok := eth["usd"]; ok {
			return toFloat(v)
		}
	}

	return 0, errors.New("price not found in response")
}

func toFloat(v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case string:
		return strconv.ParseFloat(t, 64)
	default:
		return 0, errors.New("unsupported price type")
	}
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func fmtKey(price int64) string {
	return strconv.FormatInt(price, 10)
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