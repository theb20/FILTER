      (function() {
        'use strict';
        
        const LOADING_CONFIG = {
          minDisplayTime: 1000,    // Minimum display duration (ms)
          maxDisplayTime: 5000,    // Maximum fallback timeout (ms)
          fadeOutDelay: 500        // Fade out animation duration (ms)
        };
        
        /**
         * Hide the loading screen with animation
         */
        function hideLoadingScreen() {
          const loadingScreen = document.getElementById('loading-screen');
          if (!loadingScreen || loadingScreen.classList.contains('hidden')) {
            return;
          }
          
          loadingScreen.classList.add('hidden');
          
          // Remove from DOM after animation completes
          setTimeout(function() {
            if (loadingScreen.parentNode) {
              loadingScreen.remove();
            }
          }, LOADING_CONFIG.fadeOutDelay);
        }
        
        /**
         * Initialize loading screen auto-hide on page load
         */
        window.addEventListener('load', function() {
          setTimeout(hideLoadingScreen, LOADING_CONFIG.minDisplayTime);
        });
        
        /**
         * Fallback: Force hide if load event doesn't fire
         */
        setTimeout(hideLoadingScreen, LOADING_CONFIG.maxDisplayTime);
      })();
    
