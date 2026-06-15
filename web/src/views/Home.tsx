import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Search } from "lucide-react";
import { api, type DiscoverVideo } from "@/api";
import { follows, history } from "@/store";
import { VideoGrid } from "@/components/VideoGrid";
import { Input, Spinner } from "@/components/ui";

export function Home() {
  const nav = useNavigate();
  const [q, setQ] = useState("");
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
      <form
        className="relative max-w-xl"
        onSubmit={(e) => {
          e.preventDefault();
          if (q.trim()) nav(`/app/search?q=${encodeURIComponent(q.trim())}`);
        }}
      >
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search YouTube…" className="h-10 pl-9" />
      </form>
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
