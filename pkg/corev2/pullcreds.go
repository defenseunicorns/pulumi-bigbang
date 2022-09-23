package v2

import (
	b64 "encoding/base64"
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type PullCreds struct {
	Username string
	Password string
	Registry string
}

func DeployPullCreds(ctx *pulumi.Context, namespace string, ns *corev1.Namespace, pc ...PullCreds) (*corev1.Secret, error) {

	var pullCreds string
	if len(pc) == 0 {
		c := config.New(ctx, "")
		username := config.Get(ctx, "registry.username")
		password := c.Require("registry.password")
		registry := c.Get("registry.registry")

		fmt.Printf("auths: %s:%s\n", username, password)
		encoded := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
		// .registry .username .password .email (printf "%s:%s" .username .password | b64enc)

		pullCreds = fmt.Sprintf("{\"auths\":{\"%s\":{\"username\":\"%s\",\"password\":\"%v\",\"auth\":\"%v\"}}}", registry, username, password, encoded)
	} else {
		encoded := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", pc[0].Username, pc[0].Password)))
		// .registry .username .password .email (printf "%s:%s" .username .password | b64enc)

		pullCreds = fmt.Sprintf("{\"auths\":{\"%s\":{\"username\":\"%s\",\"password\":\"%v\",\"auth\":\"%v\"}}}", pc[0].Registry, pc[0].Username, pc[0].Password, encoded)
	}

	secret, err := corev1.NewSecret(ctx, fmt.Sprintf("%s/private-registry", namespace), &corev1.SecretArgs{
		StringData: pulumi.StringMap{
			".dockerconfigjson": pulumi.String(pullCreds),
		},
		Metadata: &metav1.ObjectMetaArgs{
			Namespace: pulumi.String(namespace),
			Name:      pulumi.String("private-registry"),
		},
		Type: pulumi.String("kubernetes.io/dockerconfigjson"),
	}, pulumi.DependsOn([]pulumi.Resource{ns}))

	return secret, err
}
