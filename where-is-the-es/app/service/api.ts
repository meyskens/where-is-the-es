export interface Vehicle {
    type: string;
    number: string;
    uicNumber: string;
}

export interface CompositionResponse {
    vehicles: Vehicle[];
}

export interface Stop {
    StationName: string;
    ArrivalTime: string;
    DepartureTime: string;
    Platform: string;
    DataSources: string[];
    PrefferedDataSource: string;
    RealPlatform: string;
    RealArrivalTime: string;
    RealDepartureTime: string;
    NextDay: boolean;
    Cancelled: boolean;
}

export interface DebugStop {
    stationName: string;
    stationUIC: number;
    arrivalTime: string;
    departureTime: string;
    platform: string;
    realPlatform: string;
    realArrivalTime: string;
    realDepartureTime: string;
    isRealTime: boolean;
    cancelled: boolean;
    relevant: boolean;
    dataSources: string[];
    prefferedDataSource: string;
}

export interface DebugSourceResult {
    source: string;
    trainNumber: string;
    date: string;
    available: boolean;
    found: boolean;
    error?: string;
    stops?: DebugStop[];
}

export interface DebugSourceInfo {
    id: string;
    name: string;
    available: boolean;
}

export interface TimetableResponse {
    TrainNumber: string;
    Stops: Stop[];
    Composition: {
        Vehicles: any;
        Order: any;
    };
    IsRunning: boolean;
    Date: string;
}

export class APIService {
    private baseUrl: string;

    constructor() {
        // Use the current hostname in production, localhost:8080 in development
        this.baseUrl = process.env.NODE_ENV === 'production'
            ? window.location.origin
            : 'http://localhost:8080';
    }

    /**
     * Fetches the composition data for a specific train
     * @param trainNumber The train number to fetch composition for
     * @returns Promise with the composition data
     */
    async getTrainComposition(trainNumber: string): Promise<CompositionResponse> {
        try {
            const response = await fetch(`${this.baseUrl}/api/v1/composition/${trainNumber}`);

            if (!response.ok) {
                throw new Error(`Failed to fetch train composition: ${response.status} ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Error fetching train composition:', error);
            throw error;
        }
    }

    /**
     * Fetches the timetable data for a specific train and date
     * @param date The date in YYYY-MM-DD format
     * @param trainNumber The train number to fetch timetable for
     * @returns Promise with the timetable data
     */
    async getTrainTimetable(date: string, trainNumber: string): Promise<TimetableResponse> {
        try {
            const response = await fetch(`${this.baseUrl}/api/v1/timetable/${date}/${trainNumber}`);

            if (!response.ok) {
                throw new Error(`Failed to fetch train timetable: ${response.status} ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Error fetching train timetable:', error);
            throw error;
        }
    }

    /**
     * Fetches the list of available debug data sources.
     * @returns Promise with the list of data sources
     */
    async getDebugSources(): Promise<DebugSourceInfo[]> {
        try {
            const response = await fetch(`${this.baseUrl}/api/debug/sources`);

            if (!response.ok) {
                throw new Error(`Failed to fetch debug sources: ${response.status} ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Error fetching debug sources:', error);
            throw error;
        }
    }

    /**
     * Fetches the raw data from a specific data source for a given train.
     * @param source The data source ID (e.g. "bahn", "ns", "nmbs", ...)
     * @param trainNumber The train number to fetch data for
     * @param date Optional date in YYYY-MM-DD format. Defaults to today.
     * @returns Promise with the debug source result
     */
    async getDebugSourceData(source: string, trainNumber: string, date?: string): Promise<DebugSourceResult> {
        try {
            const url = date
                ? `${this.baseUrl}/api/debug/${source}/${trainNumber}/${date}`
                : `${this.baseUrl}/api/debug/${source}/${trainNumber}`;

            const response = await fetch(url);

            if (!response.ok) {
                throw new Error(`Failed to fetch debug data: ${response.status} ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Error fetching debug source data:', error);
            throw error;
        }
    }

}

export const apiService = new APIService();