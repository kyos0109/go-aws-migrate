package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// VPCSync ...
type VPCSync struct {
	tagsConfig []Tag
}

// VPCSyncGO ...
func VPCSyncGO(awsAccount *AWSAccount) {
	var vpcSync VPCSync

	vpcSync.tagsConfig = awsAccount.Tags

	srcVPC := getVPCsInfo(&awsAccount.Source)[0]

	vpcSync.createVPC(&awsAccount.Destination, srcVPC)

	log.Print("VPC Migrate Done.")
}

func getVPCsInfo(account *awsAuth) []ec2.Vpc {
	svc := newSVC(account)

	req := svc.DescribeVpcsRequest(&ec2.DescribeVpcsInput{
		VpcIds: []string{account.VIPCID},
	})
	result, err := req.Send(context.Background())
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return result.Vpcs
}

func (vpcSync *VPCSync) createVPC(account *awsAuth, vpcInfo ec2.Vpc) {
	svc := newSVC(account)

	req := svc.CreateVpcRequest(&ec2.CreateVpcInput{
		CidrBlock:         vpcInfo.CidrBlock,
		TagSpecifications: vpcSync.setSGTags(vpcInfo.Tags, ec2.ResourceTypeVpc),
	})
	_, err := req.Send(context.Background())
	if err != nil {
		log.Fatalln(err)
		return
	}
}

func (vpcSync *VPCSync) setSGTags(tags []ec2.Tag, resourceType ec2.ResourceType) []ec2.TagSpecification {
	tagList := &ec2.TagSpecification{}
	timeTag := &ec2.Tag{}

	timeTag.Key = aws.String("CreateAt")
	timeTag.Value = aws.String(time.Now().String())
	tags = append(tags, *timeTag)

	configTags := make([]ec2.Tag, len(vpcSync.tagsConfig))
	for i, v := range vpcSync.tagsConfig {
		configTags[i] = ec2.Tag{
			Key:   aws.String(v.Key),
			Value: aws.String(v.Value),
		}
	}
	tags = append(tags, configTags...)

	if len(tags) > 0 {
		tagList = &ec2.TagSpecification{
			Tags:         tags,
			ResourceType: resourceType,
		}
	}
	return []ec2.TagSpecification{*tagList}
}
