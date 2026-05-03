import TrainPicker from "~/components/trainpicker";
import type { Route } from "./+types/train.$trainNumber";
import { useState } from "react";
import { TrainComposition } from "~/components/composition";
import { TrainTimetable } from "~/components/timetable";
import { useParams } from "react-router";

export function meta({ params }: Route.MetaArgs) {
  return [
    { title: `Where is the European Sleeper? - Train ${params.trainNumber}` },
    { name: "description", content: `Live timetable for European Sleeper train ${params.trainNumber}` },
  ];
}

export default function TrainRoute() {
  const params = useParams();
  const [selectedTrain, setSelectedTrain] = useState(params.trainNumber || "453");
  
  const handleTrainSelect = (trainNumber: string) => {
    setSelectedTrain(trainNumber);
    // Navigation will be handled by the TrainPicker component
  };

  return (
    <>
      <TrainPicker onSelectTrain={handleTrainSelect} initialTrain={selectedTrain} />
      <div className="m-4"/>
      <TrainComposition trainNumber={selectedTrain} />
      <div className="m-4"/>
      <TrainTimetable trainNumber={selectedTrain} />
    </>
  );
}
