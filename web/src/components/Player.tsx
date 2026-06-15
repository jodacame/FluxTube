import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { api, type SubTrack } from "@/api";
import { Button } from "@/components/ui";

interface Track {
  id: number;
  name: string;
}

// Player plays a video two ways:
//  - "progressive" (default): the muxed H.264/AAC stream, natively seekable and
//    reliable in every browser, with subtitles as native <track> elements.
//  - "hls": the multi-track HLS master (selectable audio/subtitles) for clients
//    that handle adaptive fMP4 (the production target is mpv-based LumoraTV).
export function Player({ id, subs, poster }: { id: string; subs: SubTrack[]; poster?: string }) {
  const [mode, setMode] = useState<"progressive" | "hls">("progressive");
  return (
    <div className="space-y-2">
      {mode === "progressive" ? <Progressive id={id} subs={subs} poster={poster} /> : <HlsPlayer id={id} poster={poster} />}
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span>Playback:</span>
        <Button size="sm" variant={mode === "progressive" ? "default" : "outline"} onClick={() => setMode("progressive")}>
          Progressive
        </Button>
        <Button size="sm" variant={mode === "hls" ? "default" : "outline"} onClick={() => setMode("hls")}>
          Multi-track (HLS)
        </Button>
      </div>
    </div>
  );
}

function Progressive({ id, subs, poster }: { id: string; subs: SubTrack[]; poster?: string }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [sel, setSel] = useState(-1); // -1 = subtitles off

  // Take explicit control of which text track is shown so the browser never
  // displays two at once (default attribute + language auto-select).
  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    const apply = () => {
      const tt = v.textTracks;
      for (let i = 0; i < tt.length; i++) tt[i].mode = i === sel ? "showing" : "disabled";
    };
    apply();
    v.textTracks.addEventListener?.("addtrack", apply);
    return () => v.textTracks.removeEventListener?.("addtrack", apply);
  }, [sel, id]);

  return (
    <div className="space-y-2">
      <div className="overflow-hidden rounded-lg border border-border bg-black">
        <video ref={videoRef} key={`prog-${id}`} poster={poster} controls playsInline crossOrigin="anonymous" className="aspect-video w-full bg-black">
          <source src={api.progressiveUrl(id)} />
          {subs.map((s) => (
            <track key={s.lang} kind="subtitles" src={api.subUrl(id, s.lang)} srcLang={s.lang} label={s.name || s.lang} />
          ))}
        </video>
      </div>
      {subs.length > 0 && (
        <Select
          label="Subtitles"
          tracks={[{ id: -1, name: "Off" }, ...subs.map((s, i) => ({ id: i, name: (s.name || s.lang) + (s.auto ? " (auto)" : "") }))]}
          value={sel}
          onChange={setSel}
        />
      )}
    </div>
  );
}

function HlsPlayer({ id, poster }: { id: string; poster?: string }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [audioTracks, setAudioTracks] = useState<Track[]>([]);
  const [subTracks, setSubTracks] = useState<Track[]>([]);
  const [audioId, setAudioId] = useState(-1);
  const [subId, setSubId] = useState(-1);
  const [error, setError] = useState("");

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    setError("");
    const src = api.masterUrl(id);

    if (Hls.isSupported()) {
      const hls = new Hls({ enableWebVTT: true });
      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);
      hls.on(Hls.Events.MANIFEST_PARSED, () => video.play().catch(() => {}));
      hls.on(Hls.Events.AUDIO_TRACKS_UPDATED, () => {
        setAudioTracks(hls.audioTracks.map((t, i) => ({ id: i, name: t.name || t.lang || `Audio ${i + 1}` })));
        setAudioId(hls.audioTrack);
      });
      hls.on(Hls.Events.SUBTITLE_TRACKS_UPDATED, () => {
        setSubTracks(hls.subtitleTracks.map((t, i) => ({ id: i, name: t.name || t.lang || `Sub ${i + 1}` })));
        hls.subtitleTrack = -1; // start with subtitles off; user opts in
        setSubId(-1);
      });
      hls.on(Hls.Events.ERROR, (_e, data) => {
        if (data.fatal) setError(`Playback error: ${data.details}`);
      });
      return () => {
        hls.destroy();
        hlsRef.current = null;
      };
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = src;
      video.play().catch(() => {});
    } else {
      setError("HLS is not supported in this browser.");
    }
  }, [id]);

  const selectAudio = (n: number) => {
    if (hlsRef.current) hlsRef.current.audioTrack = n;
    setAudioId(n);
  };
  const selectSub = (n: number) => {
    if (hlsRef.current) hlsRef.current.subtitleTrack = n;
    setSubId(n);
  };

  return (
    <div className="space-y-2">
      <div className="overflow-hidden rounded-lg border border-border bg-black">
        <video ref={videoRef} poster={poster} controls playsInline className="aspect-video w-full bg-black" />
      </div>
      {error && <p className="text-sm text-red-400">{error} — try Progressive.</p>}
      <div className="flex flex-wrap gap-3">
        {audioTracks.length > 1 && <Select label="Audio" tracks={audioTracks} value={audioId} onChange={selectAudio} />}
        {subTracks.length > 0 && <Select label="Subtitles" tracks={[{ id: -1, name: "Off" }, ...subTracks]} value={subId} onChange={selectSub} />}
      </div>
    </div>
  );
}

function Select({ label, tracks, value, onChange }: { label: string; tracks: Track[]; value: number; onChange: (id: number) => void }) {
  return (
    <label className="flex items-center gap-2 text-sm text-muted-foreground">
      <span>{label}</span>
      <select value={value} onChange={(e) => onChange(Number(e.target.value))} className="h-8 rounded-md border border-input bg-card px-2 text-foreground outline-none">
        {tracks.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name}
          </option>
        ))}
      </select>
    </label>
  );
}
