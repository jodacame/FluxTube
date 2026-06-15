// API client mirroring the Go backend.

export interface VideoDTO {
  id: string;
  title: string;
  channel: string;
  channelId: string;
  thumbnail: string;
  duration: number;
  addedAt: number;
  active: boolean;
  state: string;
}

export interface VideoFormat {
  id: string;
  width: number;
  height: number;
  fps: number;
  codec: string;
  bitrate: number;
  hdr: boolean;
  hasAudio: boolean;
  label: string;
}
export interface AudioTrack {
  id: string;
  lang: string;
  name: string;
  codec: string;
  bitrate: number;
  default: boolean;
}
export interface SubTrack {
  lang: string;
  name: string;
  auto: boolean;
}
export interface Resolved {
  id: string;
  title: string;
  channel: string;
  channelId: string;
  duration: number;
  thumbnail: string;
  description?: string;
  video: VideoFormat[];
  audio: AudioTrack[];
  subs: SubTrack[];
  progressive: VideoFormat[];
}

export interface Settings {
  cache: { mode: string; maxSizeMB: number; segmentSeconds: number; path: string };
  quality: { defaultMaxHeight: number; preferAudioLang: string; preferSubLang: string; autoSubLangs: string[] };
  net: { listenHost: string; listenPort: number };
  youtube: { cookiesFile: string; extractorArg: string };
  discovery: { provider: string; invidiousBaseUrl: string; cacheSeconds: number };
  limits: { maxSessions: number; maxFFmpeg: number; idleTimeoutSec: number };
  apiToken: string;
}

export type RuleField = "channel" | "title" | "videoId";
export type RuleOp = "equals" | "contains" | "regex";
export type RuleAction = "reject" | "maxQuality" | "preferAudioLang" | "preferSubLang" | "cache" | "ephemeral";
export interface Rule {
  match: { field: RuleField; op: RuleOp; value: string };
  action: RuleAction;
  maxHeight?: number;
  lang?: string;
  note?: string;
}

export interface DiscoverVideo {
  id: string;
  title: string;
  channel: string;
  channelId: string;
  thumbnail: string;
  duration: number;
  views: number;
}
export interface DiscoverPage {
  videos: DiscoverVideo[];
  channels?: { id: string; name: string; thumbnail: string; subscribers: number }[];
  nextPage?: string;
}
export interface ChannelInfo {
  id: string;
  name: string;
  thumbnail: string;
  subscribers: number;
  description?: string;
}

async function j<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let msg = `${res.status}`;
    try {
      const b = await res.json();
      msg = b.error || msg;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return res.json() as Promise<T>;
}

export const api = {
  health: () => fetch("/api/health").then((r) => j<{ version: string; activeSessions: number }>(r)),

  // library
  list: () => fetch("/api/videos").then((r) => j<VideoDTO[]>(r)),
  add: (input: string) =>
    fetch("/api/videos", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(/^https?:/.test(input) ? { url: input } : { id: input }),
    }).then((r) => j<VideoDTO>(r)),
  get: (id: string) => fetch(`/api/videos/${id}`).then((r) => j<Resolved>(r)),
  stop: (id: string) => fetch(`/api/videos/${id}/stop`, { method: "POST" }),
  remove: (id: string) => fetch(`/api/videos/${id}`, { method: "DELETE" }),

  // settings / rules
  getSettings: () => fetch("/api/settings").then((r) => j<Settings>(r)),
  putSettings: (s: Settings) =>
    fetch("/api/settings", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(s) }).then((r) => j<Settings>(r)),
  getRules: () => fetch("/api/rules").then((r) => j<Rule[]>(r)),
  putRules: (rules: Rule[]) =>
    fetch("/api/rules", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(rules) }).then((r) => j<Rule[]>(r)),

  // discovery
  search: (q: string, limit = 24) => fetch(`/api/discover/search?q=${encodeURIComponent(q)}&limit=${limit}`).then((r) => j<DiscoverPage>(r)),
  trending: () => fetch(`/api/discover/trending`).then((r) => j<DiscoverPage>(r)),
  channel: (id: string) => fetch(`/api/discover/channel/${encodeURIComponent(id)}`).then((r) => j<ChannelInfo>(r)),
  channelVideos: (id: string, page = 1) => fetch(`/api/discover/channel/${encodeURIComponent(id)}/videos?page=${page}`).then((r) => j<DiscoverPage>(r)),
  related: (id: string) => fetch(`/api/discover/related/${id}`).then((r) => j<DiscoverPage>(r)),
  recommended: (channels: string[], seeds: string[]) =>
    fetch(`/api/discover/recommended`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ channels, seeds, limit: 30 }) }).then((r) => j<DiscoverPage>(r)),

  // stream urls
  masterUrl: (id: string, q?: number) => `${location.origin}/stream/${id}/master.m3u8${q ? `?q=${q}` : ""}`,
  progressiveUrl: (id: string) => `${location.origin}/stream/${id}/progressive`,
};
