export interface Vehicle {
    type: string;
    number: string;
}

export interface CompositionResponse {
    vehicles: Vehicle[];
}

export interface Stop {
    StationName: string;
    ArrivalTime: string;
    DepartureTime: string;
    Platform: string;
    DataSources: any;
    PrefferedDataSource: number;
    RealPlatform: string;
    RealArrivalTime: string;
    RealDepartureTime: string;
    NextDay: boolean;
    Cancelled: boolean;
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

}

export const apiService = new APIService();