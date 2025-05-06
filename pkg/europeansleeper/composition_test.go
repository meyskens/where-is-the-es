package europeansleeper

import (
	"testing"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"github.com/stretchr/testify/assert"
)

func Test_parseComposition(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		trainNumber string
		want        traindata.Composition
		wantErr     bool
	}{
		{
			name:        "parses valid composition",
			html:        testCompositionPage,
			trainNumber: "453",
			want: traindata.Composition{
				Vehicles: map[string]traindata.VehicleType{
					"LOC": traindata.VehicleTypeLocomotive,
					"-":   traindata.VehicleTypeCouchette,
					"21":  traindata.VehicleTypeCouchette,
					"20":  traindata.VehicleTypeCouchette,
					"19":  traindata.VehicleTypeCouchette,
					"18":  traindata.VehicleTypeSleeper,
					"17":  traindata.VehicleTypeSleeper,
					"15":  traindata.VehicleTypeCouchette,
					"14":  traindata.VehicleTypeCouchette,
					"13":  traindata.VehicleTypeCouchette,
					"12":  traindata.VehicleTypeCouchette,
					"11":  traindata.VehicleTypeCouchette,
					"10":  traindata.VehicleTypeCouchette,
					"9":   traindata.VehicleTypeCouchette,
				},
				Order: []string{"LOC", "-", "21", "20", "19", "18", "17", "15", "14", "13", "12", "11", "10", "9"},
			},
			wantErr: false,
		},
		{
			name:        "parses valid composition part 2",
			html:        testCompositionPage,
			trainNumber: "452",
			want: traindata.Composition{
				Vehicles: map[string]traindata.VehicleType{
					"LOC": traindata.VehicleTypeLocomotive,
					"-":   traindata.VehicleTypeCouchette,
					"20":  traindata.VehicleTypeCouchette,
					"19":  traindata.VehicleTypeCouchette,
					"18":  traindata.VehicleTypeSleeper,
					"17":  traindata.VehicleTypeCouchette,
					"15":  traindata.VehicleTypeCouchette,
					"14":  traindata.VehicleTypeCouchette,
					"13":  traindata.VehicleTypeCouchette,
					"12":  traindata.VehicleTypeCouchette,
					"11":  traindata.VehicleTypeCouchette,
					"10":  traindata.VehicleTypeCouchette,
					"9":   traindata.VehicleTypeCouchette,
				},
				Order: []string{"LOC", "9", "10", "11", "12", "13", "14", "15", "17", "18", "19", "20", "-", "-"},
			},
			wantErr: false,
		},
		{
			name:        "returns error for invalid HTML",
			html:        "invalid HTML",
			trainNumber: "123",
			wantErr:     true,
		},
		{
			name:        "returns error when train not found",
			html:        "<h3>Train ES 456</h3>",
			trainNumber: "123",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseComposition([]byte(tt.html), tt.trainNumber)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
