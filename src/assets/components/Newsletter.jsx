import { useState } from "react";
import { useNotify } from "../configurations/notificationContext.js";
import { DEMO_NOTICE } from "../config/site.js";

export default function Newsletter() {
  const notify = useNotify();
  const [email, setEmail] = useState("");

  const submit = (e) => {
    e.preventDefault();
    notify(DEMO_NOTICE, "demo"); // aucune inscription réelle
    setEmail("");
  };

  return (
    <section className="mx-2.5 mt-2.5 bg-news px-4 py-10 text-center" aria-labelledby="newsletter-title">
      <h2 id="newsletter-title" className="text-[28px] font-bold">Stay Up To Date</h2>
      <p className="mt-1 font-bold text-white">Subscribe to our newsletter</p>
      <form onSubmit={submit} className="mx-auto mt-5 flex max-w-md">
        <label htmlFor="newsletter-email" className="sr-only">Email address</label>
        <input
          id="newsletter-email"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="Email address"
          className="min-w-0 flex-1 rounded-l-[3px] bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-slate"
        />
        <button type="submit" className="rounded-r-[3px] bg-slate px-5 font-medium text-white hover:bg-night">
          Subscribe
        </button>
      </form>
    </section>
  );
}
