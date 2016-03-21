package main

import (
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "os/signal"
  "time"

  "github.com/garyburd/redigo/redis"
)

/*
 * Environment variables:
 * - NODE_DATACENTER - datacenter for metric keys
 * - REQUEST_ADDRESS - address of the werver to hit - ie. http://go-test-web-server.service.consul:8000/
 * - REDIS_ADDRESS - address of the Redis server - ie. redis.service.consul:6379
 */

var NODE_NAME = "-node-client-count"
var requestAddress = "http://localhost:8001/" // overried with REQUEST_ADDRESS

func main() {
  redisConn := getRedisConnection()
  if redisConn != nil {
    redisConn.Do("INCR", getNodeMetricKey())
    redisConn.Close()
  }

  makeShutdownChannel()

  for {
    makeRequest()
    time.Sleep(1000 * time.Millisecond)
  }
}

func makeRequest() {
  url := os.Getenv("REQUEST_ADDRESS")
  if url == "" {
    url = requestAddress
  }

  resp, err := http.Get(url)
  if err != nil {
    fmt.Println(err)
    return
  }

  defer resp.Body.Close()
  body, _ := ioutil.ReadAll(resp.Body)
  fmt.Println(string(body))
}

func getNodeMetricKey() string {
  return getDatacenterKey() + NODE_NAME
}

func getDatacenterKey() string {
  key := os.Getenv("NODE_DATACENTER")
  if key == "" {
    key = "default"
  }
  return key
}

func makeShutdownChannel() {
  sigch := make(chan os.Signal, 1)
  signal.Notify(sigch, os.Interrupt)

  go func() {
    <-sigch
    redisConn := getRedisConnection()
    if redisConn != nil {
      redisConn.Do("DECR", getNodeMetricKey())
    }
    redisConn.Close()
    fmt.Println("Exiting...")
    os.Exit(0)
  }()
}

func getRedisConnection() redis.Conn {
  redisAddr := os.Getenv("REDIS_ADDRESS")
  if redisAddr == "" {
    redisAddr = "localhost:6379"
  }

  redisConn, err := redis.Dial("tcp", redisAddr)
  if err != nil {
    log.Fatalf("error connecting to redis: %v", err)
    return nil
  }

  return redisConn
}
