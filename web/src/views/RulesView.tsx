import { useEffect, useState } from "react";
import { Plus, Trash2, GripVertical, Save } from "lucide-react";
import { api, type Rule, type RuleField, type RuleOp, type RuleAction } from "@/api";
import { Button, Input, Spinner } from "@/components/ui";

const FIELDS: RuleField[] = ["channel", "title", "videoId"];
const OPS: RuleOp[] = ["contains", "equals", "regex"];
const ACTIONS: RuleAction[] = ["reject", "maxQuality", "preferAudioLang", "preferSubLang", "cache", "ephemeral", "music"];

function blankRule(): Rule {
  return { match: { field: "channel", op: "contains", value: "" }, action: "maxQuality", maxHeight: 720, note: "" };
}

export function RulesView() {
  const [rules, setRules] = useState<Rule[] | null>(null);
  const [dirty, setDirty] = useState(false);
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

  const update = (i: number, r: Rule) => {
    setRules(rules.map((x, j) => (j === i ? r : x)));
    setDirty(true);
  };
  const remove = (i: number) => {
    setRules(rules.filter((_, j) => j !== i));
    setDirty(true);
  };
  const add = () => {
    setRules([...rules, blankRule()]);
    setDirty(true);
  };
  const save = async () => {
    setErr("");
    try {
      const next = await api.putRules(rules);
      setRules(next || []);
      setDirty(false);
      setSaved(true);
      setTimeout(() => setSaved(false), 1500);
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center gap-3 border-b border-border bg-card/60 px-4 py-3">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold">Per-source rules</h2>
          <p className="truncate text-xs text-muted-foreground">Evaluated in order — the first match per concern wins.</p>
        </div>
        <div className="ml-auto flex shrink-0 items-center gap-2">
          {saved && <span className="text-xs text-primary">Saved ✓</span>}
          <Button variant="outline" size="sm" onClick={add}>
            <Plus className="size-4" /> <span className="hidden sm:inline">Add</span>
          </Button>
          <Button size="sm" onClick={save} disabled={!dirty}>
            <Save className="size-4" /> <span className="hidden sm:inline">Save</span>
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="flex flex-col gap-3 p-4">
          {err && <p className="text-sm text-red-400">{err}</p>}
          {rules.length === 0 && (
            <div className="rounded-lg border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
              No rules yet. Add one to fine-tune how matching videos are handled.
            </div>
          )}
          {rules.map((r, i) => (
            <RuleRow key={i} rule={r} index={i} onChange={(x) => update(i, x)} onRemove={() => remove(i)} />
          ))}
        </div>
      </div>
    </div>
  );
}

function RuleRow({ rule, index, onChange, onRemove }: { rule: Rule; index: number; onChange: (r: Rule) => void; onRemove: () => void }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="hidden size-6 items-center justify-center text-muted-foreground sm:flex">
          <GripVertical className="size-4" />
        </div>
        <span className="w-5 text-xs tabular text-muted-foreground">{index + 1}</span>

        <Select value={rule.match.field} options={FIELDS} onChange={(v) => onChange({ ...rule, match: { ...rule.match, field: v as RuleField } })} className="w-[110px]" />
        <Select value={rule.match.op} options={OPS} onChange={(v) => onChange({ ...rule, match: { ...rule.match, op: v as RuleOp } })} className="w-[110px]" />
        <Input
          value={rule.match.value}
          onChange={(e) => onChange({ ...rule, match: { ...rule.match, value: e.target.value } })}
          placeholder="value"
          className="h-8 min-w-[140px] flex-1"
        />

        <span className="hidden text-muted-foreground sm:inline">→</span>

        <Select value={rule.action} options={ACTIONS} onChange={(v) => onChange({ ...rule, action: v as RuleAction })} className="w-[150px]" />

        {rule.action === "maxQuality" && (
          <Input type="number" value={rule.maxHeight ?? 720} onChange={(e) => onChange({ ...rule, maxHeight: Number(e.target.value) })} className="h-8 w-20" />
        )}
        {(rule.action === "preferAudioLang" || rule.action === "preferSubLang") && (
          <Input value={rule.lang ?? ""} onChange={(e) => onChange({ ...rule, lang: e.target.value })} placeholder="lang" className="h-8 w-24" />
        )}

        <Button variant="ghost" size="icon" className="ml-auto size-8 shrink-0 hover:text-red-400" onClick={onRemove}>
          <Trash2 className="size-4" />
        </Button>
      </div>
    </div>
  );
}

function Select({ value, options, onChange, className }: { value: string; options: string[]; onChange: (v: string) => void; className?: string }) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={`h-8 rounded-md border border-input bg-card px-2 text-sm outline-none ${className ?? ""}`}
    >
      {options.map((o) => (
        <option key={o} value={o}>
          {o}
        </option>
      ))}
    </select>
  );
}
