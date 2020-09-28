# go-aws-migrate

The command migrate the aws some services to another aws account.

* Support Security Groups Include Security Group ID.

* Support Security Groups Include Managed prefix lists.

* Support Update(Revoke) Sync Security Groups.

* Support Security Groups Export.

* Support Security Groups Resotre (plan).

* Support Security Groups Export Terraform (UnSupported PrefixList [#13986](https://github.com/terraform-providers/terraform-provider-aws/issues/13986 target="_blank"))

* Support Route53.


# Command
```bash
NAME:
   AWS Migrate Tools - Command Line

USAGE:
   go-aws-migrate [global options] command [command options] [arguments...] 

COMMANDS:
   Route53, r53       Route53 Copy
   SecurityGroup, sg  Security Groups Sync
   help, h            Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE, -c FILE  Load configuration from FILE (default: "config.yaml")
   --help, -h              show help (default: false)
   --version, -v           print the version (default: false)
```


# config.yaml
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
