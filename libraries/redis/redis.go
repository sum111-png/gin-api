package redis

import (
	"fmt"
	"gin-api/libraries/logging"
	"strconv"
	"strings"
	"time"

	"gin-api/libraries/config"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

type RedisDB struct {
	pool   *redis.Pool
	Config *Config
}

var obj map[string]*RedisDB

func GetRedis(redisName string) (*RedisDB, error) {
	var (
		hostCfg      string
		authCfg      string
		portCfg      string
		tmpDbCfg     string
		maxActiveCfg string
		maxIdleCfg   string
		execTimeCfg  string
		execTime     int64
	)

	cfg := config.GetConfigToJson("redis", redisName)

	hostCfg = cfg["host"].(string)
	authCfg = cfg["auth"].(string)
	portCfg = cfg["port"].(string)
	port, _ := strconv.Atoi(portCfg)
	tmpDbCfg = cfg["db"].(string)
	dbCfg, _ := strconv.Atoi(tmpDbCfg)
	maxActiveCfg = cfg["max_active"].(string)
	maxActive, _ := strconv.Atoi(maxActiveCfg)
	maxIdleCfg = cfg["max_idle"].(string)
	maxIdle, _ := strconv.Atoi(maxIdleCfg)
	execTimeCfg = cfg["exec_timeout"].(string)
	execTimeInt, _ := strconv.Atoi(execTimeCfg)
	execTime = int64(execTimeInt)

	db, err := conn(redisName, hostCfg, authCfg, port, dbCfg, maxActive, maxIdle, execTime)
	if err != nil {
		err = errors.Wrap(err, "get redis config error：")
		return nil, err
	}

	return db, nil
}

func conn(conn, host, password string, port, dbNum, maxActive, maxIdle int, execTimeout int64) (db *RedisDB, err error) {
	if len(obj) == 0 {
		obj = make(map[string]*RedisDB)
	}
	if obj[conn] != nil {
		db = obj[conn]
		return
	}

	cfg := &Config{
		Host:        host,
		Port:        port,
		Password:    password,
		DB:          dbNum,
		MaxActive:   maxActive,
		MaxIdle:     maxIdle,
		ExecTimeout: execTimeout,
	}

	db = new(RedisDB)
	db.Config = cfg
	db.pool = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(
				"tcp",
				fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
				redis.DialPassword(cfg.Password),
				redis.DialDatabase(cfg.DB),
				redis.DialConnectTimeout(time.Second*2),
				redis.DialReadTimeout(time.Second*2),
				redis.DialWriteTimeout(time.Second*2),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     cfg.MaxIdle,   // 最大的空闲连接数，表示即使没有redis连接时依然可以保持N个空闲的连接，而不被清除，随时处于待命状态
		MaxActive:   cfg.MaxActive, // 最大的激活连接数，表示同时最多有N个连接 ，为0事表示没有限制
		IdleTimeout: time.Second,   //最大的空闲连接等待时间，超过此时间后，空闲连接将被关闭
		Wait:        true,          // 当链接数达到最大后是否阻塞，如果不的话，达到最大后返回错误
	}

	obj[conn] = db

	return
}

// ConnPool 返回 redis.Pool.
// 除非必要一般不建议用这个函数, 用本库封装好的函数操作数据库.
func (db *RedisDB) ConnPool() *redis.Pool {
	return db.pool
}

// Close 释放连接资源.
func (db *RedisDB) Close() error {
	if db.pool != nil {
		return db.pool.Close()
	}
	return nil
}

// Do 执行 redis 命令
// NOTE 除非有必要(比如在一个函数内部需要执行多次 redis 操作), 否则请用该函数执行所有的操作, 这样能有效避免忘记释放资源.
func (db *RedisDB) Do(c *gin.Context, commandName string, args ...interface{}) (reply interface{}, err error) {
	if commandName == "PING" {
		return
	}
	var (
		conn    = db.pool.Get()
		argsStr []string
	)

	defer conn.Close()

	logCfg := config.GetConfigToJson("log", "log")
	headerLogField := logCfg["header_field"].(string)

	header := logging.GetLogHeader(c)
	header.LogId = c.Writer.Header().Get(headerLogField)
	header.Module = "databus/redis"

	reply, err = conn.Do(commandName, args...)

	for _, arg := range args {
		argsStr = append(argsStr, fmt.Sprint(arg))
	}

	defer func() {
		if err != nil {
			header.Error = fmt.Sprintf("redis do:[%s], error: %s", fmt.Sprint(commandName, " ", strings.Join(argsStr, " ")), err.Error())
			logging.ErrorCtx(c)
		}
	}()

	return
}
