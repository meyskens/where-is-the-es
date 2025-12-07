package europeansleeper

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/net/html"
)

func GetComposition(trainNumber string) (traindata.Composition, error) {
	// get https://europeansleeper.eu/train-composition

	resp, err := http.Get("https://europeansleeper.eu/train-composition")
	if err != nil {
		return traindata.Composition{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return traindata.Composition{}, errors.New("failed to get train composition, status code: " + resp.Status)
	}

	// read the response body
	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return traindata.Composition{}, err
	}

	return parseComposition(data, trainNumber)
}

func parseComposition(data []byte, trainNumber string) (traindata.Composition, error) {
	z := html.NewTokenizer(bytes.NewReader(data))

	// look for <h3>Train ES $TRAIN_NUM</h3>

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return traindata.Composition{}, errors.New("failed to parse train composition")
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "h3" {
				z.Next()
				if strings.Contains(string(z.Text()), "Train ES "+trainNumber) {
					break
				}
			}
		}
	}

	composition := traindata.Composition{
		Vehicles: make(map[string]traindata.VehicleType),
		Order:    []string{},
	}

L:
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			t := z.Token()
			if t.Data == "div" {
				vehicleType, number, err := parseVehicle(z)
				if err != nil {
					return traindata.Composition{}, err
				}
				composition.Vehicles[number] = vehicleType
				composition.Order = append(composition.Order, number)
			}
			if t.Data == "h3" {
				break // we are at the next train
			}
			// if <hr class="margin-y mob"> is found, this is the end
			if t.Data == "hr" {
				for _, a := range t.Attr {
					if a.Key == "class" && a.Val == "margin-y mob" {
						break L
					}
				}
			}
		}
	}

	return composition, nil
}

func parseVehicle(z *html.Tokenizer) (traindata.VehicleType, string, error) {
	var vehicleType traindata.VehicleType
	var vehicleNumber string

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return 0, "", errors.New("failed to parse vehicle")
		}
		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "h3" {
				z.Next()
				vehicleNumber = string(z.Text())
			}
			isClassText := false
			for _, a := range t.Attr {
				if a.Key == "class" && a.Val == "text-xs" {
					isClassText = true
					break
				}
			}
			if t.Data == "span" && isClassText {
				z.Next()
				text := string(z.Text())
				switch text {
				case "Locomotive":
					vehicleType = traindata.VehicleTypeLocomotive
					vehicleNumber = "LOC"
				case "Couchette", "Classic", "Comfort Standard":
					vehicleType = traindata.VehicleTypeCouchette
				case "Sleeper", "Comfort", "Comfort Plus":
					vehicleType = traindata.VehicleTypeSleeper
				case "Couchette + bikes", "Classic + bikes":
					vehicleType = traindata.VehicleTypeBikeCouchette
				case "Bistro":
					vehicleType = traindata.VehicleTypeBikeBistro
				case "Seats", "Seats + bikes", "Budget", "Budget + bikes":
					vehicleType = traindata.VehicleTypeSeats
				}
			}
		}
		if tt == html.EndTagToken {
			t := z.Token()
			if t.Data == "div" {
				break
			}
		}
	}

	return vehicleType, vehicleNumber, nil
}
