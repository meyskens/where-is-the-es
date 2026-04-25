package nmbs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
	flareSolverr    *flaresolverr.Client
	flareSolverrURL string
}

func NewNMBSFetcher(flareSolverrURL string) (*NMBSFetcher, error) {
	c, err := flaresolverr.NewClient(flaresolverr.Config{
		BaseURL: flareSolverrURL,
	})
	if err != nil {
		return nil, err
	}
	return &NMBSFetcher{
		flareSolverr:    c,
		flareSolverrURL: flareSolverrURL,
	}, nil
}

func (f *NMBSFetcher) FetchTimetable(trainNumber string, date time.Time) ([]traindata.Stop, error) {
	log.Println("NMBS: fetching timetable for train", trainNumber, "date", date.Format("2006-01-02"))

	// Get the request verification token
	token, cookies, err := f.fetchRequestVerificationToken()
	if err != nil {
		log.Println("NMBS: failed to fetch verification token for train", trainNumber, ":", err)
		return nil, fmt.Errorf("fetching verification token: %w", err)
	}
	log.Println("NMBS: got verification token (len", len(token), ") and", len(cookies), "cookies for train", trainNumber)

	resp, err := f.flareSolverr.PostRaw(flaresolverr.PostParams{
		URL:     "https://www.belgiantrain.be/api/RoutePlanner/GetDelayCertificateStationTrainsDetails",
		Cookies: cookies,
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
		log.Println("NMBS: flaresolverr POST transport failed for train", trainNumber, ":", err)
		return nil, fmt.Errorf("flaresolverr POST: %w", err)
	}

	log.Printf("NMBS: POST flaresolverr status=%q solution-status=%d message=%q version=%s for train %s", resp.Status, resp.Solution.Status, resp.Message, resp.Version, trainNumber)

	body, err := unwrapSolutionResponse(resp.Solution.Response)
	if err != nil {
		log.Println("NMBS: failed to unwrap POST solution response for train", trainNumber, ":", err)
		return nil, fmt.Errorf("unwrapping POST solution response: %w", err)
	}

	log.Println("NMBS: received", len(body), "bytes for train", trainNumber, "date", date.Format("2006-01-02"))

	stops, err := f.ParseTimetable(body)
	if err != nil {
		preview := body
		if len(preview) > 500 {
			preview = preview[:500]
		}
		log.Println("NMBS: parse failed for train", trainNumber, "date", date.Format("2006-01-02"), ":", err, "- response preview:", string(preview))
		return nil, fmt.Errorf("parse timetable: %w", err)
	}

	log.Println("NMBS: parsed", len(stops), "stops for train", trainNumber, "date", date.Format("2006-01-02"))
	return stops, nil
}

func (f *NMBSFetcher) fetchRequestVerificationToken() (string, flaresolverr.Cookies, error) {
	log.Println("NMBS: configured FlareSolverr URL:", f.flareSolverrURL)
	resp, err := f.flareSolverr.GetRaw(flaresolverr.GetParams{
		URL: "https://www.belgiantrain.be/nl/support/customer-service/delay-certificate",
	})
	if err != nil {
		log.Println("NMBS: flaresolverr GET transport failed:", err)
		return "", nil, fmt.Errorf("flaresolverr GET transport: %w", err)
	}

	log.Printf("NMBS: GET flaresolverr status=%q solution-status=%d message=%q version=%s", resp.Status, resp.Solution.Status, resp.Message, resp.Version)

	if string(resp.Status) != "ok" {
		return "", nil, fmt.Errorf("flaresolverr GET non-ok status=%q message=%q", resp.Status, resp.Message)
	}

	body, err := unwrapSolutionResponse(resp.Solution.Response)
	if err != nil {
		return "", nil, fmt.Errorf("unwrapping GET solution response: %w", err)
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

		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}

		t := z.Token()
		if t.Data != "input" {
			continue
		}

		isToken := false
		value := ""
		for _, a := range t.Attr {
			if a.Key == "name" && a.Val == "__RequestVerificationToken" {
				isToken = true
			}
			if a.Key == "value" {
				value = a.Val
			}
		}
		if isToken && value != "" {
			token = value
			break
		}
	}

	if token == "" {
		preview := body
		if len(preview) > 500 {
			preview = preview[:500]
		}
		log.Println("NMBS: token not found in response (", len(body), "bytes ), preview:", string(preview))
		return "", nil, fmt.Errorf("could not find token in %d-byte response", len(body))
	}

	return token, resp.Solution.Cookies, nil
}

// unwrapSolutionResponse converts the flaresolverr Solution.Response
// (json.RawMessage) into the actual page body bytes. The field is sometimes
// a JSON-encoded string (the typical case for HTML pages) and sometimes
// already raw bytes. Handle both.
func unwrapSolutionResponse(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty solution response")
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return []byte(s), nil
	}
	return raw, nil
}
