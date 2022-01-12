package resource

import (
	"github.com/go-redis/redis/v8"

	"github.com/why444216978/gin-api/library/cache"
	"github.com/why444216978/gin-api/library/config"
	"github.com/why444216978/gin-api/library/etcd"
	"github.com/why444216978/gin-api/library/lock"
	"github.com/why444216978/gin-api/library/logging"
	"github.com/why444216978/gin-api/library/orm"
	httpClient "github.com/why444216978/gin-api/library/rpc/http/client"
)

var (
	Config        *config.Viper
	TestDB        *orm.Orm
	ServiceLogger *logging.Logger
	RedisDefault  *redis.Client
	Etcd          *etcd.Etcd
	ClientHTTP    *httpClient.RPC
	RedisLock     lock.Locker
	RedisCache    cache.Cacher
)
