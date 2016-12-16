package pkg

//RecordInfo stores the information relevant to the record that were created
//mainly used to identify if the Route53 records needs to be updated
type RecordInfo struct {
	Target  string
	GroupID string
}
