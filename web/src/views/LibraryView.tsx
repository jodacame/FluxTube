import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { Play, Square, Trash2 } from "lucide-react";
import { api, type VideoDTO, type Resolved } from "@/api";
import { fmtDuration } from "@/util";
import { Button, Spinner } from "@/components/ui";

export function LibraryView() {
  const [videos, setVideos] = useState<VideoDTO[]>([]);
  const [selected, setSelected] = useState<string | null>(null);

  const refresh = () => api.list().then(setVideos).catch(() => {});

  useEffect(() => {
    refresh();
    const proto = location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${location.host}/api/events`);
    ws.onmessage = () => refresh();
    return () => ws.close();
  }, []);

  return (
    <PanelGroup direction="vertical" autoSaveId="ft-lib" className="h-full">
      <Panel defaultSize={64} minSize={25}>
        <div className="h-full overflow-auto">
          <table className="w-full text-sm">
            <thead className="sticky top-0 border-b border-border bg-card text-left text-xs uppercase tracking-wider text-muted-foreground">
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
                  <td colSpan={4} className="px-3 py-12 text-center text-muted-foreground">
                    No videos yet. Paste a YouTube link in the toolbar above.
                  </td>
                </tr>
              )}
              {videos.map((v) => (
                <tr
                  key={v.id}
                  onClick={() => setSelected(v.id)}
                  className={`cursor-pointer border-b border-border last:border-0 ${selected === v.id ? "bg-primary/10" : "hover:bg-accent/40"}`}
                >
                  <td className="max-w-md px-3 py-2">
                    <span className="line-clamp-1 font-medium">{v.title || v.id}</span>
                    {v.duration > 0 && <span className="ml-2 text-xs tabular text-muted-foreground">{fmtDuration(v.duration)}</span>}
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">{v.channel}</td>
                  <td className="px-3 py-2">
                    <span className={v.active ? "text-primary" : "text-muted-foreground"}>{v.active ? "streaming" : "idle"}</span>
                  </td>
                  <td className="px-3 py-2" onClick={(e) => e.stopPropagation()}>
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
      </Panel>
      <PanelResizeHandle className="flex h-px items-center justify-center bg-border transition-colors hover:bg-primary/40 data-[resize-handle-state=drag]:bg-primary/60" />
      <Panel defaultSize={36} minSize={0} collapsible>
        <DetailPanel id={selected} />
      </Panel>
    </PanelGroup>
  );
}

function DetailPanel({ id }: { id: string | null }) {
  const [info, setInfo] = useState<Resolved | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setInfo(null);
    if (!id) return;
    setLoading(true);
    api.get(id).then(setInfo).catch(() => {}).finally(() => setLoading(false));
  }, [id]);

  if (!id) {
    return <div className="flex h-full items-center justify-center text-sm text-muted-foreground">Select a video to inspect its tracks.</div>;
  }
  if (loading) {
    return (
      <div className="flex h-full items-center justify-center gap-2 text-muted-foreground">
        <Spinner /> Resolving…
      </div>
    );
  }
  if (!info) {
    return <div className="flex h-full items-center justify-center text-sm text-red-400">Could not resolve this video.</div>;
  }

  return (
    <div className="h-full space-y-4 overflow-auto p-4">
      <div className="flex gap-3">
        {info.thumbnail && <img src={info.thumbnail} alt="" className="aspect-video w-40 rounded object-cover" />}
        <div className="min-w-0">
          <h3 className="line-clamp-2 font-semibold">{info.title}</h3>
          <p className="text-sm text-muted-foreground">{info.channel}</p>
          <p className="mt-1 text-xs text-muted-foreground tabular">{fmtDuration(info.duration)}</p>
        </div>
      </div>
      <div className="grid gap-4 sm:grid-cols-3">
        <TrackList title={`Video (${info.video.length})`} items={info.video.map((v) => `${v.label} · ${v.codec}${v.hdr ? " HDR" : ""}`)} />
        <TrackList title={`Audio (${info.audio.length})`} items={info.audio.map((a) => `${a.name} · ${a.codec} ${a.bitrate}k`)} />
        <TrackList title={`Subtitles (${info.subs.length})`} items={info.subs.map((s) => `${s.name}${s.auto ? " (auto)" : ""}`)} />
      </div>
    </div>
  );
}

function TrackList({ title, items }: { title: string; items: string[] }) {
  return (
    <div>
      <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</h4>
      <ul className="space-y-1 text-sm">
        {items.length === 0 && <li className="text-muted-foreground">—</li>}
        {items.map((it, i) => (
          <li key={i} className="truncate rounded bg-card px-2 py-1">
            {it}
          </li>
        ))}
      </ul>
    </div>
  );
}
