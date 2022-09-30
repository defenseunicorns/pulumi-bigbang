#!/bin/bash

helm delete -n podinfo podinfo
kubectl delete vs -n podinfo podinfo
helm delete -n wordpress wordpress
kubectl delete vs -n wordpress wordpress

helm delete -n istio-system istio
kubectl wait -n istio-system --for=delete deployment/istiod --timeout=60s
kubectl wait -n istio-system --for=delete deployment/public --timeout=60s
helm delete -n istio-operator istio-operator
helm delete -n kyverno kyverno-policies
helm delete -n kyverno kyverno
helm delete -n neuvector neuvector
helm delete -n prisma prisma

kubectl delete ns istio-system istio-operator kyverno podinfo wordpress neuvector prisma