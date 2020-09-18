# go-aws-migrate

The command migrate the aws some services to another aws account.

Support Security Groups Include Security Group ID.

Support Security Groups Include Managed prefix lists.

Support Update(Revoke) Sync Security Groups.

Support Route53.

example config.yaml
```yaml
Setting:
  DryRun: false
  Tags:
    - Key: "Project"
      Value: "Demo"
    - Key: "Creator"
      Value: "aws-sdk-go-v2"
  Source:
    AccessKey: "AccessKey"
    SecretKey: "SecretKey"
    Region: "ap-southeast-1"
    VPCID: "VPCID"
    HostedZoneID: "Hosted Zone ID"
  Destination:
    AccessKey: "AccessKey"
    SecretKey: "SecretKey"
    Region: "ap-east-1"
    VPCID: "VPCID"
    HostedZoneID: "Hosted Zone ID" # Optional, not exsits, auto create it.
```
