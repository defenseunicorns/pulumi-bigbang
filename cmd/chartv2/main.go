package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	v2 "github.com/defenseunicorns/pulumi-bigbang/pkg/corev2"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {

	//try and read the state from the BigBang deployment

	// project := flag.String("project", "bigbang", "project name")
	// stack := flag.String("stack", "auto", "Stack name")
	namespace := flag.String("namespace", "", "Namespace to create")
	destroy := flag.Bool("destroy", false, "cleanup namespace")
	chart := flag.String("chart", "wordpress", "Chart to deploy")
	name := flag.String("name", "", "Name for helm install")
	file := flag.String("file", "", "helm values file")
	repo := flag.String("repo", "https://charts.helm.sh/stable", "helm repo to pull chart from")

	flag.Parse()

	if *name == "" {
		*name = *chart
	}

	bigbang, err := ReadBigBang("bigbang", "bigbang")

	b, _ := json.MarshalIndent(bigbang, "", "\t")
	fmt.Println(string(b))
	deployFunc := func(ctx *pulumi.Context) error {
		// conf := config.New(ctx, "")

		ns, secret, err := v2.DeployNamespace(ctx, *namespace, bigbang.Configuration.ServiceMesh.Name == api.ServieMeshIstio)

		if err != nil {
			return err
		}

		ctx.Export("namespace", ns.Metadata.Name())

		// Deploy the Chart
		_, err = v2.DeployChart(ctx, v2.Chart{
			Namespace: *namespace,
			Name:      *name,
			Chart:     *chart,
			Version:   "*",
			ValueFile: *file,
			Repo:      *repo,
		}, &bigbang, ns, secret)

		return err
	}

	ctx := context.Background()

	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, *namespace, "namespace", deployFunc)

	fmt.Printf("Created/Selected stack %q\n", *namespace)

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "kubernetes", "v3.20.5")

	// // set stack configuration specifying the AWS region to deploy
	s.SetConfig(ctx, "registry.username", auto.ConfigValue{Value: bigbang.Configuration.ImagePullSecrets[0].Username})
	s.SetConfig(ctx, "registry.password", auto.ConfigValue{Value: bigbang.Configuration.ImagePullSecrets[0].Password})
	s.SetConfig(ctx, "registry.registry", auto.ConfigValue{Value: bigbang.Configuration.ImagePullSecrets[0].Registry})
	// s.SetConfig(ctx, "keyFile", auto.ConfigValue{Value: "./public.key"})
	// s.SetConfig(ctx, "certFile", auto.ConfigValue{Value: "./public.cert"})

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Refresh succeeded!")

	if *destroy {
		fmt.Println("Starting stack destroy")

		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
		}
		fmt.Println("Stack successfully destroyed")
		os.Exit(0)
	}

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our s3 website
	_, err = s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Update succeeded!")

}

func ReadBigBang(project, stack string) (api.BigBang, error) {

	ctx := context.Background()
	bigbang, err := auto.UpsertStackInlineSource(ctx, "bigbang", "bigbang", nil)

	if err != nil {
		fmt.Printf("Error reading the Bigbang Stack: %v\n", err)
		return api.BigBang{}, err
	}

	fmt.Printf("Created/Selected stack %q\n", stack)
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
