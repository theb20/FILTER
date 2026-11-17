import type { SceneId } from "../types";

/** Texte lu par les lecteurs d'écran et affiché sous la scène : la 3D ne porte jamais seule l'information. */
export const SCENE_META: Record<SceneId, { caption: string }> = {
  "category-tree": { caption: "Les attributs (points jaunes) descendent de la catégorie parente vers ses sous-catégories." },
  "product-lifecycle": { caption: "Un produit passe de Brouillon à À valider, puis Publié, puis Archivé. Il peut être restauré en brouillon." },
  "roles-permissions": { caption: "Chaque rôle apporte ses permissions (cubes colorés) ; le membre cumule celles de tous ses rôles." },
};
