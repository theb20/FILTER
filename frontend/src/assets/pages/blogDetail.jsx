import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Clock } from "lucide-react";
import { fetchBlogArticles, fetchBlogTags } from "../lib/content.js";
import { useAsync, useProductsById } from "../lib/catalog.js";
import { adjacentArticles, relatedArticles } from "../lib/blogFilters.js";
import { usePageMeta } from "../lib/usePageMeta.js";
import { useI18n } from "../i18n/i18nContext.js";
import JsonLd from "../components/JsonLd.jsx";
import AuthorBox from "../components/blog/AuthorBox.jsx";
import PostNav from "../components/blog/PostNav.jsx";
import ShareBar from "../components/blog/ShareBar.jsx";
import ArticleCard from "../components/blog/ArticleCard.jsx";
import BlogNav from "../components/blog/BlogNav.jsx";
import BlogState from "../components/blog/BlogState.jsx";
import CitedProducts from "../components/blog/CitedProducts.jsx";
import { formatDate, postPath } from "../components/blog/format.js";

const NO_PARAMS = { cat: null, kind: null, tag: null, page: 1 };

// Gabarit d'article : fil d'Ariane, titre, image, sommaire, contenu, produits cités, articles liés.
export default function BlogDetail() {
  const { slug } = useParams();
  const { t, locale, lang } = useI18n();
  const [state, setState] = useState({ status: "loading", articles: [] });
  const BLOG_TAGS = (useAsync(fetchBlogTags, [lang]).data ?? []).map((x) => ({ slug: x.slug, label: x.name }));

  useEffect(() => {
    let alive = true;
    fetchBlogArticles()
      .then((articles) => alive && setState({ status: "ready", articles }))
      .catch(() => alive && setState({ status: "error", articles: [] }));
    return () => { alive = false; };
  }, [slug]);

  const article = state.articles.find((a) => a.slug === slug);
  const { products: citedById } = useProductsById(article?.products ?? []);
  usePageMeta(article ? `${article.title} — ${t("blog.title")}` : t("blog.title"), article ? article.excerpt : t("blog.meta.desc"));

  const origin = typeof window === "undefined" ? "" : window.location.origin;
  const wrap = (children) => (
    <div>
      <BlogNav params={NO_PARAMS} />
      {children}
    </div>
  );

  if (state.status === "loading") return wrap(<div role="status" aria-busy="true" aria-label={t("blog.loading")} className="mx-2.5 mt-2.5 h-96 animate-pulse rounded-[8px] bg-gray-200" />);
  if (state.status === "error") return wrap(<div className="mx-2.5 mt-2.5"><BlogState kind="error" onRetry={() => window.location.reload()} /></div>);
  if (!article) return wrap(<div className="mx-2.5 mt-2.5"><BlogState kind="notFound" /></div>);

  const products = article.products.map((id) => citedById[id]).filter(Boolean);
  const related = relatedArticles(article, state.articles, 3);
  const { older, newer } = adjacentArticles(article, state.articles);
  const lastIndex = article.sections.length - 1;
  const crumbs = [
    { name: t("nav.home"), to: "/" },
    { name: t("blog.title"), to: "/blog" },
    { name: article.categoryName, to: `/blog?cat=${article.category}` },
    { name: article.title, to: postPath(article.slug) },
  ];

  return wrap(
    <>
      <JsonLd data={[
        {
          "@context": "https://schema.org", "@type": "BlogPosting",
          headline: article.title, description: article.excerpt, datePublished: article.date,
          author: { "@type": "Person", name: article.author },
          ...(article.image ? { image: article.image } : {}), mainEntityOfPage: `${origin}${postPath(article.slug)}`,
        },
        {
          "@context": "https://schema.org", "@type": "BreadcrumbList",
          itemListElement: crumbs.map((c, i) => ({ "@type": "ListItem", position: i + 1, name: c.name, item: `${origin}${c.to}` })),
        },
      ]} />

      <article className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-5 shadow-card sm:p-8">
        <nav aria-label={t("blog.breadcrumb")} className="text-sm text-gray-600">
          <ol className="flex flex-wrap items-center gap-x-1.5">
            {crumbs.map((c, i) => (
              <li key={c.to} className="flex items-center gap-1.5">
                {i < crumbs.length - 1 ? <Link to={c.to} className="text-link hover:underline">{c.name}</Link> : <span aria-current="page" className="line-clamp-1 text-gray-800">{c.name}</span>}
                {i < crumbs.length - 1 && <span aria-hidden="true">/</span>}
              </li>
            ))}
          </ol>
        </nav>

        <header className="relative mt-4 flex min-h-[320px] overflow-hidden rounded-[8px] bg-night text-white sm:min-h-[420px]">
          {article.image && <img src={article.image} alt={t("blog.imageAlt", { title: article.title })} loading="eager" decoding="async" className="absolute inset-0 h-full w-full object-cover" />}
          <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/55 to-black/20" aria-hidden="true" />
          <div className="relative mt-auto w-full max-w-4xl p-5 sm:p-8">
            <span className="inline-block rounded-[3px] bg-action px-2 py-0.5 text-xs font-bold uppercase tracking-wide text-black">{article.categoryName}</span>
            <h1 className="mt-2 text-[30px] font-bold leading-tight sm:text-[42px]">{article.title}</h1>
            <p className="mt-2 text-[17px] text-white/90">{article.excerpt}</p>
            <p className="mt-3 flex flex-wrap items-center gap-x-2 text-sm text-white/90">
              <span>{t("blog.by", { author: article.author })}</span><span aria-hidden="true">·</span>
              <time dateTime={article.date}>{formatDate(locale, article.date)}</time><span aria-hidden="true">·</span>
              <span className="inline-flex items-center gap-1"><Clock size={14} aria-hidden="true" />{t("blog.readMin", { n: article.readMin })}</span>
            </p>
          </div>
        </header>

        {lang !== "fr" && <p role="note" className="mt-4 rounded-[3px] bg-info p-2.5 text-sm">{t("blog.frOnly")}</p>}

        <div className="mt-6 grid gap-8 lg:grid-cols-[240px_minmax(0,1fr)]">
          <div className="h-fit space-y-4 lg:sticky lg:top-28">
            <nav aria-label={t("blog.toc")} className="rounded-[8px] bg-gray-50 p-4">
              <p className="font-bold">{t("blog.toc")}</p>
              <ol className="mt-2 space-y-1.5 text-sm">
                {article.sections.map((s) => <li key={s.id}><a href={`#${s.id}`} className="text-link hover:underline">{s.heading}</a></li>)}
              </ol>
            </nav>
            <ShareBar title={article.title} />
          </div>
          <div className="max-w-3xl space-y-6 text-[17px] leading-relaxed">
            {article.sections.map((s, i) => (
              <section
                key={s.id}
                aria-labelledby={s.id}
                className={`scroll-mt-28 ${i === lastIndex && lastIndex > 0 ? "rounded-[8px] border-l-4 border-action bg-gray-50 p-5" : ""}`}
              >
                <h2 id={s.id} className="text-[24px] font-bold">{s.heading}</h2>
                {s.paragraphs.map((p, j) => <p key={j} className="mt-2">{p}</p>)}
              </section>
            ))}

            <ul className="flex flex-wrap gap-2 pt-2" aria-label={t("blog.tags")}>
              {article.tags.map((slug) => {
                const tag = BLOG_TAGS.find((x) => x.slug === slug);
                return tag ? <li key={slug}><Link to={`/blog?tag=${slug}`} className="inline-block rounded-full border border-gray-300 px-3 py-1 text-sm font-medium hover:border-black">#{tag.label}</Link></li> : null;
              })}
            </ul>

            {article.author && <AuthorBox author={article.author} bio={article.authorBio} />}
            <PostNav older={older} newer={newer} />
          </div>
        </div>
      </article>

      <CitedProducts products={products} titleKey="blog.products" id="post-products-title" />

      {related.length > 0 && (
        <section aria-labelledby="related-title" className="mx-2.5 mt-2.5 mb-2.5">
          <h2 id="related-title" className="text-[22px] font-bold">{t("blog.related")}</h2>
          <ul className="mt-3 grid gap-2.5 md:grid-cols-3">
            {related.map((a) => <li key={a.slug}><ArticleCard article={a} variant="vertical" /></li>)}
          </ul>
        </section>
      )}
    </>,
  );
}
