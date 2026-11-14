import { useEffect } from "react";
import Todo from "../components/Todo.jsx";

// Pages d'information (contenu à fournir) : évite les liens morts du footer.
export default function InfoPage({ title }) {
  useEffect(() => {
    document.title = `${title} — FilterFinder`;
  }, [title]);

  return (
    <section className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-8 shadow-card">
      <h1 className="text-[40px] font-bold leading-tight">{title}</h1>
      <p className="mt-4"><Todo>{title} — content to be provided</Todo></p>
    </section>
  );
}
