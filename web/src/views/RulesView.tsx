import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { api, type Rule, type RuleField, type RuleOp, type RuleAction } from "@/api";
import { Button, Card, Input, Spinner } from "@/components/ui";

const FIELDS: RuleField[] = ["channel", "title", "videoId"];
const OPS: RuleOp[] = ["equals", "contains", "regex"];
const ACTIONS: RuleAction[] = ["reject", "maxQuality", "preferAudioLang", "preferSubLang", "cache", "ephemeral"];

export function RulesView() {
  const [rules, setRules] = useState<Rule[] | null>(null);
  const [saved, setSaved] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    api.getRules().then((r) => setRules(r || [])).catch((e) => setErr(e.message));
  }, []);

  if (!rules) {
    return (
      <div className="flex items-center gap-2 p-6 text-muted-foreground">
        <Spinner /> Loading rules…
      </div>
    );
  }

  const update = (i: number, patch: Partial<Rule>) => setRules(rules.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  const updateMatch = (i: number, patch: Partial<Rule["match"]>) =>
    setRules(rules.map((r, idx) => (idx === i ? { ...r, match: { ...r.match, ...patch } } : r)));

  const add = () => setRules([...rules, { match: { field: "channel", op: "contains", value: "" }, action: "maxQuality", maxHeight: 720, note: "" }]);
  const remove = (i: number) => setRules(rules.filter((_, idx) => idx !== i));

  const save = async () => {
    setErr("");
    try {
      const next = await api.putRules(rules);
      setRules(next || []);
      setSaved(true);
      setTimeout(() => setSaved(false), 1500);
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  return (
    <div className="mx-auto max-w-3xl space-y-3 p-4">
      <p className="text-sm text-muted-foreground">Rules are evaluated in order; the first match per concern wins.</p>
      {rules.map((r, i) => (
        <Card key={i} className="flex flex-wrap items-center gap-2 p-3">
          <Select value={r.match.field} options={FIELDS} onChange={(v) => updateMatch(i, { field: v as RuleField })} />
          <Select value={r.match.op} options={OPS} onChange={(v) => updateMatch(i, { op: v as RuleOp })} />
          <Input className="w-40" value={r.match.value} onChange={(e) => updateMatch(i, { value: e.target.value })} placeholder="value" />
          <span className="text-muted-foreground">→</span>
          <Select value={r.action} options={ACTIONS} onChange={(v) => update(i, { action: v as RuleAction })} />
          {r.action === "maxQuality" && (
            <Input className="w-20" type="number" value={r.maxHeight ?? 720} onChange={(e) => update(i, { maxHeight: Number(e.target.value) })} />
          )}
          {(r.action === "preferAudioLang" || r.action === "preferSubLang") && (
            <Input className="w-20" value={r.lang ?? ""} onChange={(e) => update(i, { lang: e.target.value })} placeholder="lang" />
          )}
          <Button variant="ghost" size="icon" className="ml-auto" onClick={() => remove(i)}>
            <Trash2 className="size-4 text-red-400" />
          </Button>
        </Card>
      ))}
      <div className="flex items-center gap-3">
        <Button variant="outline" onClick={add}>
          <Plus className="size-4" /> Add rule
        </Button>
        <Button onClick={save}>Save</Button>
        {saved && <span className="text-sm text-primary">Saved ✓</span>}
      </div>
      {err && <p className="text-sm text-red-400">{err}</p>}
    </div>
  );
}

function Select({ value, options, onChange }: { value: string; options: string[]; onChange: (v: string) => void }) {
  return (
    <select value={value} onChange={(e) => onChange(e.target.value)} className="h-9 rounded-md border border-input bg-card px-2 text-sm outline-none">
      {options.map((o) => (
        <option key={o} value={o}>
          {o}
        </option>
      ))}
    </select>
  );
}
