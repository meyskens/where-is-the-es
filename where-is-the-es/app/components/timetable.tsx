import { useEffect, useState } from "react";
import { apiService, type TimetableResponse, type Stop } from "~/service/api";
import { Card, CardContent, CardHeader, CardTitle } from "~/components/ui/card";
import { Alert, AlertDescription } from "~/components/ui/alert";
import { Clock, MapPin, Calendar, ArrowDownRight, ArrowUpRight } from "lucide-react";

interface TimetableProps {
    trainNumber: string;
    date?: string;
}

export function TrainTimetable({ trainNumber, date }: TimetableProps) {
    const [timetable, setTimetable] = useState<TimetableResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Default to today's date if not provided
    const requestDate = date || "next";

    useEffect(() => {
        const fetchTimetable = async () => {
            if (!trainNumber) return;

            setLoading(true);
            setError(null);
            
            try {
                const data = await apiService.getTrainTimetable(requestDate, trainNumber);
                setTimetable(data);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to fetch timetable');
            } finally {
                setLoading(false);
            }
        };

        fetchTimetable();
    }, [trainNumber, requestDate]);

    const formatTime = (timeString: string) => {
        if (!timeString || timeString === "0001-01-01T00:00:00Z") {
            return "--:--";
        }
        try {
            const date = new Date(timeString);
            return date.toLocaleTimeString('en-GB', { 
                hour: '2-digit', 
                minute: '2-digit',
                timeZone: 'Europe/Amsterdam'
            });
        } catch {
            return "--:--";
        }
    };

    const isValidTime = (timeString: string) => {
        return timeString && timeString !== "0001-01-01T00:00:00Z";
    };

    const isStopYetToCome = (stop: Stop) => {
        // Real time is not yet available, use scheduled departure time
        const departureTime = isValidTime(stop.DepartureTime) ? 
            stop.DepartureTime : 
            stop.ArrivalTime;
        
        if (!isValidTime(departureTime)) {
            return false;
        }

        try {
            const departureDate = new Date(departureTime);
            const now = new Date();
            return departureDate > now;
        } catch {
            return false;
        }
    };

    if (loading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Clock className="h-5 w-5" />
                        Timetable - Train {trainNumber}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex items-center justify-center py-8">
                        <div className="text-muted-foreground">Loading timetable...</div>
                    </div>
                </CardContent>
            </Card>
        );
    }

    if (error) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Clock className="h-5 w-5" />
                        Timetable - Train {trainNumber}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <Alert variant="destructive">
                        <AlertDescription>{error}</AlertDescription>
                    </Alert>
                </CardContent>
            </Card>
        );
    }

    if (!timetable || !timetable.Stops || timetable.Stops.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Clock className="h-5 w-5" />
                        Timetable - ES {trainNumber}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="text-muted-foreground text-center py-8">
                        No timetable data available
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <Clock className="h-5 w-5" />
                    Timetable - Train {trainNumber}
                </CardTitle>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Calendar className="h-4 w-4" />
                    {timetable.Date ? (
                        <span>{new Date(timetable.Date).toLocaleDateString('en-GB', { 
                            year: 'numeric', 
                            month: '2-digit',
                            day: '2-digit',
                            timeZone: 'Europe/Amsterdam'
                        })}</span>
                    ) : (
                        <span>Unknown Date</span>
                    )}
                    {timetable.IsRunning && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                            Running
                        </span>
                    )}
                </div>
            </CardHeader>
            <CardContent>
                <div className="relative">
                    {/* Timeline line */}
                    <div className="absolute left-[7.44rem] top-0 bottom-0 w-0.5 bg-border"></div>
                    
                    <div className="space-y-6">
                        {timetable.Stops.map((stop: Stop, index: number) => (
                            <div key={index} className="relative flex items-start gap-4">
                                {/* Times section */}
                                <div className="w-24 flex-shrink-0 text-xs space-y-1">
                                    {/* Arrival time */}
                                    <div className={`flex items-center gap-1 ${
                                        isValidTime(stop.ArrivalTime) 
                                            ? 'text-blue-600' 
                                            : 'text-muted-foreground/50'
                                    }`}>
                                        <ArrowDownRight className="h-3 w-3" />
                                        <span className="font-medium">
                                            {formatTime(stop.ArrivalTime)}
                                        </span>
                                    </div>
                                    {/* Departure time */}
                                    <div className={`flex items-center gap-1 ${
                                        isValidTime(stop.DepartureTime) 
                                            ? 'text-green-600' 
                                            : 'text-muted-foreground/50'
                                    }`}>
                                        <ArrowUpRight className="h-3 w-3" />
                                        <span className="font-medium">
                                            {formatTime(stop.DepartureTime)}
                                        </span>
                                    </div>
                                </div>
                                
                                {/* Timeline dot */}
                                <div className="relative flex h-4 w-4 items-center justify-center">
                                    <div className={`h-3 w-3 rounded-full border-2 ${
                                        stop.Cancelled 
                                            ? 'bg-red-500 border-red-500' 
                                            : isStopYetToCome(stop)
                                                ? 'bg-blue-500 border-blue-500'
                                                : 'bg-white border-border'
                                    } relative z-10`}></div>
                                </div>
                                
                                {/* Stop information */}
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-2 mb-1">
                                        <MapPin className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                        <h3 className={`font-medium text-sm ${
                                            stop.Cancelled ? 'line-through text-red-500' : ''
                                        }`}>
                                            {stop.StationName}
                                        </h3>
                                        {stop.Cancelled && (
                                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">
                                                Cancelled
                                            </span>
                                        )}
                                    </div>
                                    
                                    {stop.Platform && (
                                        <div className="text-xs text-muted-foreground">
                                            Platform: {stop.Platform}
                                        </div>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}

export default TrainTimetable;
