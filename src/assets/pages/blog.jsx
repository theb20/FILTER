import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  BLOG_CATEGORIES, BLOG_CITED_PRODUCTS, BLOG_DOSSIERS, BLOG_GUIDES, BLOG_KINDS, BLOG_MOSAIC,
  BLOG_POPULAR, BLOG_TAGS, BLOG_TOPICS, fetchBlogArticles,
} from "../data/blogMocks.js";
import { DEMO_PRODUCTS } from "../data/demoProducts.js";
import { filterArticles, PAGE_SIZE, paginate, parseBlogParams, pickArticles, sortByDateDesc } from "../lib/blogFilters.js";
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

const ALLOWED = { categories: BLOG_CATEGORIES, kinds: BLOG_KINDS, tags: BLOG_TAGS.map((t) => t.slug) };

export default function Blog() {
  const { t } = useI18n();
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

  const base = parseBlogParams(sp, ALLOWED);
  const tagLabel = BLOG_TAGS.find((x) => x.slug === base.tag)?.label;
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
  const pageData = paginate(filterArticles(sorted, params), params.page, PAGE_SIZE);
  const origin = typeof window === "undefined" ? "" : window.location.origin;

  return (
    <div>
      <BlogNav params={params} asH1 />
      <p className="mx-2.5 mt-2.5 rounded-[3px] bg-info p-2.5 text-sm" role="note">{t("blog.demo")}</p>

      {status === "loading" && <BlogSkeleton />}
      {status === "error" && <div className="mx-2.5 mt-2.5"><BlogState kind="error" onRetry={load} /></div>}

      {status === "ready" && (
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

          <TopicsRow articles={pickArticles(articles, BLOG_TOPICS)} />
          <FeaturedMosaic articles={pickArticles(articles, BLOG_MOSAIC)} />
          <ToolBanner />

          <div className="mx-2.5 mt-2.5 grid gap-4 rounded-[4px] bg-white p-5 shadow-card lg:grid-cols-[minmax(0,1fr)_320px]">
            <LatestArticles params={params} pageData={pageData} />
            <Sidebar popular={pickArticles(articles, BLOG_POPULAR)} />
          </div>

          <GuidesRow articles={pickArticles(articles, BLOG_GUIDES)} />
          <DossiersSection articles={pickArticles(articles, BLOG_DOSSIERS)} />
          <CitedProducts products={BLOG_CITED_PRODUCTS.map((id) => DEMO_PRODUCTS.find((p) => p.id === id)).filter(Boolean)} />
          <TagCloud params={params} />
        </>
      )}
    </div>
  );
}
