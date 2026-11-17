import { createContext, useContext } from "react";

export const GuideContext = createContext(null);

export const useGuide = () => {
  const c = useContext(GuideContext);
  if (!c) throw new Error("useGuide doit être utilisé dans <GuideProvider>");
  return c;
};
