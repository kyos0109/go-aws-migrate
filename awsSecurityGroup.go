package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const (
	sgSelf        = "sg-self"
	sgDefaultName = "default"
)

// BuildSGMapType ...
type BuildSGMapType map[*string][]ec2.IpPermission

// SGIdNameMapType ...
type SGIdNameMapType map[string]string

var (
	// ID: Name
	sgIDMameMap = make(SGIdNameMapType)

	// Name: ID
	newSGNameIDMap = make(SGIdNameMapType)

	ipps  = make(BuildSGMapType)
	ippes = make(BuildSGMapType)

	ippEmpty = &ec2.IpPermission{}
)

func newSVC(account *awsAuth) *ec2.Client {
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

	return ec2.New(cfg)
}

// SecurityGroupSyncGO ...
func SecurityGroupSyncGO(awsAccount *AWSAccount) {
	var awssync AWSSync

	awssync.perfixListMap = make(map[string]*PerfixList)

	if len(sourceSGID) > 0 {
		awssync.sourceSGLists = GetFilterSGListByIds(&awsAccount.Source, sourceSGID)
		awssync.GetPerfixLists(&awsAccount.Source)
	} else {
		awssync.sourceSGLists = GetSGList(&awsAccount.Source)
		awssync.GetPerfixLists(&awsAccount.Source)
	}

	awssync.CreateAndSyncSGList(&awsAccount.Destination, &awsAccount.DryRun)
}

// GetFilterSGListByNames ...
func GetFilterSGListByNames(account *awsAuth, names ...string) []ec2.SecurityGroup {
	svc := newSVC(account)

	req := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{
		GroupNames: names,
	})
	result, err := req.Send(context.Background())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroupName.Malformed":
				fallthrough
			case "InvalidGroup.NotFound":
				exitErrorf("%s.", aerr.Message())
			}
		}
		exitErrorf("Unable to get descriptions for security groups, %v", err)
	}

	log.Println("Successfully get security group, filter by name")
	return result.SecurityGroups
}

// GetFilterSGListByIds ...
func GetFilterSGListByIds(account *awsAuth, groupIds ...string) []ec2.SecurityGroup {
	svc := newSVC(account)

	req := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{
		GroupIds: groupIds,
	})
	result, err := req.Send(context.Background())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroupId.Malformed":
				fallthrough
			case "InvalidGroup.NotFound":
				exitErrorf("%s.", aerr.Message())
			}
		}
		exitErrorf("Unable to get descriptions for security groups, %v", err)
	}

	log.Println("Successfully get security group, filter by id")
	return result.SecurityGroups
}

// GetSGList ...
func GetSGList(account *awsAuth) []ec2.SecurityGroup {
	svc := newSVC(account)

	req := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{})
	result, err := req.Send(context.Background())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroupId.Malformed":
				fallthrough
			case "InvalidGroup.NotFound":
				exitErrorf("%s.", aerr.Message())
			}
		}
		exitErrorf("Unable to get descriptions for security groups, %v", err)
	}

	log.Println("Successfully get security group list")
	return result.SecurityGroups
}

// PerfixList ...
type PerfixList struct {
	OldPerfixListID   *string
	newPerfixListID   *string
	ManagedPrefixList ec2.ManagedPrefixList
	PrefixListEntry   []ec2.PrefixListEntry
}

// GetPerfixLists ...
func (awssync *AWSSync) GetPerfixLists(account *awsAuth) {
	svc := newSVC(account)

	for _, sg := range awssync.sourceSGLists {
		for _, ipp := range sg.IpPermissions {
			for _, plids := range ipp.PrefixListIds {
				if _, ok := awssync.perfixListMap[*plids.PrefixListId]; ok {
					continue
				}

				p := new(PerfixList)
				p.OldPerfixListID = plids.PrefixListId
				req := svc.GetManagedPrefixListEntriesRequest(&ec2.GetManagedPrefixListEntriesInput{
					PrefixListId: p.OldPerfixListID,
				})
				result, err := req.Send(context.Background())
				if err != nil {
					log.Println("Get PerfixList Error", err)
				}

				p.PrefixListEntry = result.Entries

				desReq := svc.DescribeManagedPrefixListsRequest(&ec2.DescribeManagedPrefixListsInput{
					PrefixListIds: []string{*p.OldPerfixListID},
				})
				perfixListInfo, err := desReq.Send(context.Background())
				if err != nil {
					log.Println("Describe Perfix Error", err)
				}

				p.ManagedPrefixList = perfixListInfo.PrefixLists[0]

				awssync.perfixListMap[*p.OldPerfixListID] = p

				log.Printf("Found PerfixList: %v, Add To Sync Data", *p.OldPerfixListID)
			}
		}
	}
	return
}

func (awssync *AWSSync) createPerfixList(svc *ec2.Client) {
	for i, v := range awssync.perfixListMap {
		PerfixListAddr := convertAddPerfixList(v.PrefixListEntry)
		tags := []ec2.Tag{}

		req := svc.CreateManagedPrefixListRequest(&ec2.CreateManagedPrefixListInput{
			AddressFamily:     v.ManagedPrefixList.AddressFamily,
			Entries:           PerfixListAddr,
			PrefixListName:    v.ManagedPrefixList.PrefixListName,
			MaxEntries:        v.ManagedPrefixList.MaxEntries,
			TagSpecifications: setSGTags(tags, "prefix-list"),
		})
		result, err := req.Send(context.Background())
		if err != nil {
			log.Println("Create PerfixList Error", err)
		}

		awssync.perfixListMap[i].newPerfixListID = result.PrefixList.PrefixListId
	}
}

func convertAddPerfixList(plist []ec2.PrefixListEntry) []ec2.AddPrefixListEntry {
	var newPlist []ec2.AddPrefixListEntry

	for _, v := range plist {

		pAddr := &ec2.AddPrefixListEntry{
			Cidr:        v.Cidr,
			Description: v.Description,
		}

		newPlist = append(newPlist, *pAddr)
	}
	return newPlist
}

func deletSGDefaultValue(sgList []ec2.SecurityGroup) []ec2.SecurityGroup {
	// delete non vpc id default sg from source
	for i, sg := range sgList {
		if aws.StringValue(sg.GroupName) == "default" && aws.StringValue(sg.VpcId) == "" {
			log.Printf("This Source Security Group: %v(%v), Not Found VPC ID, Ignore SYNC.", *sg.GroupName, *sg.GroupId)
			copy(sgList[i:], sgList[i+1:])
			sgList[len(sgList)-1] = ec2.SecurityGroup{}
			sgList = sgList[:len(sgList)-1]
		}
	}

	// delete Security Group include Security Group ID, and copy to new map.
	for i, sg := range sgList {
		if len(sg.IpPermissions) == 0 {
			continue
		}

		ippSlice := []ec2.IpPermission{}
		ippeSlice := []ec2.IpPermission{}

		for _, ipp := range sg.IpPermissions {
			if len(ipp.UserIdGroupPairs) > 0 {
				if !updateMode && aws.StringValue(sg.GroupName) == sgDefaultName {
					continue
				}

				ipps[sg.GroupId] = append(ipps[sg.GroupId], ipp)

			} else {
				ippSlice = append(ippSlice, ipp)
			}
		}

		for _, ippe := range sg.IpPermissionsEgress {
			if len(ippe.UserIdGroupPairs) > 0 {
				if !updateMode && aws.StringValue(sg.GroupName) == sgDefaultName {
					continue
				}

				ippes[sg.GroupId] = append(ippes[sg.GroupId], ippe)

			} else {
				ippeSlice = append(ippeSlice, ippe)
			}
		}

		sgList[i].IpPermissions = ippSlice
		sgList[i].IpPermissionsEgress = ippeSlice
	}

	return sgList
}

func replaceGroupID(ippMap BuildSGMapType) BuildSGMapType {
	newBuildSG := make(BuildSGMapType)

	for gid, data := range ippMap {
		gName, ok := sgIDMameMap[aws.StringValue(gid)]
		if !ok {
			log.Println("Not Found Old Security Group ID In Map, ID:", *gid)
			break
		}

		newGID, ok := newSGNameIDMap[gName]
		if !ok {
			log.Println("Not Found New Security Group Name In Map, Name:", gName)
			break
		}

		newBuildSG[aws.String(newGID)] = data
	}

	for _, ipp := range newBuildSG {
		for ii, ips := range ipp {
			ugps := []ec2.UserIdGroupPair{}
			for _, ugp := range ips.UserIdGroupPairs {
				gName, ok := sgIDMameMap[aws.StringValue(ugp.GroupId)]
				if !ok {
					log.Println("Not Found Old Security Group ID In Map, From UserIdGroupPairs, ID:", *ugp.GroupId)
					break
				}

				newID, ok := newSGNameIDMap[gName]
				if !ok {
					log.Println("Not Found New Security Group Name In Map, From UserIdGroupPairs, Name:", gName)
					break
				}
				ugp.GroupId = aws.String(newID)
				ugps = append(ugps, ugp)
			}
			ipp[ii].UserIdGroupPairs = ugps
		}
	}

	return newBuildSG
}

func (awssync *AWSSync) replacePerfixListID(sgList []ec2.SecurityGroup) []ec2.SecurityGroup {
	for _, sg := range sgList {
		for _, ipp := range sg.IpPermissions {
			for _, plist := range ipp.PrefixListIds {
				plist.PrefixListId = awssync.perfixListMap[*plist.PrefixListId].newPerfixListID
			}
		}
	}
	return sgList
}

func setSGTags(tags []ec2.Tag, resourceType ec2.ResourceType) []ec2.TagSpecification {
	tagList := &ec2.TagSpecification{}
	timeTag := &ec2.Tag{}

	timeTag.Key = aws.String("CreateAt")
	timeTag.Value = aws.String(time.Now().String())

	tags = append(tags, *timeTag)

	if len(tags) > 0 {
		tagList = &ec2.TagSpecification{
			Tags:         tags,
			ResourceType: resourceType,
		}
	}
	return []ec2.TagSpecification{*tagList}
}

func appendSGUGPRule(svc *ec2.Client) {
	if len(ipps) > 0 {
		newUgpMap := replaceGroupID(ipps)

		for gid, ipps := range newUgpMap {
			req := svc.AuthorizeSecurityGroupIngressRequest(&ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:       gid,
				IpPermissions: ipps,
			})
			_, err := req.Send(context.Background())
			if err != nil {
				exitErrorf("Unable to append set security group %q ingress, %v", *gid, err)
			}
		}

	}

	if len(ippes) > 0 {
		newUgpMap := replaceGroupID(ippes)

		for gid, ipps := range newUgpMap {
			req := svc.AuthorizeSecurityGroupEgressRequest(&ec2.AuthorizeSecurityGroupEgressInput{
				GroupId:       gid,
				IpPermissions: ipps,
			})
			_, err := req.Send(context.Background())
			if err != nil {
				exitErrorf("Unable to append set security group %q Egress, %v", *gid, err)
			}
		}
	}
}

func updateSG(account *awsAuth, groupName string) *string {
	svc := newSVC(account)

	existSG := GetFilterSGListByNames(account, groupName)[0]

	if len(existSG.IpPermissions) > 0 {
		req := svc.RevokeSecurityGroupIngressRequest(&ec2.RevokeSecurityGroupIngressInput{
			GroupId:       existSG.GroupId,
			IpPermissions: existSG.IpPermissions,
		})
		_, err := req.Send(context.Background())
		if err != nil {
			exitErrorf("Unable to revoke security group %q Ingress, %v", *existSG.GroupId, err)
		}
	}

	if len(existSG.IpPermissionsEgress) > 0 {
		req := svc.RevokeSecurityGroupEgressRequest(&ec2.RevokeSecurityGroupEgressInput{
			GroupId:       existSG.GroupId,
			IpPermissions: existSG.IpPermissionsEgress,
		})
		_, err := req.Send(context.Background())
		if err != nil {
			exitErrorf("Unable to revoke security group %q Egress, %v", *existSG.GroupId, err)
		}
	}

	log.Printf("Successfully update security group %q", *existSG.GroupId)

	return existSG.GroupId
}

// CreateAndSyncSGList ...
func (awssync *AWSSync) CreateAndSyncSGList(account *awsAuth, dryRun *bool) {
	svc := newSVC(account)

	defaultSGID := GetFilterSGListByNames(account, sgDefaultName)[0]

	newSrcSGList := deletSGDefaultValue(awssync.sourceSGLists)

	awssync.createPerfixList(svc)

	newSrcSGList = awssync.replacePerfixListID(newSrcSGList)

	for _, sg := range newSrcSGList {
		var resGroupID *string

		sgIDMameMap[*sg.GroupId] = *sg.GroupName

		req := svc.CreateSecurityGroupRequest(&ec2.CreateSecurityGroupInput{
			DryRun:            dryRun,
			GroupName:         sg.GroupName,
			Description:       sg.Description,
			VpcId:             aws.String(account.VIPCID),
			TagSpecifications: setSGTags(sg.Tags, ec2.ResourceTypeSecurityGroup),
		})
		createRes, err := req.Send(context.Background())
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case "InvalidVpcID.NotFound":
					exitErrorf("Unable to find VPC with ID %q.", account.VIPCID)
				case "InvalidGroup.Duplicate":
					if updateMode {
						resGroupID = updateSG(account, *sg.GroupName)
						break
					}
					exitErrorf("Security group %q already exists.", *sg.GroupName)
				case "DryRunOperation":
					exitErrorf("DryRunOperation to create security group %q, %v", *sg.GroupName, err)
				case "InvalidParameterValue":
					if aws.StringValue(sg.GroupName) == sgDefaultName {
						log.Print("Found default group, switch id.")
						if updateMode {
							updateSG(account, sgDefaultName)
						}
						resGroupID = defaultSGID.GroupId
						break
					}
					exitErrorf("InvalidParameterValue to create security group %q, %v", *sg.GroupName, err)
				default:
					exitErrorf("Unable to create security group %q, %v", *sg.GroupName, err)
				}
			}
		} else {
			resGroupID = createRes.GroupId
		}

		// clean create Security Group default value
		if !updateMode {
			reqRevoke := svc.RevokeSecurityGroupEgressRequest(&ec2.RevokeSecurityGroupEgressInput{
				GroupId: resGroupID,
				IpPermissions: []ec2.IpPermission{
					{
						FromPort:   aws.Int64(-1),
						IpProtocol: aws.String("-1"),
						IpRanges: []ec2.IpRange{
							{
								CidrIp: aws.String("0.0.0.0/0"),
							},
						},
					},
				},
			})
			_, err = reqRevoke.Send(context.Background())
			if err != nil {
				exitErrorf("Revoke Default Security Group Rule Egress Error", err)
			}
		}

		// all new security group id
		newSGNameIDMap[*sg.GroupName] = *resGroupID

		if !updateMode {
			log.Printf("Created security group %s(%s) with VPC %s.\n",
				aws.StringValue(sg.GroupName), aws.StringValue(resGroupID), account.VIPCID)
		}

		if len(sg.IpPermissions) > 0 {
			req := svc.AuthorizeSecurityGroupIngressRequest(&ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:       resGroupID,
				IpPermissions: setSelfSecurityGroupID(sg.IpPermissions, resGroupID),
			})
			_, err := req.Send(context.Background())
			if err != nil {
				exitErrorf("Unable to set security group %q ingress, %v", *sg.GroupName, err)
			}
		}

		if len(sg.IpPermissionsEgress) > 0 {
			req := svc.AuthorizeSecurityGroupEgressRequest(&ec2.AuthorizeSecurityGroupEgressInput{
				GroupId:       resGroupID,
				IpPermissions: setSelfSecurityGroupID(sg.IpPermissionsEgress, resGroupID),
			})
			_, err := req.Send(context.Background())
			if err != nil {
				exitErrorf("Unable to set security group %q Egress, %v", *sg.GroupName, err)
			}
		}
	}

	appendSGUGPRule(svc)

	log.Println("Successfully set security group ingress")
}

func setSelfSecurityGroupID(ips []ec2.IpPermission, groupID *string) []ec2.IpPermission {
	for _, ipp := range ips {
		for _, ugp := range ipp.UserIdGroupPairs {
			if aws.StringValue(ugp.GroupId) == sgSelf {
				ugp.GroupId = groupID
			}
		}
	}
	return ips
}

// CleanSecurityGroupRule ...
func CleanSecurityGroupRule(account *awsAuth) {
	c := askForConfirmation("Doooooooooooooooooooooooooooooooooooooon't, Are You Sure?")
	if !c {
		fmt.Println("Bye...")
		os.Exit(0)
	}

	cc := askForConfirmation(fmt.Sprintf("AccessKey: %s, Sure?", account.AccessKey))
	if !cc {
		fmt.Println("Bye...")
		os.Exit(0)
	}

	svc := newSVC(account)

	req := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{})
	result, err := req.Send(context.Background())
	if err != nil {
		exitErrorf("Get All Security Groups Failed", err)
	}

	log.Print("Do It.")

	for _, v := range result.SecurityGroups {
		req := svc.RevokeSecurityGroupIngressRequest(&ec2.RevokeSecurityGroupIngressInput{
			GroupId:       v.GroupId,
			IpPermissions: v.IpPermissions,
		})
		_, err := req.Send(context.Background())
		if err != nil {
			log.Println("Revoke Security Group Ingress Error", err)
		}

		reqE := svc.RevokeSecurityGroupEgressRequest(&ec2.RevokeSecurityGroupEgressInput{
			GroupId:       v.GroupId,
			IpPermissions: v.IpPermissionsEgress,
		})
		_, err = reqE.Send(context.Background())
		if err != nil {
			log.Println("Revoke Security Group Egress Error", err)
		}
	}
	log.Print("Done.")
	os.Exit(0)
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
