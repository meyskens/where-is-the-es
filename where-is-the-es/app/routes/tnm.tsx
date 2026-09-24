import { useEffect, useState } from "react";
import { apiService, type DebugSourceInfo, type DebugSourceResult, type DebugStop } from "~/service/api";
import { Card, CardContent, CardHeader, CardTitle } from "~/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { AlertCircle, RefreshCw, Train, Glasses } from "lucide-react";

export function meta() {
  return [
    { title: "Train Nerd Mode — Where is the European Sleeper?" },
    { name: "robots", content: "noindex, nofollow" },
  ];
}

const trainOptions = [
  { value: "453", label: "ES 453 Brussels → Praha" },
  { value: "452", label: "ES 452 Praha → Brussels" },
  { value: "474", label: "ES 474 Berlin → Paris" },
  { value: "475", label: "ES 475 Paris → Berlin" },
  { value: "400", label: "ES 400 Milano → Brussels" },
  { value: "401", label: "ES 401 Brussels → Milano" },
];

function formatTime(timeString: string): string {
  if (!timeString) return "--:--";
  try {
    const d = new Date(timeString);
    if (d.getTime() === 0 || isNaN(d.getTime())) return "--:--";
    return d.toLocaleTimeString("en-GB", {
      hour: "2-digit",
      minute: "2-digit",
      timeZone: "Europe/Amsterdam",
    });
  } catch {
    return "--:--";
  }
}

export default function TNMDebug() {
  const [sources, setSources] = useState<DebugSourceInfo[]>([]);
  const [selectedTrain, setSelectedTrain] = useState("453");
  const [results, setResults] = useState<Record<string, DebugSourceResult | null>>({});
  const [loading, setLoading] = useState<Record<string, boolean>>({});
  const [sourcesError, setSourcesError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSources = async () => {
      try {
        const data = await apiService.getDebugSources();
        setSources(data);
      } catch (err) {
        setSourcesError(err instanceof Error ? err.message : "Failed to fetch sources");
      }
    };
    fetchSources();
  }, []);

  useEffect(() => {
    if (sources.length === 0) return;
    let cancelled = false;

    const loadAll = async () => {
      setLoading({});
      setResults({});
      for (const s of sources) {
        if (!s.available) continue;
        setLoading((prev) => ({ ...prev, [s.id]: true }));
        try {
          const result = await apiService.getDebugSourceData(s.id, selectedTrain);
          if (cancelled) return;
          setResults((prev) => ({ ...prev, [s.id]: result }));
        } catch (err) {
          if (cancelled) return;
          setResults((prev) => ({
            ...prev,
            [s.id]: {
              source: s.id,
              trainNumber: selectedTrain,
              date: "",
              available: false,
              found: false,
              error: err instanceof Error ? err.message : "Failed to load",
            },
          }));
        } finally {
          if (!cancelled) setLoading((prev) => ({ ...prev, [s.id]: false }));
        }
      }
    };

    loadAll();
    return () => { cancelled = true; };
  }, [sources, selectedTrain]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-2xl font-bold flex items-center gap-2">
            <Glasses className="h-6 w-6" />
            Train Nerd Mode
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Raw data from for raw people.
          </p>
        </div>
        <div className="flex items-end gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-500 mb-1">Train</label>
            <select
              className="border border-gray-300 rounded-md px-3 py-2 bg-white shadow-sm"
              value={selectedTrain}
              onChange={(e) => setSelectedTrain(e.target.value)}
            >
              {trainOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

        </div>
      </div>

      {sourcesError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Failed to load data sources</AlertTitle>
          <AlertDescription>{sourcesError}</AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4">
        {sources.map((source) => {
          const result = results[source.id];
          const isLoading = loading[source.id];
          return (
            <SourceCard
              key={source.id}
              source={source}
              result={result}
              isLoading={!!isLoading}
            />
          );
        })}
        {sources.length === 0 && !sourcesError && (
          <Card>
            <CardContent className="text-center text-gray-500 py-8">
              <Train className="h-8 w-8 mx-auto mb-2 opacity-50" />
              Loading data sources...
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}

interface SourceCardProps {
  source: DebugSourceInfo;
  result: DebugSourceResult | null | undefined;
  isLoading: boolean;
}

function SourceCard({ source, result, isLoading }: SourceCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <span>{source.name}</span>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-normal ${
                source.available
                  ? "bg-green-100 text-green-700"
                  : "bg-gray-100 text-gray-500"
              }`}
            >
              {source.available ? "available" : "not configured"}
            </span>
          </CardTitle>
          {isLoading && (
            <span className="text-xs text-gray-500 flex items-center gap-1">
              <RefreshCw className="h-3.5 w-3.5 animate-spin" />
              Loading...
            </span>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {!source.available && (
          <p className="text-sm text-gray-400">This data source is not configured on the server.</p>
        )}
        {source.available && result && result.error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{result.error}</AlertDescription>
          </Alert>
        )}
        {source.available && result && !result.error && (
          <div className="space-y-3">
            <div className="flex gap-4 text-xs text-gray-500">
              <span>Train: <strong>{result.trainNumber}</strong></span>
              <span>Date: <strong>{result.date || "—"}</strong></span>
              <span>Stops: <strong>{result.stops?.length ?? 0}</strong></span>
              {!result.found && (
                <span className="text-amber-600">not in cache</span>
              )}
            </div>
            {result.stops && result.stops.length > 0 ? (
              <StopsTable stops={result.stops} />
            ) : (
              <p className="text-sm text-gray-400">No stops in cache for this source.</p>
            )}
          </div>
        )}
        {source.available && !result && !isLoading && (
          <p className="text-sm text-gray-400">Loading cached data...</p>
        )}
      </CardContent>
    </Card>
  );
}

interface StopsTableProps {
  stops: DebugStop[];
}

function StopsTable({ stops }: StopsTableProps) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-gray-500 border-b border-gray-200">
            <th className="py-1.5 pr-3">Station</th>
            <th className="py-1.5 pr-3">UIC</th>
            <th className="py-1.5 pr-3">Arr</th>
            <th className="py-1.5 pr-3">Dep</th>
            <th className="py-1.5 pr-3">Plat</th>
            <th className="py-1.5 pr-3">Real Arr</th>
            <th className="py-1.5 pr-3">Real Dep</th>
            <th className="py-1.5 pr-3">Real Plat</th>
            <th className="py-1.5 pr-3">RT</th>
            <th className="py-1.5 pr-3">Cxl</th>
            <th className="py-1.5">Rel</th>
          </tr>
        </thead>
        <tbody>
          {stops.map((stop, i) => (
            <tr
              key={i}
              className={`border-b border-gray-100 ${stop.cancelled ? "opacity-50 line-through" : ""}`}
            >
              <td className="py-1.5 pr-3 font-medium">{stop.stationName || "—"}</td>
              <td className="py-1.5 pr-3 text-gray-500 font-mono text-xs">
                {stop.stationUIC || "—"}
              </td>
              <td className="py-1.5 pr-3">{formatTime(stop.arrivalTime)}</td>
              <td className="py-1.5 pr-3">{formatTime(stop.departureTime)}</td>
              <td className="py-1.5 pr-3">{stop.platform || "—"}</td>
              <td className="py-1.5 pr-3">
                {stop.isRealTime ? formatTime(stop.realArrivalTime) : "—"}
              </td>
              <td className="py-1.5 pr-3">
                {stop.isRealTime ? formatTime(stop.realDepartureTime) : "—"}
              </td>
              <td className="py-1.5 pr-3">
                {stop.isRealTime ? stop.realPlatform || "—" : "—"}
              </td>
              <td className="py-1.5 pr-3">
                {stop.isRealTime ? (
                  <span className="text-green-600">✓</span>
                ) : (
                  <span className="text-gray-300">—</span>
                )}
              </td>
              <td className="py-1.5">
                {stop.cancelled ? (
                  <span className="text-red-600">✗</span>
                ) : (
                  <span className="text-gray-300">—</span>
                )}
              </td>
              <td className="py-1.5">
                {stop.relevant ? (
                  <span className="text-green-600" title="This stop was used in the merged timetable">✓</span>
                ) : (
                  <span className="text-amber-500" title="This stop was discarded (no country/name match)">✗</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
