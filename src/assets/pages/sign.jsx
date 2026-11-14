import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useNotify } from "../configurations/notificationContext.js";
import { DEMO_NOTICE } from "../config/site.js";

const AuthSplit = () => {
  const notify = useNotify();

  useEffect(() => {
    document.title = "Connexion — Filter Finder";
  }, []);
  const [isLogin, setIsLogin] = useState(true);
  const [formData, setFormData] = useState({
    fullName: "",
    email: "",
    password: "",
  });
  const [errors, setErrors] = useState({});
  const isLoading = false;
  const navigate = useNavigate();

  const toggleForm = () => {
    setIsLogin(!isLogin);
    setErrors({});
    setFormData({ fullName: "", email: "", password: "" });
  };

  const validateForm = () => {
    const newErrors = {};
    if (!isLogin && !formData.fullName.trim()) {
      newErrors.fullName = "Le nom complet est obligatoire";
    }
    if (!formData.email.trim()) {
      newErrors.email = "L'email est obligatoire";
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = "L'email n'est pas valide";
    }
    if (!formData.password) {
      newErrors.password = "Le mot de passe est obligatoire";
    } else if (formData.password.length < 8) {
      newErrors.password = "Le mot de passe doit contenir au moins 8 caractères";
    }
    return newErrors;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    const newErrors = validateForm();
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }
    // Aucune authentification réelle : on ne simule pas de succès.
    notify(DEMO_NOTICE, "demo");
  };

  const handleGoogleAuth = () => {
    notify(DEMO_NOTICE, "demo");
  };

  const handleInputChange = (e) => {
    const name = e.target.name;
    const value = e.target.value;
    setFormData((prev) => {
      const updated = { ...prev };
      updated[name] = value;
      return updated;
    });
    if (errors[name]) {
      setErrors((prev) => {
        const updated = { ...prev };
        updated[name] = "";
        return updated;
      });
    }
  };

  return (
    <div className="auth-container flex items-center justify-center h-screen bg-gray-100 dark:bg-gray-950">
      <div className="w-full lg:rounded-2xl overflow-hidden flex flex-col md:flex-row md:w-[60vw] md:h-[60vh] shadow-2xl relative">
        <div className="hidden md:flex w-full md:w-1/2 bg-side from-amber-500 to-orange-600 relative overflow-hidden">
          <div className="absolute inset-0 opacity-10">
            <div className="absolute top-10 left-10 w-32 h-32 bg-white rounded-full blur-3xl"></div>
            <div className="absolute bottom-10 right-10 w-40 h-40 bg-white rounded-full blur-3xl"></div>
          </div>
          <div className="m-8 p-4 text-white flex flex-col space-y-3 h-full z-10 relative">
            <div>
              <h2 className="text-3xl font-bold mb-2">Bienvenue chez <br/>Filter Finder</h2>
              <p className="text-white/90 text-sm">Votre spécialiste des filtres automobiles.</p>
            </div>
            
          </div>
        </div>

        <div className="flex-1 flex items-center justify-center bg-white/10 backdrop-blur-md dark:bg-gray-900/10 p-8 m-5 lg:m-0 rounded-2xl lg:rounded-none relative z-10">
          <div className="w-full max-w-md">
            <h1 className="text-3xl font-bold text-gray-100 dark:text-white mb-2 transition-all">
              {isLogin ? "Connectez-vous" : "Inscrivez-vous"}
            </h1>
            <p className="text-gray-200 dark:text-gray-300 mb-6">
              {isLogin
                ? "Connectez-vous pour accéder à toutes les fonctionnalités de Filter Finder."
                : "Inscrivez-vous pour commencer à utiliser Filter Finder."}
            </p>

            <p className="mb-4 rounded-lg bg-amber-100 text-amber-900 text-sm px-3 py-2" role="note">
              {DEMO_NOTICE}
            </p>

            <form className="space-y-4" onSubmit={handleSubmit} noValidate>
              {!isLogin && (
                <div>
                  <input
                    type="text"
                    name="fullName"
                    aria-label="Nom complet"
                    placeholder="Nom complet"
                    value={formData.fullName}
                    onChange={handleInputChange}
                    className={`w-full p-3 border ${
                      errors.fullName ? "border-red-500" : "border-gray-300 dark:border-gray-700"
                    } rounded-lg bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 transition-all`}
                  />
                  {errors.fullName && (
                    <p className="text-red-500 text-sm mt-1">{errors.fullName}</p>
                  )}
                </div>
              )}
              <div>
                <input
                  type="email"
                  name="email"
                  aria-label="Email"
                  autoComplete="email"
                  placeholder="Email"
                  value={formData.email}
                  onChange={handleInputChange}
                  className={`w-full p-3 border ${
                    errors.email ? "border-red-500" : "border-gray-300 dark:border-gray-700"
                  } rounded-lg bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 transition-all`}
                />
                {errors.email && (
                  <p className="text-red-500 text-sm mt-1">{errors.email}</p>
                )}
              </div>
              <div>
                <input
                  type="password"
                  name="password"
                  aria-label="Mot de passe"
                  autoComplete={isLogin ? "current-password" : "new-password"}
                  placeholder={isLogin ? "Mot de passe" : "Créer un mot de passe"}
                  value={formData.password}
                  onChange={handleInputChange}
                  className={`w-full p-3 border ${
                    errors.password ? "border-red-500" : "border-gray-300 dark:border-gray-700"
                  } rounded-lg bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-amber-500 transition-all`}
                />
                {errors.password && (
                  <p className="text-red-500 text-sm mt-1">{errors.password}</p>
                )}
              </div>

              {isLogin && (
                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={() => navigate("/reset")}
                    className="text-sm text-amber-500 hover:text-amber-600 font-medium focus:outline-none"
                  >
                    Mot de passe oublié?
                  </button>
                </div>
              )}

              <button
                type="submit"
                disabled={isLoading}
                className="w-full py-3 bg-amber-500 hover:bg-amber-600 text-white font-bold rounded-lg transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed transform hover:scale-[1.02] active:scale-[0.98]"
              >
                {isLoading ? (
                  <span className="flex items-center justify-center">
                    <svg className="animate-spin h-5 w-5 mr-2" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Chargement...
                  </span>
                ) : (
                  <>{isLogin ? "Se connecter" : "S'inscrire"}</>
                )}
              </button>

              <div className="relative my-6">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-300 dark:border-gray-700"></div>
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-4 bg-white dark:bg-gray-900 rounded-xl text-gray-500 dark:text-gray-400">
                    Ou continuez avec
                  </span>
                </div>
              </div>

              <button
                type="button"
                onClick={handleGoogleAuth}
                disabled={isLoading}
                className="w-full py-3 px-4 border border-gray-300 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-200 font-medium hover:bg-gray-50 dark:hover:bg-gray-700 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center space-x-3 transform hover:scale-[1.02] active:scale-[0.98]"
              >
                <svg className="w-5 h-5" viewBox="0 0 24 24">
                  <path
                    fill="#4285F4"
                    d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  />
                  <path
                    fill="#34A853"
                    d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                  />
                  <path
                    fill="#EA4335"
                    d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                  />
                </svg>
                <span>Continuer avec Google</span>
              </button>
            </form>

            <p className="mt-4 text-center text-gray-200 dark:text-gray-300">
              {isLogin ? "Vous n'avez pas de compte?" : "Vous avez déjà un compte?"}{" "}
              <button
                type="button"
                onClick={toggleForm}
                className="text-amber-500 font-semibold hover:underline focus:outline-none"
              >
                {isLogin ? "Inscription" : "Connexion"}
              </button>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AuthSplit;