package main

import (
	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	bbv2 "github.com/defenseunicorns/pulumi-bigbang/pkg/corev2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		conf := config.New(ctx, "")
		configFile := conf.Get("config")
		if configFile == "" {
			configFile = "./config.yaml"
		}
		config, err := api.LoadConfiguration(configFile)
		if err != nil {
			return err
		}
		_, err = bbv2.DeployBigBang(ctx, *config)
		return err
	})
}
