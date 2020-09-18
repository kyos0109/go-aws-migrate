package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// SubnetSync ...
type SubnetSync struct {
	tagsConfig []Tag
	srcVPCID   string
}

// SubnetSyncGO ...
func SubnetSyncGO(awsAccount *AWSAccount) {
	var subnetSync SubnetSync

	subnetSync.tagsConfig = awsAccount.Tags
	subnetSync.srcVPCID = awsAccount.Source.VIPCID

	subnets := getSubnetsInfo(&awsAccount.Source)
	newSubnets := subnetSync.filterSubnetByVPCID(subnets)

	subnetSync.createSubnets(&awsAccount.Destination, newSubnets)

	log.Print("Subnet Migrate Done.")
}

func (subnetSync *SubnetSync) createSubnets(account *awsAuth, subnets []ec2.Subnet) {
	svc := newSVC(account)

	for _, subnet := range subnets {
		req := svc.CreateSubnetRequest(&ec2.CreateSubnetInput{
			AvailabilityZoneId: subnet.AvailabilityZoneId,
			CidrBlock:          subnet.CidrBlock,
			VpcId:              subnet.VpcId,
			TagSpecifications:  subnetSync.setSGTags(subnet.Tags, ec2.ResourceTypeSubnet),
		})
		_, err := req.Send(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func getSubnetsInfo(account *awsAuth) []ec2.Subnet {
	svc := newSVC(account)

	req := svc.DescribeSubnetsRequest(&ec2.DescribeSubnetsInput{})
	result, err := req.Send(context.Background())
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return result.Subnets
}

func (subnetSync *SubnetSync) filterSubnetByVPCID(subnets []ec2.Subnet) []ec2.Subnet {
	var newSubnets []ec2.Subnet

	for _, subnet := range subnets {
		if aws.StringValue(subnet.VpcId) == subnetSync.srcVPCID {
			newSubnets = append(newSubnets, subnet)
		}
	}

	return newSubnets
}

func (subnetSync *SubnetSync) setSGTags(tags []ec2.Tag, resourceType ec2.ResourceType) []ec2.TagSpecification {
	tagList := &ec2.TagSpecification{}
	timeTag := &ec2.Tag{}

	timeTag.Key = aws.String("CreateAt")
	timeTag.Value = aws.String(time.Now().String())
	tags = append(tags, *timeTag)

	configTags := make([]ec2.Tag, len(subnetSync.tagsConfig))
	for i, v := range subnetSync.tagsConfig {
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
