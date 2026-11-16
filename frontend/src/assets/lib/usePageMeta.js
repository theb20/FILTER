import { useEffect } from "react";

// Pose le titre et la meta description de la page, puis restaure les anciennes valeurs au départ.
// (Application côté navigateur : pour le référencement, prévoir un pré-rendu ou du SSR.)
export function usePageMeta(title, description) {
  useEffect(() => {
    const previousTitle = document.title;
    document.title = title;
    let tag = document.querySelector('meta[name="description"]');
    const created = !tag;
    if (!tag) {
      tag = document.createElement("meta");
      tag.setAttribute("name", "description");
      document.head.appendChild(tag);
    }
    const previousDescription = tag.getAttribute("content");
    tag.setAttribute("content", description);
    return () => {
      document.title = previousTitle;
      if (created) tag.remove();
      else tag.setAttribute("content", previousDescription ?? "");
    };
  }, [title, description]);
}
