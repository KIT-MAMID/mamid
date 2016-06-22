package masterslaveprotocol

type HostPort struct {
	Hostname string
	Port uint
}

type Mongod struct {
	Port uint
	ReplSetName string
	Targets []HostPort
	CurrentError error
}