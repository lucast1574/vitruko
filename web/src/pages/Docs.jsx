import React from "react";
import Inspector from "./Inspector";
export default function Docs({
  heading = "Documentación de la API de Vitruko",
}) {
  return (
    <div className="docs-page">
      <div className="page-title">
        <div>
          <h1>{heading}</h1>
          <p>Un enlace público. Una respuesta JSON lista para integrar.</p>
        </div>
        <a
          className="text-link"
          href="https://api.vitruko.lat/openapi.json"
          target="_blank"
          rel="noreferrer"
        >
          OpenAPI
        </a>
      </div>
      <section className="docs-auth">
        <h2>Extraer un producto</h2>
        <p>
          Base URL: <code>https://api.vitruko.lat</code>. Envía{" "}
          <code>POST /v1/extract</code>, un cuerpo JSON con <code>url</code> y
          tu clave en <code>X-API-Key</code>. También se admite{" "}
          <code>Authorization: Bearer TU_API_KEY</code>.
        </p>
        <pre>{`curl "https://api.vitruko.lat/v1/extract" \\
  -H "X-API-Key: TU_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"url":"https://tienda.com/products/articulo"}'`}</pre>
        <p>
          La URL debe ser HTTPS pública, sin credenciales ni puertos
          personalizados, de hasta 2.048 caracteres. Tu clave necesita el
          permiso <code>extract</code>. Nunca envíes la clave en la URL ni la
          publiques en un repositorio.
        </p>
      </section>
      <section>
        <h2>Contrato de respuesta</h2>
        <pre>{`{
  "title": "Cafetera",
  "imageUrl": "https://tienda.com/cafetera-frente.jpg",
  "description": "Descripción publicada por la tienda",
  "url": "https://tienda.com/products/cafetera",
  "images": ["https://tienda.com/cafetera-frente.jpg", "https://tienda.com/cafetera-lateral.jpg"],
  "brand": "Casa",
  "price": { "amount": "129.90", "currency": "PEN" },
  "availability": "in_stock",
  "quality": {
    "productDetected": true,
    "missingFields": [],
    "fieldSources": {
      "title": "json_ld", "description": "json_ld", "images": "json_ld",
      "brand": "json_ld", "price": "json_ld", "availability": "json_ld"
    },
    "warnings": [],
    "extractedAt": "2026-11-07T12:00:00Z"
  }
}`}</pre>
        <p>Este es un ejemplo de respuesta completa. Los campos originales <code>title</code>, <code>imageUrl</code>, <code>description</code> y <code>url</code> se mantienen. La descripción, imagen y marca pueden estar vacías. <code>images</code> contiene hasta 20 URLs distintas y su primera imagen coincide con <code>imageUrl</code>. Vitruko devuelve enlaces a las imágenes; no descarga ni almacena esos archivos.</p>
        <p><code>price</code> es <code>null</code> cuando no hay un importe y una moneda publicados que correspondan a una oferta identificable. <code>price.amount</code> es un string decimal en unidades principales: <code>"129.90"</code> PEN significa S/ 129.90, no centavos. No inferimos la moneda por el dominio o el símbolo, no sumamos envío ni convertimos divisas. Un rango de precios no se presenta como un precio único.</p>
        <p><code>availability</code> puede ser <code>in_stock</code>, <code>out_of_stock</code>, <code>preorder</code>, <code>backorder</code>, <code>discontinued</code> o <code>unknown</code>. Refleja lo que publicó la tienda al consultarla; no es una garantía de inventario actual.</p>
        <h3>Calidad y procedencia</h3>
        <p><code>quality.productDetected</code> indica si se identificaron datos de producto en la fuente. <code>missingFields</code> enumera campos ausentes o sin confirmar. <code>fieldSources</code> indica el origen de cada campo disponible: <code>json_ld</code>, <code>microdata</code>, <code>open_graph</code>, <code>html_meta</code>, <code>html_title</code>, <code>product_meta</code>, <code>shopify_json</code> o <code>vtex_json</code>. No es una puntuación de certeza ni una validación comercial.</p>
        <p><code>quality.extractedAt</code> usa UTC y se conserva al responder desde caché. Las advertencias posibles son <code>product_not_confirmed</code>, <code>multiple_products</code>, <code>multiple_offers</code>, <code>price_range</code> y <code>variant_not_confirmed</code>. Si hay varias ofertas o variantes, no elijas una por tu cuenta sin confirmar cuál corresponde al enlace.</p>
        <p><code>url</code> conserva el enlace normalizado: elimina seguimiento conocido y mantiene los identificadores de producto y variante. Una URL de variante se conserva, pero la extracción de su precio y stock depende de que la fuente permita identificar esa oferta. Trata los campos ausentes como datos incompletos y ofrece confirmación o edición en tu aplicación.</p>
      </section>
      <section>
        <h2>Prueba una petición real</h2>
        <p>
          Utiliza tu propia clave. Solo se consulta la tienda cuando pulsas
          «Extraer producto». Esta operación consume una petición de tu cuenta.
        </p>
        <Inspector embedded />
      </section>
      <section>
        <h2>Cuotas y caché</h2>
        <p>
          Las cabeceras <code>X-Quota-Limit</code>,{" "}
          <code>X-Quota-Remaining</code> y <code>X-Quota-Reset</code> indican tu
          cuota. Para Scale, las primeras dos indican <code>unlimited</code>. El
          período Free es el mes UTC; los planes pagados usan su período de
          facturación.
        </p>
        <p>
          <code>X-Cache: HIT</code> indica una respuesta en memoria de hasta
          cinco minutos; <code>MISS</code>, una consulta nueva. Ambas cuentan
          una petición. Los errores posteriores a admitir la clave también
          cuentan. Los rechazos de clave, permisos o cuota no se cobran. Respeta{" "}
          <code>Retry-After</code> cuando la API lo indique.
        </p>
      </section>
      <section>
        <h2>Errores que debes manejar</h2>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>HTTP</th>
                <th>Qué significa</th>
                <th>Qué hacer</th>
              </tr>
            </thead>
            <tbody>
              {[
                [
                  400,
                  "URL o JSON inválido",
                  "Revisa url y envía un solo objeto JSON.",
                ],
                [401, "Clave ausente o revocada", "Comprueba X-API-Key."],
                [
                  403,
                  "Permiso faltante o cuenta suspendida",
                  "Usa una clave con extract y una cuenta activa.",
                ],
                [405, "Método incorrecto", "Usa POST."],
                [
                  415,
                  "Formato incorrecto",
                  "Envía Content-Type: application/json.",
                ],
                [
                  422,
                  "Ficha no accesible",
                  "Conserva el enlace y ofrece edición manual.",
                ],
                [
                  429,
                  "Cuota, velocidad o límite temporal de la tienda",
                  "Respeta Retry-After. source_rate_limited indica un límite de la tienda, no de tu plan.",
                ],
                [
                  503,
                  "Servicio ocupado o renovación pendiente",
                  "Reintenta con espera; revisa tu facturación.",
                ],
              ].map(([code, meaning, action]) => (
                <tr key={code}>
                  <td>{code}</td>
                  <td>{meaning}</td>
                  <td>{action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <section>
        <h2>Alcance y uso de los datos</h2>
        <p>
          Vitruko lee Open Graph, JSON-LD y microdatos; complementa fichas
          compatibles con fuentes públicas de Shopify y VTEX. La consulta tiene
          un límite de tiempo de doce segundos. No ejecuta JavaScript, no usa
          sesiones de comprador ni supera CAPTCHA o bloqueos de tiendas. La
          disponibilidad puede variar.
        </p>
        <p>
          Los nombres, imágenes y descripciones pertenecen a sus titulares.
          Comprueba las condiciones de la fuente y tus permisos antes de
          republicar información. Las imágenes se devuelven como URL; mostrarlas
          puede conectar el navegador del visitante con la tienda. Revisa los
          datos antes de guardarlos como definitivos.
        </p>
      </section>
    </div>
  );
}
