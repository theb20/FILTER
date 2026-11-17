export type Placement = "auto" | "top" | "bottom" | "left" | "right";

/** Une étape : `target` = valeur de l'attribut `data-guide` de l'élément à montrer (absent = étape centrée, sans projecteur). */
export type SceneId = "category-tree" | "product-lifecycle" | "roles-permissions";

export type GuideStep = { id: string; /** Scène 3D pédagogique affichée dans la bulle (chargée à la demande). */ scene?: SceneId; target?: string; title: string; text: string; placement?: Placement };

export type GuideTour = {
  id: string;
  title: string;
  summary: string;
  /** Page où la visite se lance. */
  path: string;
  match: (pathname: string) => boolean;
  /** Permission requise (comme dans la navigation) ; absente = tout membre. */
  permission?: string;
  /** Se lance seule à la première visite de la page (sinon : bouton « Découvrir cette section », signalé par une pastille). */
  auto?: boolean;
  steps: GuideStep[];
};

export type GuidePrefs = { enabled: boolean; animations: "auto" | "off"; seen: string[] };
