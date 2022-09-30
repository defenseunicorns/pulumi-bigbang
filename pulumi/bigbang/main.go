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

		ns, secret, err := bbv2.DeployNamespace(ctx, "podinfo", config.ServiceMesh.Name == api.ServieMeshIstio,
			bbv2.PullCreds{
				Username: config.ImagePullSecrets[0].Username,
				Password: config.ImagePullSecrets[0].Password,
				Registry: config.ImagePullSecrets[0].Registry,
			})

		if err != nil {
			return err
		}

		ctx.Export("namespace", ns.Metadata.Name())

		// Deploy the Chart
		_, err = bbv2.DeployChart(ctx, bbv2.Chart{
			Namespace: "podinfo",
			Name:      "podinfo",
			Chart:     "podinfo",
			Version:   "*",
			ValueFile: "",
			Repo:      "https://stefanprodan.github.io/podinfo",
		}, &api.BigBang{Configuration: *config}, ns, secret)

		return err
	})
}
