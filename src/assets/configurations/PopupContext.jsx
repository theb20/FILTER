import { useState } from 'react';
import { PopupContext } from './popupContext.js';

export const PopupProvider = ({ children }) => {
  const [activePopup, setActivePopup] = useState(null);
  const [popupPayload, setPopupPayload] = useState(null);

  const openPopup = (name, payload = null) => {
    setPopupPayload(payload || null);
    setActivePopup(name);
  };

  const closePopup = () => {
    setActivePopup(null);
    setPopupPayload(null);
  };

  return (
    <PopupContext.Provider 
      value={{ 
        activePopup, 
        setActivePopup, 
        openPopup,
        closePopup, 
        popupPayload 
      }}
    >
      {children}
    </PopupContext.Provider>
  );
};
