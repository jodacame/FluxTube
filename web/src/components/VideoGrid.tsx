import { Link } from "react-router-dom";
import type { DiscoverVideo } from "@/api";
import { fmtDuration, fmtViews, shortName } from "@/util";

export function VideoGrid({ videos }: { videos: DiscoverVideo[] }) {
  if (videos.length === 0) {
    return <p className="py-10 text-center text-sm text-muted-foreground">Nothing to show.</p>;
  }
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {videos.map((v) => (
        <VideoCard key={v.id} v={v} />
      ))}
    </div>
  );
}

function VideoCard({ v }: { v: DiscoverVideo }) {
  return (
    <Link to={`/app/watch/${v.id}`} className="group block">
      <div className="relative overflow-hidden rounded-lg border border-border bg-card">
        <img src={v.thumbnail} alt="" loading="lazy" className="aspect-video w-full object-cover transition-transform group-hover:scale-[1.03]" />
        {v.duration > 0 && (
          <span className="absolute bottom-1.5 right-1.5 rounded bg-black/80 px-1.5 py-0.5 text-xs font-medium tabular text-white">
            {fmtDuration(v.duration)}
          </span>
        )}
      </div>
      <div className="mt-2 px-0.5">
        <p className="line-clamp-2 text-sm font-medium leading-snug">{shortName(v.title, 90)}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {v.channel}
          {v.views > 0 && <span> · {fmtViews(v.views)} views</span>}
        </p>
      </div>
    </Link>
  );
}
