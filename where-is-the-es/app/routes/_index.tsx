import TrainPicker from "~/components/trainpicker";
import type { Route } from "./+types/home";
import { useState } from "react";
import { TrainComposition } from "~/components/composition";
import { Train } from "lucide-react";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Where is the European Sleeper?" },
    { name: "description", content: "A live timetable for the European Sleeper trains" },
  ];
}

export default function Home() {
  const [selectedTrain, setSelectedTrain] = useState("453");
  
  const handleTrainSelect = (trainNumber: string) => {
    setSelectedTrain(trainNumber);
    // You can add any additional logic here, like fetching data based on the selected train
    console.log(`Selected train: ${trainNumber}`);
  };

  return (
    <>
    <TrainPicker onSelectTrain={handleTrainSelect} />
    <div className="m-4"/>
    <TrainComposition trainNumber={selectedTrain} />
    </>
  );
}
