import React, { useEffect, useState } from 'react';
import { apiService, type CompositionResponse, type Vehicle } from '../service/api';
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "../components/ui/alert";
import { AlertCircle, Info } from "lucide-react";

interface TrainCompositionProps {
  trainNumber: string;
}

const vehicleTypeToImage: Record<string, string> = {
  Locomotive: '/images/186.webp',
  Sleeper: '/images/WLABmz-7070-b.webp',
  Couchette: '/images/Bvcmz248-euro-a.webp',
  Seats: '/images/Bm-euro-b.gif',
  "Bike Couchette": '/images/BDcm-a.webp',
  default: '/images/Bvcmz248-euro-b.webp',
};

export const TrainComposition: React.FC<TrainCompositionProps> = ({ trainNumber }) => {
  const [composition, setComposition] = useState<CompositionResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchComposition = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await apiService.getTrainComposition(trainNumber);
        setComposition(data);
      } catch (err) {
        setError('Failed to load train composition. Please try again later.');
        console.error('Error fetching train composition:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchComposition();
  }, [trainNumber]);

  const getVehicleImage = (vehicle: Vehicle): string => {
    return vehicleTypeToImage[vehicle.type] || vehicleTypeToImage.default;
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center p-6 space-y-4">
        <div className="flex flex-col items-center space-y-4 w-full max-w-lg">
          <Skeleton className="h-8 w-1/2 mb-4" />
          <p className="text-sm text-muted-foreground">Loading train composition...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive" className="mt-4">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  if (!composition || composition.vehicles.length === 0) {
    return (
      <Alert className="mt-4">
        <Info className="h-4 w-4" />
        <AlertTitle>No data</AlertTitle>
        <AlertDescription>No composition data available for this train.</AlertDescription>
      </Alert>
    );
  }

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="text-xl font-bold">ES {trainNumber} Composition</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap overflow-x-auto py-2">
          {composition.vehicles.map((vehicle, index) => (
            <div key={index} className="flex flex-col items-center">
              <div className="p-0 m-0 bg-muted/20 flex items-end" style={{ height: '60px' }}>
                <img 
                  src={getVehicleImage(vehicle)} 
                  alt={vehicle.type}
                  style={vehicle.type == "Locomotive" ? { width: '189px', height: '58px' } : { width: 'auto', height: '41px' }}
                />
              </div>
              <div className="text-center mt-2">
                <div className="text-2xl">{vehicle.number}</div>
                <div className="text-sm text-muted-foreground">{vehicle.type}</div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};