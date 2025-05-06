package traindata

type VehicleType int

const (
	VehicleTypeUnknown VehicleType = iota
	VehicleTypeLocomotive
	VehicleTypeCouchette
	VehicleTypeSleeper
	VehicleTypeBikeCouchette
	VehicleTypeBikeBistro
)

func (v VehicleType) String() string {
	switch v {
	case VehicleTypeLocomotive:
		return "Locomotive"
	case VehicleTypeCouchette:
		return "Couchette"
	case VehicleTypeSleeper:
		return "Sleeper"
	case VehicleTypeBikeCouchette:
		return "Bike Couchette"
	case VehicleTypeBikeBistro:
		return "Bike Bistro"
	default:
		return "Unknown"
	}
}

type Composition struct {
	Vehicles map[string]VehicleType
	Order    []string
}

func (c *Composition) ToBrowser() CompositionBrowser {
	vehicles := make([]CompositionBrowserVehicle, 0, len(c.Order))

	for _, number := range c.Order {
		vehicles = append(vehicles, CompositionBrowserVehicle{
			Type:   c.Vehicles[number].String(),
			Number: number,
		})
	}

	return CompositionBrowser{
		Vehicles: vehicles,
	}
}

type CompositionBrowser struct {
	Vehicles []CompositionBrowserVehicle `json:"vehicles"`
}

type CompositionBrowserVehicle struct {
	Type   string `json:"type"`
	Number string `json:"number"`
}
