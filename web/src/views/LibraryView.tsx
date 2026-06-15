import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Play, Square, Trash2, Link2 } from "lucide-react";
import { api, type VideoDTO } from "@/api";
import { fmtDuration } from "@/util";
import { Button, Input, Spinner } from "@/components/ui";

export function LibraryView() {
  const [videos, setVideos] = useState<VideoDTO[]>([]);
  const [link, setLink] = useState("");
  const [adding, setAdding] = useState(false);
  const [err, setErr] = useState("");

  const refresh = () => api.list().then(setVideos).catch(() => {});

  useEffect(() => {
    refresh();
    const ws = openEvents(refresh);
    return ws;
  }, []);

  const add = async () => {
    const v = link.trim();
    if (!v || adding) return;
    setAdding(true);
    setErr("");
    try {
      await api.add(v);
      setLink("");
      refresh();
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="mx-auto max-w-screen-xl space-y-4 p-4">
      <div className="flex gap-2">
        <div className="relative max-w-md flex-1">
          <Link2 className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={link}
            onChange={(e) => setLink(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && add()}
            placeholder="Paste a YouTube ID or URL…"
            className="pl-8"
          />
        </div>
        <Button onClick={add} disabled={adding || !link.trim()}>
          {adding ? <Spinner /> : <Plus className="size-4" />} Add
        </Button>
      </div>
      {err && <p className="text-sm text-red-400">{err}</p>}

      <div className="overflow-hidden rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead className="border-b border-border bg-card/50 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th className="px-3 py-2 font-medium">Title</th>
              <th className="px-3 py-2 font-medium">Channel</th>
              <th className="px-3 py-2 font-medium">State</th>
              <th className="px-3 py-2 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {videos.length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-10 text-center text-muted-foreground">
                  No videos yet. Paste a YouTube link above.
                </td>
              </tr>
            )}
            {videos.map((v) => (
              <tr key={v.id} className="border-b border-border last:border-0 hover:bg-accent/40">
                <td className="max-w-md px-3 py-2">
                  <Link to={`/app/watch/${v.id}`} className="line-clamp-1 font-medium hover:text-primary">
                    {v.title || v.id}
                  </Link>
                  {v.duration > 0 && <span className="ml-2 text-xs text-muted-foreground tabular">{fmtDuration(v.duration)}</span>}
                </td>
                <td className="px-3 py-2 text-muted-foreground">{v.channel}</td>
                <td className="px-3 py-2">
                  <span className={v.active ? "text-primary" : "text-muted-foreground"}>{v.active ? "streaming" : "idle"}</span>
                </td>
                <td className="px-3 py-2">
                  <div className="flex justify-end gap-1">
                    <Link to={`/app/watch/${v.id}`}>
                      <Button variant="ghost" size="icon" title="Play">
                        <Play className="size-4" />
                      </Button>
                    </Link>
                    <Button variant="ghost" size="icon" title="Stop" onClick={() => api.stop(v.id).then(refresh)}>
                      <Square className="size-4" />
                    </Button>
                    <Button variant="ghost" size="icon" title="Delete" onClick={() => api.remove(v.id).then(refresh)}>
                      <Trash2 className="size-4 text-red-400" />
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// openEvents subscribes to the live WebSocket and refreshes on any event.
function openEvents(onEvent: () => void): () => void {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const ws = new WebSocket(`${proto}://${location.host}/api/events`);
  ws.onmessage = () => onEvent();
  return () => ws.close();
}
