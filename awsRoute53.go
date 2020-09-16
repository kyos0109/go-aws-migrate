package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/route53"
)

type route53Sync struct {
	dstHostedZone    *route53.HostedZone
	srcHostedZone    *route53.HostedZone
	srcRecordListRes *route53.ListResourceRecordSetsResponse
}

func newRoute53SVC(account *awsAuth) *route53.Client {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithCredentialsProvider{
			CredentialsProvider: aws.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     account.AccessKey,
					SecretAccessKey: account.SecretKey,
					Source:          "config file",
					// SessionToken:    "",
				},
			},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config, %v", err)
		os.Exit(1)
	}

	cfg.Region = account.Region

	// Credentials retrieve will be called automatically internally to the SDK
	// service clients created with the cfg value.
	_, err = cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get credentials, %v", err)
		os.Exit(1)
	}

	return route53.New(cfg)
}

func getDNSRecordList(awsAuth *awsAuth) (*route53.ListResourceRecordSetsResponse, *route53.HostedZone) {
	svc := newRoute53SVC(awsAuth)

	reqGet := svc.GetHostedZoneRequest(&route53.GetHostedZoneInput{
		Id: &awsAuth.HostedZoneID,
	})
	hostZone, err := reqGet.Send(context.Background())
	if err != nil {
		log.Println(err.Error())
		return nil, nil
	}

	listParams := &route53.ListResourceRecordSetsInput{
		HostedZoneId: &awsAuth.HostedZoneID, // Required
		// MaxItems:              aws.String("100"),
		// StartRecordIdentifier: aws.String("Sample update."),
		// StartRecordName:       aws.String("com."),
		// StartRecordType:       aws.String("CNAME"),
	}

	reqList := svc.ListResourceRecordSetsRequest(listParams)
	respList, err := reqList.Send(context.Background())
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		log.Println(err.Error())
		return nil, nil
	}

	// Pretty-print the response data.
	return respList, hostZone.HostedZone
}

func (r53sync *route53Sync) createHostedZone(awsAuth *awsAuth) *route53.HostedZone {
	svc := newRoute53SVC(awsAuth)

	reqCreate := svc.CreateHostedZoneRequest(&route53.CreateHostedZoneInput{
		CallerReference: aws.String(time.Now().String()),
		VPC: &route53.VPC{
			VPCId:     &awsAuth.VIPCID,
			VPCRegion: route53.VPCRegion(awsAuth.Region),
		},
		HostedZoneConfig: &route53.HostedZoneConfig{
			Comment:     r53sync.srcHostedZone.Config.Comment,
			PrivateZone: r53sync.srcHostedZone.Config.PrivateZone,
		},
		Name: r53sync.srcHostedZone.Name,
	})
	result, err := reqCreate.Send(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	return result.HostedZone
}

func (r53sync *route53Sync) createRecord(account *awsAuth) {
	svc := newRoute53SVC(account)

	rrChangeList := []route53.Change{}

	for _, v := range r53sync.srcRecordListRes.ResourceRecordSets {
		rrChange := route53.Change{
			Action: route53.ChangeActionCreate,
			ResourceRecordSet: &route53.ResourceRecordSet{
				HealthCheckId:           v.HealthCheckId,
				TrafficPolicyInstanceId: v.TrafficPolicyInstanceId,
				Failover:                v.Failover,
				Region:                  v.Region,
				AliasTarget:             v.AliasTarget,
				GeoLocation:             v.GeoLocation,
				MultiValueAnswer:        v.MultiValueAnswer,
				Name:                    v.Name,
				Type:                    v.Type,
				ResourceRecords:         v.ResourceRecords,
				TTL:                     v.TTL,
				Weight:                  v.Weight,
				SetIdentifier:           v.SetIdentifier,
			},
		}

		rrChangeList = append(rrChangeList, rrChange)
	}

	log.Print("Create Resource Record.")

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: rrChangeList,
			Comment: aws.String("Create By aws-golang-sdk-v2, " + time.Now().String()),
		},
		HostedZoneId: r53sync.dstHostedZone.Id,
	}

	req := svc.ChangeResourceRecordSetsRequest(params)
	_, err := req.Send(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

}

func (r53sync *route53Sync) removeSrcDefaultRecord() {
	for i, v := range r53sync.srcRecordListRes.ResourceRecordSets {
		if v.Type == route53.RRTypeNs {
			copy(r53sync.srcRecordListRes.ResourceRecordSets[i:], r53sync.srcRecordListRes.ResourceRecordSets[i+1:])
			r53sync.srcRecordListRes.ResourceRecordSets[len(r53sync.srcRecordListRes.ResourceRecordSets)-1] = route53.ResourceRecordSet{}
			r53sync.srcRecordListRes.ResourceRecordSets = r53sync.srcRecordListRes.ResourceRecordSets[:len(r53sync.srcRecordListRes.ResourceRecordSets)-1]
		}
	}

	for i, v := range r53sync.srcRecordListRes.ResourceRecordSets {
		if v.Type == route53.RRTypeSoa {
			copy(r53sync.srcRecordListRes.ResourceRecordSets[i:], r53sync.srcRecordListRes.ResourceRecordSets[i+1:])
			r53sync.srcRecordListRes.ResourceRecordSets[len(r53sync.srcRecordListRes.ResourceRecordSets)-1] = route53.ResourceRecordSet{}
			r53sync.srcRecordListRes.ResourceRecordSets = r53sync.srcRecordListRes.ResourceRecordSets[:len(r53sync.srcRecordListRes.ResourceRecordSets)-1]
		}
	}
}

// Route53SyncGO ...
func Route53SyncGO(awsAccount *AWSAccount) {
	var r53sync route53Sync

	r53sync.srcRecordListRes, r53sync.srcHostedZone = getDNSRecordList(&awsAccount.Source)

	r53sync.removeSrcDefaultRecord()

	if len(awsAccount.Destination.HostedZoneID) > 0 {
		_, r53sync.dstHostedZone = getDNSRecordList(&awsAccount.Destination)
	} else {
		log.Println("Not Host Zone, Ceate It.")
		r53sync.dstHostedZone = r53sync.createHostedZone(&awsAccount.Destination)
		r53sync.createRecord(&awsAccount.Destination)
	}

	log.Print("Done.")
}
