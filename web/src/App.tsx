import { Routes, Route, Link, useNavigate, useLocation } from "react-router-dom";
import { useState } from "react";
import { Play, Search, Library, Settings as SettingsIcon, ScrollText, Tv } from "lucide-react";
import { cn } from "@/util";
import { Input } from "@/components/ui";
import { Home } from "@/views/Home";
import { SearchView } from "@/views/SearchView";
import { ChannelView } from "@/views/ChannelView";
import { Watch } from "@/views/Watch";
import { LibraryView } from "@/views/LibraryView";
import { SettingsView } from "@/views/SettingsView";
import { RulesView } from "@/views/RulesView";

function Brand() {
  return (
    <Link to="/app" className="flex select-none items-center gap-1.5 text-[15px] font-medium">
      <span className="flex size-6 items-center justify-center rounded bg-primary text-primary-foreground">
        <Play className="size-3.5" fill="currentColor" />
      </span>
      <span>
        Flux<b className="text-primary">Tube</b>
      </span>
    </Link>
  );
}

function Header() {
  const nav = useNavigate();
  const loc = useLocation();
  const [q, setQ] = useState("");
  const isManage = !loc.pathname.startsWith("/app");

  return (
    <header className="sticky top-0 z-20 flex items-center gap-3 border-b border-border bg-card/80 px-3 py-2 backdrop-blur">
      <Brand />
      {!isManage && (
        <form
          className="relative ml-2 hidden max-w-md flex-1 sm:block"
          onSubmit={(e) => {
            e.preventDefault();
            if (q.trim()) nav(`/app/search?q=${encodeURIComponent(q.trim())}`);
          }}
        >
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search YouTube…" className="pl-8" />
        </form>
      )}
      <nav className="ml-auto flex items-center gap-1">
        <HeaderLink to="/app" active={!isManage} icon={Tv} label="Watch" />
        <HeaderLink to="/" active={isManage} icon={Library} label="Manage" />
      </nav>
    </header>
  );
}

function HeaderLink({ to, active, icon: Icon, label }: { to: string; active: boolean; icon: typeof Tv; label: string }) {
  return (
    <Link
      to={to}
      className={cn(
        "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
        active ? "bg-primary/15 text-foreground" : "text-muted-foreground hover:bg-accent"
      )}
    >
      <Icon className={cn("size-4", active && "text-primary")} />
      <span className="hidden sm:inline">{label}</span>
    </Link>
  );
}

function ManageTabs() {
  const loc = useLocation();
  const tabs = [
    { to: "/", label: "Library", icon: Library },
    { to: "/rules", label: "Rules", icon: ScrollText },
    { to: "/settings", label: "Settings", icon: SettingsIcon },
  ];
  return (
    <div className="flex gap-1 border-b border-border px-3 py-2">
      {tabs.map((t) => (
        <Link
          key={t.to}
          to={t.to}
          className={cn(
            "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors",
            loc.pathname === t.to ? "bg-accent text-foreground" : "text-muted-foreground hover:bg-accent/60"
          )}
        >
          <t.icon className="size-4" />
          {t.label}
        </Link>
      ))}
    </div>
  );
}

export default function App() {
  const loc = useLocation();
  const isManage = !loc.pathname.startsWith("/app");
  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      {isManage && <ManageTabs />}
      <main className="flex-1">
        <Routes>
          <Route path="/" element={<LibraryView />} />
          <Route path="/rules" element={<RulesView />} />
          <Route path="/settings" element={<SettingsView />} />
          <Route path="/app" element={<Home />} />
          <Route path="/app/search" element={<SearchView />} />
          <Route path="/app/channel/:id" element={<ChannelView />} />
          <Route path="/app/watch/:id" element={<Watch />} />
        </Routes>
      </main>
    </div>
  );
}
