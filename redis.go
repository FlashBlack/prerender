package main

import (
	"encoding/json"
	"github.com/go-redis/redis/v7"
	"time"
)

type RedisClient struct {
	c *redis.Client
}

func NewRedisClient() *RedisClient {
	c := redis.NewClient(&redis.Options{
		Addr:     "192.168.1.16:6379",
		Password: "",
		DB:       0,
	})

	if err := c.Ping().Err(); err != nil {
		panic("Unable to connect to redis " + err.Error())
	}

	return &RedisClient{c}
}

func (client *RedisClient) GetKey(key string, src interface{}) error {
	val, err := client.c.Get(key).Result()
	if err == redis.Nil || err != nil {
		return err
	}

	err = json.Unmarshal([]byte(val), &src)
	if err != nil {
		return err
	}

	return nil
}

func (client *RedisClient) SetKey(key string, value interface{}, expiration time.Duration) error {
	cacheEntry, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = client.c.Set(key, cacheEntry, expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func (client *RedisClient) Close() {
	_ = client.c.Close()
}
