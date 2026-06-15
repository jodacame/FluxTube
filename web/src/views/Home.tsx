import { useEffect, useState } from "react";
import { api, type DiscoverVideo } from "@/api";
import { follows, history } from "@/store";
import { VideoGrid } from "@/components/VideoGrid";
import { Spinner } from "@/components/ui";

export function Home() {
  const [recommended, setRecommended] = useState<DiscoverVideo[]>([]);
  const [trending, setTrending] = useState<DiscoverVideo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fch = follows.list();
    const tasks: Promise<void>[] = [
      api.trending().then((p) => setTrending(p.videos || [])).catch(() => {}),
    ];
    if (fch.length > 0) {
      tasks.push(
        api.recommended(fch, history.list()).then((p) => setRecommended(p.videos || [])).catch(() => {})
      );
    }
    Promise.all(tasks).finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-20 text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-screen-2xl space-y-8 p-4">
      {recommended.length > 0 && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">Recommended for you</h2>
          <VideoGrid videos={recommended} />
        </section>
      )}
      <section>
        <h2 className="mb-3 text-lg font-semibold">Trending</h2>
        <VideoGrid videos={trending} />
      </section>
    </div>
  );
}
