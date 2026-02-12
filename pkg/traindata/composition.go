package traindata

import (
	"strconv"
)

type VehicleType int

const (
	VehicleTypeUnknown VehicleType = iota
	VehicleTypeLocomotive
	VehicleTypeCouchette
	VehicleTypeSleeper
	VehicleTypeBikeCouchette
	VehicleTypeBikeBistro
	VehicleTypeSeats
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
	case VehicleTypeSeats:
		return "Seats"
	default:
		return "Unknown"
	}
}

type Composition struct {
	Vehicles          map[string]VehicleType
	Order             []string
	UICNumbers        map[int]string
	UICNumbersOnOrder []string
}

func (c *Composition) ToBrowser() CompositionBrowser {
	vehicles := make([]CompositionBrowserVehicle, 0, len(c.Order))

	i := 0
	for _, number := range c.Order {
		uicNum := ""
		needsOrder := false
		iNum, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			needsOrder = true
		}
		if !needsOrder {
			n, ok := c.UICNumbers[int(iNum)]
			if ok {
				uicNum = n

			} else {
				needsOrder = true
			}
		}

		if needsOrder && i >= 2 && i-2 < len(c.UICNumbersOnOrder) {
			uicNum = c.UICNumbersOnOrder[i-2]
		}
		vehicles = append(vehicles, CompositionBrowserVehicle{
			Type:      c.Vehicles[number].String(),
			Number:    number,
			UICNumber: uicNum,
		})
		i++
	}

	return CompositionBrowser{
		Vehicles: vehicles,
	}
}

type CompositionBrowser struct {
	Vehicles []CompositionBrowserVehicle `json:"vehicles"`
}

type CompositionBrowserVehicle struct {
	Type      string `json:"type"`
	Number    string `json:"number"`
	UICNumber string `json:"uicNumber"`
}
