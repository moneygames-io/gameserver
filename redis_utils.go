package main

import (
	"github.com/go-redis/redis"
	"github.com/op/go-logging"
	"time"
)

func connectToRedis(addr string, log *logging.Logger) *redis.Client {
	var client *redis.Client
	for {
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
		})
		_, err := client.Ping().Result()
		if err != nil {
			log.Error("Failed to connect to redis: %v", err)
		} else {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Info("Connected to %v", addr)
	return client
}
