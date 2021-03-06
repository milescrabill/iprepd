package iprepd

import (
	"io/ioutil"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mozilla.org/mozlogrus"
	yaml "gopkg.in/yaml.v2"
)

type serverRuntime struct {
	cfg             serverCfg
	redis           redisLink
	versionResponse []byte
}

type serverCfg struct {
	Listen string
	Redis  struct {
		Addr         string
		Replicas     []string
		ReadTimeout  int
		WriteTimeout int
		DialTimeout  int
	}
	Auth struct {
		DisableAuth bool
		Hawk        map[string]string
		APIKey      map[string]string
	}
	Violations []Violation
	Decay      struct {
		Points   int
		Interval time.Duration
	}
	Exceptions struct {
		File []string
		AWS  bool
	}
	VersionResponse string
}

func (cfg *serverCfg) validate() error {
	if cfg.VersionResponse == "" {
		cfg.VersionResponse = "./version.json"
	}
	if cfg.Redis.ReadTimeout == 0 {
		cfg.Redis.ReadTimeout = 50
	}
	if cfg.Redis.WriteTimeout == 0 {
		cfg.Redis.WriteTimeout = 50
	}
	if cfg.Redis.DialTimeout == 0 {
		cfg.Redis.DialTimeout = 100
	}
	return nil
}

func (cfg *serverCfg) getViolation(v string) *Violation {
	for _, x := range cfg.Violations {
		if x.Name == v {
			return &x
		}
	}
	return nil
}

var sruntime serverRuntime

func init() {
	mozlogrus.Enable("iprepd")
	rand.Seed(time.Now().Unix())
}

func loadCfg(confpath string) (ret serverCfg, err error) {
	buf, err := ioutil.ReadFile(confpath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(buf, &ret)
	if err != nil {
		return
	}
	return ret, ret.validate()
}

// StartDaemon starts a new instance of iprepd using configuration file confpath.
func StartDaemon(confpath string) {
	log.Infof("starting daemon")

	var err error
	sruntime.cfg, err = loadCfg(confpath)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.redis, err = newRedisLink(sruntime.cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.versionResponse, err = ioutil.ReadFile(sruntime.cfg.VersionResponse)
	if err != nil {
		log.Warnf(err.Error())
	}
	go startExceptions()
	err = startAPI()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
