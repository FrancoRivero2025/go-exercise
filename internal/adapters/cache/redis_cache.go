package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	lastValues map[domain.Pair]domain.LTP
}

func NewRedisCache(addr, password string, db int, ttl time.Duration) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisCache{
		client:     rdb,
		ttl:        ttl,
		lastValues: make(map[domain.Pair]domain.LTP),
	}
}

func (r *RedisCache) Get(pair domain.Pair) (ltp domain.LTP, found bool) {
	defer func() {
		if rec := recover(); rec != nil {
			log.GetInstance().Debug("Recovered from panic in Get: %v", rec)
			ltp = domain.LTP{}
			found = false
		}
	}()

	ctx := context.Background()
	val, err := r.client.Get(ctx, string(pair)).Result()
	if err == redis.Nil {
		return domain.LTP{}, false
	}
	if err != nil {
		panic("Redis get error: " + err.Error())
	}

	if err := json.Unmarshal([]byte(val), &ltp); err != nil {
		panic("JSON unmarshal error: " + err.Error())
	}
	return ltp, true
}

func (r *RedisCache) Set(pair domain.Pair, ltp domain.LTP) {
	defer func() {
		if rec := recover(); rec != nil {
			log.GetInstance().Debug("Recovered from panic in Set: %v", rec)
			r.lastValues[pair] = ltp
			log.GetInstance().Debug("Stored last value for %s as fallback", pair)
		}
	}()

	ctx := context.Background()
	if ltp.Timestamp.IsZero() {
		ltp.Timestamp = time.Now()
	}
	data, err := json.Marshal(ltp)
	if err != nil {
		panic("JSON marshal error: " + err.Error())
	}

	if err := r.client.Set(ctx, string(pair), data, r.ttl).Err(); err != nil {
		panic("Redis set error: " + err.Error())
	}

	r.lastValues[pair] = ltp
}
