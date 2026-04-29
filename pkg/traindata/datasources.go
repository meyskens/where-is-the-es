package traindata

type DataSource int

const (
	DataSourceNMBS DataSource = iota
	DataSourceNS
	DataSourceDB
	DataSourceCD
)

func (d DataSource) String() string {
	switch d {
	case DataSourceNMBS:
		return "nmbs"
	case DataSourceNS:
		return "ns"
	case DataSourceDB:
		return "db"
	case DataSourceCD:
		return "cd"
	default:
		return "unknown"
	}
}

func DataSourcesToStrings(sources []DataSource) []string {
	result := make([]string, len(sources))
	for i, s := range sources {
		result[i] = s.String()
	}
	return result
}
