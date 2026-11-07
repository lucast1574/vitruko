import React, { useState } from "react";
import { ArrowUpRight } from "lucide-react";
const fields = {
  title: "Nombre",
  description: "Descripción",
  images: "Imágenes",
  brand: "Marca",
  price: "Precio y moneda",
  availability: "Disponibilidad",
};
const sources = {
  json_ld: "Datos estructurados de la página",
  microdata: "Microdatos de la página",
  open_graph: "Vista previa de la página",
  html_meta: "Metadatos de la página",
  html_title: "Título de la página",
  product_meta: "Metadatos de producto",
  shopify_json: "Ficha pública de Shopify",
  vtex_json: "Ficha pública de VTEX",
};
const stock = {
  in_stock: "Disponible",
  out_of_stock: "Agotado",
  preorder: "Preventa",
  backorder: "Bajo pedido",
  discontinued: "Descontinuado",
  unknown: "Disponibilidad no publicada",
};
const warnings = {
  product_not_confirmed:
    "La página ofrece una vista previa general; no se pudo confirmar una ficha de producto.",
  multiple_products:
    "Hay varios productos y no se pudo identificar uno de forma inequívoca.",
  multiple_offers: "Hay varias ofertas; no se eligió un precio arbitrario.",
  price_range: "La tienda publica un rango de precios, no un importe único.",
  variant_not_confirmed:
    "No se pudo confirmar la oferta de la variante indicada en el enlace.",
};
export default function ProductPreview({ result }) {
  const images = result.images?.length
    ? result.images
    : result.imageUrl
      ? [result.imageUrl]
      : [];
  const [selected, setSelected] = useState(0),
    [failed, setFailed] = useState(false),
    [expanded, setExpanded] = useState(false);
  const quality = result.quality;
  return (
    <article className="extracted-product">
      {images[selected] && !failed ? (
        <img
          src={images[selected]}
          alt={`${result.title} · imagen ${selected + 1}`}
          referrerPolicy="no-referrer"
          onError={() => setFailed(true)}
        />
      ) : (
        <div className="product-image-empty">
          {failed
            ? "No se pudo cargar esta imagen de la tienda. Puedes elegir otra."
            : "La tienda no publicó una imagen."}
        </div>
      )}
      {images.length > 1 && (
        <div className="product-gallery" aria-label="Imágenes del producto">
          {images.map((image, i) => (
            <button
              key={image}
              type="button"
              aria-label={`Ver imagen ${i + 1}`}
              aria-pressed={selected === i}
              onClick={() => {
                setSelected(i);
                setFailed(false);
              }}
            >
              <img
                src={image}
                alt=""
                loading="lazy"
                referrerPolicy="no-referrer"
              />
              <span>{i + 1}</span>
            </button>
          ))}
        </div>
      )}
      {result.brand && <span className="product-brand">{result.brand}</span>}
      <h2>{result.title}</h2>
      <div className="product-commercial">
        <div>
          <span>Precio publicado</span>
          <strong>
            {result.price
              ? `${result.price.currency} ${result.price.amount}`
              : "No disponible"}
          </strong>
        </div>
        <span className="product-stock" data-stock={result.availability}>
          {stock[result.availability] || stock.unknown}
        </span>
      </div>
      <p
        className={
          expanded ? "product-description" : "product-description is-condensed"
        }
      >
        {result.description || "La tienda no publicó una descripción."}
      </p>
      {result.description?.length > 260 && (
        <button
          type="button"
          className="product-description-toggle"
          aria-expanded={expanded}
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? "Ver menos" : "Ver descripción completa"}
        </button>
      )}
      <a
        className="text-link"
        href={result.url}
        target="_blank"
        rel="noreferrer"
      >
        Abrir producto <ArrowUpRight size={16} />
      </a>
      {quality && (
        <details className="product-quality">
          <summary>Calidad y origen de los datos</summary>
          <p>
            {quality.productDetected
              ? "La fuente publicó datos identificados como producto."
              : "Ficha de producto sin confirmar."}
          </p>
          {quality.missingFields.length > 0 && (
            <p>
              Sin publicar o sin confirmar:{" "}
              {quality.missingFields.map((f) => fields[f] || f).join(", ")}.
            </p>
          )}
          {quality.warnings.map((w) => (
            <p key={w}>{warnings[w] || w}</p>
          ))}
          <dl>
            {Object.entries(quality.fieldSources).map(([field, source]) => (
              <div key={field}>
                <dt>{fields[field] || field}</dt>
                <dd>{sources[source] || source}</dd>
              </div>
            ))}
          </dl>
          <p>
            Extraído el{" "}
            <time dateTime={quality.extractedAt}>
              {new Date(quality.extractedAt).toLocaleString("es-PE")}
            </time>
            . El precio y el stock pueden cambiar; confirma la información en la
            tienda.
          </p>
        </details>
      )}
    </article>
  );
}
