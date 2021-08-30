package config

import "github.com/go-redis/redis/v8"

var Redis *redis.Client

func CreateRedisClient() {

	redis := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})
	Redis = redis
}
