import { useState } from "react";
import { ChevronDownIcon } from "@heroicons/react/24/solid";

interface TrainPickerProps {
  onSelectTrain: (trainNumber: string) => void;
  initialTrain?: string;
}

const TrainPicker = ({ onSelectTrain, initialTrain = "453" }: TrainPickerProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedTrain, setSelectedTrain] = useState(initialTrain);

  const trainOptions = [
    { value: "453", label: "ES 453 Brussels → Praha" },
    { value: "452", label: "ES 452 Praha → Brussels" },
  ];

  const handleSelect = (value: string) => {
    setSelectedTrain(value);
    onSelectTrain(value);
    setIsOpen(false);
  };

  // Safe-guard in case the selected train is not in the options
  const selectedLabel = trainOptions.find(option => option.value === selectedTrain)?.label || "";
  
  return (
    <div className="relative w-full max-w-md">
      <div 
        className="flex items-center justify-between p-3 border border-gray-300 rounded-md bg-white shadow-sm cursor-pointer hover:bg-gray-50"
        onClick={() => setIsOpen(!isOpen)}
      >
        <div className="flex items-center">
          <span className="font-medium">{selectedLabel}</span>
        </div>
        <ChevronDownIcon 
          className={`h-5 w-5 text-gray-500 transition-transform ${isOpen ? 'rotate-180' : ''}`} 
        />
      </div>

      {isOpen && (
        <div className="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-md shadow-lg">
          {trainOptions.map((option) => (
            <div
              key={option.value}
              className={`p-3 cursor-pointer ${
                selectedTrain === option.value
                  ? "bg-blue-50 text-blue-700"
                  : "hover:bg-gray-50"
              }`}
              onClick={() => handleSelect(option.value)}
            >
              <span>{option.label}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default TrainPicker;