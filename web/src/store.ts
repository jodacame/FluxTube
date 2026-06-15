// Client-side state the headless backend deliberately does not keep: followed
// channels and watch history live in the browser (NewPipe/FreeTube model).

const FOLLOWS = "ft.follows";
const HISTORY = "ft.history";

function read(key: string): string[] {
  try {
    return JSON.parse(localStorage.getItem(key) || "[]");
  } catch {
    return [];
  }
}
function write(key: string, v: string[]) {
  localStorage.setItem(key, JSON.stringify(v));
}

export const follows = {
  list: () => read(FOLLOWS),
  has: (id: string) => read(FOLLOWS).includes(id),
  toggle: (id: string) => {
    const cur = read(FOLLOWS);
    const next = cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id];
    write(FOLLOWS, next);
    return next.includes(id);
  },
};

export const history = {
  list: () => read(HISTORY),
  push: (id: string) => {
    const cur = read(HISTORY).filter((x) => x !== id);
    write(HISTORY, [id, ...cur].slice(0, 100));
  },
};
