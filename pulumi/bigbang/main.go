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
		var config *api.Configuration
		var err error
		if configFile != "" {
			config, err = api.LoadConfiguration(configFile)
			if err != nil {
				return err
			}
		} else {
			username := conf.Get("username")
			password := conf.Get("password")
			config = &api.Configuration{
				Policy: api.PolicyConfiguration{
					Name: api.PolicyKyverno,
				},
				ServiceMesh: api.ServiceMeshConfiguration{
					Name: api.ServieMeshIstio,
					Gateways: []struct {
						Name string
						Tls  struct {
							Key      string "yaml:\"key\""
							KeyFile  string "yaml:\"keyFile\""
							Cert     string "yaml:\"cert\""
							CertFile string "yaml:\"certFile\""
						} "yaml:\"tls\""
						Domain string
					}{
						struct {
							Name string
							Tls  struct {
								Key      string "yaml:\"key\""
								KeyFile  string "yaml:\"keyFile\""
								Cert     string "yaml:\"cert\""
								CertFile string "yaml:\"certFile\""
							} "yaml:\"tls\""
							Domain string
						}{
							Name:   "public",
							Domain: "bigbang.dev",
							Tls: struct {
								Key      string "yaml:\"key\""
								KeyFile  string "yaml:\"keyFile\""
								Cert     string "yaml:\"cert\""
								CertFile string "yaml:\"certFile\""
							}{
								KeyFile:  "https://raw.githubusercontent.com/defenseunicorns/pulumi-bigbang/main/public.key",
								CertFile: "https://raw.githubusercontent.com/defenseunicorns/pulumi-bigbang/main/public.cert",
							},
						},
					},
				},
				Development: true,
				CommonConfig: api.CommonConfig{
					ImagePullSecrets: []struct {
						Registry string "yaml:\"registry\""
						Username string "yaml:\"username\""
						Password string "yaml:\"password\""
					}{struct {
						Registry string "yaml:\"registry\""
						Username string "yaml:\"username\""
						Password string "yaml:\"password\""
					}{
						Registry: "registry1.dso.mil",
						Username: username,
						Password: password,
					}},
				},
			}
		}
		if configFile == "" {
			configFile = "./config.yaml"
		}

		_, err = bbv2.DeployBigBang(ctx, *config)
		return err
	})
}
