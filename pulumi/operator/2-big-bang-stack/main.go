package main

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the Pulumi API token and AWS creds.
		config := config.New(ctx, "")
		pulumiAccessToken := config.Require("pulumiAccessToken")

		// Create the creds as Kubernetes Secrets.
		accessToken, err := corev1.NewSecret(ctx, "accesstoken", &corev1.SecretArgs{
			StringData: pulumi.StringMap{"accessToken": pulumi.String(pulumiAccessToken)},
		})
		if err != nil {
			return err
		}

		username := config.Require("username")
		password := config.Require("password")

		// repo1Creds, err := corev1.NewSecret(ctx, "aws-creds", &corev1.SecretArgs{
		// 	Metadata: metav1.ObjectMetaPtr(&metav1.ObjectMetaArgs{
		// 		Name: pulumi.String("aws-creds"),
		// 	}).ToObjectMetaPtrOutput(),
		// 	StringData: pulumi.StringMap{
		// 		"bigbang:username": pulumi.String(username),
		// 		"bigbang:password": pulumi.String(password),
		// 	},
		// })
		// if err != nil {
		// 	return err
		// }

		fmt.Println(pulumiAccessToken)

		// Deploy Big Bang through the operator
		_, err = apiextensions.NewCustomResource(ctx, "bb-stack",
			&apiextensions.CustomResourceArgs{
				Metadata: metav1.ObjectMetaPtr(&metav1.ObjectMetaArgs{
					Name: pulumi.String("bb"),
				}).ToObjectMetaPtrOutput(),
				ApiVersion: pulumi.String("pulumi.com/v1"),
				Kind:       pulumi.String("Stack"),
				OtherFields: kubernetes.UntypedArgs{
					"spec": map[string]interface{}{
						"accessTokenSecret": accessToken.Metadata.Name(),
						"stack":             "mikevanhemert/bb",
						"projectRepo":       "https://github.com/defenseunicorns/pulumi-bigbang",
						"branch":            "pulumi-chart",
						"repoDir":           "/pulumi/bigbang",
						// "envSecrets":        []interface{}{repo1Creds.Metadata.Name()},
						"config": map[string]string{
							"policy.enforce":                       "true",
							"policy.name":                          "kyverno",
							"monitoring.name":                      "prometheus",
							"serviceMesh.name":                     "istio",
							"serviceMesh.gateways[0].domain":       "bigbang.dev",
							"serviceMesh.gateways[0].name":         "public",
							"serviceMesh.gateways[0].tls.keyFile":  "/home/mikaelvanhemert/Documents/repos/hncd/pulumi-bigbang/public.key",
							"serviceMesh.gateways[0].tls.certFile": "/home/mikaelvanhemert/Documents/repos/hncd/pulumi-bigbang/public.cert",
							"development":                          "true",
							"bigbang:username":                     username,
							"bigbang:password":                     password,
						},
						"destroyOnFinalize": true,
					},
				},
			}, pulumi.DependsOn([]pulumi.Resource{accessToken}))

		return nil
	})
}

// "policy:
//     enforce: true
//     name: kyverno
// monitoring:
//     name: prometheus
// serviceMesh:
//     name: istio
//     gateways:
//     - domain: bigbang.dev
//         name: public
//         tls:
//             keyFile: /home/mikaelvanhemert/Documents/repos/hncd/pulumi-bigbang/public.key
//             certFile: /home/mikaelvanhemert/Documents/repos/hncd/pulumi-bigbang/public.cert
// development: true
// global:
//     imagePullSecrets:
//     - registry: registry1.dso.mil
//         username: MVanhemert
//         password: kbYOz379ADJaq8hfCJfLGh2oVgmVcD9i
// "
