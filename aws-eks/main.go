package main

import (
    "fmt"
    "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
    "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
    "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/autoscaling"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {

        prefix := "test"
        tags := make(map[string]string)
        tags["region"] = getEnv(ctx, "aws:region", "unknown")
        tags["version"] = getEnv(ctx, "tags:version", "unknown")
        tags["author"] = getEnv(ctx, "tags:author", "unknown")

        // Create new vpc
        cidrBlock := getEnv(ctx, "vpc:cidr-block", "unknown")
        vpcName := prefix + "-vpc"
        tags["Name"] = vpcName
        
        vpcArgs := &ec2.VpcArgs{
            CidrBlock:          pulumi.String(cidrBlock),
            EnableDnsHostnames: pulumi.Bool(true),
			InstanceTenancy:    pulumi.String("default"),
            Tags:               pulumi.ToStringMap(tags),
        }

        vpc, err := ec2.NewVpc(ctx, vpcName, vpcArgs)
        if err != nil {
            fmt.Println(err.Error())
            return err
        }

        ctx.Export("vpcId", vpc.ID())

        // Create subnets

        // Create 3 Private Subnets
        var privSubnetName string
        
        privSubnetName = prefix + "-priv-subnet-1"
		tags["Name"] = privSubnetName
		privSubnet1, err := ec2.NewSubnet(ctx, privSubnetName, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.1.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1a"),
			Tags:             pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        privSubnetName = prefix + "-priv-subnet-2"
		tags["Name"] = privSubnetName
		privSubnet2, err := ec2.NewSubnet(ctx, privSubnetName, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.2.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1b"),
			Tags:             pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        privSubnetName = prefix + "-priv-subnet-3"
		tags["Name"] = privSubnetName
		privSubnet3, err := ec2.NewSubnet(ctx, privSubnetName, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.3.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1c"),
			Tags:             pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        // Create 3 Public Subnets
        var pubSubnetName string
        pubSubnetName = prefix + "-pub-subnet-1"
		tags["Name"] = pubSubnetName
		pubSubnet1, err := ec2.NewSubnet(ctx, pubSubnetName, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.4.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1a"),
			Tags:             pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        pubSubnetName = prefix + "-pub-subnet-2"
		tags["Name"] = pubSubnetName
		pubSubnet2, err := ec2.NewSubnet(ctx, pubSubnetName, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.5.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1b"),
			Tags:             pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        pubSubnetName = prefix + "-pub-subnet-3"
		tags["Name"] = pubSubnetName
		pubSubnet3, err := ec2.NewSubnet(ctx, pubSubnetName, &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.6.0/24"),
			AvailabilityZone: pulumi.String("eu-west-1c"),
			Tags:             pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        // Resource: Elastic IP
        // EIP for NAT GW
        eipName := prefix + "-eip1"
		eip1, err := ec2.NewEip(ctx, eipName, &ec2.EipArgs{
			Vpc: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

        // NAT Gateway with EIP
        netGwName := prefix + "-natgw"
		tags["Name"] = netGwName
		natGw1, err := ec2.NewNatGateway(ctx, netGwName, &ec2.NatGatewayArgs{
			AllocationId: eip1.ID(),
			// NAT must reside in public subnet for private instance internet access
			SubnetId: pubSubnet1.ID(),
			Tags:     pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        // IGW for the Public Subnets
        igwName := prefix + "-igw"
		tags["Name"] = igwName
		igw1, err := ec2.NewInternetGateway(ctx, igwName, &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags:  pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        // Private Route Table for Private Subnets
        privRtbName := prefix + "-rtb-private-1"
		tags["Name"] = privRtbName
		privateRouteTable, err := ec2.NewRouteTable(ctx, privRtbName, &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					// To Internet via NAT
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: natGw1.ID(),
				},
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        // Public Route Table for Public Subnets
        pubRtbName := prefix + "-rtb-public-1"
		tags["Name"] = pubRtbName
		publicRouteTable, err := ec2.NewRouteTable(ctx, pubRtbName, &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				// To Internet via IGW
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igw1.ID(),
				},
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

        // Associate Private Subs with Private Route Tables
		_, err = ec2.NewRouteTableAssociation(ctx, prefix + "-rtb-assoc-priv-1", &ec2.RouteTableAssociationArgs{
			SubnetId:     privSubnet1.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, prefix + "-rtb-assoc-priv-2", &ec2.RouteTableAssociationArgs{
			SubnetId:     privSubnet2.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, prefix + "-rtb-assoc-priv-3", &ec2.RouteTableAssociationArgs{
			SubnetId:     privSubnet3.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		// Associate Public Subs with Public Route Tables
		_, err = ec2.NewRouteTableAssociation(ctx, prefix + "-rtb-assoc-pub-1", &ec2.RouteTableAssociationArgs{
			SubnetId:     pubSubnet1.ID(),
			RouteTableId: publicRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, prefix + "-rtb-assoc-pub-2", &ec2.RouteTableAssociationArgs{
			SubnetId:     pubSubnet2.ID(),
			RouteTableId: publicRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, prefix + "-rtb-assoc-pub-3", &ec2.RouteTableAssociationArgs{
			SubnetId:     pubSubnet3.ID(),
			RouteTableId: publicRouteTable.ID(),
		})
		if err != nil {
			return err
		}

        eksIamRoleName := prefix + "-eks-iam-eksRole"
        // IAM Role for EKS
		eksIamRole, err := iam.NewRole(ctx, eksIamRoleName, &iam.RoleArgs{
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
			_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf(prefix + "-rpa-%d", i), &iam.RolePolicyAttachmentArgs{
				PolicyArn: pulumi.String(eksPolicy),
				Role:      eksIamRole.Name,
			})
			if err != nil {
				return err
			}
		}

		// Create a Security Group for EKS
        clusterSgName := prefix + "-cluster-sg"
		tags["Name"] = clusterSgName
		clusterSg, err := ec2.NewSecurityGroup(ctx, clusterSgName, &ec2.SecurityGroupArgs{
			VpcId: vpc.ID(),
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
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}


        // Create an EKS cluster.
        var subnetIds pulumi.StringArray
        eksClusterName := prefix + "-cluster"
        
        eksCluster, err := eks.NewCluster(ctx, eksClusterName, &eks.ClusterArgs{
			Name: pulumi.String(eksClusterName),
			Version: pulumi.String("1.23"),
            RoleArn: pulumi.StringOutput(eksIamRole.Arn),
            VpcConfig: &eks.ClusterVpcConfigArgs{
				PublicAccessCidrs: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
				SecurityGroupIds: pulumi.StringArray{
					clusterSg.ID().ToStringOutput(),
				},
				SubnetIds: pulumi.StringArray(
					append(
						subnetIds,
						privSubnet1.ID(),
						privSubnet2.ID(),
						privSubnet3.ID(),
						pubSubnet1.ID(),
						pubSubnet2.ID(),
						pubSubnet3.ID(),
					),
                ),
			},
        })

        if err != nil {
            return err
        }

        // Export EKS cluster endpoint
        ctx.Export("eksEndpoint", eksCluster.Endpoint)

        // Create EKS NodeGroup Role
		nodeGroupRole, err := iam.NewRole(ctx, prefix + "-nodegroup-iam-role", &iam.RoleArgs{
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
		nodeGroupPolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		}
		for i, nodeGroupPolicy := range nodeGroupPolicies {
			_, err := iam.NewRolePolicyAttachment(ctx, fmt.Sprintf(prefix + "-ngpa-%d", i), &iam.RolePolicyAttachmentArgs{
				Role:      nodeGroupRole.Name,
				PolicyArn: pulumi.String(nodeGroupPolicy),
			})
			if err != nil {
				return err
			}
		}

		// Add EKS Node Groups

		var privsubnetIds pulumi.StringArray

		// Create EKS Node Group 1
        eksWorkerGroup1Name := prefix + "-worker-group-1"
		tags["Name"] = eksWorkerGroup1Name
		eksWorkerGroup1, err := eks.NewNodeGroup(ctx, eksWorkerGroup1Name, &eks.NodeGroupArgs{
			ClusterName: eksCluster.Name,
			SubnetIds: pulumi.StringArray(
				append(
					privsubnetIds,
					privSubnet1.ID(),
					privSubnet2.ID(),
					privSubnet3.ID(),
				),
            ),
			NodeRoleArn: pulumi.StringInput(nodeGroupRole.Arn),
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MinSize:     pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}

		eksWg1AsgName := eksWorkerGroup1.Resources.Index(pulumi.Int(0)).AutoscalingGroups().Index(pulumi.Int(0)).Name().Elem()

		ctx.Export("eksWg1AsgName", eksWg1AsgName)

		// Create EKS Node Group 2
        eksWorkerGroup2Name := prefix + "-worker-group-2"
		tags["Name"] = eksWorkerGroup2Name
		eksWorkerGroup2, err := eks.NewNodeGroup(ctx, eksWorkerGroup2Name, &eks.NodeGroupArgs{
			ClusterName: eksCluster.Name,
			SubnetIds: pulumi.StringArray(
				append(
					privsubnetIds,
					privSubnet1.ID(),
					privSubnet2.ID(),
					privSubnet3.ID(),
				)),
			NodeRoleArn: pulumi.StringInput(nodeGroupRole.Arn),
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(1),
				MinSize:     pulumi.Int(1),
				MaxSize:     pulumi.Int(1),
			},
			Tags: pulumi.ToStringMap(tags),
		})
		if err != nil {
			return err
		}
		eksWg2AsgName := eksWorkerGroup2.Resources.Index(pulumi.Int(0)).AutoscalingGroups().Index(pulumi.Int(0)).Name().Elem()

		ctx.Export("eksWg2AsgName", eksWg2AsgName)

		// Create load balancer target group for EKS
		eksTgName := prefix + "-eks-tg"
		eksTg, err := lb.NewTargetGroup(ctx, eksTgName, &lb.TargetGroupArgs{
			VpcId:    vpc.ID(),
			Port:     pulumi.Int(30443),
			Protocol: pulumi.String("HTTPS"),
			HealthCheck: &lb.TargetGroupHealthCheckArgs{
				Protocol:           pulumi.String("HTTPS"),
				Path:               pulumi.String("/health"),
				Matcher:            pulumi.String("200-299"),
				HealthyThreshold:   pulumi.Int(3),
				Interval:           pulumi.Int(30),
				UnhealthyThreshold: pulumi.Int(3),
				Timeout:            pulumi.Int(15),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("eksTgArn", eksTg.Arn)

		// Attach load balancer target group to EKS asg groups
		eksAsg1AttachementName := prefix + "-eks-asg1-attach"
		_, err = autoscaling.NewAttachment(ctx, eksAsg1AttachementName, &autoscaling.AttachmentArgs{
			AutoscalingGroupName: eksWg1AsgName,
			LbTargetGroupArn:     eksTg.Arn,
		})
		if err != nil {
			return err
		}
		
		eksAsg2AttachementName := prefix + "-eks-asg2-attach"
		_, err = autoscaling.NewAttachment(ctx, eksAsg2AttachementName, &autoscaling.AttachmentArgs{
			AutoscalingGroupName: eksWg2AsgName,
			LbTargetGroupArn:     eksTg.Arn,
		})
		if err != nil {
			return err
		}
        
		// Add ALB for EKS
		eksPubAlbName := prefix + "-eks-pub-alb"
		eksPubAlb, err := lb.NewLoadBalancer(ctx, eksPrivAlbName, &lb.LoadBalancerArgs{
			LoadBalancerType: pulumi.String("application"),
			Internal: pulumi.Bool(false),
			SubnetMappings: lb.LoadBalancerSubnetMappingArray{
				&lb.LoadBalancerSubnetMappingArgs{
					SubnetId:     pububnet1.ID(),
				},
				&lb.LoadBalancerSubnetMappingArgs{
					SubnetId:     pubSubnet2.ID(),
				},
				&lb.LoadBalancerSubnetMappingArgs{
					SubnetId:     pubSubnet3.ID(),
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("eksPubAlbArn", eksPubAlb.Arn)

		// Add HTTP Listender to forward traffic to EKS target group
		_, err = lb.NewListener(ctx, "httpListener", &lb.ListenerArgs{
			LoadBalancerArn: eksPubAlb.Arn,
			Port:            pulumi.Int(443),
			Protocol:        pulumi.String("HTTPS"),
			DefaultActions: lb.ListenerDefaultActionArray{
				&lb.ListenerDefaultActionArgs{
					Type:           pulumi.String("forward"),
					TargetGroupArn: eksTg.Arn,
				},
			},
		})
		
        return nil
    })

}

func getEnv(ctx *pulumi.Context, key string, fallback string) string {
    if value, ok := ctx.GetConfig(key); ok {
        return value
    }
    return fallback
}
