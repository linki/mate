package aws

type Record struct {
	HostedZoneID *string //Hosted Zone ID, as  set by AWS
	GroupID      *string //Group ID for the TXT record
}
