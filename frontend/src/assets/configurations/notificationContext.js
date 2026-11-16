import { createContext, useContext } from "react";

export const NotificationContext = createContext(null);

// Hook : const notify = useNotify(); notify("message", "info" | "success" | "error" | "demo")
export const useNotify = () => {
  const notify = useContext(NotificationContext);
  if (!notify) {
    throw new Error("useNotify doit être utilisé à l'intérieur de NotificationProvider");
  }
  return notify;
};
