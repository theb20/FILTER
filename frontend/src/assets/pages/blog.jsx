import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { BLOG_KINDS } from "../data/blog.js";
import { fetchBlogArticles, fetchBlogCategories, fetchBlogTags } from "../lib/content.js";
import { useAsync, useProductsById } from "../lib/catalog.js";
import { filterArticles, PAGE_SIZE, paginate, parseBlogParams, sortByDateDesc } from "../lib/blogFilters.js";
import { usePageMeta } from "../lib/usePageMeta.js";
import { useI18n } from "../i18n/i18nContext.js";
import JsonLd from "../components/JsonLd.jsx";
import BlogNav from "../components/blog/BlogNav.jsx";
import BlogSkeleton from "../components/blog/BlogSkeleton.jsx";
import BlogState from "../components/blog/BlogState.jsx";
import CitedProducts from "../components/blog/CitedProducts.jsx";
import DossiersSection from "../components/blog/DossiersSection.jsx";
import FeaturedMosaic from "../components/blog/FeaturedMosaic.jsx";
import GuidesRow from "../components/blog/GuidesRow.jsx";
import LatestArticles from "../components/blog/LatestArticles.jsx";
import Sidebar from "../components/blog/Sidebar.jsx";
import TagCloud from "../components/blog/TagCloud.jsx";
import ToolBanner from "../components/blog/ToolBanner.jsx";
import TopicsRow from "../components/blog/TopicsRow.jsx";
import { postPath } from "../components/blog/format.js";

export default function Blog() {
  const { t, lang } = useI18n();
  const [sp] = useSearchParams();
  const [state, setState] = useState({ status: "loading", articles: [] });

  const load = useCallback(() => {
    setState({ status: "loading", articles: [] });
    fetchBlogArticles()
      .then((articles) => setState({ status: "ready", articles }))
      .catch(() => setState({ status: "error", articles: [] }));
  }, []);
  useEffect(() => { load(); }, [load]);

  usePageMeta(t("blog.meta.title"), t("blog.meta.desc"));

  const categories = useAsync(() => fetchBlogCategories().catch(() => []), [lang]).data ?? [];
  const tags = useAsync(() => fetchBlogTags().catch(() => []), [lang]).data ?? [];
  const base = parseBlogParams(sp, { categories: categories.map((c) => c.slug), kinds: BLOG_KINDS, tags: tags.map((x) => x.slug) });
  const tagLabel = tags.find((x) => x.slug === base.tag)?.name;
  const params = { ...base, tagLabel };

  // Après un changement d'étiquette ou de page, ramène la vue sur la liste des articles.
  const lastKey = useRef(`${base.tag}|${base.page}`);
  const ready = state.status === "ready";
  useEffect(() => {
    const key = `${base.tag}|${base.page}`;
    if (ready && key !== lastKey.current) document.getElementById("latest-title")?.scrollIntoView();
    lastKey.current = key;
  }, [base.tag, base.page, ready]);

  const { status, articles } = state;
  const sorted = sortByDateDesc(articles);
  // Sélections de la page, calculées d'après les réglages de chaque article (backoffice) : mis en avant, guides, dossiers.
  const featured = sorted.filter((a) => a.featured);
  const mosaic = (featured.length ? featured : sorted).slice(0, 5);
  const topics = categories.map((c) => sorted.find((a) => a.category === c.slug)).filter(Boolean).slice(0, 4);
  const guides = sorted.filter((a) => a.guide).slice(0, 4);
  const dossiers = sorted.filter((a) => a.dossier).slice(0, 3);
  const citedIds = [...new Set(sorted.flatMap((a) => a.products))].slice(0, 4);
  const { products: citedById } = useProductsById(citedIds);
  const pageData = paginate(filterArticles(sorted, params), params.page, PAGE_SIZE);
  const origin = typeof window === "undefined" ? "" : window.location.origin;

  return (
    <div>
      <BlogNav params={params} asH1 />

      {status === "loading" && <BlogSkeleton />}
      {status === "error" && <div className="mx-2.5 mt-2.5"><BlogState kind="error" onRetry={load} /></div>}

      {status === "ready" && articles.length === 0 && (
        <div role="status" className="mx-2.5 mt-2.5 rounded-[8px] bg-white p-8 text-center shadow-card">
          <h2 className="text-[20px] font-semibold">{t("blog.nothing.title")}</h2>
          <p className="mt-1 text-gray-700">{t("blog.nothing.text")}</p>
        </div>
      )}

      {status === "ready" && articles.length > 0 && (
        <>
          <JsonLd data={{
            "@context": "https://schema.org",
            "@type": "Blog",
            name: t("blog.title"),
            description: t("blog.meta.desc"),
            url: `${origin}/blog`,
            blogPost: sorted.slice(0, 10).map((a) => ({
              "@type": "BlogPosting", headline: a.title, datePublished: a.date,
              author: { "@type": "Person", name: a.author }, url: `${origin}${postPath(a.slug)}`,
            })),
          }} />

          <TopicsRow articles={topics} />
          <FeaturedMosaic articles={mosaic} />
          <ToolBanner />

          <div data-guide="blog-latest" className="mx-2.5 mt-2.5 grid gap-4 rounded-[4px] bg-white p-5 shadow-card lg:grid-cols-[minmax(0,1fr)_320px]">
            <LatestArticles params={params} pageData={pageData} />
            <Sidebar popular={sorted.slice(0, 5)} />
          </div>

          <GuidesRow articles={guides} />
          <DossiersSection articles={dossiers} />
          <CitedProducts products={citedIds.map((id) => citedById[id]).filter(Boolean)} />
          <TagCloud params={params} />
        </>
      )}
    </div>
  );
}
