package main

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/runyontr/pulumi-bigbang/pkg/api"

	v2 "github.com/runyontr/pulumi-bigbang/pkg/corev2"
)

func main() {
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

		stack := conf.Get("stack")
		if stack == "" {
			stack = "k3d"
		}

		project := conf.Get("project")
		if project == "" {
			project = "bb"
		}

		bigbang, err := ReadBigBang(ctx, stack, project)
		if err != nil {
			return err
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

func ReadBigBang(ctx *pulumi.Context, stack, project string) (api.BigBang, error) {

	s, err := pulumi.NewStackReference(ctx, stack, &pulumi.StackReferenceArgs{})

	if err != nil {
		return api.BigBang{
			Configuration: api.Configuration{},
			Packages:      make([]api.BigBangPackage, 0),
		}, nil
	}

	if err != nil {
		fmt.Printf("Error reading the Bigbang Stack: %v\n", err)
		return api.BigBang{}, err
	}

	fmt.Printf("Got the outputs, but here they are: %v\n", s.Outputs)
	return s.Outputs.ApplyT(func(o map[string]interface{}) (api.BigBang, error) {
		config := api.NewConfiguration(o["bigbang"].Value.(string))
		return api.BigBang{
			Configuration: config,
			Packages:      make([]api.BigBangPackage, 0),
		}, nil
	})

}
