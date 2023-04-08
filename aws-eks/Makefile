# eks
# karpenter
# vault
# logstash, elasticsearch, kibana
# prometheus, thanos
# grafana
# loki
# opentelemetry

# commands
# --------------------------------
# make startup
# make build
# make apply
# make create_karpenter_provider
# --------------------------------

MAKEFLAGS += --warn-undefined-variables --no-print-directory
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := startup
.DELETE_ON_ERROR:
.SUFFIXES:

export KARPENTER_VERSION = "v0.27.0"

.PHONY: startup
startup:
	@pulumi up

.PHONY: build
build: helmfile_build prepare_karpenter_config create_karpenter_crd create_vault_sa

.PHONY: apply
apply: helmfile_install

.PHONY: destroy
destroy:
	@pushd helm; export KUBECONFIG=../.kubeconfig; helmfile destroy; popd
	@pulumi destroy

.PHONY: helmfile_build
helmfile_build:
	@pushd helm; helmfile build; popd

.PHONY: helmfile_install
helmfile_install:
	@pushd helm; export KUBECONFIG=../.kubeconfig; helmfile sync; popd

.PHONY: helmfile_destroy
helmfile_destroy:
	@pushd helm; export KUBECONFIG=../.kubeconfig; helmfile destroy; popd

.PHONY: create_vault_sa
create_vault_sa:
	@export KUBECONFIG=.kubeconfig; \
	export VAULT_IAM_ROLE_ARN=`pulumi stack output vaultIAMRoleArn 2>/dev/null`; \
	export VAULT_KMS_KEY_ALIAS=`pulumi stack output vaultKmsKeyAlias 2>/dev/null`; \
	kubectl create ns vault; \
	kubectl -n vault create sa vault-sa --dry-run=client -o yaml | yq -r '.metadata.annotations."eks.amazonaws.com/role-arn"=env(VAULT_IAM_ROLE_ARN)' | kubectl apply -f - ; \
	cat helm/manifests/vault-values-template.yaml | sed "s|kms_key_id = .*|kms_key_id = \"$$VAULT_KMS_KEY_ALIAS\"|" > helm/manifests/.vault-values.yaml;

# https://repost.aws/knowledge-center/eks-install-karpenter
.PHONY: prepare_karpenter_config
prepare_karpenter_config:
	@export EKS_CLUSTER_NAME=`pulumi stack output eksClusterName 2>/dev/null`; \
	export EKS_CLUSTER_ENDPOINT=`pulumi stack output eksClusterEndpoint 2>/dev/null`; \
	export KARPENTER_INTERRUPTION_QUEUE_NAME=`pulumi stack output karpenterInterruptionQueueName 2>/dev/null`; \
	export KARPENTER_CONTROLLER_ROLE_ARN=`pulumi stack output karpenterControllerIAMRoleArn 2>/dev/null`; \
	export EKS_WORKER_GROUP_1_NAME=`pulumi stack output eksWorkerGroup1Name 2>/dev/null`; \
	cat helm/manifests/karpenter-values-template.yaml | \
	yq '.settings.aws.clusterEndpoint = env(EKS_CLUSTER_ENDPOINT)' | \
	yq '.settings.aws.clusterName = env(EKS_CLUSTER_NAME)' | \
	yq '.settings.aws.interruptionQueueName = env(KARPENTER_INTERRUPTION_QUEUE_NAME)' | \
	yq '.serviceAccount.annotations."eks.amazonaws.com/role-arn" = env(KARPENTER_CONTROLLER_ROLE_ARN)' | \
	yq '.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[1].matchExpressions[0].values[0] = env(EKS_WORKER_GROUP_1_NAME)' \
	> helm/manifests/.karpenter-values.yaml; \
	sed "s/EKS_CLUSTER_NAME/$$EKS_CLUSTER_NAME/g" helm/manifests/karpenter-provider-template.yaml > helm/manifests/.karpenter-provider-values.yaml

.PHONY: create_karpenter_crd
create_karpenter_crd:
	@export KUBECONFIG=.kubeconfig; \
	kubectl create -f https://raw.githubusercontent.com/aws/karpenter/${KARPENTER_VERSION}/pkg/apis/crds/karpenter.sh_provisioners.yaml; \
	kubectl create -f https://raw.githubusercontent.com/aws/karpenter/${KARPENTER_VERSION}/pkg/apis/crds/karpenter.k8s.aws_awsnodetemplates.yaml; \

.PHONY: create_karpenter_provider
create_karpenter_provider:
	@export KUBECONFIG=.kubeconfig; \
	kubectl create -f helm/manifests/.karpenter-provider-values.yaml

 .PHONY: vault_init
vault_init:
	@export KUBECONFIG=.kubeconfig; \
	kubectl -n vault exec vault-0 -- vault operator init -format=json > .vault-cluster-keys.json; \
	kubectl -n vault exec -it vault-1 -- vault operator raft join http://vault-0.vault-internal:8200; \
	kubectl -n vault exec -it vault-2 -- vault operator raft join http://vault-0.vault-internal:8200

# @VAULT_UNSEAL_KEY=$$(jq -r ".unseal_keys_b64[]" .vault-cluster-keys.json) && \
# kubectl -n vault exec vault-0 -- vault operator unseal $$VAULT_UNSEAL_KEY
# kubectl -n vault exec -it vault-1 -- vault operator unseal $$VAULT_UNSEAL_KEY && \
# kubectl -n vault exec -it vault-2 -- vault operator unseal $$VAULT_UNSEAL_KEY

# Login first time with token
# VAULT_TOKEN=$$(jq -r ".root_token" .vault-cluster-keys.json)
# vault login

.PHONY: vault_enable_auth_k8s_dev
vault_enable_auth_k8s_dev:
	@export KUBECONFIG=.kubeconfig; \
	kubectl -n app create sa rbac-app; \
	DEV_VAULT_SA_NAME=$$(kubectl -n app get sa rbac-app --output jsonpath="{.secrets[*]['name']}") && \
	DEV_SA_JWT_TOKEN=$$(kubectl -n app get secret $$DEV_VAULT_SA_NAME --output 'go-template={{ .data.token }}' | base64 --decode) && \
	echo $$DEV_SA_JWT_TOKEN > .dev-cluster-jwt-token; \
	kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.certificate-authority-data}' | base64 --decode > .dev-cluster-ca.crt; \
	kubectl cluster-info | grep "control plane" | awk '{print $$NF}' > .dev-cluster-ep; \
	kubectl cp .dev-cluster-jwt-token vault/vault-0:tmp/; \
	kubectl cp .dev-cluster-ep vault/vault-0:tmp/; \
	kubectl cp .dev-cluster-ca.crt vault/vault-0:tmp/; \
	export VAULT_TOKEN=$$(jq -r ".root_token" .vault-cluster-keys.json); \
	kubectl -n vault exec -it vault-0 -- vault login env(VAULT_TOKEN); \
	kubectl -n vault exec -it vault-0 -- vault secrets enable -path=dev kv-v2; \
	kubectl -n vault exec -it vault-0 -- vault policy write dev-app-policy - <<EOF path "dev/*" { capabilities = ["read"] }EOF; \
	kubectl -n vault exec -it vault-0 -- vault auth enable --path=dev-cluster kubernetes; \
	kubectl -n vault exec -it vault-0 -- vault write auth/dev-cluster/config token_reviewer_jwt="$$(cat /tmp/.dev-cluster-jwt-token)" kubernetes_host="$$(cat /tmp/.dev-cluster-ep)" kubernetes_ca_cert=@/tmp/.dev-cluster-ca.crt issuer="https://kubernetes.default.svc.cluster.local"; \
	kubectl -n vault exec -it vault-0 -- vault write auth/dev-cluster/role/dev-app bound_service_account_names=rbac-app bound_service_account_namespaces=app policies=dev-app-policy ttl=24h
