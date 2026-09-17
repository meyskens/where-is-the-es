package traindata

type DataSource int

const (
	// DataSourceUnknown is the zero value of DataSource. It represents a
	// stop for which no data source has been selected as preferred. It MUST
	// stay first in this iota block so that unset PrefferedDataSource fields
	// do not accidentally read as DataSourceNMBS.
	DataSourceUnknown DataSource = iota
	DataSourceNMBS
	DataSourceNS
	DataSourceDB
	DataSourceCD
	DataSourceSZ
	DataSourceSNCFGC
	DataSourceArenaways
)

func (d DataSource) String() string {
	switch d {
	case DataSourceUnknown:
		return "unknown"
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
