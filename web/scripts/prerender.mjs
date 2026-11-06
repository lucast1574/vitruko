import { createServer } from "vite";
import { readFile, writeFile, mkdir } from "node:fs/promises";
import { Resvg } from "@resvg/resvg-js";
import { pages, siteOrigin, notFound } from "../src/public/content.js";

const template = await readFile("dist/index.html", "utf8");
const escape = (value) =>
  String(value)
    .replaceAll("&", "&amp;")
    .replaceAll('"', "&quot;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
function graph(path, page) {
  const url = siteOrigin + (path === "/" ? "/" : path);
  const brand = {
    "@type": "Organization",
    "@id": siteOrigin + "/#organization",
    name: "Vitruko",
    url: siteOrigin + "/",
    logo: {
      "@type": "ImageObject",
      url: siteOrigin + "/icon-512.png",
      width: 512,
      height: 512,
    },
    email: "hola@nexode.app",
  };
  const website = {
    "@type": "WebSite",
    "@id": siteOrigin + "/#website",
    name: "Vitruko",
    url: siteOrigin + "/",
    inLanguage: "es",
    publisher: { "@id": brand["@id"] },
  };
  const webPage = {
    "@type": "WebPage",
    "@id": url + "#webpage",
    url,
    name: page.title,
    description: page.description,
    inLanguage: "es",
    isPartOf: { "@id": website["@id"] },
    publisher: { "@id": brand["@id"] },
    primaryImageOfPage: {
      "@type": "ImageObject",
      url: siteOrigin + "/og-vitruko.png",
      width: 1200,
      height: 630,
    },
  };
  if (path === "/") webPage.mainEntity = { "@id": siteOrigin + "/#api" };
  else webPage.breadcrumb = { "@id": url + "#breadcrumbs" };
  const data = [brand, website, webPage];
  if (path === "/")
    data.push({
      "@type": "Service",
      "@id": siteOrigin + "/#api",
      name: "Vitruko API",
      serviceType: "API de extracción de fichas de producto",
      url: siteOrigin + "/",
      description: page.description,
      provider: { "@id": brand["@id"] },
    });
  else
    data.push({
      "@type": "BreadcrumbList",
      "@id": url + "#breadcrumbs",
      itemListElement: [
        {
          "@type": "ListItem",
          position: 1,
          name: "Vitruko",
          item: siteOrigin + "/",
        },
        ...(page.operation
          ? [
              {
                "@type": "ListItem",
                position: 2,
                name: "Documentación",
                item: siteOrigin + "/documentacion",
              },
            ]
          : []),
        {
          "@type": "ListItem",
          position: page.operation ? 3 : 2,
          name:
            page.navTitle ||
            page.heading ||
            page.title.replace(" | Vitruko", ""),
          item: url,
        },
      ],
    });
  return { "@context": "https://schema.org", "@graph": data };
}
function document(page, path, html, indexable = true) {
  const canonical = siteOrigin + (path === "/" ? "/" : path);
  const head = `<title>${escape(page.title)}</title>
<meta name="description" content="${escape(page.description)}">
<meta name="robots" content="${indexable ? "index,follow,max-image-preview:large" : "noindex,nofollow"}">
${indexable ? `<link rel="canonical" href="${canonical}">` : ""}
<meta property="og:type" content="website"><meta property="og:locale" content="es_ES"><meta property="og:site_name" content="Vitruko">
<meta property="og:title" content="${escape(page.title)}"><meta property="og:description" content="${escape(page.description)}"><meta property="og:url" content="${canonical}">
<meta property="og:image" content="${siteOrigin}/og-vitruko.png"><meta property="og:image:width" content="1200"><meta property="og:image:height" content="630"><meta property="og:image:alt" content="Vitruko: API de productos y vistas previas">
<meta name="twitter:card" content="summary_large_image"><meta name="twitter:title" content="${escape(page.title)}"><meta name="twitter:description" content="${escape(page.description)}"><meta name="twitter:image" content="${siteOrigin}/og-vitruko.png"><meta name="twitter:image:alt" content="Vitruko: API de productos y vistas previas">
<link rel="icon" href="/favicon-96.png" sizes="96x96" type="image/png"><link rel="icon" href="/logo.svg" type="image/svg+xml"><link rel="apple-touch-icon" href="/apple-touch-icon.png">
${indexable ? `<script type="application/ld+json">${JSON.stringify(graph(path, page)).replaceAll("<", "\\u003c")}</script>` : ""}`;
  return template
    .replace(/<title>[\s\S]*?<\/title>/, " ")
    .replace(/<meta name="description"[^>]*>/, "")
    .replace(/<link rel="icon"[^>]*>/, "")
    .replace("</head>", head + "</head>")
    .replace(
      '<div id="root"></div>',
      html
        ? `<div id="root" data-prerendered="true">${html}</div>`
        : '<div id="root"></div>',
    );
}
const logo = await readFile("public/logo.svg", "utf8");
for (const [file, size] of [
  ["favicon-96.png", 96],
  ["apple-touch-icon.png", 180],
  ["icon-512.png", 512],
]) {
  const svg = new Resvg(logo, { fitTo: { mode: "width", value: size } });
  await writeFile("dist/" + file, svg.render().asPng());
}
await writeFile(
  "dist/og-vitruko.png",
  new Resvg(await readFile("public/og-vitruko.svg", "utf8")).render().asPng(),
);
// The private app is a separate noindex shell: no marketing canonical leaks
// into authentication, accounts, shared maps or arbitrary unknown paths.
await writeFile(
  "dist/shell.html",
  document(
    {
      title: "Vitruko | Tu cuenta y tus productos",
      description:
        "Accede a tu cuenta de Vitruko para gestionar claves, consumo y extracciones.",
    },
    "/",
    null,
    false,
  ),
);
const server = await createServer({
  server: { middlewareMode: true },
  appType: "custom",
  ssr: { noExternal: ["gsap"] },
});
try {
  const { render } = await server.ssrLoadModule("/src/entry-server.jsx");
  for (const [path, page] of Object.entries(pages)) {
    const dir = path === "/" ? "dist" : "dist" + path;
    await mkdir(dir, { recursive: true });
    await writeFile(dir + "/index.html", document(page, path, render(path)));
  }
  await writeFile(
    "dist/404.html",
    document(notFound, "/404", render("/404"), false),
  );
} finally {
  await server.close();
}
await writeFile("dist/routes.json", JSON.stringify(Object.keys(pages)));
await writeFile(
  "dist/sitemap.xml",
  `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${Object.keys(
    pages,
  )
    .map((p) => `<url><loc>${siteOrigin}${p === "/" ? "/" : p}</loc></url>`)
    .join("")}</urlset>\n`,
);
await writeFile(
  "dist/robots.txt",
  `User-agent: *\nAllow: /\nDisallow: /console/\nDisallow: /v1/\nDisallow: /stripe/\n\nSitemap: ${siteOrigin}/sitemap.xml\n`,
);
console.log(
  `Prerendered ${Object.keys(pages).length} public pages, sitemap, robots, structured data and social images.`,
);
