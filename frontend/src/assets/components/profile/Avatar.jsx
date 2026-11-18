import { useMyImage } from "../../lib/account.js";

const initials = (u) => (((u?.firstName?.[0] ?? "") + (u?.lastName?.[0] ?? "")) || u?.email?.[0] || "?").toUpperCase();

// Photo de profil du compte connecté, ou ses initiales tant qu'il n'y en a pas.
export default function Avatar({ user, size = 40, className = "" }) {
  const url = useMyImage("avatar");
  const style = { width: size, height: size, fontSize: Math.round(size * 0.38) };
  if (url) return <img src={url} alt="" style={style} className={`shrink-0 rounded-full object-cover ${className}`} />;
  return (
    <span style={style} className={`flex shrink-0 items-center justify-center rounded-full bg-action font-bold text-black ${className}`} aria-hidden="true">
      {initials(user)}
    </span>
  );
}
