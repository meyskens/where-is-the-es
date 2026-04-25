package europeansleeper

type TCResponse struct {
	Services []struct {
		ID            string `json:"id"`
		TrainNumber   string `json:"train_number"`
		DepartureDate string `json:"departure_date"`
		Timetable     struct {
			Stops []struct {
				Arrival      string `json:"arrival"`
				Departure    string `json:"departure"`
				Station      string `json:"station"`
				StationUIC   int    `json:"station_uic"`
				Cancelled    bool   `json:"cancelled"`
				ServiceStop  bool   `json:"service_stop"`
				DayInService int    `json:"day_in_service"`
			} `json:"stops"`
		} `json:"timetable"`
		Composition []struct {
			CarriageOrder int  `json:"carriage_order"`
			Number        any  `json:"number"`
			IsLocomotive  bool `json:"is_locomotive"`
			CarriageID    any  `json:"carriage_id"`
			Carriage      struct {
				UICNumber string `json:"uic_number"`
			} `json:"carriage"`
		} `json:"composition"`
	} `json:"services"`
}
