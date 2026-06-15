import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { api, type Resolved, type DiscoverVideo } from "@/api";
import { history } from "@/store";
import { fmtViews } from "@/util";
import { Player } from "@/components/Player";
import { Spinner } from "@/components/ui";

export function Watch() {
  const { id = "" } = useParams();
  const [info, setInfo] = useState<Resolved | null>(null);
  const [related, setRelated] = useState<DiscoverVideo[]>([]);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setErr("");
    setInfo(null);
    history.push(id);
    api
      .get(id)
      .then(setInfo)
      .catch((e) => setErr(e.message))
      .finally(() => setLoading(false));
    api.related(id).then((p) => setRelated(p.videos || [])).catch(() => {});

    // Stop the session when leaving this video so it doesn't linger as
    // "streaming" and its cache is released promptly.
    return () => {
      api.stop(id).catch(() => {});
    };
  }, [id]);

  return (
    <div className="mx-auto grid max-w-screen-2xl gap-6 p-4 lg:grid-cols-[1fr_360px]">
      <div className="min-w-0 space-y-3">
        {loading ? (
          <div className="flex aspect-video items-center justify-center gap-2 rounded-lg border border-border text-muted-foreground">
            <Spinner /> Resolving…
          </div>
        ) : err ? (
          <div className="flex aspect-video items-center justify-center rounded-lg border border-border p-4 text-center text-sm text-red-400">
            {err}
          </div>
        ) : (
          <Player id={id} subs={info?.subs ?? []} poster={info?.thumbnail} />
        )}

        {info && (
          <div className="space-y-2">
            <h1 className="text-lg font-semibold leading-snug">{info.title}</h1>
            <Link to={`/app/channel/${info.channelId}`} className="inline-block text-sm font-medium text-muted-foreground hover:text-foreground">
              {info.channel}
            </Link>
            <div className="flex flex-wrap gap-2 pt-1 text-xs text-muted-foreground">
              {info.video[0] && <Badge>{info.video[0].label}</Badge>}
              {info.audio.length > 0 && <Badge>{info.audio.length} audio</Badge>}
              {info.subs.length > 0 && <Badge>{info.subs.length} subtitles</Badge>}
            </div>
            {info.description && <Description text={info.description} />}
          </div>
        )}
      </div>

      <aside className="space-y-3">
        <h2 className="text-sm font-semibold text-muted-foreground">Related</h2>
        {related.map((v) => (
          <Link key={v.id} to={`/app/watch/${v.id}`} className="flex gap-2 rounded-md p-1 hover:bg-accent">
            <img src={v.thumbnail} alt="" className="aspect-video w-36 shrink-0 rounded object-cover" />
            <div className="min-w-0">
              <p className="line-clamp-2 text-sm font-medium leading-snug">{v.title}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {v.channel}
                {v.views > 0 && <span> · {fmtViews(v.views)}</span>}
              </p>
            </div>
          </Link>
        ))}
      </aside>
    </div>
  );
}

function Badge({ children }: { children: React.ReactNode }) {
  return <span className="rounded-full bg-accent px-2 py-0.5 font-medium text-accent-foreground">{children}</span>;
}

// Description shows the full text, collapsed by default with a toggle so long
// descriptions don't get cut off permanently.
function Description({ text }: { text: string }) {
  const [open, setOpen] = useState(false);
  const long = text.length > 280;
  return (
    <div className="rounded-md bg-card p-3 text-sm text-muted-foreground">
      <p className={`whitespace-pre-wrap break-words ${open || !long ? "" : "line-clamp-4"}`}>{text}</p>
      {long && (
        <button onClick={() => setOpen((v) => !v)} className="mt-2 text-xs font-medium text-primary hover:underline">
          {open ? "Show less" : "Show more"}
        </button>
      )}
    </div>
  );
}
