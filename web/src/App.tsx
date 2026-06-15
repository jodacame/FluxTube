import { Routes, Route, useNavigate } from "react-router-dom";
import { useState } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { Play, Link2, GripVertical } from "lucide-react";
import { Button, Input } from "@/components/ui";
import { Sidebar, MobileNav } from "@/components/Sidebar";
import { StatusBar } from "@/components/StatusBar";
import { Home } from "@/views/Home";
import { SearchView } from "@/views/SearchView";
import { ChannelView } from "@/views/ChannelView";
import { Watch } from "@/views/Watch";
import { LibraryView } from "@/views/LibraryView";
import { SettingsView } from "@/views/SettingsView";
import { RulesView } from "@/views/RulesView";
import { extractId } from "@/util";

function Toolbar() {
  const nav = useNavigate();
  const [link, setLink] = useState("");

  // Playing does not save to the library: a YouTube id/url goes straight to the
  // player; anything else is treated as a search query.
  const go = () => {
    const v = link.trim();
    if (!v) return;
    const id = extractId(v);
    setLink("");
    if (id) nav(`/app/watch/${id}`);
    else nav(`/app/search?q=${encodeURIComponent(v)}`);
  };

  return (
    <div className="flex shrink-0 items-center gap-2 border-b border-border bg-card/60 px-2 py-2 sm:px-3">
      <div className="flex shrink-0 select-none items-center gap-1.5 pr-1">
        <span className="flex size-6 items-center justify-center rounded bg-primary text-primary-foreground">
          <Play className="size-3.5" fill="currentColor" />
        </span>
        <span className="hidden font-medium sm:inline">
          Flux<b className="text-primary">Tube</b>
        </span>
      </div>
      <div className="mx-1 hidden h-6 w-px bg-border sm:block" />
      <div className="flex min-w-0 flex-1 items-center gap-2 sm:max-w-md">
        <div className="relative flex-1">
          <Link2 className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={link} onChange={(e) => setLink(e.target.value)} onKeyDown={(e) => e.key === "Enter" && go()} placeholder="Play a YouTube ID/URL or search…" className="h-8 pl-8" />
        </div>
        <Button size="sm" className="h-8" onClick={go} disabled={!link.trim()}>
          <Play className="size-4" fill="currentColor" />
          <span className="hidden sm:inline">Go</span>
        </Button>
      </div>
      <div className="ml-auto hidden items-center gap-1.5 pr-1 text-xs text-muted-foreground md:flex">
        <span className="size-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px] shadow-emerald-400" />
        live
      </div>
    </div>
  );
}

export default function App() {
  return (
    <div className="flex h-screen flex-col bg-background text-foreground">
      <Toolbar />
      <MobileNav />
      <PanelGroup direction="horizontal" autoSaveId="ft-shell" className="flex-1">
        <Panel defaultSize={18} minSize={13} maxSize={28} className="hidden lg:block">
          <Sidebar />
        </Panel>
        <PanelResizeHandle className="hidden w-px bg-border transition-colors hover:bg-primary/40 data-[resize-handle-state=drag]:bg-primary/60 lg:flex">
          <span className="relative flex w-px items-center justify-center">
            <span className="absolute z-10 flex h-4 w-3 items-center justify-center rounded-sm border border-border bg-border">
              <GripVertical className="size-2.5" />
            </span>
          </span>
        </PanelResizeHandle>
        <Panel defaultSize={82} className="min-w-0">
          <div className="h-full overflow-y-auto">
            <Routes>
              <Route path="/" element={<LibraryView />} />
              <Route path="/rules" element={<RulesView />} />
              <Route path="/settings" element={<SettingsView />} />
              <Route path="/app" element={<Home />} />
              <Route path="/app/search" element={<SearchView />} />
              <Route path="/app/channel/:id" element={<ChannelView />} />
              <Route path="/app/watch/:id" element={<Watch />} />
            </Routes>
          </div>
        </Panel>
      </PanelGroup>
      <StatusBar />
    </div>
  );
}
