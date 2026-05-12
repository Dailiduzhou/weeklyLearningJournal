package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/go-kratos/gateway/client"
	configLoader "github.com/go-kratos/gateway/config"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"

	_ "github.com/go-kratos/gateway/middleware/cors"
	_ "github.com/go-kratos/gateway/middleware/logging"
	_ "github.com/go-kratos/gateway/middleware/rewrite"
	_ "github.com/go-kratos/gateway/middleware/tracing"
	_ "go.uber.org/automaxprocs"
)

var (
	flagConf    string
	flagAddr    string
	flagEtcd    string
	flagName    string
	flagVersion string
)

func init() {
	flag.StringVar(&flagConf, "conf", "../../configs", "config path, eg: -conf config.yaml")
	flag.StringVar(&flagAddr, "addr", ":8080", "gateway listen address, eg: -addr :8080")
	flag.StringVar(&flagEtcd, "etcd", "127.0.0.1:2379", "etcd endpoints, eg: -etcd 127.0.0.1:2379")
	flag.StringVar(&flagName, "name", "gateway", "gateway service name")
	flag.StringVar(&flagVersion, "version", "v1.0.0", "gateway service version")
}

func newDiscovery(etcdEndpoints []string) (registry.Discovery, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: etcdEndpoints,
	})
	if err != nil {
		return nil, err
	}
	return etcd.New(client), nil
}

func main() {
	flag.Parse()

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.name", flagName,
		"service.version", flagVersion,
	)

	log.SetLogger(logger)

	endpoints := []string{flagEtcd}

	dis, err := newDiscovery(endpoints)
	if err != nil {
		log.Fatalf("failed to create etcd discovery: %v", err)
	}

	clientFactory := client.NewFactory(dis)
	p, err := proxy.New(clientFactory, middleware.Create)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	confPath := flagConf + "/config.yaml"
	confLoader, err := configLoader.NewFileLoader(confPath, "")
	if err != nil {
		log.Fatalf("failed to create config loader: %v", err)
	}
	defer confLoader.Close()

	bc, err := confLoader.Load(context.Background())
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	bc.Name = flagName
	bc.Version = flagVersion

	buildContext := client.NewBuildContext(bc)
	if err := p.Update(buildContext, bc); err != nil {
		log.Fatalf("failed to update config: %v", err)
	}

	reloader := func() error {
		bc, err := confLoader.Load(context.Background())
		if err != nil {
			log.Errorf("failed to reload config: %v", err)
			return err
		}
		bc.Name = flagName
		bc.Version = flagVersion
		buildContext := client.NewBuildContext(bc)
		if err := p.Update(buildContext, bc); err != nil {
			log.Errorf("failed to update config: %v", err)
			return err
		}
		log.Infof("config reloaded successfully")
		return nil
	}
	confLoader.Watch(reloader)

	var serverHandler http.Handler = p
	srv := server.NewProxy(serverHandler, flagAddr)

	app := kratos.New(
		kratos.Name(flagName),
		kratos.Version(flagVersion),
		kratos.Logger(logger),
		kratos.Server(srv),
	)

	log.Infof("gateway starting, listen on: %s", flagAddr)
	if err := app.Run(); err != nil {
		log.Errorf("gateway stopped with error: %v", err)
	}
}
