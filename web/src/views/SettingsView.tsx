import { useEffect, useState } from "react";
import { api, type Settings } from "@/api";
import { Button, Input, Card, Spinner } from "@/components/ui";

export function SettingsView() {
  const [s, setS] = useState<Settings | null>(null);
  const [saved, setSaved] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    api.getSettings().then(setS).catch((e) => setErr(e.message));
  }, []);

  if (!s) {
    return (
      <div className="flex items-center gap-2 p-6 text-muted-foreground">
        <Spinner /> Loading settings…
      </div>
    );
  }

  const save = async () => {
    setErr("");
    try {
      const next = await api.putSettings(s);
      setS(next);
      setSaved(true);
      setTimeout(() => setSaved(false), 1500);
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <Card className="space-y-4 p-4">
        <h3 className="font-semibold">Quality</h3>
        <Field label="Default max height (p)">
          <Input
            type="number"
            value={s.quality.defaultMaxHeight}
            onChange={(e) => setS({ ...s, quality: { ...s.quality, defaultMaxHeight: Number(e.target.value) } })}
          />
        </Field>
        <Field label="Preferred audio language">
          <Input
            value={s.quality.preferAudioLang}
            onChange={(e) => setS({ ...s, quality: { ...s.quality, preferAudioLang: e.target.value } })}
            placeholder="e.g. en"
          />
        </Field>
        <Field label="Auto-caption languages (comma separated)">
          <Input
            value={s.quality.autoSubLangs.join(", ")}
            onChange={(e) => setS({ ...s, quality: { ...s.quality, autoSubLangs: e.target.value.split(",").map((x) => x.trim()).filter(Boolean) } })}
          />
        </Field>
      </Card>

      <Card className="space-y-3 p-4">
        <h3 className="font-semibold">Music</h3>
        <label className="flex items-center gap-3 text-sm">
          <input
            type="checkbox"
            className="size-4 accent-[var(--primary)]"
            checked={s.music.autoSave}
            onChange={(e) => setS({ ...s, music: { ...s.music, autoSave: e.target.checked } })}
          />
          <span>
            Auto-save audio when it's music
            <span className="block text-xs text-muted-foreground">
              Detects songs (Music category, “- Topic”/Vevo channels) and saves the audio automatically for instant replay.
            </span>
          </span>
        </label>
      </Card>

      <Card className="space-y-4 p-4">
        <h3 className="font-semibold">YouTube</h3>
        <Field label="Cookies file (optional, unlocks restricted videos)">
          <Input
            value={s.youtube.cookiesFile}
            onChange={(e) => setS({ ...s, youtube: { ...s.youtube, cookiesFile: e.target.value } })}
            placeholder="/config/cookies.txt"
          />
        </Field>
      </Card>

      <Card className="space-y-4 p-4">
        <h3 className="font-semibold">Cache & limits</h3>
        <Field label="Segment seconds">
          <Input
            type="number"
            value={s.cache.segmentSeconds}
            onChange={(e) => setS({ ...s, cache: { ...s.cache, segmentSeconds: Number(e.target.value) } })}
          />
        </Field>
        <Field label="Idle timeout (seconds)">
          <Input
            type="number"
            value={s.limits.idleTimeoutSec}
            onChange={(e) => setS({ ...s, limits: { ...s.limits, idleTimeoutSec: Number(e.target.value) } })}
          />
        </Field>
        <Field label="Max concurrent ffmpeg">
          <Input
            type="number"
            value={s.limits.maxFFmpeg}
            onChange={(e) => setS({ ...s, limits: { ...s.limits, maxFFmpeg: Number(e.target.value) } })}
          />
        </Field>
      </Card>

      <Card className="space-y-4 p-4">
        <h3 className="font-semibold">Security</h3>
        <Field label="API token (guards /api/*; blank = open)">
          <Input
            value={s.apiToken}
            onChange={(e) => setS({ ...s, apiToken: e.target.value })}
            placeholder="leave blank for no auth"
          />
        </Field>
      </Card>

      {err && <p className="text-sm text-red-400">{err}</p>}
      <div className="flex items-center gap-3">
        <Button onClick={save}>Save</Button>
        {saved && <span className="text-sm text-primary">Saved ✓</span>}
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-sm text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}
