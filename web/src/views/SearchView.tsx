import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Music } from "lucide-react";
import { api, type DiscoverVideo } from "@/api";
import { VideoGrid } from "@/components/VideoGrid";
import { Spinner } from "@/components/ui";

export function SearchView() {
  const [params] = useSearchParams();
  const q = params.get("q") || "";
  const music = params.get("music") === "1";
  const [videos, setVideos] = useState<DiscoverVideo[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  useEffect(() => {
    if (!q) return;
    setLoading(true);
    setErr("");
    api
      .search(q, music)
      .then((p) => setVideos(p.videos || []))
      .catch((e) => setErr(e.message))
      .finally(() => setLoading(false));
  }, [q, music]);

  return (
    <div className="mx-auto max-w-screen-2xl space-y-4 p-4">
      <h2 className="flex items-center gap-2 text-lg font-semibold">
        {music && <Music className="size-5 text-primary" />}
        {music ? "Songs" : "Results"} for “{q}”
      </h2>
      {loading ? (
        <div className="flex items-center gap-2 py-16 text-muted-foreground">
          <Spinner /> Searching…
        </div>
      ) : err ? (
        <p className="text-sm text-red-400">{err}</p>
      ) : (
        <VideoGrid videos={videos} audio={music} />
      )}
    </div>
  );
}
