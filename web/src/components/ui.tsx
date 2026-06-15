import { cn } from "@/util";
import { Loader2 } from "lucide-react";

export function Button({
  className,
  variant = "default",
  size = "md",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "ghost" | "outline" | "destructive";
  size?: "sm" | "md" | "icon";
}) {
  const variants = {
    default: "bg-primary text-primary-foreground hover:opacity-90",
    ghost: "hover:bg-accent text-foreground",
    outline: "border border-border hover:bg-accent text-foreground",
    destructive: "text-red-400 hover:bg-red-500/10",
  };
  const sizes = { sm: "h-8 px-2.5 text-sm", md: "h-9 px-3.5 text-sm", icon: "h-9 w-9" };
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center gap-1.5 rounded-md font-medium transition-colors disabled:pointer-events-none disabled:opacity-50",
        variants[variant],
        sizes[size],
        className
      )}
      {...props}
    />
  );
}

export function Input({ className, ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:ring-2 focus:ring-ring/50",
        className
      )}
      {...props}
    />
  );
}

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn("size-4 animate-spin", className)} />;
}

export function Card({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rounded-lg border border-border bg-card", className)} {...props} />;
}
