package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/kms"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/sqs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	// "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/autoscaling"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// tags := make(map[string]string)
		tags := NewTags(ctx)

		// Get current account id
		current, err := GetCurrent(ctx)
		if err != nil {
			return err
		}
		// ctx.Export("accountId", current.accountId)

		// Create VPC
		vpcCidrBlock := getEnv(ctx, "vpc:cidr-block", "unknown")
		vpc, err := NewVpc(ctx, NewPrefixName("vpc"), vpcCidrBlock)
		if err != nil {
			return err
		}
		ctx.Export("vpcId", vpc.vpcId)

		// Create 3 private subnets
		privSubnet1, err := NewSubnet(ctx, NewPrefixName("priv-subnet-1"), vpc.vpcId, "10.0.1.0/24", "eu-west-1a")
		if err != nil {
			return err
		}
		privSubnet2, err := NewSubnet(ctx, NewPrefixName("priv-subnet-2"), vpc.vpcId, "10.0.2.0/24", "eu-west-1b")
		if err != nil {
			return err
		}
		privSubnet3, err := NewSubnet(ctx, NewPrefixName("priv-subnet-3"), vpc.vpcId, "10.0.3.0/24", "eu-west-1c")
		if err != nil {
			return err
		}

		// Create 3 public subnets
		pubSubnet1, err := NewSubnet(ctx, NewPrefixName("pub-subnet-1"), vpc.vpcId, "10.0.4.0/24", "eu-west-1a")
		if err != nil {
			return err
		}
		pubSubnet2, err := NewSubnet(ctx, NewPrefixName("pub-subnet-2"), vpc.vpcId, "10.0.5.0/24", "eu-west-1b")
		if err != nil {
			return err
		}
		pubSubnet3, err := NewSubnet(ctx, NewPrefixName("pub-subnet-3"), vpc.vpcId, "10.0.6.0/24", "eu-west-1c")
		if err != nil {
			return err
		}

		// Create an Elastic IP
		eipNatgw, err := NewEip(ctx, NewPrefixName("eip-natgw"))
		if err != nil {
			return err
		}

		// Create NAT Gateway
		natGw, err := NewNatGw(ctx, eipNatgw.eipId, pubSubnet1.subnetId)
		if err != nil {
			return err
		}

		// Create Internet Gateway
		igw, err := NewIgw(ctx, vpc.vpcId)
		if err != nil {
			return err
		}

		// Create Private Routing Table
		privateRouteTable, err := NewRoutingTable(ctx, NewPrefixName("rtb-private"), vpc.vpcId, natGw.ngwId)
		if err != nil {
			return err
		}

		// Create Public Routing Table
		publicRouteTable, err := NewRoutingTable(ctx, NewPrefixName("rtb-public"), vpc.vpcId, igw.igwId)
		if err != nil {
			return err
		}

		// Associate Private Subs with Private Route Tables
		_, err = NewRoutingTableAssoc(ctx, NewPrefixName("rtb-assoc-priv-1"), privSubnet1.subnetId, privateRouteTable.routingtableId)
		if err != nil {
			return err
		}
		_, err = NewRoutingTableAssoc(ctx, NewPrefixName("rtb-assoc-priv-2"), privSubnet2.subnetId, privateRouteTable.routingtableId)
		if err != nil {
			return err
		}
		_, err = NewRoutingTableAssoc(ctx, NewPrefixName("rtb-assoc-priv-3"), privSubnet3.subnetId, privateRouteTable.routingtableId)
		if err != nil {
			return err
		}

		// Associate Public Subs with Public Route Tables
		_, err = NewRoutingTableAssoc(ctx, NewPrefixName("rtb-assoc-pub-1"), pubSubnet1.subnetId, publicRouteTable.routingtableId)
		if err != nil {
			return err
		}
		_, err = NewRoutingTableAssoc(ctx, NewPrefixName("rtb-assoc-pub-2"), pubSubnet2.subnetId, publicRouteTable.routingtableId)
		if err != nil {
			return err
		}
		_, err = NewRoutingTableAssoc(ctx, NewPrefixName("rtb-assoc-pub-3"), pubSubnet3.subnetId, publicRouteTable.routingtableId)
		if err != nil {
			return err
		}

		// Add ALB for EKS
		// Create a Security Group for private alb
		eksAlbSgName := NewPrefixName("eks-alb-sg")
		tags["Name"] = eksAlbSgName
		eksAlbSg, err := ec2.NewSecurityGroup(ctx, eksAlbSgName, &ec2.SecurityGroupArgs{
			VpcId: vpc.vpcId,
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(30443),
					ToPort:     pulumi.Int(30443),
					CidrBlocks: pulumi.StringArray{pulumi.String("10.0.0.0/16")},
				},
			},
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(443),
					ToPort:     pulumi.Int(443),
					CidrBlocks: pulumi.StringArray{pulumi.String("10.0.0.0/16")},
				},
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

		eksPrivAlbName := NewPrefixName("eks-priv-alb")
		eksPrivAlb, err := lb.NewLoadBalancer(ctx, eksPrivAlbName, &lb.LoadBalancerArgs{
			LoadBalancerType: pulumi.String("application"),
			Internal:         pulumi.Bool(true),
			SubnetMappings: lb.LoadBalancerSubnetMappingArray{
				&lb.LoadBalancerSubnetMappingArgs{
					SubnetId: privSubnet1.subnetId,
				},
				&lb.LoadBalancerSubnetMappingArgs{
					SubnetId: privSubnet2.subnetId,
				},
				&lb.LoadBalancerSubnetMappingArgs{
					SubnetId: privSubnet3.subnetId,
				},
			},
			SecurityGroups: pulumi.StringArray{
				eksAlbSg.ID().ToStringOutput(),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("eksPrivAlbArn", eksPrivAlb.Arn)

		// EKS Cluster IAM Role
		eksClusterIamRoleName := NewPrefixName("eks-iam-eksRole")
		eksClusterIamRole, err := iam.NewRole(ctx, eksClusterIamRoleName, &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
		    "Version": "2008-10-17",
		    "Statement": [{
		        "Sid": "",
		        "Effect": "Allow",
		        "Principal": {
		            "Service": "eks.amazonaws.com"
		        },
		        "Action": "sts:AssumeRole"
		    }]
		}`),
		})
		if err != nil {
			return err
		}

		eksPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
			"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
		}
		for i, eksPolicy := range eksPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, NewPrefixName(fmt.Sprintf("rpa-%d", i)), &iam.RolePolicyAttachmentArgs{
				Role:      eksClusterIamRole.ID(),
				PolicyArn: pulumi.String(eksPolicy),
			})
			if err != nil {
				return err
			}
		}

		// Create an EKS cluster.
		var eksSubnetIds pulumi.StringArray
		eksVersion := getEnv(ctx, "eks:version", "unknown")
		eksClusterName := NewPrefixName("eks-cluster")

		eksCluster, err := eks.NewCluster(ctx, eksClusterName, &eks.ClusterArgs{
			Name:    pulumi.String(eksClusterName),
			Version: pulumi.String(eksVersion),
			RoleArn: pulumi.StringOutput(eksClusterIamRole.Arn),
			VpcConfig: &eks.ClusterVpcConfigArgs{
				PublicAccessCidrs: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
				EndpointPrivateAccess: pulumi.Bool(true),
				SubnetIds: pulumi.StringArray(
					append(
						eksSubnetIds,
						privSubnet1.subnetId,
						privSubnet2.subnetId,
						privSubnet3.subnetId,
						pubSubnet1.subnetId,
						pubSubnet2.subnetId,
						pubSubnet3.subnetId,
					),
				),
			},
		})

		if err != nil {
			return err
		}

		eksclusterSecurityGroupId := eksCluster.VpcConfig.ClusterSecurityGroupId().Elem()
		eksOidcIssuer := eksCluster.Identities.Index(pulumi.Int(0)).Oidcs().Index(pulumi.Int(0)).Issuer().Elem()
		ctx.Export("eksClusterName", pulumi.String(eksClusterName))
		ctx.Export("eksClusterArn", eksCluster.Arn)
		ctx.Export("eksClusterEndpoint", eksCluster.Endpoint)
		ctx.Export("eksOidcIssuer", eksOidcIssuer)
		ctx.Export("eksClusterSgId", eksclusterSecurityGroupId)

		// Associate an IAM OIDC provider with the EKS cluster
		_, err = iam.NewOpenIdConnectProvider(ctx, fmt.Sprintf("iam-oidc-provider-%s", eksClusterName), &iam.OpenIdConnectProviderArgs{
			ClientIdLists: pulumi.StringArray{
				pulumi.String("sts.amazonaws.com"),
			},
			ThumbprintLists: pulumi.StringArray{
				pulumi.String("9E99A48A9960B14926BB7F3B02E22DA2B0AB7280"),
			},
			Url: eksOidcIssuer,
		})
		if err != nil {
			return err
		}

		// Create kms key for vault server running in EKS cluster
		vaultKmsKey, err := NewKmsKey(ctx, NewPrefixName("vault-kms-key"))
		vaultKmsKeyAlias, err := kms.NewAlias(ctx, "alias/vault-seal", &kms.AliasArgs{
			TargetKeyId: vaultKmsKey.kmsKeyId,
		})
		ctx.Export("vaultKmsKeyAlias", vaultKmsKeyAlias.ID())
		vaultKmsPolicyStatement := vaultKmsKey.kmsKeyArn.ApplyT(func(kmsKeyarn string) (string, error) {
			vaultKmsPolicyJson, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Sid": "AllowKmsKeyUsage",
						"Action": []string{
							"kms:Decrypt",
							"kms:Encrypt",
							"kms:DescribeKey",
						},
						"Effect": "Allow",
						"Resource": []string{
							kmsKeyarn,
						},
					},
				},
			})
			if err != nil {
				return "", err
			}
			return string(vaultKmsPolicyJson), nil

		})

		vaultKmsPolicyName := NewPrefixName("vault-kms-policy")
		vaultKmsPolicy, err := iam.NewPolicy(ctx, vaultKmsPolicyName, &iam.PolicyArgs{
			Description: pulumi.String(vaultKmsPolicyName),
			Policy:      vaultKmsPolicyStatement,
		})
		if err != nil {
			return err
		}

		ctx.Export("vaultKmsKeyArn", vaultKmsKey.kmsKeyArn)
		ctx.Export("vaultKmsKeyPolicyArn", vaultKmsPolicy.Arn)

		vaultIAMRolePolicy := eksCluster.Identities.Index(pulumi.Int(0)).Oidcs().Index(pulumi.Int(0)).Issuer().Elem().ApplyT(func(issuer string) (string, error) {
			eksOIDCSub := strings.ReplaceAll(issuer, "https://", "") + ":sub"
			eksOIDCAud := strings.ReplaceAll(issuer, "https://", "") + ":aud"
			eksOIDCArn := "arn:aws:iam::" + string(current.accountId) + ":oidc-provider/" + strings.ReplaceAll(issuer, "https://", "")

			vaultIAMRolePolicyJson, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Action": []string{
							"sts:AssumeRoleWithWebIdentity",
						},
						"Effect": "Allow",
						"Principal": map[string]interface{}{
							"Federated": eksOIDCArn,
						},
						"Condition": map[string]interface{}{
							"StringEquals": map[string]interface{}{
								eksOIDCSub: "system:serviceaccount:vault:vault-sa",
								eksOIDCAud: "sts.amazonaws.com",
							},
						},
					},
				},
			})
			if err != nil {
				return "", err
			}
			return string(vaultIAMRolePolicyJson), nil
		})

		// Create vault IAM Role
		vaultIAMRole, err := iam.NewRole(ctx, NewPrefixName("vault-iam-role"), &iam.RoleArgs{
			AssumeRolePolicy: vaultIAMRolePolicy,
		})
		if err != nil {
			return err
		}
		ctx.Export("vaultIAMRoleArn", vaultIAMRole.Arn)

		_, err = iam.NewRolePolicyAttachment(ctx, NewPrefixName("vault-kms-policy-attach"), &iam.RolePolicyAttachmentArgs{
			Role:      vaultIAMRole.ID(),
			PolicyArn: vaultKmsPolicy.Arn,
		})
		if err != nil {
			return err
		}

		// Create the EC2 NodeGroup Role
		karpenterInstanceNodeRole, err := iam.NewRole(ctx, fmt.Sprintf("karpenterInstanceNodeRole-%s", eksClusterName), &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
		    "Version": "2012-10-17",
		    "Statement": [{
		        "Sid": "",
		        "Effect": "Allow",
		        "Principal": {
		            "Service": "ec2.amazonaws.com"
		        },
		        "Action": "sts:AssumeRole"
		    }]
		}`),
		})
		if err != nil {
			return err
		}
		ctx.Export("karpenterInstanceNodeRoleArn", karpenterInstanceNodeRole.Arn)

		nodeGroupPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
			"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
			"arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy",
		}
		for i, nodeGroupPolicy := range nodeGroupPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, NewPrefixName(fmt.Sprintf("ngpa-%d", i)), &iam.RolePolicyAttachmentArgs{
				Role:      karpenterInstanceNodeRole.ID(),
				PolicyArn: pulumi.String(nodeGroupPolicy),
			})
			if err != nil {
				return err
			}
		}

		// Karpenter node instance Profile
		karpenterNodeInstanceProfileName := fmt.Sprintf("KarpenterNodeInstanceProfile-%s", eksClusterName)
		karpenterNodeInstanceProfile, err := iam.NewInstanceProfile(ctx, karpenterNodeInstanceProfileName, &iam.InstanceProfileArgs{
			Name: pulumi.String(karpenterNodeInstanceProfileName),
			Role: karpenterInstanceNodeRole.Name,
		})
		if err != nil {
			return err
		}
		ctx.Export("karpenterNodeInstanceProfileName", karpenterNodeInstanceProfile.Name)
		ctx.Export("karpenterNodeInstanceProfileArn", karpenterNodeInstanceProfile.Arn)

		// Karpenter interruption queue
		karpenterInterruptionQueueName := fmt.Sprintf("karpenterInterruptionQueue-%s", eksClusterName)
		karpenterInterruptionQueue, err := sqs.NewQueue(ctx, karpenterInterruptionQueueName, &sqs.QueueArgs{
			Name:                    pulumi.String(karpenterInterruptionQueueName),
			MessageRetentionSeconds: pulumi.Int(300),
		})
		if err != nil {
			return err
		}
		ctx.Export("karpenterInterruptionQueueName", pulumi.String(karpenterInterruptionQueueName))
		ctx.Export("karpenterInterruptionQueueArn", karpenterInterruptionQueue.Arn)

		// karpenter interruption queue policy
		karpenterInterruptionQueuePolicy := karpenterInterruptionQueue.Arn.ApplyT(func(karpenterInterruptionQueueArn string) (string, error) {
			karpenterInterruptionQueuePolicyJson, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Action": []string{
							"sqs:SendMessage",
						},
						"Effect": "Allow",
						"Principal": map[string]interface{}{
							"Service": []string{
								"events.amazonaws.com",
								"sqs.amazonaws.com",
							},
						},
						"Resource": karpenterInterruptionQueueArn,
					},
				},
			})
			if err != nil {
				return "", err
			}
			return string(karpenterInterruptionQueuePolicyJson), nil
		})

		_, err = sqs.NewQueuePolicy(ctx, fmt.Sprintf("karpenter-interruption-queue-policy-%s", eksClusterName), &sqs.QueuePolicyArgs{
			QueueUrl: karpenterInterruptionQueue.Url,
			Policy:   karpenterInterruptionQueuePolicy,
		})
		if err != nil {
			return err
		}

		// Scheduled Change Rule
		scheduledChangeRule, err := cloudwatch.NewEventRule(ctx, "scheduled-change-rule", &cloudwatch.EventRuleArgs{
			Name: pulumi.String("scheduled-change-rule"),
			EventPattern: pulumi.String(`{
              "source": [
                "aws.ec2"
              ],
              "detail-type": [
                "AWS Health Event"
              ]
            }`),
		})
		if err != nil {
			return err
		}

		_, err = cloudwatch.NewEventTarget(ctx, "scheduled-change-rule-target", &cloudwatch.EventTargetArgs{
			Rule: scheduledChangeRule.Name,
			Arn:  karpenterInterruptionQueue.Arn,
		})
		if err != nil {
			return err
		}

		// Spot Interruption Rule
		spotInterruptionRule, err := cloudwatch.NewEventRule(ctx, "spot-interruption-rule", &cloudwatch.EventRuleArgs{
			Name: pulumi.String("spot-interruption-rule"),
			EventPattern: pulumi.String(`{
              "source": [
                "aws.ec2"
              ],
              "detail-type": [
                "EC2 Spot Instance Interruption Warning"
              ]
            }`),
		})
		if err != nil {
			return err
		}

		_, err = cloudwatch.NewEventTarget(ctx, "spot-interruption-rule-target", &cloudwatch.EventTargetArgs{
			Rule: spotInterruptionRule.Name,
			Arn:  karpenterInterruptionQueue.Arn,
		})
		if err != nil {
			return err
		}

		// EC2 Instance Rebalance Rule
		rebalanceRule, err := cloudwatch.NewEventRule(ctx, "rebalance-rule", &cloudwatch.EventRuleArgs{
			Name: pulumi.String("rebalance-rule"),
			EventPattern: pulumi.String(`{
              "source": [
                "aws.ec2"
              ],
              "detail-type": [
                "EC2 Instance Rebalance Recommendation"
              ]
            }`),
		})
		if err != nil {
			return err
		}

		_, err = cloudwatch.NewEventTarget(ctx, "rebalance-rule-target", &cloudwatch.EventTargetArgs{
			Rule: rebalanceRule.Name,
			Arn:  karpenterInterruptionQueue.Arn,
		})
		if err != nil {
			return err
		}

		// EC2 Instance State Change Rule
		instanceStateChangeRule, err := cloudwatch.NewEventRule(ctx, "instance-state-change-rule", &cloudwatch.EventRuleArgs{
			Name: pulumi.String("instance-state-change-rule"),
			EventPattern: pulumi.String(`{
              "source": [
                "aws.ec2"
              ],
              "detail-type": [
                "EC2 Instance State-change Notification"
              ]
            }`),
		})
		if err != nil {
			return err
		}

		_, err = cloudwatch.NewEventTarget(ctx, "instance-state-change-rule-target", &cloudwatch.EventTargetArgs{
			Rule: instanceStateChangeRule.Name,
			Arn:  karpenterInterruptionQueue.Arn,
		})
		if err != nil {
			return err
		}

		// Karpenter controller iam role
		karpenterControllerIAMAssumeRolePolicy := eksOidcIssuer.ApplyT(func(issuer string) (string, error) {
			eksOIDCSub := strings.ReplaceAll(issuer, "https://", "") + ":sub"
			eksOIDCAud := strings.ReplaceAll(issuer, "https://", "") + ":aud"
			eksOIDCArn := "arn:aws:iam::" + string(current.accountId) + ":oidc-provider/" + strings.ReplaceAll(issuer, "https://", "")

			karpenterControllerIAMAssumeRolePolicyJson, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Action": []string{
							"sts:AssumeRoleWithWebIdentity",
						},
						"Effect": "Allow",
						"Principal": map[string]interface{}{
							"Federated": eksOIDCArn,
						},
						"Condition": map[string]interface{}{
							"StringEquals": map[string]interface{}{
								eksOIDCSub: "system:serviceaccount:karpenter:karpenter",
								eksOIDCAud: "sts.amazonaws.com",
							},
						},
					},
				},
			})
			if err != nil {
				return "", err
			}
			return string(karpenterControllerIAMAssumeRolePolicyJson), nil
		})
		karpenterControllerIAMRole, err := iam.NewRole(ctx, NewPrefixName("karpenter-controller-iam-role"), &iam.RoleArgs{
			AssumeRolePolicy: karpenterControllerIAMAssumeRolePolicy,
		})
		if err != nil {
			return err
		}
		ctx.Export("karpenterControllerIAMRoleArn", karpenterControllerIAMRole.Arn)

		karpenterControllerIamPolicy := pulumi.All(karpenterInterruptionQueue.Arn, karpenterInstanceNodeRole.Arn, eksCluster.Arn).ApplyT(
			func(args []interface{}) (string, error) {
				karpenterInterruptionQueueArn := args[0].(string)
				karpenterInstanceNodeRoleArn := args[1].(string)
				eksClusterArn := args[2].(string)
				karpenterControllerPolicyStatement, err := json.Marshal(map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						{
							"Action": []string{
								// Write Operations
								"ec2:CreateFleet",
								"ec2:CreateLaunchTemplate",
								"ec2:CreateTags",
								"ec2:DeleteLaunchTemplate",
								"ec2:RunInstances",
								// Read Operations
								"ec2:DescribeAvailabilityZones",
								"ec2:DescribeImages",
								"ec2:DescribeInstances",
								"ec2:DescribeInstanceTypeOfferings",
								"ec2:DescribeInstanceTypes",
								"ec2:DescribeLaunchTemplates",
								"ec2:DescribeSecurityGroups",
								"ec2:DescribeSpotPriceHistory",
								"ec2:DescribeSubnets",
								"pricing:GetProducts",
								"ssm:GetParameter",
							},
							"Effect":   "Allow",
							"Resource": "*",
							"Sid":      "Karpenter",
						},
						{
							"Action": []string{
								"ec2:TerminateInstances",
							},
							"Condition": map[string]interface{}{
								"StringEquals": map[string]interface{}{
									"ec2:ResourceTag/Name": "*karpenter*",
								},
							},
							"Effect":   "Allow",
							"Resource": "*",
							"Sid":      "ConditionalEC2Termination",
						},
						{
							"Action": []string{
								// Write Operations
								"sqs:DeleteMessage",
								// Read Operations
								"sqs:GetQueueAttributes",
								"sqs:GetQueueUrl",
								"sqs:ReceiveMessage",
							},
							"Effect":   "Allow",
							"Resource": karpenterInterruptionQueueArn,
							"Sid":      "KarpenterInterruptionQueue",
						},
						{
							"Action": []string{
								"iam:PassRole",
							},
							"Effect":   "Allow",
							"Resource": karpenterInstanceNodeRoleArn,
							"Sid":      "PassNodeIamRole",
						},
						{
							"Action": []string{
								"eks:DescribeCluster",
							},
							"Effect":   "Allow",
							"Resource": eksClusterArn,
							"Sid":      "EKSClusterEndpointLookup",
						},
					},
				})
				if err != nil {
					return "", err
				}
				return string(karpenterControllerPolicyStatement), nil
			})
		_, err = iam.NewRolePolicy(ctx, NewPrefixName("karpenter-controller-iam-policy"), &iam.RolePolicyArgs{
			Role:   karpenterControllerIAMRole.ID(),
			Policy: karpenterControllerIamPolicy,
		})
		if err != nil {
			return err
		}

		// Add karpenter tags to subnets
		_, err = ec2.NewTag(ctx, NewPrefixName("priv-subnet-1"), &ec2.TagArgs{
			Key:        pulumi.String("karpenter.sh/discovery"),
			Value:      pulumi.String(eksClusterName),
			ResourceId: privSubnet1.subnetId,
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewTag(ctx, NewPrefixName("priv-subnet-2"), &ec2.TagArgs{
			Key:        pulumi.String("karpenter.sh/discovery"),
			Value:      pulumi.String(eksClusterName),
			ResourceId: privSubnet2.subnetId,
		})
		if err != nil {
			return err
		}
		_, err = ec2.NewTag(ctx, NewPrefixName("priv-subnet-3"), &ec2.TagArgs{
			Key:        pulumi.String("karpenter.sh/discovery"),
			Value:      pulumi.String(eksClusterName),
			ResourceId: privSubnet3.subnetId,
		})
		if err != nil {
			return err
		}

		// Resource: EKS Node Groups
		// Purpose: Amazon EKS managed node groups automate the provisioning and lifecycle management of nodes (Amazon EC2 instances) for Amazon EKS Kubernetes clusters.
		// Docs: https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html

		// Create Node Group 1
		// Create a Security Group for EKS worker nodes
		eksWorkerSgName := NewPrefixName("eks-worker-sg")
		tags["Name"] = eksWorkerSgName
		tags["karpenter.sh/discovery"] = eksClusterName
		eksWorkerSg, err := ec2.NewSecurityGroup(ctx, eksWorkerSgName, &ec2.SecurityGroupArgs{
			VpcId: vpc.vpcId,
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol: pulumi.String("tcp"),
					FromPort: pulumi.Int(443),
					ToPort:   pulumi.Int(443),
					SecurityGroups: pulumi.StringArray{
						eksAlbSg.ID().ToStringOutput(),
					},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol: pulumi.String("tcp"),
					FromPort: pulumi.Int(1025),
					ToPort:   pulumi.Int(65535),
					SecurityGroups: pulumi.StringArray{
						eksAlbSg.ID().ToStringOutput(),
					},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("10.0.0.0/16")},
				},
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}
		ctx.Export("eksWorkerSgId", eksWorkerSg.ID())

		eksWorkerGroup1Name := NewPrefixName("worker-group-1")
		ctx.Export("eksWorkerGroup1Name", pulumi.String(eksWorkerGroup1Name))

		// EC2 launch template for eks worker
		// https://www.pulumi.com/registry/packages/aws/api-docs/ec2/launchtemplate/#inputs
		// Create eks worker group 1 user data
		eksWorkerGroup1Ami := getEnv(ctx, "eks:ami-id", "unknown")
		eksClusterCA := eksCluster.CertificateAuthority.Data().Elem().ApplyT(func(caData string) string {
			return caData
		})
		eksClusterEP := eksCluster.Endpoint.ApplyT(func(endPoint string) string {
			return endPoint
		})
		eksWorkerGroup1UserData := pulumi.All(eksClusterCA, eksClusterEP).ApplyT(
			func(args []interface{}) (string, error) {
				caData := args[0].(string)
				ep := args[1].(string)
				tempData := fmt.Sprintf(`Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="userdata.txt"

#!/bin/bash
set -ex
B64_CLUSTER_CA=%s
API_SERVER_URL=%s
DNS_CLUSTER_IP=172.20.0.10
/etc/eks/bootstrap.sh %s --b64-cluster-ca $B64_CLUSTER_CA --apiserver-endpoint $API_SERVER_URL --dns-cluster-ip $DNS_CLUSTER_IP --kubelet-extra-args '--node-labels=karpenter.sh/discovery=%s,managed-by=karpenter,eks.amazonaws.com/nodegroup-image=%s,eks.amazonaws.com/capacityType=ON_DEMAND,eks.amazonaws.com/nodegroup=%s'
--//--
`, caData, ep, eksClusterName, eksClusterName, eksWorkerGroup1Ami, eksWorkerGroup1Name)

				eksKubeconfig := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: %s
  name: eks-cluster
contexts:
- context:
    cluster: eks-cluster
    user: eks-cluster
  name: eks-cluster
current-context: eks-cluster
kind: Config
preferences: {}
users:
- name: eks-cluster
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - --region
      - eu-west-1
      - eks
      - get-token
      - --cluster-name
      - %s
      command: aws
      env: null
`, caData, ep, eksClusterName)
				err := ioutil.WriteFile(".kubeconfig", []byte(eksKubeconfig), 0600)
				if err != nil {
					panic(err)
				}

				return string(base64.StdEncoding.EncodeToString([]byte(tempData))), nil
			},
		)

		// Create eks woker group 1 launch template
		eksWorkerGroup1LaunchTemplateName := NewPrefixName("worker-group-1-launch-template")
		eksWorkerGroup1LaunchTemplate, err := ec2.NewLaunchTemplate(ctx, eksWorkerGroup1LaunchTemplateName, &ec2.LaunchTemplateArgs{
			ImageId:      pulumi.String(eksWorkerGroup1Ami),
			InstanceType: pulumi.String("t3.medium"),
			VpcSecurityGroupIds: pulumi.StringArray{
				eksclusterSecurityGroupId,
				eksWorkerSg.ID().ToStringOutput(),
			},
			TagSpecifications: ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags:         pulumi.ToStringMap(tags),
				},
			},
			UserData: eksWorkerGroup1UserData.(pulumi.StringOutput),
		}, pulumi.DependsOn([]pulumi.Resource{eksCluster}))
		if err != nil {
			return err
		}
		ctx.Export("eksWorkerGroup1LaunchTemplate", eksWorkerGroup1LaunchTemplate.ID())

		// Create eks worker group 1
		var privsubnetIds pulumi.StringArray
		tags["Name"] = eksWorkerGroup1Name
		tags[fmt.Sprintf("kubernetes.io/cluster/%s", eksClusterName)] = "owned"
		eksWorkerGroup1, err := eks.NewNodeGroup(ctx, eksWorkerGroup1Name, &eks.NodeGroupArgs{
			ClusterName:   eksCluster.Name,
			NodeGroupName: pulumi.String(eksWorkerGroup1Name),
			NodeRoleArn:   pulumi.StringInput(karpenterInstanceNodeRole.Arn),
			SubnetIds: pulumi.StringArray(
				append(
					privsubnetIds,
					privSubnet1.subnetId,
					privSubnet2.subnetId,
					privSubnet3.subnetId,
				),
			),
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(3),
				MinSize:     pulumi.Int(1),
				MaxSize:     pulumi.Int(3),
			},
			LaunchTemplate: &eks.NodeGroupLaunchTemplateArgs{
				Version: pulumi.String("$Latest"),
				Id:      eksWorkerGroup1LaunchTemplate.ID().ToStringOutput(),
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

		// Export eks worker group 1 autoscaling group name
		eksWg1AsgName := eksWorkerGroup1.Resources.Index(pulumi.Int(0)).AutoscalingGroups().Index(pulumi.Int(0)).Name().Elem()
		ctx.Export("eksWg1AsgName", eksWg1AsgName)

		return nil
	})

}
