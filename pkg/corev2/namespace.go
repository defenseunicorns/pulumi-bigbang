package v2

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func DeployNamespace(ctx *pulumi.Context, namespace string, istioInjection bool, pc ...PullCreds) (*corev1.Namespace, *corev1.Secret, error) {

	injection := "disabled"
	if istioInjection {
		injection = "enabled"
	}

	ns, err := corev1.NewNamespace(ctx, namespace, &corev1.NamespaceArgs{
		ApiVersion: pulumi.String("v1"),
		Kind:       pulumi.String("Namespace"),
		Metadata: &metav1.ObjectMetaArgs{
			Labels: pulumi.StringMap{
				"istio-injection": pulumi.String(injection),
			},
			Name: pulumi.String(namespace),
		},
	})
	if err != nil {
		return ns, nil, err
	}

	secret, err := DeployPullCreds(ctx, namespace, ns, pc...)

	//TODO Add default network Policies

	return ns, secret, nil
}

// Control plane.  When should apps be able to talk to the control plane?
/*
{{- if .Values.bigbang.networkPolicies.enabled -}}
{{- range $ns := compact (splitList " " (include "uniqueNamespaces" (merge (dict "default" false "constraint" "network.allowControlPlaneEgress") .))) -}}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: controlplane
  namespace: {{ $ns }}
  labels:
    app.kubernetes.io/name: {{ $ns }}
    {{- include "commonLabels" $ | nindent 4 }}
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - ipBlock:
        cidr: {{ $.Values.bigbang.networkPolicies.controlPlaneCidr }}
        {{- if eq $.Values.bigbang.networkPolicies.controlPlaneCidr "0.0.0.0/0" }}
        # ONLY Block requests to cloud metadata IP
        except:
        - 169.254.169.254/32
        {{- end }}
    {{ if dig "networkPolicies" "controlPlaneNode" false $.Values.bigbang }}
    ports:
    - port: {{ $.Values.bigbang.networkPolicies.controlPlaneNode }}
    {{- end -}}
---
{{ end -}}
{{- end -}}



Default Deny:

apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: defaultdeny
  namespace: {{ $ns }}
  labels:
    app.kubernetes.io/name: {{ $ns }}
    {{- include "commonLabels" $ | nindent 4 }}
spec:
  podSelector: {}
  policyTypes:
  - Egress
  - Ingress
  egress: []
  ingress: []



  Needs DNS:

  apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: dns
  namespace: {{ $ns }}
  labels:
    app.kubernetes.io/name: {{ $ns }}
    {{- include "commonLabels" $ | nindent 4 }}
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - port: 53
      protocol: UDP
    {{- if $.Values.bigbang.openshift }}
    - port: 5353
      protocol: UDP
    {{- end }}


Probably should be captured in the Ingress function in Istio, not here:

apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "resourceName" (printf "%s-gateway-%s" $pkg (print $i)) }}
  namespace: {{ dig "namespace" "name" $pkg $vals }}
  labels:
    app.kubernetes.io/name: {{ $pkg }}
    {{- include "commonLabels" $ | nindent 4 }}
spec:
  podSelector:
    {{- toYaml $selector | nindent 4 }}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          app.kubernetes.io/name: istio-controlplane
      podSelector:
        matchLabels:
          istio: ingressgateway
    {{- include "exposedPorts" $host | nindent 4 }}


	//Allow all pods to talk to eachother in the same namespace:

	apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: intranamespace
  namespace: {{ $ns }}
  labels:
    app.kubernetes.io/name: {{ $ns }}
    {{- include "commonLabels" $ | nindent 4 }}
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
  ingress:
  - from:
    - podSelector: {}
  egress:
  - to:
    - podSelector: {}


// Should be part of the Istio "MutateNamespace" call or whatever

apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: istiod
  namespace: {{ $ns }}
  labels:
    app.kubernetes.io/name: {{ $ns }}
    {{- include "commonLabels" $ | nindent 4 }}
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - ports:
    - port: 15012
    to:
    - namespaceSelector:
        matchLabels:
          app.kubernetes.io/name: istio-controlplane
      podSelector:
        matchLabels:
          istio: pilot


*/
