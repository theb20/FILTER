// @ts-check
/** Date lisible dans la langue du site. @param {string} locale @param {string} iso */
export const formatDate = (locale, iso) =>
  new Intl.DateTimeFormat(locale, { day: "numeric", month: "short", year: "numeric" }).format(new Date(`${iso}T12:00:00`));

/** Chemin d'un article. @param {string} slug */
export const postPath = (slug) => `/blog/${slug}`;
