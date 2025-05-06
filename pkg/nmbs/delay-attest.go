package nmbs

import (
	"bytes"
	"fmt"
	"net/url"
	"time"

	"github.com/astrocode-id/go-flaresolverr"
	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/net/html"
)

/*
What whitchcraft is this?

Oh yeah... SNCB/NMBS GTFS data only contains their own trains. However their routing engine
contains way more information as it is fetching data from EMMA which is also aware of non SNCB trains
as well as any charter trains.

The best information gets exposed by "accident" in the delay certificate.
Sadly this data is now behind an annpying cloudflare wall.

But nothing can stop us from scraping it!
*/

/*
curl 'https://www.belgiantrain.be/api/RoutePlanner/GetDelayCertificateStationTrainsDetails' \
-X POST \
-H 'Content-Type: application/x-www-form-urlencoded; charset=UTF-8' \
-H 'X-Requested-With: XMLHttpRequest' \
-H 'Origin: https://www.belgiantrain.be' \
 -H 'Referer: https://www.belgiantrain.be/nl/support/customer-service/delay-certificate' \
  -H 'Sec-Fetch-Dest: empty' \
  -H 'Sec-Fetch-Mode: cors' \
  -H 'Sec-Fetch-Site: same-origin' \
  -H 'Priority: u=0' \
  -H 'TE: trailers' \
  --data-raw '__RequestVerificationToken=HlP0QHBCRf549XksO9jRvGxc781KyY1VUR3OEAluEFOM4PsxrP-DMQupCzKvMouqp4cK1DRxe3riSrG4sfU5Vocu7rCitx54KTYAlrNJX901&ParameterSearchByStationName=ByStationName&ParameterSearchByTrainNumber=ByTrainNumber&ParameterDaysAllowedInPast=7&DatasourceId=07DFC82039D54C7F85021417DECF2AB8&SearchModeType=ByTrainNumber&TrainNumber=453&ByTrainDate=05%2F05%2F2025&StationId=&DirectionType=ArrivalBoard&ByStationDate=06%2F05%2F2025&SearchByStationTime=09%3A00+-+10%3A00'
*/

type NMBSFetcher struct {
	flareSolverr *flaresolverr.Client
}

func NewNMBSFetcher(flareSolverrURL string) (*NMBSFetcher, error) {
	c, err := flaresolverr.NewClient(flaresolverr.Config{
		BaseURL: flareSolverrURL,
	})
	if err != nil {
		return nil, err
	}
	return &NMBSFetcher{
		flareSolverr: c,
	}, nil
}

func (f *NMBSFetcher) FetchTimetable(trainNumber string, date time.Time) ([]traindata.Stop, error) {
	// Get the request verification token
	token, err := f.fetchRequestVerificationToken()
	if err != nil {
		return nil, err
	}

	resp, err := f.flareSolverr.Post(flaresolverr.PostParams{
		URL: "https://www.belgiantrain.be/api/RoutePlanner/GetDelayCertificateStationTrainsDetails",
		PostData: url.Values{
			"__RequestVerificationToken":   {token},
			"ParameterSearchByStationName": {"ByStationName"},
			"ParameterSearchByTrainNumber": {"ByTrainNumber"},
			"ParameterDaysAllowedInPast":   {"7"},
			"DatasourceId":                 {"07DFC82039D54C7F85021417DECF2AB8"},
			"SearchModeType":               {"ByTrainNumber"},
			"TrainNumber":                  {trainNumber},
			"ByTrainDate":                  {date.Format("01/02/2006")},
			"StationId":                    {""},
			"DirectionType":                {"ArrivalBoard"},
			"ByStationDate":                {date.Format("01/02/2006")},
			"SearchByStationTime":          {"09:00+-+10:00"},
		},
	})

	if err != nil {
		return nil, err
	}

	return f.ParseTimetable(resp)
}

func (f *NMBSFetcher) fetchRequestVerificationToken() (string, error) {
	body, err := f.flareSolverr.Get(flaresolverr.GetParams{
		URL: "https://www.belgiantrain.be/nl/support/customer-service/delay-certificate",
	})
	if err != nil {
		return "", err
	}

	// Find the token in the body
	// <input name="__RequestVerificationToken" type="hidden" value="$VALUE" />

	token := ""

	z := html.NewTokenizer(bytes.NewReader(body))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "input" {
				for _, a := range t.Attr {
					if a.Key == "name" && a.Val == "__RequestVerificationToken" {
						// Read the next token
						tt = z.Next()
						if tt == html.SelfClosingTagToken {
							t = z.Token()
							for _, a := range t.Attr {
								if a.Key == "value" {
									token = a.Val
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if token == "" {
		return "", fmt.Errorf("could not find token")
	}

	return token, nil
}
