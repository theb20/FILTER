
import { useState, useEffect } from "react";
import { Mail, ArrowUpRight, Paperclip } from "lucide-react";
import { SITE, DEMO_NOTICE } from "../config/site.js";

const ContactPage = () => {
  const [index, setIndex] = useState(0);
  const [showForm, setShowForm] = useState(false);
  const [sent, setSent] = useState(false);
  const [profile, setProfile] = useState(null); // null | "particulier" | "entreprise"
  const emptyForm = {
    name: "",
    email: "",
    phone: "",
    company: "",
    siret: "",
    role: "",
    message: "",
    file: null,
    consent: false ,
  };
  const inputClass =
    "w-full bg-white rounded-2xl px-5 py-4 text-sm outline-none focus:ring-2 focus:ring-amber-500";
  const isCompany = profile === "entreprise";
  const [formData, setFormData] = useState(emptyForm);

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({ ...prev, [name]: type === "checkbox" ? checked : value }));
  };

  const handleFileChange = (e) =>
    setFormData((prev) => ({ ...prev, file: e.target.files[0] || null }));

  const handleSubmit = (e) => {
    e.preventDefault();
    // Aucun envoi réel : à brancher (API / EmailJS...). Rien n'est transmis.
    setFormData(emptyForm);
    setSent(true);
  };

  useEffect(() => {
    document.title = "Contact - Filter Finder";
  }, []);

  const words = [
    "Vous avez des questions ?",
    "Besoin d'assistance ?",
    "Parlons de votre projet.",
  ];

  useEffect(() => {
    const interval = setInterval(() => {
      setIndex((prev) => (prev + 1) % words.length);
    }, 2500);

    return () => clearInterval(interval);
  }, []);

  const isOpen = () => {
    const hour = new Date().getHours();
    return hour >= 9 && hour < 18;
  };

  return (
    <main className="min-h-screen bg-white px-4 py-4 sm:px-6 lg:px-10">

      <style>{`
        @keyframes wordChange {
          from {
            opacity: 0;
            transform: translateY(10px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        .word-change {
          display: inline-block;
          animation: wordChange 0.4s ease;
        }
      `}</style>

      {/* HERO */}
      <section className="grid grid-cols-1 lg:grid-cols-2 gap-3 lg:gap-4">

        {/* LEFT PANEL */}
        <div className="relative overflow-hidden bg-[#F6F6F6] rounded-[28px] min-h-[560px] lg:h-full lg:min-h-screen flex items-center">

          {showForm && !profile ? (
            <div className="word-change relative z-10 w-full px-8 py-16 sm:px-12 lg:px-12 xl:px-16 max-w-2xl">
              <h2 className="text-4xl sm:text-5xl font-semibold tracking-tight text-black mb-3">
                Vous êtes ?
              </h2>
              <p className="text-gray-500 mb-8">
                Choisissez votre profil pour un formulaire adapté.
              </p>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {[
                  { id: "particulier", label: "Particulier", desc: "Une question, un besoin personnel" },
                  { id: "entreprise", label: "Entreprise", desc: "Un projet ou un partenariat professionnel" },
                ].map((opt) => (
                  <button
                    key={opt.id}
                    type="button"
                    onClick={() => setProfile(opt.id)}
                    className="text-left bg-white rounded-2xl p-6 hover:ring-2 hover:ring-amber-500 transition group"
                  >
                    <span className="flex items-center justify-between text-lg font-semibold text-black">
                      {opt.label}
                      <ArrowUpRight
                        size={19}
                        className="group-hover:rotate-45 transition-transform duration-300"
                      />
                    </span>
                    <span className="block text-sm text-gray-500 mt-2">{opt.desc}</span>
                  </button>
                ))}
              </div>

              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="mt-6 text-sm text-gray-500 hover:text-black transition"
              >
                Annuler
              </button>
            </div>
          ) : showForm ? (
            <form
              onSubmit={handleSubmit}
              className="word-change relative z-10 w-full px-8 py-16 sm:px-12 lg:px-12 xl:px-16 max-w-2xl"
            >
              {sent ? (
                <div>
                  <h2 className="text-4xl sm:text-5xl font-semibold tracking-tight text-black mb-4">
                    Démonstration
                  </h2>
                  <p className="text-gray-500 text-lg leading-8 mb-8">
                    {DEMO_NOTICE} Votre message n'a pas été envoyé.
                  </p>
                  <button
                    type="button"
                    onClick={() => {
                      setSent(false);
                      setProfile(null);
                      setShowForm(false);
                    }}
                    className="text-sm font-medium underline underline-offset-4"
                  >
                    Retour
                  </button>
                </div>
              ) : (
                <>
                  <h2 className="text-4xl sm:text-5xl font-semibold tracking-tight text-black mb-4">
                    {isCompany ? "Contact entreprise" : "Écrivez-nous"}
                  </h2>
                  <p className="mb-6 rounded-lg bg-amber-100 text-amber-900 text-sm px-3 py-2" role="note">
                    {DEMO_NOTICE}
                  </p>

                  <div className="space-y-4">
                    {isCompany && (
                      <>
                        <input
                          type="text"
                          name="company"
                          required
                          placeholder="Nom de l'entreprise"
                          value={formData.company}
                          onChange={handleChange}
                          className={inputClass}
                        />
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                          <input
                            type="text"
                            name="siret"
                            inputMode="numeric"
                            pattern="[0-9 ]{9,17}"
                            placeholder="SIRET (optionnel)"
                            value={formData.siret}
                            onChange={handleChange}
                            className={inputClass}
                          />
                          <input
                            type="text"
                            name="role"
                            placeholder="Fonction"
                            value={formData.role}
                            onChange={handleChange}
                            className={inputClass}
                          />
                        </div>
                      </>
                    )}
                    <input
                      type="text"
                      name="name"
                      required
                      placeholder={isCompany ? "Nom du contact" : "Nom complet"}
                      value={formData.name}
                      onChange={handleChange}
                      className={inputClass}
                    />
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <input
                        type="email"
                        name="email"
                        required
                        placeholder={isCompany ? "Email professionnel" : "Email"}
                        value={formData.email}
                        onChange={handleChange}
                        className={inputClass}
                      />
                      <input
                        type="tel"
                        name="phone"
                        required={isCompany}
                        placeholder={isCompany ? "Téléphone" : "Téléphone (optionnel)"}
                        value={formData.phone}
                        onChange={handleChange}
                        className={inputClass}
                      />
                    </div>
                    <textarea
                      name="message"
                      required
                      rows={5}
                      placeholder={isCompany ? "Décrivez votre projet ou votre besoin" : "Votre message"}
                      value={formData.message}
                      onChange={handleChange}
                      className={`${inputClass} resize-none`}
                    />
                    <label className="flex items-center gap-3 w-full bg-white rounded-2xl px-5 py-4 text-sm text-gray-500 cursor-pointer focus-within:ring-2 focus-within:ring-amber-500">
                      <Paperclip size={18} className="shrink-0 text-black" />
                      <span className="truncate">
                        {formData.file ? formData.file.name : "Joindre un fichier (optionnel)"}
                      </span>
                      <input
                        type="file"
                        name="file"
                        onChange={handleFileChange}
                        className="sr-only"
                      />
                    </label>

                    <label className="flex items-start gap-3 text-sm text-gray-600 cursor-pointer">
                      <input
                        type="checkbox"
                        name="consent"
                        required
                        checked={formData.consent}
                        onChange={handleChange}
                        className="mt-1 h-4 w-4 accent-black"
                      />
                      <span>
                        J'accepte que mes données soient utilisées pour traiter ma demande
                        conformément à la politique de confidentialité.
                      </span>
                    </label>
                  </div>

                  <div className="flex items-center gap-6 mt-6">
                    <button
                      type="submit"
                      className="inline-flex items-center gap-3 bg-black text-white px-7 py-4 rounded-full hover:bg-amber-500 transition-all duration-300 group"
                    >
                      <span className="text-sm font-medium">Envoyer</span>
                      <ArrowUpRight
                        size={19}
                        className="group-hover:rotate-45 transition-transform duration-300"
                      />
                    </button>

                    <button
                      type="button"
                      onClick={() => setProfile(null)}
                      className="text-sm text-gray-500 hover:text-black transition"
                    >
                      Changer de profil
                    </button>
                  </div>
                </>
              )}
            </form>
          ) : (
            <div className="relative z-10 px-8 py-16 sm:px-12 lg:px-12 xl:px-16 max-w-2xl">

              <h1 className="text-5xl sm:text-6xl xl:text-[68px] leading-[1.05] font-semibold tracking-[-0.055em] text-black mb-8">
                <span key={words[index]} className="word-change">
                  {words[index]}
                </span>
              </h1>

              <p className="text-gray-500 text-base sm:text-lg leading-8 max-w-md mb-9">
                Une question, un besoin ou une idée ?
                L'équipe Filter Finder est là pour vous accompagner
                et vous apporter une réponse adaptée.
              </p>

              <button
                type="button"
                onClick={() => setShowForm(true)}
                className="inline-flex items-center gap-3 bg-black text-white px-7 py-4 rounded-full hover:bg-amber-500 transition-all duration-300 group"
              >
                <span className="text-sm font-medium">
                  Entrer en contact
                </span>

                <ArrowUpRight
                  size={19}
                  className="group-hover:rotate-45 transition-transform duration-300"
                />
              </button>

              {/* Decorative text */}
              <div className="mt-20 hidden sm:block">
                <p className="text-xs text-gray-400 uppercase tracking-[0.2em]">
                  Filter Finder — Toujours à votre écoute
                </p>
              </div>

            </div>
          )}
        </div>

        {/* RIGHT PANEL */}
        <div className="relative overflow-hidden bg-[#FFE85C] rounded-[28px] min-h-[500px] lg:h-full lg:min-h-screen">

          <img
            src="https://i.pinimg.com/736x/bf/1a/ab/bf1aab6b6a6b937d4c22cefa9388c14d.jpg"
            alt="Illustration de contact"
            className="absolute inset-0 w-full h-full object-cover object-center"
          />

          {/* Decorative overlay */}
          <div className="absolute top-8 left-8 bg-white/90 backdrop-blur-sm rounded-full px-5 py-3 text-sm font-medium text-black">
            Let's connect !
          </div>

          <div className="absolute bottom-8 right-8 bg-white rounded-full p-4 shadow-lg">
            <Mail size={25} className="text-black" />
          </div>

        </div>

      </section>

      {/* CONTACT SECTION */}
      <section
        id="contact-form"
        className="py-24 px-4 sm:px-8 lg:px-16"
      >
        <div className="max-w-3xl mx-auto text-center">

          <span className="text-amber-500 text-sm font-semibold uppercase tracking-widest">
            Contact
          </span>

          <h2 className="text-4xl sm:text-5xl font-semibold tracking-tight mt-4 mb-6">
            Faisons connaissance.
          </h2>

          <p className="text-gray-500 text-lg leading-8">
            Notre équipe est disponible pour répondre à vos questions
            et vous accompagner dans vos démarches.
          </p>

          {SITE.email ? (
            <a
              href={`mailto:${SITE.email}`}
              className="inline-flex items-center gap-3 mt-8 bg-black text-white px-7 py-4 rounded-full hover:bg-amber-500 transition"
            >
              <Mail size={18} />
              Nous envoyer un email
            </a>
          ) : (
            <p className="mt-8 text-sm text-gray-500">
              Adresse email de contact bientôt disponible.
            </p>
          )}

        </div>
      </section>

    </main>
  );
};

export default ContactPage;
