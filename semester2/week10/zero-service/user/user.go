package main

import (
	"flag"
	"fmt"

	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/internal/config"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/internal/server"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/internal/svc"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/user/user"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		user.RegisterUserServer(grpcServer, server.NewUserServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
