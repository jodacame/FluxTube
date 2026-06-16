import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { api, type SubTrack } from "@/api";
import { Button } from "@/components/ui";

interface Cue {
  s: number;
  e: number;
  text: string;
}

// parseVTT extracts cues from a WebVTT document.
function parseVTT(text: string): Cue[] {
  const cues: Cue[] = [];
  const re = /(\d+):(\d+):(\d+)\.(\d+)\s*-->\s*(\d+):(\d+):(\d+)\.(\d+)/;
  const blocks = text.replace(/\r/g, "").split("\n\n");
  for (const b of blocks) {
    const lines = b.split("\n").filter((l) => l.trim() !== "");
    const ti = lines.findIndex((l) => l.includes("-->"));
    if (ti < 0) continue;
    const m = lines[ti].match(re);
    if (!m) continue;
    const s = +m[1] * 3600 + +m[2] * 60 + +m[3] + +m[4] / 1000;
    const e = +m[5] * 3600 + +m[6] * 60 + +m[7] + +m[8] / 1000;
    const body = lines
      .slice(ti + 1)
      .join("\n")
      .replace(/<[^>]+>/g, ""); // strip styling tags
    cues.push({ s, e, text: body });
  }
  return cues;
}

// useActiveCue renders the current subtitle ourselves so exactly one line shows,
// regardless of browser/hls.js native-track quirks.
function useActiveCue(videoRef: React.RefObject<HTMLVideoElement>, vttUrl: string | null): string {
  const [cues, setCues] = useState<Cue[]>([]);
  const [text, setText] = useState("");

  useEffect(() => {
    setText("");
    if (!vttUrl) {
      setCues([]);
      return;
    }
    let cancelled = false;
    fetch(vttUrl)
      .then((r) => (r.ok ? r.text() : ""))
      .then((t) => {
        if (!cancelled) setCues(parseVTT(t));
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [vttUrl]);

  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    const onTime = () => {
      const t = v.currentTime;
      const cue = cues.find((c) => t >= c.s && t <= c.e);
      setText(cue ? cue.text : "");
    };
    v.addEventListener("timeupdate", onTime);
    return () => v.removeEventListener("timeupdate", onTime);
  }, [cues, videoRef]);

  return text;
}

function SubtitleLayer({ text }: { text: string }) {
  if (!text) return null;
  return (
    <div className="pointer-events-none absolute inset-x-0 bottom-[8%] flex justify-center px-4">
      <span className="whitespace-pre-line rounded bg-black/70 px-2 py-1 text-center text-[clamp(14px,2.4vw,24px)] leading-snug text-white">
        {text}
      </span>
    </div>
  );
}

export function Player({ id, subs, poster }: { id: string; subs: SubTrack[]; poster?: string }) {
  const [mode, setMode] = useState<"progressive" | "hls">("progressive");
  const [sub, setSub] = useState(-1); // index into subs, -1 = off

  const subUrl = sub >= 0 && sub < subs.length ? api.subUrl(id, subs[sub].lang) : null;

  return (
    <div className="space-y-2">
      {mode === "progressive" ? (
        <Progressive id={id} poster={poster} subUrl={subUrl} />
      ) : (
        <HlsPlayer id={id} poster={poster} subUrl={subUrl} />
      )}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Playback:</span>
          <Button size="sm" variant={mode === "progressive" ? "default" : "outline"} onClick={() => setMode("progressive")}>
            Progressive
          </Button>
          <Button size="sm" variant={mode === "hls" ? "default" : "outline"} onClick={() => setMode("hls")}>
            Multi-track (HLS)
          </Button>
        </div>
        {subs.length > 0 && (
          <Select
            label="Subtitles"
            options={["Off", ...subs.map((s) => (s.name || s.lang) + (s.auto ? " (auto)" : ""))]}
            value={sub + 1}
            onChange={(i) => setSub(i - 1)}
          />
        )}
      </div>
    </div>
  );
}

function Progressive({ id, poster, subUrl }: { id: string; poster?: string; subUrl: string | null }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const cue = useActiveCue(videoRef, subUrl);
  return (
    <div className="relative overflow-hidden rounded-lg border border-border bg-black">
      <video ref={videoRef} key={`prog-${id}`} src={api.progressiveUrl(id)} poster={poster} controls playsInline className="aspect-video w-full bg-black" />
      <SubtitleLayer text={cue} />
    </div>
  );
}

function HlsPlayer({ id, poster, subUrl }: { id: string; poster?: string; subUrl: string | null }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const cue = useActiveCue(videoRef, subUrl);
  const [audioTracks, setAudioTracks] = useState<string[]>([]);
  const [audioId, setAudioId] = useState(-1);
  const [error, setError] = useState("");

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    setError("");
    const src = api.masterUrl(id);
    if (Hls.isSupported()) {
      // We render subtitles ourselves, so disable hls.js native subtitle handling.
      const hls = new Hls({ renderTextTracksNatively: false });
      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);
      hls.on(Hls.Events.MANIFEST_PARSED, () => video.play().catch(() => {}));
      hls.on(Hls.Events.AUDIO_TRACKS_UPDATED, () => {
        setAudioTracks(hls.audioTracks.map((t, i) => t.name || t.lang || `Audio ${i + 1}`));
        setAudioId(hls.audioTrack);
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

  const selectAudio = (i: number) => {
    if (hlsRef.current) hlsRef.current.audioTrack = i;
    setAudioId(i);
  };

  return (
    <div className="space-y-2">
      <div className="relative overflow-hidden rounded-lg border border-border bg-black">
        <video ref={videoRef} poster={poster} controls playsInline className="aspect-video w-full bg-black" />
        <SubtitleLayer text={cue} />
      </div>
      {error && <p className="text-sm text-red-400">{error} — try Progressive.</p>}
      {audioTracks.length > 1 && <Select label="Audio" options={audioTracks} value={audioId} onChange={selectAudio} />}
    </div>
  );
}

function Select({ label, options, value, onChange }: { label: string; options: string[]; value: number; onChange: (i: number) => void }) {
  return (
    <label className="flex items-center gap-2 text-sm text-muted-foreground">
      <span>{label}</span>
      <select value={value} onChange={(e) => onChange(Number(e.target.value))} className="h-8 rounded-md border border-input bg-card px-2 text-foreground outline-none">
        {options.map((o, i) => (
          <option key={i} value={i}>
            {o}
          </option>
        ))}
      </select>
    </label>
  );
}
