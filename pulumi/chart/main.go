package main

import (
	"context"
	"fmt"

	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	v2 "github.com/defenseunicorns/pulumi-bigbang/pkg/corev2"
)

func main() {
	// This is hard coded to work for me
	bigbang, err := ReadBigBang("runyontr/bb/local-bigbang", "bb")
	if err != nil {
		panic(err)
	}

	pulumi.Run(func(ctx *pulumi.Context) error {

		//make the config
		conf := config.New(ctx, "")

		//namespace
		namespace := conf.Get("namespace")
		if namespace == "" {
			namespace = "default"
		}

		file := conf.Get("file")

		//chart name
		chart := conf.Require("chart")

		name := conf.Get("name")
		if name == "" {
			name = chart
		}

		//namespace
		repo := conf.Get("repo")
		if repo == "" {
			repo = "https://charts.helm.sh/stable"
		}

		ns, secret, err := v2.DeployNamespace(ctx, namespace, bigbang.Configuration.ServiceMesh.Name == api.ServieMeshIstio,
			v2.PullCreds{
				Username: bigbang.Configuration.ImagePullSecrets[0].Username,
				Password: bigbang.Configuration.ImagePullSecrets[0].Password,
				Registry: bigbang.Configuration.ImagePullSecrets[0].Registry,
			})

		if err != nil {
			return err
		}

		ctx.Export("namespace", ns.Metadata.Name())

		// Deploy the Chart
		_, err = v2.DeployChart(ctx, v2.Chart{
			Namespace: namespace,
			Name:      name,
			Chart:     chart,
			Version:   "*",
			ValueFile: file,
			Repo:      repo,
		}, &bigbang, ns, secret)

		return err
	})
}

func ReadBigBang(stack, project string) (api.BigBang, error) {

	// s, err := pulumi.NewStackReference(ctx, stack, &pulumi.StackReferenceArgs{})
	ctx := context.Background()
	bigbang, err := auto.UpsertStackInlineSource(ctx, stack, project, nil)

	outs, err := bigbang.Outputs(ctx)
	if err != nil {
		fmt.Printf("Error getting outputs :%v\n", err)
		return api.BigBang{
			Configuration: api.Configuration{},
			Packages:      make([]api.BigBangPackage, 0),
		}, nil
	}
	fmt.Printf("Got the outputs, but here they are: %v\n", outs)
	for k, v := range outs {
		fmt.Printf("%v: %v\n", k, v)
	}
	config := api.NewConfiguration(outs["bigbang"].Value.(string))
	return api.BigBang{
		Configuration: config,
		Packages:      make([]api.BigBangPackage, 0),
	}, nil

}
