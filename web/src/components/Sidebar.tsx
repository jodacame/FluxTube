import { Link, useLocation } from "react-router-dom";
import { Compass, Library, ScrollText, Settings as SettingsIcon } from "lucide-react";
import { cn } from "@/util";

export const NAV = [
  { to: "/app", label: "Discover", icon: Compass, match: (p: string) => p.startsWith("/app") },
  { to: "/", label: "Library", icon: Library, match: (p: string) => p === "/" },
  { to: "/rules", label: "Rules", icon: ScrollText, match: (p: string) => p === "/rules" },
  { to: "/settings", label: "Settings", icon: SettingsIcon, match: (p: string) => p === "/settings" },
];

// MobileNav is a horizontal nav shown when the resizable sidebar is hidden.
export function MobileNav() {
  const { pathname } = useLocation();
  return (
    <nav className="flex gap-1 overflow-x-auto border-b border-border bg-sidebar px-2 py-1.5 lg:hidden">
      {NAV.map((n) => {
        const active = n.match(pathname);
        return (
          <Link
            key={n.to}
            to={n.to}
            className={cn(
              "flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors",
              active ? "bg-primary/15 text-foreground" : "text-muted-foreground hover:bg-accent/60"
            )}
          >
            <n.icon className={cn("size-4", active && "text-primary")} />
            {n.label}
          </Link>
        );
      })}
    </nav>
  );
}

export function Sidebar() {
  const { pathname } = useLocation();
  return (
    <div className="flex h-full flex-col bg-sidebar">
      <nav className="flex flex-col gap-0.5 p-2 pt-3">
        {NAV.map((n) => {
          const active = n.match(pathname);
          return (
            <Link
              key={n.to}
              to={n.to}
              className={cn(
                "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors",
                active ? "bg-primary/15 text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
              )}
            >
              <n.icon className={cn("size-4", active && "text-primary")} />
              {n.label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
