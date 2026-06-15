import { useEffect, useState } from "react";
import { Activity, Heart } from "lucide-react";
import { api } from "@/api";

const REPO = "https://github.com/jodacame/fluxtube";
const SPONSOR = "https://github.com/sponsors/jodacame";

function GitHubMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden>
      <path d="M12 .5a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.3-1.3-1.7-1.3-1.7-1.1-.7 0-.7 0-.7 1.2 0 1.8 1.2 1.8 1.2 1.1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.8-1.6-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2 0-.3-.5-1.5.2-3.1 0 0 1-.3 3.3 1.2a11.5 11.5 0 0 1 6 0C17.3 4.7 18.3 5 18.3 5c.7 1.6.2 2.8.1 3.1.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .5Z" />
    </svg>
  );
}

export function StatusBar() {
  const [version, setVersion] = useState("");
  const [active, setActive] = useState(0);

  useEffect(() => {
    const pull = () => api.health().then((h) => {
      setVersion(h.version);
      setActive(h.activeSessions);
    }).catch(() => {});
    pull();
    const id = setInterval(pull, 5000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="flex h-7 shrink-0 items-center gap-3 border-t border-border bg-card px-3 text-xs text-muted-foreground">
      <span className="flex items-center gap-1.5 tabular">
        <Activity className="size-3" /> {active} active
      </span>
      <div className="mx-1 hidden h-3.5 w-px bg-border sm:block" />
      <a href={REPO} target="_blank" rel="noreferrer" className="hidden items-center gap-1.5 transition-colors hover:text-foreground sm:flex" title="GitHub">
        <GitHubMark className="size-3.5" />
        <span className="tabular">{version}</span>
      </a>
      <a href={SPONSOR} target="_blank" rel="noreferrer" className="hidden items-center gap-1 text-pink-400 transition-opacity hover:opacity-80 sm:flex" title="Sponsor">
        <Heart className="size-3 fill-current" /> Sponsor
      </a>
      <span className="ml-auto flex select-none items-center gap-1 text-muted-foreground/70">
        Flux<b className="text-primary">Tube</b>
      </span>
    </div>
  );
}
