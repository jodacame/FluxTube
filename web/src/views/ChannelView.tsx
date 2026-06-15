import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api, type DiscoverVideo, type ChannelInfo } from "@/api";
import { follows } from "@/store";
import { fmtViews } from "@/util";
import { VideoGrid } from "@/components/VideoGrid";
import { Button, Spinner } from "@/components/ui";

export function ChannelView() {
  const { id = "" } = useParams();
  const [info, setInfo] = useState<ChannelInfo | null>(null);
  const [videos, setVideos] = useState<DiscoverVideo[]>([]);
  const [loading, setLoading] = useState(true);
  const [following, setFollowing] = useState(false);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      api.channel(id).then(setInfo).catch(() => {}),
      api.channelVideos(id).then((p) => setVideos(p.videos || [])).catch(() => {}),
    ]).finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (info) setFollowing(follows.has(info.id));
  }, [info]);

  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-20 text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-screen-2xl space-y-5 p-4">
      <div className="flex items-center gap-4">
        {info?.thumbnail && <img src={info.thumbnail} alt="" className="size-16 rounded-full object-cover" />}
        <div className="min-w-0">
          <h1 className="truncate text-xl font-semibold">{info?.name || id}</h1>
          {info && info.subscribers > 0 && <p className="text-sm text-muted-foreground">{fmtViews(info.subscribers)} subscribers</p>}
        </div>
        {info && (
          <Button
            variant={following ? "outline" : "default"}
            className="ml-auto"
            onClick={() => setFollowing(follows.toggle(info.id))}
          >
            {following ? "Following" : "Follow"}
          </Button>
        )}
      </div>
      <VideoGrid videos={videos} />
    </div>
  );
}
