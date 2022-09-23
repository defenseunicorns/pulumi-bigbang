package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/runyontr/pulumi-bigbang/pkg/api"
	bbv2 "github.com/runyontr/pulumi-bigbang/pkg/corev2"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		//make the config
		conf := config.New(ctx, "")
		username := conf.Require("username")
		password := conf.Require("password")

		_, err := bbv2.DeployBigBang(ctx, api.Configuration{
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
							KeyFile:  "/Users/tom/Workspaces/defenseunicorns/pulumi/pulumi-bigbang/public.key",
							CertFile: "/Users/tom/Workspaces/defenseunicorns/pulumi/pulumi-bigbang/public.cert",
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
		})
		return err
	})
}
