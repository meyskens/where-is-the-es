package nmbs

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"github.com/meyskens/where-is-the-es/pkg/traindata"
	"golang.org/x/net/html"
)

/*

<input type="hidden" id="trainSearchErrorResult" value="false" />

<div id="delayCertificateTrainRouteDetails" class="well theme-white marg-top-sm-20 custom-planner-route-detailed planner-delay">

    <ol>
        <li class="planner-dtl__item planner-head">
            <div>Aankomst</div>
            <div>Vertrek</div>
            <div>Tussenstops</div>
        </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                19:22
                            </div>
                                <div class="delay-txt">+ 1</div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot__icon-wrapper">

                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                <div class="train-origin-station" data-origin-station-time="19:22" data-origin-station-uic="8814001" data-origin-station-name="BRUSSEL-ZUID">BRUSSEL-ZUID</div>
                            </div>
                            <div id="trip-message-text" class="planner-dtl__detail">
                                INT 453 richting PRAHA HL.N.
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                19:58
                            </div>
                                <div class="delay-txt">+ 4</div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                20:02
                            </div>
                                <div class="delay-txt">+ 5</div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>ANTWERPEN-CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                20:41
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                20:44
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>ROOSENDAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                21:19
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                21:22
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>ROTTERDAM CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                21:40
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                21:42
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DEN HAAG HOLLANDS SPOOR</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                22:28
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                22:34
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>AMSTERDAM CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                23:08
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                23:13
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>AMERSFOORT CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                23:48
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                23:52
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DEVENTER</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                06:16
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                06:20
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>BERLIN HBF</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                06:27
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                06:29
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>BERLIN OSTBAHNHOF</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                08:50
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                08:54
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DRESDEN HBF</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                09:21
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                09:23
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>BAD SCHANDAU</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                09:41
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                09:46
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DECIN HL.N.</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item planner-dtl__item--transfer  ">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                11:24
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div class="train-destination-station" data-destination-station-time="11:24" data-destination-station-uic="5457076" data-destination-station-name="PRAHA HL.N.">PRAHA HL.N.</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>

    </ol>
</div>
*/

var ErrNoTime = errors.New("no time found")

func (f *NMBSFetcher) ParseTimetable(body []byte) ([]traindata.Stop, error) {
	z := html.NewTokenizer(bytes.NewReader(body))

	var stops []traindata.Stop

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "li" {
				stop, err := f.parseStop(z)
				if err != nil {
					return nil, err
				}

				if stop.StationName != "" {
					stops = append(stops, stop)
				}
			}

			// if trainSearchErrorResult is true, return an error
			if t.Data == "input" {
				for _, a := range t.Attr {
					if a.Key == "id" && a.Val == "trainSearchErrorResult" {
						for _, a := range t.Attr {
							if a.Key == "value" && a.Val == "true" {
								return nil, errors.New("trainSearchErrorResult returned true")
							}
						}
					}
				}
			}

		}
	}
	if len(stops) == 0 {
		return nil, errors.New("no stops found")
	}

	return stops, nil
}

func (f *NMBSFetcher) parseStop(z *html.Tokenizer) (traindata.Stop, error) {
	var stop traindata.Stop
	var err error

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "div" {
				for _, a := range t.Attr {
					if a.Key == "class" && a.Val == "planner-dtl__arrival" {
						stop.ArrivalTime, stop.RealArrivalTime, err = f.parseTime(z)
						if err != nil && err != ErrNoTime {
							return stop, err
						}
					}
					if a.Key == "class" && a.Val == "planner-dtl__departure" {
						stop.DepartureTime, stop.RealDepartureTime, err = f.parseTime(z)
						if err != nil && err != ErrNoTime {
							return stop, err
						}
					}
					if a.Key == "class" && a.Val == "planner-dtl__lbl mobile-full-width" {
						stop.StationName, err = f.parseStationName(z)
						if err != nil {
							return stop, err
						}
					}
				}
			}
			if t.Data == "li" {
				for _, a := range t.Attr {
					if a.Key == "class" && a.Val == "planner-dtl__item planner-head" {
						break
					}
				}
			}
		}

		if tt == html.EndTagToken && z.Token().Data == "li" {
			break
		}
	}

	return stop, nil
}

func (f *NMBSFetcher) parseTime(z *html.Tokenizer) (time.Time, time.Time, error) {
	var timePlanned time.Time
	var timeReal time.Time

	timeString := ""

	innerDIVs := 1
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.TextToken {
			timeString += string(z.Text())
		}

		if tt == html.StartTagToken && z.Token().Data == "div" {
			innerDIVs++
		}

		if tt == html.EndTagToken && z.Token().Data == "div" {
			innerDIVs--
			if innerDIVs == 0 {
				break
			}
		}
	}

	timeString = strings.ReplaceAll(timeString, "\n", "")
	timeString = strings.ReplaceAll(timeString, "Vertrek in", "")
	timeString = strings.ReplaceAll(timeString, "Aangekomen in", "")
	timeString = strings.TrimSpace(timeString)

	if timeString == "" {
		return time.Time{}, time.Time{}, ErrNoTime
	}

	delayString := "0"
	if strings.Contains(timeString, "+") {
		delayString = strings.Split(timeString, "+")[1]
		timeString = strings.Split(timeString, "+")[0]

		timeString = strings.TrimSpace(timeString)
		delayString = strings.TrimSpace(delayString)
	}

	// Parse the time string
	parsedTime, err := time.Parse("15:04", timeString)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	timePlanned = parsedTime
	timeReal = parsedTime

	if delayString != "0" {
		delay, err := time.ParseDuration(delayString + "m")
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		timeReal = parsedTime.Add(delay)
	}

	return timePlanned, timeReal, nil
}

func (f *NMBSFetcher) parseStationName(z *html.Tokenizer) (string, error) {
	var stationName string

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.TextToken {
			stationName += string(z.Text())

		}

		if tt == html.EndTagToken && z.Token().Data == "div" {
			break
		}
	}

	stationName = strings.ReplaceAll(stationName, "Van", "")
	stationName = strings.ReplaceAll(stationName, "\n", "")
	stationName = strings.TrimSpace(stationName)

	if stationName == "" {
		return "", errors.New("no station name found")
	}

	return stationName, nil
}
