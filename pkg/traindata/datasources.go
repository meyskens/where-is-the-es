package traindata

type DataSource int

const (
	DataSourceNMBS DataSource = iota
	DataSourceNS
	DataSourceDB
	DataSourceCD
	DataSourceSZ
	DataSourceSNCFGC
	DataSourceArenaways
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
	case DataSourceSZ:
		return "SŽ"
	case DataSourceSNCFGC:
		return "sncf-gc"
	case DataSourceArenaways:
		return "arenaways"
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
