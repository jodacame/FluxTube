import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { cn } from "@/util";

interface Track {
  id: number;
  name: string;
  lang?: string;
}

// Player loads an HLS master with hls.js (or native HLS on Safari) and exposes
// the audio and subtitle tracks the stream carries for selection — the MKV-like
// multi-track experience, but seekable.
export function Player({ src, poster }: { src: string; poster?: string }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [audioTracks, setAudioTracks] = useState<Track[]>([]);
  const [subTracks, setSubTracks] = useState<Track[]>([]);
  const [audioId, setAudioId] = useState<number>(-1);
  const [subId, setSubId] = useState<number>(-1);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    setError("");

    if (Hls.isSupported()) {
      const hls = new Hls({ enableWebVTT: true, lowLatencyMode: false });
      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().catch(() => {});
      });
      hls.on(Hls.Events.AUDIO_TRACKS_UPDATED, () => {
        setAudioTracks(hls.audioTracks.map((t, i) => ({ id: i, name: t.name || t.lang || `Audio ${i + 1}`, lang: t.lang })));
        setAudioId(hls.audioTrack);
      });
      hls.on(Hls.Events.SUBTITLE_TRACKS_UPDATED, () => {
        setSubTracks(hls.subtitleTracks.map((t, i) => ({ id: i, name: t.name || t.lang || `Sub ${i + 1}`, lang: t.lang })));
        setSubId(hls.subtitleTrack);
      });
      hls.on(Hls.Events.ERROR, (_e, data) => {
        if (data.fatal) setError(`Playback error: ${data.details}`);
      });
      return () => {
        hls.destroy();
        hlsRef.current = null;
      };
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = src; // native HLS (Safari)
      video.play().catch(() => {});
    } else {
      setError("HLS is not supported in this browser.");
    }
  }, [src]);

  const selectAudio = (id: number) => {
    if (hlsRef.current) hlsRef.current.audioTrack = id;
    setAudioId(id);
  };
  const selectSub = (id: number) => {
    if (hlsRef.current) hlsRef.current.subtitleTrack = id;
    setSubId(id);
  };

  return (
    <div className="space-y-2">
      <div className="overflow-hidden rounded-lg border border-border bg-black">
        <video ref={videoRef} poster={poster} controls playsInline className="aspect-video w-full bg-black" />
      </div>
      {error && <p className="text-sm text-red-400">{error}</p>}
      <div className="flex flex-wrap gap-3">
        {audioTracks.length > 1 && (
          <TrackSelect label="Audio" tracks={audioTracks} value={audioId} onChange={selectAudio} />
        )}
        {subTracks.length > 0 && (
          <TrackSelect label="Subtitles" tracks={[{ id: -1, name: "Off" }, ...subTracks]} value={subId} onChange={selectSub} />
        )}
      </div>
    </div>
  );
}

function TrackSelect({ label, tracks, value, onChange }: { label: string; tracks: Track[]; value: number; onChange: (id: number) => void }) {
  return (
    <label className={cn("flex items-center gap-2 text-sm text-muted-foreground")}>
      <span>{label}</span>
      <select
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="h-8 rounded-md border border-input bg-card px-2 text-foreground outline-none"
      >
        {tracks.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name}
          </option>
        ))}
      </select>
    </label>
  );
}
